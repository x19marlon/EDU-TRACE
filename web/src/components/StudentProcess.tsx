"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { GroupStudent, listGroupSubmissions, streamStudentAnalysis, SubmissionMeta } from "@/lib/api";
import { Card, Spinner } from "@/components/ui";
import { KindPill, StatusPill, SubmissionDetail } from "@/components/SubmissionView";
import { formatDateTime, formatFeedback } from "@/lib/format";

type Filter = "all" | "official" | "attempt";

function Stat({ value, label, tone }: { value: string | number; label: string; tone: string }) {
  return (
    <div className={`rounded-2xl px-3 py-3 text-center ${tone}`}>
      <p className="text-xl font-extrabold tabular-nums">{value}</p>
      <p className="mt-0.5 text-[11px] font-semibold">{label}</p>
    </div>
  );
}

/** Análisis con IA del proceso, para el profesor. Se pide a demanda. */
function AnalysisCard({ groupId, studentId, total }: { groupId: string; studentId: string; total: number }) {
  const [text, setText] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [timeMs, setTimeMs] = useState<number | null>(null);
  const acc = useRef("");

  const run = async () => {
    acc.current = "";
    setText("");
    setError(null);
    setTimeMs(null);
    setLoading(true);
    try {
      const meta = await streamStudentAnalysis(groupId, studentId, (t) => {
        acc.current += t;
        setText(acc.current);
      });
      setTimeMs(meta.time_ms);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo generar el análisis");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rounded-3xl border border-lavender-100 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <h3 className="font-bold">En qué puede mejorar</h3>
          <p className="text-xs text-ink-soft">
            La IA revisa sus intentos y entregas y te propone en qué enfocarte. Tú decides qué hacer con ello.
          </p>
        </div>
        <button
          onClick={run}
          disabled={loading || total === 0}
          className="flex items-center gap-2 rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white shadow-soft
            transition hover:bg-lavender-700 disabled:opacity-40"
        >
          {loading && <Spinner className="h-4 w-4" />}
          {loading ? "Analizando..." : text ? "Volver a analizar" : "Analizar con IA"}
        </button>
      </div>
      {error && <p className="mt-3 rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>}
      {(text || loading) && (
        <div className="mt-3 rounded-2xl bg-lavender-50/70 p-4">
          {loading && !text && <p className="text-sm font-semibold text-lavender-700">Revisando el proceso...</p>}
          <div className="text-sm leading-relaxed" dangerouslySetInnerHTML={{ __html: formatFeedback(text) }} />
          {!loading && (
            <p className="mt-3 border-t border-lavender-200 pt-2 text-xs text-ink-faint">
              Análisis generado por IA{timeMs ? ` en ${(timeMs / 1000).toFixed(1)} s` : ""} a partir de los datos del
              estudiante. Puede equivocarse: úsalo como apoyo, la valoración es tuya.
            </p>
          )}
        </div>
      )}
    </div>
  );
}

/** Proceso de un estudiante en la clase: intentos de compilación y entregas oficiales. */
export default function StudentProcess({
  groupId,
  student,
  assignmentTitles,
}: {
  groupId: string;
  student: GroupStudent;
  assignmentTitles: Map<string, string>;
}) {
  const [items, setItems] = useState<SubmissionMeta[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>("all");
  const [selectedId, setSelectedId] = useState<string | null>(null);

  useEffect(() => {
    listGroupSubmissions(groupId, student.id, "all")
      .then(setItems)
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar el proceso"));
  }, [groupId, student.id]);

  const stats = useMemo(() => {
    const list = items ?? [];
    const attempts = list.filter((m) => m.kind === "attempt");
    const compiled = attempts.filter((m) => m.success).length;
    const counts = new Map<string, number>();
    list.forEach((m) => m.error_summary && counts.set(m.error_summary, (counts.get(m.error_summary) ?? 0) + 1));
    const topErrors = Array.from(counts.entries()).sort((a, b) => b[1] - a[1]).slice(0, 3);
    return {
      attempts: attempts.length,
      official: list.length - attempts.length,
      compiledPct: attempts.length ? Math.round((compiled / attempts.length) * 100) : null,
      topErrors,
    };
  }, [items]);

  const visible = (items ?? []).filter((m) => filter === "all" || m.kind === filter);
  const tabs: [Filter, string][] = [
    ["all", `Todo (${(items ?? []).length})`],
    ["official", `Entregas oficiales (${stats.official})`],
    ["attempt", `Intentos (${stats.attempts})`],
  ];

  return (
    <Card title={`Proceso de ${student.name}`} className="rise-in lg:col-span-3">
      {error ? (
        <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>
      ) : !items ? (
        <p className="text-sm text-ink-soft">Cargando...</p>
      ) : (
        <div className="space-y-4">
          <div className="grid grid-cols-3 gap-2">
            <Stat value={stats.attempts} label="intentos" tone="bg-lavender-50 text-lavender-700" />
            <Stat
              value={stats.compiledPct === null ? "—" : `${stats.compiledPct}%`}
              label="intentos que compilaron"
              tone="bg-mint-50 text-mint-700"
            />
            <Stat value={stats.official} label="entregas oficiales" tone="bg-peach-50 text-peach-700" />
          </div>

          {stats.topErrors.length > 0 && (
            <div>
              <h3 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Errores de compilación más frecuentes</h3>
              <ul className="space-y-1">
                {stats.topErrors.map(([msg, n]) => (
                  <li key={msg} className="flex items-center gap-2 rounded-xl bg-peach-50 px-3 py-1.5 text-sm">
                    <code className="min-w-0 flex-1 truncate font-mono text-xs text-peach-700" title={msg}>{msg}</code>
                    <span className="text-xs font-bold text-peach-700">{n}×</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <AnalysisCard groupId={groupId} studentId={student.id} total={items.length} />

          <div>
            <div role="tablist" className="mb-2 flex flex-wrap gap-1">
              {tabs.map(([f, label]) => (
                <button
                  key={f}
                  role="tab"
                  aria-selected={filter === f}
                  onClick={() => setFilter(f)}
                  className={`rounded-full px-3.5 py-1.5 text-sm font-bold transition-colors ${
                    filter === f ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
            {visible.length === 0 ? (
              <p className="text-sm text-ink-soft">
                {filter === "attempt"
                  ? "Todavía no hay intentos guardados de este estudiante en la clase."
                  : filter === "official"
                    ? "Este estudiante aún no ha hecho una entrega oficial."
                    : "Todavía no hay intentos ni entregas de este estudiante en la clase."}
              </p>
            ) : (
              <ul className="max-h-80 space-y-1 overflow-auto pr-1">
                {visible.map((m) => (
                  <li key={m.id}>
                    <button
                      onClick={() => setSelectedId(m.id)}
                      className={`flex w-full flex-wrap items-center gap-x-3 gap-y-1 rounded-2xl px-3 py-2.5 text-left transition-colors ${
                        selectedId === m.id
                          ? "bg-lavender-100"
                          : m.kind === "attempt"
                            ? "hover:bg-lavender-50"
                            : "bg-lavender-50/70 hover:bg-lavender-50"
                      }`}
                    >
                      <KindPill meta={m} />
                      <span className="text-xs text-ink-soft">{formatDateTime(m.created_at)}</span>
                      {m.assignment_id && (
                        <span className="rounded-full bg-lavender-100 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
                          {assignmentTitles.get(m.assignment_id) ?? "Taller"}
                        </span>
                      )}
                      {m.error_summary && (
                        <code className="max-w-[16rem] truncate font-mono text-[11px] text-peach-700" title={m.error_summary}>
                          {m.error_summary}
                        </code>
                      )}
                      <span className="flex-1" />
                      <StatusPill meta={m} />
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>

          {selectedId && (
            <div className="border-t border-lavender-100 pt-4">
              <SubmissionDetail
                id={selectedId}
                studentName={student.name}
                assignmentTitle={(() => {
                  const aid = items.find((m) => m.id === selectedId)?.assignment_id;
                  return aid ? assignmentTitles.get(aid) ?? "Taller" : undefined;
                })()}
              />
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
