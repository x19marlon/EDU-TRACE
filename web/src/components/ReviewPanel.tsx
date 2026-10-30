"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import {
  AuditEntry,
  discardReviewDraft,
  FeedbackOrigin,
  generateReviewDraft,
  getReview,
  getSubmissionAudit,
  RAELevel,
  Review,
  saveReview,
  SubmissionMeta,
} from "@/lib/api";
import { Spinner } from "@/components/ui";
import { formatDateTime, formatDeadline, formatDuration, formatFeedback, LEVEL_STYLE, performanceBand, startedDays } from "@/lib/format";

const GRADE_MAX = 5;

const LEVELS: RAELevel[] = ["bajo", "medio", "alto"];

const Style: Record<string, string> = {
  alta: "bg-mint-100 text-mint-700", media: "bg-amber-100 text-amber-800", baja: "bg-peach-100 text-peach-700",
};

const IDLE_MS = 5 * 60 * 1000;
const POLL_MS = 8000;

const ORIGIN_TEXT: Record<FeedbackOrigin, string> = {
  ia: "El estudiante verá el texto de la IA sin cambios, marcado como «propuesto por IA y aprobado por tu profesor».",
  ia_editada: "El estudiante verá que la IA propuso el texto y que tú lo editaste y aprobaste.",
  docente: "El estudiante verá que escribiste tú la retroalimentación.",
};

export const ACTION_LABEL: Record<string, string> = {
  propuesta_ia: "Propuesta de la IA",
  propuesta_descartada: "Propuesta descartada",
  borrador_guardado: "Borrador guardado",
  aprobada: "Aprobada y devuelta",
  devuelta_actualizada: "Devolución actualizada",
  anotaciones_enviadas: "Marcas enviadas",
  modo_ia: "Modo de IA",
  casos_reejecutados: "Casos re-ejecutados",
};

function useReviewTimer() {
  const total = useRef(0);
  const sent = useRef(0);
  const lastInput = useRef(Date.now());
  useEffect(() => {
    const touch = () => {
      lastInput.current = Date.now();
    };
    const events = ["keydown", "pointerdown", "pointermove", "wheel"] as const;
    events.forEach((e) => window.addEventListener(e, touch, { passive: true }));
    const id = window.setInterval(() => {
      if (document.visibilityState === "visible" && Date.now() - lastInput.current < IDLE_MS) total.current += 1000;
    }, 1000);
    return () => {
      events.forEach((e) => window.removeEventListener(e, touch));
      window.clearInterval(id);
    };
  }, []);
  return {
    take: () => total.current - sent.current,
    commit: (ms: number) => {
      sent.current += ms;
    },
  };
}

function minutes(ms: number): string {
  const m = Math.round(ms / 60000);
  return m < 1 ? "menos de 1 min" : `${m} min`;
}

function ReviewHistory({ submissionId, refreshKey }: { submissionId: string; refreshKey: number }) {
  const [open, setOpen] = useState(false);
  const [entries, setEntries] = useState<AuditEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    if (!open) return;
    getSubmissionAudit(submissionId)
      .then(setEntries)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar el historial"));
  }, [open, submissionId, refreshKey]);
  return (
    <details className="text-xs" onToggle={(e) => setOpen((e.target as HTMLDetailsElement).open)}>
      <summary className="cursor-pointer font-bold text-ink-soft">Historial de esta revisión (bitácora)</summary>
      {error && <p className="mt-2 text-peach-700">{error}</p>}
      {entries && entries.length === 0 && <p className="mt-2 text-ink-faint">Aún no hay intervenciones registradas.</p>}
      {entries && entries.length > 0 && (
        <ol className="mt-2 space-y-1.5 border-l-2 border-lavender-100 pl-3">
          {entries.map((e) => (
            <li key={e.id}>
              <span className="font-bold text-ink">{ACTION_LABEL[e.action] ?? e.action}</span>
              <span className="text-ink-faint">
                {" "}
                · {e.actor_name} · {formatDateTime(e.at)}
              </span>
              {e.detail && <p className="text-ink-soft">{e.detail}</p>}
            </li>
          ))}
        </ol>
      )}
    </details>
  );
}

function parseGrade(text: string): number | null | "invalid" {
  const t = text.trim().replace(",", ".");
  if (t === "") return null;
  const n = Number(t);
  if (!Number.isFinite(n) || n < 0 || n > GRADE_MAX) return "invalid";
  return Math.round(n * 10) / 10;
}


function LateNotice({
  lateSeconds,
  dueAt,
  penalty,
  onApply,
}: {
  lateSeconds: number;
  dueAt?: string;
  penalty?: number;
  onApply: (total: number) => boolean;
}) {
  const [applied, setApplied] = useState(false);
  const days = startedDays(lateSeconds);
  const total = penalty ? Math.round(penalty * days * 10) / 10 : 0;
  return (
    <div className="space-y-1.5 rounded-2xl bg-peach-50 p-3 text-sm">
      <p className="font-bold text-peach-700">
        Entrega tardía: {formatDuration(lateSeconds)} después de la fecha límite ({days} {days === 1 ? "día iniciado" : "días iniciados"}).
      </p>
      {dueAt && <p className="text-xs text-ink-soft">La fecha límite era el {formatDeadline(dueAt)}.</p>}
      {total > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-ink-soft">
            Penalización sugerida por el taller: −{penalty!.toFixed(1)} × {days} = <span className="font-bold text-ink">−{total.toFixed(1)}</span>.
            Tú decides si aplicarla.
          </span>
          <button
            type="button"
            disabled={applied}
            onClick={() => setApplied(onApply(total))}
            className="rounded-full bg-white px-3 py-1 text-xs font-bold text-peach-700 ring-1 ring-peach-200 transition hover:bg-peach-100 disabled:opacity-60"
          >
            {applied ? `Penalización aplicada (−${total.toFixed(1)})` : `Restar ${total.toFixed(1)} a la nota`}
          </button>
        </div>
      )}
    </div>
  );
}

export default function ReviewPanel({
  submissionId,
  onSaved,
  meta,
}: {
  submissionId: string;
  onSaved?: () => void;
  meta?: SubmissionMeta;
}) {
  const [review, setReview] = useState<Review | null>(null);
  const [feedback, setFeedback] = useState("");
  const [grade, setGrade] = useState("");
  const [levels, setLevels] = useState<Record<string, RAELevel | "">>({});
  const [busy, setBusy] = useState<"draft" | "save" | "discard" | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [historyKey, setHistoryKey] = useState(0);
  const timer = useReviewTimer();
  const loaded = useRef<{ feedback: string; grade: string }>({ feedback: "", grade: "" });
  // consistencia de datos
  const apply = useCallback((r: Review) => {
    const g = r.grade === undefined ? "" : r.grade.toFixed(1);
    loaded.current = { feedback: r.feedback, grade: g };
    setReview(r);
    setFeedback(r.feedback);
    setGrade(g);
    setLevels(Object.fromEntries((r.raes ?? []).map((x) => [x.rae_id, x.level ?? ""])));
    setHistoryKey((k) => k + 1);
  }, []);
  useEffect(() => {
    setReview(null);
    setError(null);
    getReview(submissionId)
      .then(apply)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar la revisión"));
  }, [submissionId, apply]);

  const feedbackRef = useRef(feedback);
  const gradeRef = useRef(grade);
  feedbackRef.current = feedback;
  gradeRef.current = grade;

  const waiting = !!review && !review.ai_at && (review.ai_job?.status === "pendiente" || review.ai_job?.status === "procesando");
  useEffect(() => {
    if (!waiting) return;
    const id = window.setInterval(async () => {
      try {
        const r = await getReview(submissionId);
        if (!r.ai_at) {
          setReview(r);
          return;
        }
        const dirty = feedbackRef.current !== loaded.current.feedback || gradeRef.current !== loaded.current.grade;
        if (dirty) setReview(r);
        else apply(r);
        onSaved?.();
      } catch {
        /* se reintenta en el siguiente ciclo */
      }
    }, POLL_MS);
    return () => window.clearInterval(id);
  }, [waiting, submissionId, apply, onSaved]);
  const draft = async () => {
    setBusy("draft");
    setError(null);
    try {
      apply(await generateReviewDraft(submissionId));
      onSaved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo generar la propuesta");
    } finally {
      setBusy(null);
    }
  };

  const discard = async () => {
    if (!window.confirm("¿Descartar la propuesta de la IA? Se quitará de tu formulario si no la modificaste y quedará registrado en la bitácora.")) return;
    setBusy("discard");
    setError(null);
    try {
      const elapsed = timer.take();
      apply(await discardReviewDraft(submissionId, elapsed));
      timer.commit(elapsed);
      onSaved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo descartar la propuesta");
    } finally {
      setBusy(null);
    }
  };

  const save = async (returnIt: boolean, useAIGrade = false) => {
    const g = useAIGrade ? review?.ai_grade ?? null : parseGrade(grade);
    if (g === "invalid") {
      setError(`La nota debe ser un número entre 0.0 y ${GRADE_MAX.toFixed(1)}`);
      return;
    }
    if (returnIt && !feedback.trim()) {
      setError("Escribe la retroalimentación antes de devolverla");
      return;
    }
    setBusy("save");
    setError(null);
    try {
      const elapsed = timer.take();
      apply(await saveReview(submissionId, { feedback, grade: g, return: returnIt, rae_levels: levels, elapsed_ms: elapsed }));
      timer.commit(elapsed);
      onSaved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo guardar la revisión");
    } finally {
      setBusy(null);
    }
  };

  if (!review) {
    return error ? (
      <p className="rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>
    ) : (
      <p className="flex items-center gap-2 text-sm text-ink-soft">
        <Spinner className="h-4 w-4 border-lavender-500" /> Cargando revisión...
      </p>
    );
  }

  const hasAI = !!review.ai_at;
  const returned = review.status === "returned";
  const origin: FeedbackOrigin =
    hasAI && review.ai_feedback ? (feedback.trim() === review.ai_feedback.trim() ? "ia" : "ia_editada") : "docente";
  const job = review.ai_job;

  return (
    <div className="space-y-4 rounded-3xl border border-lavender-200 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h4 className="flex items-center gap-2 font-bold">
          Tu revisión
          <span
            className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${
              returned ? "bg-mint-100 text-mint-700" : "bg-amber-50 text-amber-800"
            }`}
          >
            {returned ? "Aprobada y devuelta" : "Pendiente de revisión"}
          </span>
        </h4>
        <button
          onClick={draft}
          disabled={busy !== null}
          className="flex items-center gap-2 rounded-full bg-lavender-100 px-4 py-1.5 text-sm font-bold text-lavender-700
            transition hover:bg-lavender-200 disabled:opacity-50"
        >
          {busy === "draft" && <Spinner className="h-4 w-4 border-lavender-700" />}
          {busy === "draft" ? "Generando (≈ 1 min)..." : hasAI ? "Volver a proponer con IA" : "Generar propuesta con IA"}
        </button>
      </div>

      {waiting && (
        <p className="flex items-center gap-2 rounded-2xl bg-lavender-50 px-4 py-2.5 text-sm font-semibold text-lavender-700">
          <Spinner className="h-4 w-4 border-lavender-700" />
          {job?.status === "procesando"
            ? "La IA está preparando la propuesta de esta entrega. Aparecerá aquí sola."
            : `La propuesta de la IA está en cola${job && job.ahead > 0 ? ` (${job.ahead} antes)` : ""}. Puedes empezar a revisar mientras tanto.`}
        </p>
      )}
      {!hasAI && job?.status === "error" && (
        <p className="rounded-2xl bg-peach-50 px-4 py-2.5 text-sm text-peach-700">
          <span className="font-bold">La propuesta automática falló ({job.attempts} {job.attempts === 1 ? "intento" : "intentos"}):</span> {job.error}.
          Puedes generarla manualmente o revisar sin IA; la entrega del estudiante está intacta.
        </p>
      )}
      {!hasAI && review.ai_discarded_at && (
        <p className="rounded-2xl bg-lavender-50/70 px-4 py-2 text-xs text-ink-soft">
          Descartaste la propuesta de la IA el {formatDateTime(review.ai_discarded_at)}.
        </p>
      )}

      {hasAI && (
        <div className="rounded-2xl bg-lavender-50/70 p-4">
          <p className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">
            Propuesta IA solo accesible por ti
          </p>
          <div className="flex flex-wrap items-center gap-3">
            <span className="text-3xl font-extrabold tabular-nums text-lavender-700">
              {review.ai_grade === undefined ? "—" : review.ai_grade.toFixed(1)}
              <span className="text-base font-bold text-ink-faint"> / {GRADE_MAX.toFixed(1)}</span>
            </span>
            {review.ai_grade !== undefined && (
              <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${performanceBand(review.ai_grade).tone}`}>
                {performanceBand(review.ai_grade).label} {performanceBand(review.ai_grade).range}
              </span>
            )}
            {review.ai_confidence && (
              <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${Style[review.ai_confidence]}`}>
                Confianza {review.ai_confidence}
              </span>
            )}
          </div>
          {review.ai_reason && <p className="mt-2 text-sm text-ink-soft">{review.ai_reason}</p>}
          {review.ai_grade === undefined && (
            <p className="mt-2 text-xs text-peach-700">Error del modelo al calcular la nota tentativa.</p>
          )}
          {!returned && (
            <button
              onClick={discard}
              disabled={busy !== null}
              className="mt-3 rounded-full bg-white px-3 py-1 text-xs font-bold text-peach-700 ring-1 ring-peach-200 transition hover:bg-peach-50 disabled:opacity-50"
            >
              {busy === "discard" ? "Descartando..." : "Descartar propuesta de la IA"}
            </button>
          )}
        </div>
      )}

      
      <div className="space-y-2">
        <label className="block text-xs font-bold uppercase tracking-wider text-ink-soft" htmlFor={`fb-${submissionId}`}>
          Retroalimentación para el estudiante
        </label>
        <textarea
          id={`fb-${submissionId}`}
          value={feedback}
          onChange={(e) => setFeedback(e.target.value)}
          maxLength={20000}
          placeholder="Escribe tu retroalimentación o genera una propuesta con IA y edítala."
          className="h-56 w-full resize-y rounded-2xl border border-lavender-100 bg-white p-3 text-sm
            focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100"
        />
        {feedback.trim() && (
          <details className="text-xs text-ink-soft">
            <summary className="cursor-pointer font-semibold">Vista previa</summary>
            <div
              className="mt-2 rounded-2xl bg-mint-50 p-3 text-sm text-ink"
              dangerouslySetInnerHTML={{ __html: formatFeedback(feedback) }}
            />
          </details>
        )}
        <label className="flex items-center gap-2 text-sm font-semibold text-ink-soft">
          Nota / calificación tentativa
          <input
            value={grade}
            onChange={(e) => setGrade(e.target.value)}
            inputMode="decimal"
            placeholder={`0.0 – ${GRADE_MAX.toFixed(1)}`}
            className="w-24 rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-center text-sm font-bold tabular-nums
              focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
          <span className="text-xs font-normal text-ink-faint">(opcional)</span>
          {(() => {
            const g = parseGrade(grade);
            if (g === null || g === "invalid") return null;
            const band = performanceBand(g);
            return (
              <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${band.tone}`}>
                {band.label} {band.range}
              </span>
            );
          })()}
        </label>
        {meta?.late_seconds ? (
          <LateNotice
            lateSeconds={meta.late_seconds}
            dueAt={meta.due_at}
            penalty={meta.late_penalty}
            onApply={(total) => {
              const g = parseGrade(grade);
              if (g === null || g === "invalid") {
                setError("Escribe primero una nota válida para restarle la penalización");
                return false;
              }
              setError(null);
              setGrade(Math.max(0, Math.round((g - total) * 10) / 10).toFixed(1));
              return true;
            }}
          />
        ) : null}
      </div>

      {(review.raes ?? []).length > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-bold uppercase tracking-wider text-ink-soft">Nivel de logro por RAE</p>
          <ul className="space-y-1.5">
            {review.raes.map((r) => (
              <li key={r.rae_id} className="flex flex-wrap items-center gap-x-3 gap-y-1.5 rounded-2xl bg-lavender-50/60 px-3 py-2">
                <span className="min-w-0 flex-1 text-sm">
                  <span className="font-bold text-lavender-700">{r.code}</span>{" "}
                  <span className="text-ink-soft">{r.description}</span>
                </span>
                {r.ai_level && <span className="text-[11px] font-semibold text-ink-faint">IA propone: {r.ai_level}</span>}
                <div role="radiogroup" aria-label={`Nivel de ${r.code}`} className="flex gap-1 rounded-full bg-white p-0.5 ring-1 ring-lavender-100">
                  {LEVELS.map((l) => (
                    <button
                      key={l}
                      type="button"
                      role="radio"
                      aria-checked={levels[r.rae_id] === l}
                      onClick={() => setLevels((prev) => ({ ...prev, [r.rae_id]: prev[r.rae_id] === l ? "" : l }))}
                      className={`rounded-full px-3 py-1 text-xs font-bold capitalize transition ${
                        levels[r.rae_id] === l ? LEVEL_STYLE[l] : "text-ink-soft hover:bg-lavender-50"
                      }`}
                    >
                      {l}
                    </button>
                  ))}
                </div>
              </li>
            ))}
          </ul>
        </div>
      )}

      {error && (
        <div className="rounded-2xl bg-peach-50 px-3 py-2 text-xs text-peach-700 font-medium">
          <span className="font-bold block">Aviso del sistema (RNF-15): {error}</span>
          <span className="text-[11px] text-peach-600 block mt-0.5">
            La entrega del estudiante se conserva intacta en el servidor (RNF-14). Puedes completar la evaluación manualmente.
          </span>
        </div>
      )}

      <div className="flex flex-wrap gap-2">
        {review.ai_grade !== undefined && (
          <button
            onClick={() => save(true, true)}
            disabled={busy !== null}
            className="rounded-full bg-mint-600 px-4 py-2 text-sm font-bold text-white transition hover:bg-mint-700 disabled:opacity-50"
          >
            Confirmar nota de la IA ({review.ai_grade.toFixed(1)}) y devolver asignación a estudiante
          </button>
        )}
        <button
          onClick={() => save(true)}
          disabled={busy !== null}
          className="rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white transition hover:bg-lavender-700 disabled:opacity-50"
        >
          {returned ? "Actualizar lo devuelto" : "Aprobar y devolver con mi nota"}
        </button>
        <button
          onClick={() => save(false)}
          disabled={busy !== null}
          className="rounded-full bg-white px-4 py-2 text-sm font-bold text-ink-soft ring-1 ring-lavender-200 transition
            hover:bg-lavender-50 disabled:opacity-50"
        >
          Guardar sin devolver a estudiante
        </button>
      </div>

      {feedback.trim() && <p className="text-xs text-ink-soft">{ORIGIN_TEXT[origin]}</p>}

      <p className="text-xs text-ink-faint">
        {returned && review.returned_at
          ? `Aprobada y devuelta al estudiante el ${formatDateTime(review.returned_at)}. `
          : "Aún no la ve el estudiante. "}
        {review.review_ms > 0 && `Tiempo de revisión registrado: ${minutes(review.review_ms)}. `}
        {review.grade_source === "ai_confirmed" && "Nota de la IA confirmada por ti."}
        {review.grade_source === "profesor" &&
          review.ai_grade !== undefined &&
          `Nota modificada por ti (la IA proponía ${review.ai_grade.toFixed(1)}).`}
      </p>

      <ReviewHistory submissionId={submissionId} refreshKey={historyKey} />
    </div>
  );
}
