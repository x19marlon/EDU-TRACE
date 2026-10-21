"use client";

import { useCallback, useEffect, useState } from "react";
import { generateReviewDraft, getReview, Review, saveReview } from "@/lib/api";
import { Spinner } from "@/components/ui";
import { formatDateTime, formatFeedback } from "@/lib/format";

const GRADE_MAX = 5;

const Style: Record<string, string> = {
  alta: "bg-mint-100 text-mint-700", media: "bg-amber-100 text-amber-800", baja: "bg-peach-100 text-peach-700",
};

function parseGrade(text: string): number | null | "invalid" {
  const t = text.trim().replace(",", ".");
  if (t === "") return null;
  const n = Number(t);
  if (!Number.isFinite(n) || n < 0 || n > GRADE_MAX) return "invalid";
  return Math.round(n * 10) / 10;
}


export default function ReviewPanel({ submissionId, onSaved }: { submissionId: string; onSaved?: () => void }) {
  const [review, setReview] = useState<Review | null>(null);
  const [feedback, setFeedback] = useState("");
  const [grade, setGrade] = useState("");
  const [busy, setBusy] = useState<"draft" | "save" | null>(null);
  const [error, setError] = useState<string | null>(null);
  // consistencia de datos
  const apply = useCallback((r: Review) => {
    setReview(r);
    setFeedback(r.feedback);
    setGrade(r.grade === undefined ? "" : r.grade.toFixed(1)); }, []);
  useEffect(() => {
    setReview(null);
    setError(null);
    getReview(submissionId)
      .then(apply)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar la revisión"));
  }, [submissionId, apply]);

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
      apply(await saveReview(submissionId, { feedback, grade: g, return: returnIt }));
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

  return (
    <div className="space-y-4 rounded-3xl border border-lavender-200 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h4 className="font-bold">Tu revisión</h4>
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
        </label>
      </div>

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
          {returned ? "Actualizar lo devuelto" : "Devolver con mi nota"}
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

      <p className="text-xs text-ink-faint">
        {returned && review.returned_at
          ? `Devuelta al estudiante el ${formatDateTime(review.returned_at)}. `
          : "Aún no la ve el estudiante. "}
        {review.grade_source === "ai_confirmed" && "Nota de la IA confirmada por ti."}
        {review.grade_source === "profesor" &&
          review.ai_grade !== undefined &&
          `Nota modificada por ti (la IA proponía ${review.ai_grade.toFixed(1)}).`}
      </p>
    </div>
  );
}
