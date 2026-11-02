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
import AuditItem from "@/components/AuditItem";
import { formatDateTime, formatShortDuration } from "@/lib/format";

function fmtMs(ms: number): string {
  if (ms < 1000) return `${ms} ms`;
  if (ms < 60000) return `${Math.round(ms / 1000)} s`;
  return formatShortDuration(ms / 1000);
}

type Verdict = "ok" | "over" | "none" | "empty";

const VERDICT: Record<Verdict, [string, string]> = {
  ok: ["A tiempo", "bg-mint-100 text-mint-700"],
  over: ["Con demoras", "bg-peach-100 text-peach-700"],
  none: ["Sin meta", "bg-lavender-50 text-ink-soft"],
  empty: ["Sin datos", "bg-lavender-50 text-ink-soft"],
};

const ESTIMATE_VERDICT: Record<Verdict, [string, string]> = {
  ok: ["Dentro de lo estimado", "bg-mint-100 text-mint-700"],
  over: ["Más de lo estimado", "bg-amber-100 text-amber-800"],
  none: ["Sin estimación", "bg-lavender-50 text-ink-soft"],
  empty: ["Sin datos", "bg-lavender-50 text-ink-soft"],
};

function verdict(hasData: boolean, limit: number | null | undefined, over: boolean): Verdict {
  if (!hasData) return "empty";
  if (!limit) return "none";
  return over ? "over" : "ok";
}

function Indicator({ title, value, detail, chip }: { title: string; value: string; detail: string; chip: [string, string] }) {
  const [label, tone] = chip;
  return (
    <div className="rounded-2xl bg-lavender-50/60 p-3">
      <div className="flex items-start justify-between gap-2">
        <p className="text-xs font-bold text-ink-soft">{title}</p>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${tone}`}>{label}</span>
      </div>
      <p className="mt-1 text-xl font-extrabold tabular-nums">{value}</p>
      <p className="mt-1 text-xs leading-relaxed text-ink-soft">{detail}</p>
    </div>
  );
}

function statLine(s: Stat): string {
  return s.count === 0 ? "—" : `${fmtMs(s.avg_ms)} en promedio`;
}

function reviewText(m: GroupMetrics): string {
  const r = m.review_time;
  if (r.count === 0) return "Todavía no hay revisiones medidas. Cuando revises entregas, aquí verás cuánto tiempo les dedicas.";
  const base =
    r.count === 1
      ? "Llevas 1 revisión medida."
      : `Llevas ${r.count} revisiones medidas. La mitad te tomó menos de ${fmtMs(r.p50_ms)} y la más larga, ${fmtMs(r.max_ms)}.`;
  if (m.review_estimate.count === 0) return base;
  return `${base} La IA calculaba unos ${fmtMs(m.review_estimate.p50_ms)} por entrega y ${m.over_estimate} de ${m.review_estimate.count} te tomaron más.`;
}

function turnaroundText(m: GroupMetrics): string {
  const t = m.turnaround;
  const limit = m.thresholds.feedback_hours;
  const parts = [];
  if (t.count === 0) parts.push("Todavía no has devuelto retroalimentación en esta clase.");
  else if (t.count === 1) parts.push("Ya devolviste 1 retroalimentación.");
  else parts.push(`Ya devolviste ${t.count} retroalimentaciones. La mitad llegó en menos de ${fmtMs(t.p50_ms)} y la más demorada tardó ${fmtMs(t.max_ms)}.`);
  if (t.over && limit) parts.push(`${t.over} ${t.over === 1 ? "pasó" : "pasaron"} de las ${limit} h que tienes como meta.`);
  if (m.pending_reviews) {
    const late = m.pending_over && limit ? ` y ${m.pending_over} ya ${m.pending_over === 1 ? "pasó" : "pasaron"} de las ${limit} h` : "";
    parts.push(`Faltan ${m.pending_reviews} por devolver: la más antigua lleva ${fmtMs(m.oldest_pending_ms)} esperando${late}.`);
  }
  return parts.join(" ");
}

function instantText(i: GroupMetrics["instant"]): string {
  const chats = i.chats > 0 ? ` También han enviado ${i.chats} ${i.chats === 1 ? "mensaje" : "mensajes"} al tutor.` : "";
  if (i.requests === 0) return `Tus estudiantes todavía no le han pedido retroalimentación directamente a la IA.${chats}`;
  const percent = (ok: number, checked: number) => (checked ? Math.round((ok / checked) * 100) : 100);
  const failed = i.failures > 0 ? ` ${i.failures} ${i.failures === 1 ? "pedido falló" : "pedidos fallaron"}.` : "";
  return (
    `Tus estudiantes le han pedido retroalimentación directamente ${i.requests} ${i.requests === 1 ? "vez" : "veces"}. ` +
    `Suele empezar a responder en ${fmtMs(i.first_token.avg_ms)} y termina en ${fmtMs(i.total.avg_ms)}.${failed} ` +
    `El ${percent(i.sections_ok, i.sections_checked)} % de las respuestas trajo sus tres secciones y el ${percent(i.quality_ok, i.quality_checked)} % ` +
    `incluyó la observación de calidad; cuando algo faltó, EduTrace lo agregó automáticamente.${chats}`
  );
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
      <div className="space-y-1.5 text-xs leading-relaxed text-ink-soft">
        <p>Aquí puedes configurar los tiempos que quieres manejar para tu materia. Estos ajustes se aplicarán a todas las clases que tengas asociadas.</p>
        <p>
          Por ahora, se han establecido por defecto 5 horas para procesar cada lote de entregas y otras 5 horas como tiempo máximo entre la
          entrega de un estudiante y la recepción de su retroalimentación. Puedes ajustar estos tiempos según las necesidades de tu materia.
        </p>
        <p>
          El tiempo de revisión queda a tu criterio, podrás decidir cuánto dedicarle a cada propuesta teniendo en cuenta la estimación de tiempo
          que te proporciona la IA.
        </p>
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-2">
        {field("De la entrega a la retroalimentación, máximo", "h", feedback, setFeedback)}
        {field("Procesar cada lote, máximo", "h", batch, setBatch)}
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
        "En el modo inmediato, los estudiantes recibirán directamente la retroalimentación y podrán interactuar con la IA sin necesidad de que revises o apruebes su contenido previamente.\n\n¿Estás seguro de que deseas continuar?"
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
          "La IA prepara un borrador de cada entrega y el estudiante solo ve lo que tú apruebas."
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
  return total === 0 ? "sin respuestas" : `${Math.round((c.si / total) * 100)} % entendió (${c.si} sí, ${c.parcial} en parte, ${c.no} no)`;
}

const COMPREHENSION_VERDICT: Record<GroupMetrics["comprehension"]["veredicto"], [string, string]> = {
  cumple: ["Se entiende bien", "bg-mint-100 text-mint-700"],
  no_cumple: ["Les está costando", "bg-peach-100 text-peach-700"],
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
        <p className="font-bold text-ink-soft">¿Los estudiantes entienden la retroalimentación?</p>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${tone}`}>{label}</span>
      </div>
      <p className="text-ink-soft">
        Después de cada retroalimentación les preguntamos si la entendieron. En cada taller buscamos que al menos el{" "}
        {Math.round(goal.min_si * 100)} % diga que sí y que no más del {Math.round(goal.max_no * 100)} % diga que no, con {goal.min_respuestas}{" "}
        respuestas o más.
      </p>
      {data.por_taller.length === 0 ? (
        <p className="text-ink-soft">
          Aún nadie ha respondido. Cuando lo hagan, aquí verás el resumen, sin nombres.
        </p>
      ) : (
        <ul className="space-y-1.5">
          {data.por_taller.map((t) => (
            <li key={t.assignment_id || "libre"} className="rounded-xl bg-white px-2.5 py-2">
              <div className="flex flex-wrap items-center gap-x-2 gap-y-1">
                <span className="min-w-0 flex-1 truncate font-bold">{t.title}</span>
                <span className="text-ink-soft">
                  {pct(t.si, t.total)} % sí, {pct(t.parcial, t.total)} % en parte, {pct(t.no, t.total)} % no
                </span>
                <span
                  className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${
                    !t.suficientes ? "bg-lavender-50 text-ink-soft" : t.cumple ? "bg-mint-100 text-mint-700" : "bg-peach-100 text-peach-700"
                  }`}
                >
                  {!t.suficientes ? `${t.total} de ${goal.min_respuestas} respuestas` : t.cumple ? "Se entiende" : "Les cuesta"}
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
          En las revisiones devueltas: {share(data.revisiones)}. En la IA inmediata: {share(data.inmediata)}.
        </p>
      )}
      {data.comentarios.length > 0 && (
        <ul className="space-y-1">
          {data.comentarios.map((c, i) => (
            <li key={i} className="rounded-xl bg-white px-2.5 py-1.5">
              <span className="font-bold">{c.understood === "no" ? "No entendió" : c.understood === "parcial" ? "En parte" : "Entendió"}:</span>{" "}
              {c.comment} <span className="text-ink-faint">({formatDateTime(c.at)})</span>
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
            Cada entrega oficial pasa automáticamente a la cola de revisión. La IA ejecuta los casos de prueba y prepara una propuesta de
            retroalimentación que podrás revisar, editar, aprobar o descartar.
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
                title="Tiempo de revisión por entrega"
                value={statLine(metrics.review_time)}
                detail={reviewText(metrics)}
                chip={
                  ESTIMATE_VERDICT[
                    verdict(
                      metrics.review_time.count > 0,
                      metrics.review_estimate.count > 0 ? metrics.review_estimate.p50_ms : undefined,
                      metrics.review_time.p50_ms > metrics.review_estimate.p50_ms
                    )
                  ]
                }
              />
              <Indicator
                title="De la entrega a la retroalimentación"
                value={statLine(metrics.turnaround)}
                detail={turnaroundText(metrics)}
                chip={
                  VERDICT[
                    verdict(
                      metrics.turnaround.count + metrics.pending_reviews > 0,
                      t?.feedback_hours,
                      metrics.turnaround.over + metrics.pending_over > 0
                    )
                  ]
                }
              />
            </div>
            <div className="rounded-2xl bg-lavender-50/60 p-3">
              <p className="text-xs font-bold text-ink-soft">Procesamiento de cada lote</p>
              <p className="mt-1 text-xs leading-relaxed text-ink-soft">
                Aquí ves cuánto tarda la IA en preparar las propuestas de cada taller.
                {t?.batch_hours ? ` La idea es que ninguna entrega tarde más de ${t.batch_hours} h.` : ""}
                {metrics.queue.run.count > 0
                  ? ` En promedio, cada entrega espera ${fmtMs(metrics.queue.wait.avg_ms)} en la cola y su propuesta tarda ${fmtMs(metrics.queue.run.avg_ms)}.`
                  : ""}
              </p>
              {metrics.batches.length === 0 ? (
                <p className="mt-1 text-xs text-ink-soft">Todavía no hay entregas oficiales en la cola.</p>
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
                          {b.done} de {b.submissions} listas{b.failed ? `, ${b.failed} con error` : ""}
                          {b.waiting ? `, ${b.waiting} en cola` : ""}
                          {skipped ? "" : `. La más demorada tardó ${fmtMs(b.worst_ms)}`}
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
                <p className="font-bold text-ink-soft">Decisiones sobre la IA</p>
                <p className="mt-1 leading-relaxed text-ink-soft">
                  De las propuestas de la IA, aprobaste {metrics.decisions.ia} sin cambios, editaste {metrics.decisions.ia_editada} y descartaste{" "}
                  {metrics.decisions.descartadas}. Además, escribiste {metrics.decisions.docente} por tu cuenta.
                </p>
              </div>
              <div className="rounded-2xl bg-lavender-50/60 p-3 text-xs">
                <p className="font-bold text-ink-soft">IA inmediata</p>
                <p className="mt-1 leading-relaxed text-ink-soft">{instantText(metrics.instant)}</p>
              </div>
            </div>
            <Comprehension data={metrics.comprehension} />
          </div>
        )}
      </Card>

      <Card title="Bitácora de intervenciones" className="rise-in lg:col-span-5" bodyClassName="p-5 pt-3">
        {!audit ? (
          <p className="text-sm text-ink-soft">Cargando...</p>
        ) : audit.length === 0 ? (
          <p className="text-sm text-ink-soft">Aún no hay intervenciones registradas en esta clase.</p>
        ) : (
          <ul tabIndex={0} aria-label="Bitácora de la clase" className="max-h-96 space-y-1 overflow-auto rounded-xl pr-1 focus:outline-none focus-visible:ring-4 focus-visible:ring-lavender-100">
            {audit.map((e) => (
              <AuditItem key={e.id} entry={e} className="rounded-xl px-3 py-2 odd:bg-lavender-50/60" />
            ))}
          </ul>
        )}
      </Card>
    </div>
  );
}
