"use client";

import { useCallback, useEffect, useState } from "react";
import {
  AIMode,
  AIStatus,
  AuditEntry,
  getAIStatus,
  getGroupAudit,
  getGroupMetrics,
  GroupMetrics,
  RatingCount,
  retryDrafts,
  setGroupAIMode,
  setThresholds,
  Stat,
  Thresholds,
} from "@/lib/api";
import { Card, SmallButton, Spinner } from "@/components/ui";
import { ACTION_LABEL } from "@/components/ReviewPanel";
import { formatDateTime, formatShortDuration } from "@/lib/format";

function fmtMs(ms: number): string {
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60000) return `${Math.round(ms / 1000)} s`;
  return formatShortDuration(ms / 1000);
}

type Verdict = "ok" | "over" | "none" | "empty";

const VERDICT: Record<Verdict, [string, string]> = {
  ok: ["Cumple", "bg-mint-100 text-mint-700"],
  over: ["No cumple", "bg-peach-100 text-peach-700"],
  none: ["Sin umbral", "bg-lavender-50 text-ink-soft"],
  empty: ["Sin datos", "bg-lavender-50 text-ink-soft"],
};

function verdict(hasData: boolean, limit: number | null | undefined, over: boolean): Verdict {
  if (!hasData) return "empty";
  if (!limit) return "none";
  return over ? "over" : "ok";
}

function Indicator({ id, title, value, detail, state }: { id: string; title: string; value: string; detail: string; state: Verdict }) {
  const [label, tone] = VERDICT[state];
  return (
    <div className="rounded-2xl bg-lavender-50/60 p-3">
      <div className="flex items-start justify-between gap-2">
        <p className="text-xs font-bold text-ink-soft">
          <span className="text-lavender-700">{id}</span> · {title}
        </p>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${tone}`}>{label}</span>
      </div>
      <p className="mt-1 text-xl font-extrabold tabular-nums">{value}</p>
      <p className="text-[11px] text-ink-soft">{detail}</p>
    </div>
  );
}

function statLine(s: Stat): string {
  return s.count === 0 ? "—" : `${fmtMs(s.avg_ms)} promedio`;
}

function statDetail(s: Stat, unit: string): string {
  if (s.count === 0) return `Aún no hay ${unit}.`;
  return `${s.count} ${unit} · mediana ${fmtMs(s.p50_ms)} · 90 % en menos de ${fmtMs(s.p90_ms)} · máximo ${fmtMs(s.max_ms)}${
    s.over ? ` · ${s.over} sobre el umbral` : ""
  }`;
}

function ThresholdsForm({ courseId, initial, onSaved }: { courseId: string; initial: Thresholds; onSaved: () => void }) {
  const toText = (v?: number | null) => (v ? String(v) : "");
  const [feedback, setFeedback] = useState(toText(initial.feedback_hours));
  const [batch, setBatch] = useState(toText(initial.batch_hours));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const parse = (t: string) => (t.trim() ? Number(t.replace(",", ".")) : null);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await setThresholds(courseId, { review_minutes: initial.review_minutes ?? null, feedback_hours: parse(feedback), batch_hours: parse(batch) });
      onSaved();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudieron guardar los umbrales");
    } finally {
      setBusy(false);
    }
  };

  const field = (label: string, unit: string, value: string, set: (v: string) => void) => (
    <label className="flex items-center gap-2 text-xs font-semibold text-ink-soft">
      {label}
      <input
        value={value}
        onChange={(e) => set(e.target.value)}
        inputMode="decimal"
        placeholder="—"
        className="w-16 rounded-full border border-lavender-100 bg-white px-2 py-1 text-center text-sm font-bold text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
      />
      {unit}
    </label>
  );

  return (
    <form onSubmit={save} className="space-y-2 rounded-2xl border border-lavender-100 p-3">
      <p className="text-xs text-ink-soft">
        Umbrales acordados para la materia; aplican a todas sus clases. Quedaron acordadas 5 h para procesar cada lote y 5 h entre la
        entrega y la retroalimentación. El tiempo de revisión no lleva umbral: lo decides tú con la estimación de la IA.
      </p>
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        {field("Entrega → retroalimentación ≤", "h", feedback, setFeedback)}
        {field("Procesar cada lote ≤", "h", batch, setBatch)}
        <button
          type="submit"
          disabled={busy}
          className="rounded-full bg-lavender-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-lavender-700 disabled:opacity-50"
        >
          {busy ? "Guardando..." : "Guardar umbrales"}
        </button>
      </div>
      {error && <p className="text-xs font-semibold text-peach-700">{error}</p>}
    </form>
  );
}

function AIModeSwitch({ groupId, mode, onChange }: { groupId: string; mode: AIMode; onChange: (m: AIMode) => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const choose = async (next: AIMode) => {
    if (next === mode) return;
    if (
      next === "inmediata" &&
      !window.confirm(
        "En modo inmediato los estudiantes reciben la retroalimentación y el chat de la IA sin tu aprobación. Esto no cumple RNF-06 (validación docente). ¿Continuar?"
      )
    )
      return;
    setBusy(true);
    setError(null);
    try {
      onChange((await setGroupAIMode(groupId, next)).ai_mode);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo cambiar el modo");
    } finally {
      setBusy(false);
    }
  };
  const option = (value: AIMode, title: string, text: string) => (
    <button
      type="button"
      role="radio"
      aria-checked={mode === value}
      disabled={busy}
      onClick={() => choose(value)}
      className={`flex-1 rounded-2xl p-3 text-left transition ${
        mode === value ? "bg-lavender-600 text-white shadow-soft" : "bg-lavender-50 text-ink hover:bg-lavender-100"
      }`}
    >
      <span className="block text-sm font-bold">{title}</span>
      <span className={`block text-xs ${mode === value ? "text-lavender-100" : "text-ink-soft"}`}>{text}</span>
    </button>
  );
  return (
    <div className="space-y-2">
      <div role="radiogroup" aria-label="Retroalimentación de la IA para el estudiante" className="flex flex-col gap-2 sm:flex-row">
        {option(
          "supervisada",
          "Supervisada (recomendada)",
          "La IA prepara un borrador de cada entrega y el estudiante solo ve lo que tú apruebas. Cumple RNF-06."
        )}
        {option(
          "inmediata",
          "Inmediata",
          "El estudiante pide retroalimentación y chat a la IA mientras practica, sin tu aprobación previa."
        )}
      </div>
      {error && <p className="text-xs font-semibold text-peach-700">{error}</p>}
    </div>
  );
}

function share(c: RatingCount): string {
  const total = c.si + c.parcial + c.no;
  return total === 0 ? "sin respuestas" : `${Math.round((c.si / total) * 100)} % entendió (${c.si} sí · ${c.parcial} en parte · ${c.no} no)`;
}

const COMPREHENSION_VERDICT: Record<GroupMetrics["comprehension"]["veredicto"], [string, string]> = {
  cumple: ["Cumple la meta", "bg-mint-100 text-mint-700"],
  no_cumple: ["No cumple la meta", "bg-peach-100 text-peach-700"],
  pocas_respuestas: ["Pocas respuestas", "bg-lavender-50 text-ink-soft"],
};

function pct(n: number, total: number): number {
  return total ? Math.round((n / total) * 100) : 0;
}

function Comprehension({ data }: { data: GroupMetrics["comprehension"] }) {
  const goal = data.meta;
  const [label, tone] = COMPREHENSION_VERDICT[data.veredicto];
  return (
    <div className="space-y-2 rounded-2xl bg-lavender-50/60 p-3 text-xs">
      <div className="flex items-start justify-between gap-2">
        <p className="font-bold text-ink-soft">
          <span className="text-lavender-700">RNF-09</span> · ¿Los estudiantes entienden la retroalimentación?
        </p>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${tone}`}>{label}</span>
      </div>
      <p className="text-ink-soft">
        Meta por taller: al menos {Math.round(goal.min_si * 100)} % de «sí» y como máximo {Math.round(goal.max_no * 100)} % de «no», con{" "}
        {goal.min_respuestas} respuestas o más.
      </p>
      {data.por_taller.length === 0 ? (
        <p className="text-ink-soft">
          Aún nadie respondió. Cada retroalimentación le pregunta al estudiante si la entendió; aquí verás el resumen sin nombres.
        </p>
      ) : (
        <ul className="space-y-1.5">
          {data.por_taller.map((t) => (
            <li key={t.assignment_id || "libre"} className="rounded-xl bg-white px-2.5 py-2">
              <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="min-w-0 flex-1 truncate font-bold">{t.title}</span>
                <span className="text-ink-soft">
                  {pct(t.si, t.total)} % sí · {pct(t.parcial, t.total)} % en parte · {pct(t.no, t.total)} % no
                </span>
                <span
                  className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                    !t.suficientes ? "bg-lavender-50 text-ink-soft" : t.cumple ? "bg-mint-100 text-mint-700" : "bg-peach-100 text-peach-700"
                  }`}
                >
                  {!t.suficientes ? `${t.total} de ${goal.min_respuestas} respuestas` : t.cumple ? "Cumple" : "No cumple"}
                </span>
              </div>
              <div className="mt-1.5 flex h-1.5 overflow-hidden rounded-full bg-lavender-100" aria-hidden>
                <div className="bg-mint-400" style={{ width: `${pct(t.si, t.total)}%` }} />
                <div className="bg-amber-300" style={{ width: `${pct(t.parcial, t.total)}%` }} />
                <div className="bg-peach-400" style={{ width: `${pct(t.no, t.total)}%` }} />
              </div>
            </li>
          ))}
        </ul>
      )}
      {data.por_taller.length > 0 && (
        <p className="text-ink-faint">
          Revisiones devueltas: {share(data.revisiones)} · IA inmediata: {share(data.inmediata)}
        </p>
      )}
      {data.comentarios.length > 0 && (
        <ul className="space-y-1">
          {data.comentarios.map((c, i) => (
            <li key={i} className="rounded-xl bg-white px-2.5 py-1.5">
              <span className="font-bold">{c.understood === "no" ? "No entendió" : c.understood === "parcial" ? "En parte" : "Entendió"}:</span>{" "}
              {c.comment} <span className="text-ink-faint">· {formatDateTime(c.at)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

export default function ClassOversight({
  groupId,
  mode,
  onModeChange,
  refreshKey,
}: {
  groupId: string;
  mode: AIMode;
  onModeChange: (m: AIMode) => void;
  refreshKey: number;
}) {
  const [metrics, setMetrics] = useState<GroupMetrics | null>(null);
  const [status, setStatus] = useState<AIStatus | null>(null);
  const [audit, setAudit] = useState<AuditEntry[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState(false);
  const [retrying, setRetrying] = useState(false);

  const load = useCallback(async () => {
    try {
      const [m, s, a] = await Promise.all([getGroupMetrics(groupId), getAIStatus(), getGroupAudit(groupId)]);
      setMetrics(m);
      setStatus(s);
      setAudit(a);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudieron cargar los indicadores");
    }
  }, [groupId]);

  useEffect(() => {
    load();
  }, [load, refreshKey, mode]);

  useEffect(() => {
    if (!metrics || metrics.queue.pending + metrics.queue.running === 0) return;
    const id = window.setInterval(load, 15000);
    return () => window.clearInterval(id);
  }, [metrics, load]);

  const retry = async () => {
    setRetrying(true);
    try {
      await retryDrafts(groupId);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo reintentar");
    } finally {
      setRetrying(false);
    }
  };

  const failedHere = status?.failed.find((f) => f.group_id === groupId);
  const t = metrics?.thresholds;

  return (
    <div className="grid grid-cols-1 gap-5 lg:grid-cols-5">
      <Card title="Retroalimentación con IA" className="rise-in lg:col-span-2 self-start">
        <div className="space-y-4">
          <AIModeSwitch groupId={groupId} mode={mode} onChange={onModeChange} />
          {status && !status.available && (
            <p role="alert" className="rounded-2xl bg-peach-50 px-4 py-2.5 text-sm text-peach-700">
              <span className="font-bold">La IA no está disponible ahora.</span> Las entregas se guardan intactas y sus propuestas esperan en
              la cola; también puedes revisar sin IA.
            </p>
          )}
          {failedHere && (
            <div role="alert" className="space-y-2 rounded-2xl bg-peach-50 px-4 py-3 text-sm text-peach-700">
              <p>
                <span className="font-bold">
                  La IA no pudo preparar {failedHere.failed} {failedHere.failed === 1 ? "propuesta" : "propuestas"} de esta clase.
                </span>{" "}
                Último error: {failedHere.last_error}
              </p>
              <SmallButton onClick={retry}>{retrying ? "Reintentando..." : "Reintentar ahora"}</SmallButton>
            </div>
          )}
          {metrics && (
            <div className="grid grid-cols-4 gap-2 text-center">
              {[
                { v: metrics.queue.pending, l: "en cola" },
                { v: metrics.queue.running, l: "procesando" },
                { v: metrics.queue.done, l: "listas" },
                { v: metrics.queue.failed, l: "con error" },
              ].map((x) => (
                <div key={x.l} className="rounded-2xl bg-lavender-50 py-2">
                  <p className="text-lg font-extrabold tabular-nums">{x.v}</p>
                  <p className="text-[10px] font-semibold text-ink-soft">{x.l}</p>
                </div>
              ))}
            </div>
          )}
          <p className="text-xs text-ink-faint">
            Cada entrega oficial entra sola a la cola (RF-01, RF-16): la IA ejecuta los casos de prueba y prepara una propuesta que
            revisas, editas, apruebas o descartas.
          </p>
        </div>
      </Card>

      <Card
        title="Indicadores de la clase"
        className="rise-in lg:col-span-3"
        action={<SmallButton onClick={() => setEditing((e) => !e)}>{editing ? "Cerrar" : "Umbrales"}</SmallButton>}
      >
        {error && <p className="mb-3 rounded-2xl bg-peach-50 px-4 py-2 text-sm text-peach-700">{error}</p>}
        {!metrics ? (
          <p className="flex items-center gap-2 text-sm text-ink-soft">
            <Spinner className="h-4 w-4 border-lavender-500" /> Calculando...
          </p>
        ) : (
          <div className="space-y-3">
            {editing && (
              <ThresholdsForm
                courseId={metrics.course_id}
                initial={metrics.thresholds}
                onSaved={() => {
                  setEditing(false);
                  load();
                }}
              />
            )}
            <div className="grid gap-2 sm:grid-cols-2">
              <Indicator
                id="RNF-08"
                title="Tiempo de revisión por entrega (frente a lo que estimó la IA)"
                value={statLine(metrics.review_time)}
                detail={`${statDetail(metrics.review_time, "revisiones medidas")}${
                  metrics.review_estimate.count
                    ? ` · la IA estimó una mediana de ${fmtMs(metrics.review_estimate.p50_ms)}; ${metrics.over_estimate} de ${metrics.review_estimate.count} tomaron más`
                    : ""
                }`}
                state={verdict(
                  metrics.review_time.count > 0,
                  metrics.review_estimate.count > 0 ? metrics.review_estimate.p50_ms : undefined,
                  metrics.review_time.p50_ms > metrics.review_estimate.p50_ms
                )}
              />
              <Indicator
                id="RNF-11"
                title={`Entrega → retroalimentación${t?.feedback_hours ? ` (≤ ${t.feedback_hours} h)` : ""}`}
                value={statLine(metrics.turnaround)}
                detail={`${statDetail(metrics.turnaround, "devueltas")}${
                  metrics.pending_reviews
                    ? ` · ${metrics.pending_reviews} por devolver (la más antigua hace ${fmtMs(metrics.oldest_pending_ms)}${
                        metrics.pending_over ? `, ${metrics.pending_over} ya pasaron el umbral` : ""
                      })`
                    : ""
                }`}
                state={verdict(
                  metrics.turnaround.count + metrics.pending_reviews > 0,
                  t?.feedback_hours,
                  metrics.turnaround.over + metrics.pending_over > 0
                )}
              />
            </div>
            <div className="rounded-2xl bg-lavender-50/60 p-3">
              <div className="flex items-start justify-between gap-2">
                <p className="text-xs font-bold text-ink-soft">
                  <span className="text-lavender-700">RNF-10</span> · Procesamiento de cada lote
                  {t?.batch_hours ? ` (≤ ${t.batch_hours} h por entrega)` : ""}
                </p>
                <span className="text-[11px] text-ink-faint">
                  IA: espera {statLine(metrics.queue.wait)} · propuesta {statLine(metrics.queue.run)}
                </span>
              </div>
              {metrics.batches.length === 0 ? (
                <p className="mt-1 text-[11px] text-ink-soft">Aún no hay entregas oficiales en la cola.</p>
              ) : (
                <ul className="mt-2 space-y-1">
                  {metrics.batches.map((b) => {
                    const skipped = b.done + b.failed + b.waiting === 0;
                    const [label, tone] = skipped
                      ? ["Revisado sin IA", "bg-lavender-50 text-ink-soft"]
                      : b.finished
                        ? VERDICT[verdict(true, t?.batch_hours, b.over)]
                        : ["En curso", "bg-lavender-100 text-lavender-700"];
                    return (
                      <li key={b.assignment_id || "libre"} className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs">
                        <span className="min-w-0 flex-1 truncate font-bold">{b.title}</span>
                        <span className="text-ink-soft">
                          {b.done}/{b.submissions} listas{b.failed ? ` · ${b.failed} con error` : ""}
                          {b.waiting ? ` · ${b.waiting} en cola` : ""}
                          {skipped ? "" : ` · peor demora ${fmtMs(b.worst_ms)}`}
                        </span>
                        <span className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${tone}`}>{label}</span>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
            <div className="grid gap-2 sm:grid-cols-2">
              <div className="rounded-2xl bg-lavender-50/60 p-3 text-xs">
                <p className="font-bold text-ink-soft">
                  <span className="text-lavender-700">RF-10 a RF-13</span> · Decisiones sobre la IA
                </p>
                <p className="mt-1 text-ink-soft">
                  {metrics.decisions.ia} aprobadas sin cambios · {metrics.decisions.ia_editada} editadas · {metrics.decisions.docente}{" "}
                  escritas por ti · {metrics.decisions.descartadas} descartadas
                </p>
              </div>
              <div className="rounded-2xl bg-lavender-50/60 p-3 text-xs">
                <p className="font-bold text-ink-soft">
                  <span className="text-lavender-700">RF-02/03 y RF-05/07</span> · IA inmediata
                </p>
                <p className="mt-1 text-ink-soft">
                  {metrics.instant.requests === 0
                    ? "Sin pedidos directos de los estudiantes."
                    : `${metrics.instant.requests} pedidos · primer texto en ${fmtMs(metrics.instant.first_token.avg_ms)} · completa en ${fmtMs(
                        metrics.instant.total.avg_ms
                      )} · ${metrics.instant.failures} fallos · ${
                        metrics.instant.sections_checked
                          ? Math.round((metrics.instant.sections_ok / metrics.instant.sections_checked) * 100)
                          : 100
                      } % trajo las 3 secciones y ${
                        metrics.instant.quality_checked
                          ? Math.round((metrics.instant.quality_ok / metrics.instant.quality_checked) * 100)
                          : 100
                      } % la observación de calidad sin completar (lo que faltó se agregó solo)`}
                  {metrics.instant.chats > 0 && ` · ${metrics.instant.chats} mensajes de chat`}
                </p>
              </div>
            </div>
            <Comprehension data={metrics.comprehension} />
          </div>
        )}
      </Card>

      <Card title="Bitácora de intervenciones (RF-20)" className="rise-in lg:col-span-5" bodyClassName="p-5 pt-3">
        {!audit ? (
          <p className="text-sm text-ink-soft">Cargando...</p>
        ) : audit.length === 0 ? (
          <p className="text-sm text-ink-soft">Aún no hay intervenciones registradas en esta clase.</p>
        ) : (
          <ul className="max-h-72 space-y-1 overflow-auto pr-1">
            {audit.map((e) => (
              <li key={e.id} className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 rounded-xl px-2 py-1.5 text-xs odd:bg-lavender-50/60">
                <span className="w-32 shrink-0 text-ink-faint">{formatDateTime(e.at)}</span>
                <span className="font-bold">{ACTION_LABEL[e.action] ?? e.action}</span>
                <span className="text-ink-soft">
                  por {e.actor_name}
                  {e.student_name && ` · ${e.student_name}`}
                  {e.assignment_title && ` · ${e.assignment_title}`}
                </span>
                {e.detail && <span className="text-ink-soft">— {e.detail}</span>}
              </li>
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
