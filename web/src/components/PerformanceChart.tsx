"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Card } from "@/components/ui";
import { Assignment, getGroupPerformance, GroupPerformance, PerformanceAttention, PerformanceCategory, StudentPerformance } from "@/lib/api";

const LEVELS = ["Bajo", "Medio", "Alto"];
const EFFORT_MAX = 9; // por encima se dibuja en el borde superior ("9+")

const ATTENTION: Record<PerformanceAttention, { label: string; tone: string }> = {
  prioritaria: { label: "Atención prioritaria", tone: "bg-peach-100 text-peach-700" },
  seguimiento: { label: "Seguimiento", tone: "bg-amber-100 text-amber-900" },
  pendiente: { label: "Pendiente tuya", tone: "bg-lavender-100 text-lavender-700" },
  va_bien: { label: "Va bien", tone: "bg-mint-100 text-mint-700" },
};

// Márgenes en píxeles. El SVG se dibuja al ancho real del contenedor para que el
// texto no se encoja en pantallas pequeñas.
const M = { top: 12, right: 16, bottom: 46 };
const MIN_CELL_LABEL_WIDTH = 150; // en celdas más angostas el nombre va solo en el tooltip
const TOOLTIP_W = 256; // w-64
const TOOLTIP_H = 250;

function geometry(width: number) {
  const W = Math.max(280, width);
  const H = W < 560 ? 340 : 420;
  const left = W < 560 ? 52 : 64;
  const PW = W - left - M.right;
  const PH = H - M.top - M.bottom;
  return {
    W,
    H,
    left,
    PW,
    PH,
    x: (grade: number) => left + (grade / 5) * PW,
    y: (effort: number) => M.top + PH - (Math.min(effort, EFFORT_MAX) / EFFORT_MAX) * PH,
  };
}

type Point = StudentPerformance & { grade: number; cx: number; cy: number };

const fmt = (n: number) => n.toLocaleString("es-CO", { minimumFractionDigits: 1, maximumFractionDigits: 1 });
const plural = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

function AttentionPill({ attention }: { attention: PerformanceAttention }) {
  const a = ATTENTION[attention];
  return <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${a.tone}`}>{a.label}</span>;
}

function CategoryCard({
  category,
  students,
  onSelect,
}: {
  category: PerformanceCategory;
  students: StudentPerformance[];
  onSelect: (id: string) => void;
}) {
  return (
    <li className="flex flex-col gap-2 rounded-2xl bg-lavender-50/50 p-4 ring-1 ring-lavender-100">
      <div className="flex flex-wrap items-center gap-2">
        <h4 className="font-bold">{category.title}</h4>
        <AttentionPill attention={category.attention} />
      </div>
      <div className="flex flex-wrap gap-1.5">
        {students.map((s) => (
          <button
            key={s.id}
            type="button"
            onClick={() => onSelect(s.id)}
            className="rounded-full bg-white px-2.5 py-1 text-xs font-bold text-lavender-700 ring-1 ring-lavender-200 transition hover:bg-lavender-100"
          >
            {s.name}
            {s.grade !== null && <span className="ml-1.5 font-semibold tabular-nums text-ink-soft">{fmt(s.grade)}</span>}
          </button>
        ))}
      </div>
      <p className="text-sm leading-relaxed text-ink">
        <b>Cómo va:</b> {category.status}
      </p>
      <p className="text-sm leading-relaxed text-ink">
        <b>Qué necesita:</b> {category.needs}
      </p>
    </li>
  );
}

export default function PerformanceChart({
  groupId,
  assignments,
  refreshKey,
  onSelectStudent,
}: {
  groupId: string;
  assignments: Assignment[];
  refreshKey: number;
  onSelectStudent: (id: string) => void;
}) {
  const [assignmentId, setAssignmentId] = useState("");
  const [data, setData] = useState<GroupPerformance | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [hover, setHover] = useState<string | null>(null);
  const [asTable, setAsTable] = useState(false);
  const [width, setWidth] = useState(720);
  const box = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    getGroupPerformance(groupId, assignmentId || undefined)
      .then((d) => !cancelled && setData(d))
      .catch((err) => !cancelled && setError(err instanceof Error ? err.message : "No se pudo cargar el gráfico"));
    return () => {
      cancelled = true;
    };
  }, [groupId, assignmentId, refreshKey]);

  const hasChart = !!data && !asTable;
  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const ro = new ResizeObserver(([entry]) => setWidth(Math.round(entry.contentRect.width)));
    ro.observe(el);
    return () => ro.disconnect();
  }, [hasChart]);

  const { W, H, left, PW, PH, x, y } = useMemo(() => geometry(width), [width]);

  const categories = useMemo(() => new Map((data?.categories ?? []).map((c) => [c.key, c])), [data]);

  const points = useMemo(() => {
    // Estudiantes en la misma posición se separan en horizontal para que no se tapen.
    const seen = new Map<string, number>();
    return (data?.students ?? [])
      .filter((s) => s.grade !== null)
      .map((s): Point => {
        const grade = s.grade as number;
        const key = `${Math.round(x(grade) / 8)}:${Math.round(y(s.effort) / 8)}`;
        const n = seen.get(key) ?? 0;
        seen.set(key, n + 1);
        const offset = n === 0 ? 0 : (n % 2 ? 1 : -1) * Math.ceil(n / 2) * 14;
        return { ...s, grade, cx: Math.min(left + PW - 8, Math.max(left + 8, x(grade) + offset)), cy: y(s.effort) };
      });
  }, [data, x, y, left, PW]);

  const groups = useMemo(
    () =>
      (data?.categories ?? [])
        .map((category) => ({ category, students: (data?.students ?? []).filter((s) => s.category === category.key) }))
        .filter((g) => g.students.length > 0),
    [data]
  );

  const hovered = points.find((p) => p.id === hover);
  const hoveredCategory = hovered && categories.get(hovered.category);
  const showNames = points.length <= (W < 560 ? 4 : 16);
  const gradeBounds = data ? [0, ...data.grade_cuts, 5] : [];
  const effortBounds = data ? [0, ...data.effort_cuts, EFFORT_MAX] : [];
  const cellTitle = (row: number, col: number) => data?.categories.find((c) => c.effort_band === row && c.grade_band === col)?.title ?? "";

  return (
    <Card
      title="Nota vs. esfuerzo"
      className="rise-in"
      action={
        <div className="flex flex-wrap items-center justify-end gap-2">
          <select
            value={assignmentId}
            onChange={(e) => setAssignmentId(e.target.value)}
            className="max-w-[14rem] rounded-xl border border-lavender-200 bg-white px-3 py-1.5 text-xs font-semibold text-ink"
            aria-label="Filtrar por taller"
          >
            <option value="">Todos los talleres</option>
            {assignments.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => setAsTable((v) => !v)}
            className="rounded-xl bg-lavender-50 px-3 py-1.5 text-xs font-bold text-lavender-700 hover:bg-lavender-100"
          >
            {asTable ? "Ver gráfico" : "Ver tabla"}
          </button>
        </div>
      }
    >
      {data && (
        <p className="mb-3 text-xs text-ink-soft">
          Cada punto es un estudiante. <b>Nota</b>: promedio de las notas que pusiste, en borrador o devueltas (no cuenta la propuesta de la
          IA sin revisar): baja &lt; {fmt(data.grade_cuts[0])}, media hasta {fmt(data.grade_cuts[1] - 0.1)}, alta desde {fmt(data.grade_cuts[1])}.{" "}
          <b>Esfuerzo</b>: intentos de compilación por taller: bajo &lt; {data.effort_cuts[0]}, medio {data.effort_cuts[0]}–{data.effort_cuts[1] - 1},
          alto desde {data.effort_cuts[1]}. Haz clic en un estudiante para ver su proceso.
        </p>
      )}

      {error && <p className="rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>}
      {!data && !error && <p className="text-sm text-ink-soft">Cargando...</p>}

      {data && asTable && (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="text-xs text-ink-soft">
              <tr>
                <th className="py-2 pr-3 font-semibold">Estudiante</th>
                <th className="py-2 pr-3 font-semibold">Nota</th>
                <th className="py-2 pr-3 font-semibold">Intentos</th>
                <th className="py-2 pr-3 font-semibold">Por taller</th>
                <th className="py-2 pr-3 font-semibold">Categoría</th>
                <th className="py-2 font-semibold">Qué necesita</th>
              </tr>
            </thead>
            <tbody>
              {data.students.map((s) => {
                const c = categories.get(s.category);
                return (
                  <tr key={s.id} className="border-t border-lavender-100 align-top">
                    <td className="py-2 pr-3">
                      <button type="button" className="text-left font-bold text-lavender-700 hover:underline" onClick={() => onSelectStudent(s.id)}>
                        {s.name}
                      </button>
                    </td>
                    <td className="py-2 pr-3 tabular-nums">{s.grade !== null ? fmt(s.grade) : "Sin nota"}</td>
                    <td className="py-2 pr-3 tabular-nums">{s.attempts}</td>
                    <td className="py-2 pr-3 tabular-nums">{fmt(s.effort)}</td>
                    <td className="py-2 pr-3 font-semibold">{c?.title}</td>
                    <td className="min-w-[16rem] py-2 text-ink-soft">{c?.needs}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {data && !asTable && (
        <div ref={box} className="relative">
          <svg
            width={W}
            height={H}
            viewBox={`0 0 ${W} ${H}`}
            className="block"
            role="group"
            aria-label="Gráfico de dispersión de nota frente a esfuerzo por estudiante. La misma información está en las categorías de abajo y en la vista de tabla."
          >
            {/* 9 cuadrantes: la columna de nota baja se resalta como zona de atención */}
            {[0, 1, 2].map((row) =>
              [0, 1, 2].map((col) => {
                const x0 = x(gradeBounds[col]);
                const x1 = x(gradeBounds[col + 1]);
                const y0 = y(effortBounds[row + 1]);
                const y1 = y(effortBounds[row]);
                return (
                  <g key={`${row}-${col}`}>
                    <rect x={x0} y={y0} width={x1 - x0} height={y1 - y0} className={col === 0 ? "fill-peach-50" : "fill-white"} />
                    {x1 - x0 >= MIN_CELL_LABEL_WIDTH && (
                      <text x={x0 + 8} y={y0 + 16} className="fill-ink-soft text-[11px] font-semibold">
                        {cellTitle(row, col)}
                      </text>
                    )}
                  </g>
                );
              })
            )}

            {data.grade_cuts.map((g) => (
              <line key={`gx${g}`} x1={x(g)} x2={x(g)} y1={M.top} y2={M.top + PH} className="stroke-lavender-200" strokeDasharray="4 4" />
            ))}
            {data.effort_cuts.map((e) => (
              <line key={`ey${e}`} x1={left} x2={left + PW} y1={y(e)} y2={y(e)} className="stroke-lavender-200" strokeDasharray="4 4" />
            ))}
            <rect x={left} y={M.top} width={PW} height={PH} fill="none" className="stroke-lavender-200" />

            {[0, 1, 2, 3, 4, 5].map((g) => (
              <text key={`tx${g}`} x={x(g)} y={M.top + PH + 16} textAnchor="middle" className="fill-ink-soft text-[11px] tabular-nums">
                {fmt(g)}
              </text>
            ))}
            <text x={left + PW / 2} y={H - 8} textAnchor="middle" className="fill-ink-soft text-[12px] font-semibold">
              Nota del docente (0 a 5)
            </text>

            {[0, 1, 2].map((row) => (
              <text
                key={`ly${row}`}
                x={left - 10}
                y={(y(effortBounds[row]) + y(effortBounds[row + 1])) / 2 + 4}
                textAnchor="end"
                className="fill-ink-soft text-[12px] font-bold"
              >
                {LEVELS[row]}
              </text>
            ))}
            {[...data.effort_cuts, EFFORT_MAX].map((e) => (
              <text key={`ty${e}`} x={left - 10} y={y(e) + 4} textAnchor="end" className="fill-ink-soft text-[10px] tabular-nums">
                {e === EFFORT_MAX ? `${e}+` : e}
              </text>
            ))}
            {W >= 560 && (
              <text transform={`translate(14 ${M.top + PH / 2}) rotate(-90)`} textAnchor="middle" className="fill-ink-soft text-[12px] font-semibold">
                Esfuerzo (intentos por taller)
              </text>
            )}

            {points.map((p) => {
              const flip = p.cx > left + PW - 70;
              return (
                <g
                  key={p.id}
                  role="button"
                  tabIndex={0}
                  aria-label={`${p.name}: nota ${fmt(p.grade)}, ${fmt(p.effort)} intentos por taller, ${categories.get(p.category)?.title}. Ver su proceso.`}
                  className="cursor-pointer outline-none"
                  onPointerEnter={() => setHover(p.id)}
                  onPointerLeave={() => setHover(null)}
                  onFocus={() => setHover(p.id)}
                  onBlur={() => setHover(null)}
                  onClick={() => onSelectStudent(p.id)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      onSelectStudent(p.id);
                    }
                  }}
                >
                  <circle cx={p.cx} cy={p.cy} r={14} fill="transparent" />
                  <circle cx={p.cx} cy={p.cy} r={hover === p.id ? 8 : 6} className="fill-lavender-600 stroke-white transition-all" strokeWidth={2} />
                  {showNames && (
                    <text
                      x={flip ? p.cx - 11 : p.cx + 11}
                      y={p.cy + 4}
                      textAnchor={flip ? "end" : "start"}
                      className="pointer-events-none fill-ink text-[11px] font-bold"
                    >
                      {p.name.split(" ")[0]}
                    </text>
                  )}
                </g>
              );
            })}
          </svg>

          {hovered && hoveredCategory && (
            <div
              className="pointer-events-none absolute z-10 w-64 rounded-2xl bg-white p-3 text-xs shadow-lift ring-1 ring-lavender-100"
              style={{
                left: Math.min(Math.max(hovered.cx - TOOLTIP_W / 2, 0), Math.max(0, W - TOOLTIP_W)),
                ...(hovered.cy > TOOLTIP_H ? { top: hovered.cy - 14, transform: "translateY(-100%)" } : { top: hovered.cy + 14 }),
              }}
            >
              <p className="text-sm font-bold text-ink">{hovered.name}</p>
              <p className="mt-0.5 font-semibold text-lavender-700">{hoveredCategory.title}</p>
              <dl className="mt-2 grid grid-cols-[1fr_auto] gap-x-3 gap-y-0.5 text-ink-soft">
                <dt>Nota</dt>
                <dd className="text-right font-bold tabular-nums text-ink">{fmt(hovered.grade)}</dd>
                <dt>Talleres con nota</dt>
                <dd className="text-right tabular-nums text-ink">{hovered.graded_count}</dd>
                <dt>Intentos</dt>
                <dd className="text-right tabular-nums text-ink">{hovered.attempts}</dd>
                <dt>Por taller</dt>
                <dd className="text-right tabular-nums text-ink">{fmt(hovered.effort)}</dd>
              </dl>
              <p className="mt-2 leading-relaxed text-ink">
                <b>Qué necesita:</b> {hoveredCategory.needs}
              </p>
            </div>
          )}
        </div>
      )}

      {data && data.students.length === 0 && <p className="mt-3 text-sm text-ink-soft">Todavía no hay estudiantes en esta clase.</p>}

      {groups.length > 0 && (
        <section className="mt-5" aria-labelledby={`categorias-${groupId}`}>
          <h3 id={`categorias-${groupId}`} className="font-bold">
            Cómo va cada estudiante
          </h3>
          <p className="mt-0.5 text-xs text-ink-soft">
            {plural(data?.students.length ?? 0, "estudiante", "estudiantes")}, primero quienes necesitan más atención. Es una orientación a partir de
            sus notas e intentos: no reemplaza tu criterio ni describe a la persona.
          </p>
          <ul className="mt-3 grid gap-3 lg:grid-cols-2">
            {groups.map(({ category, students }) => (
              <CategoryCard key={category.key} category={category} students={students} onSelect={onSelectStudent} />
            ))}
          </ul>
        </section>
      )}

      {data && (
        <details className="mt-4 rounded-2xl ring-1 ring-lavender-100">
          <summary className="cursor-pointer rounded-2xl px-4 py-3 text-sm font-bold hover:bg-lavender-50">Qué significa cada categoría</summary>
          <ul className="space-y-3 px-4 pb-4">
            {data.categories.map((c) => (
              <li key={c.key} className="border-t border-lavender-100 pt-3 first:border-t-0 first:pt-0">
                <p className="flex flex-wrap items-center gap-2 text-sm font-bold">
                  {c.title} <AttentionPill attention={c.attention} />
                </p>
                <p className="mt-1 text-sm text-ink-soft">{c.status}</p>
                <p className="mt-1 text-sm text-ink-soft">
                  <b>Qué necesita:</b> {c.needs}
                </p>
              </li>
            ))}
          </ul>
        </details>
      )}
    </Card>
  );
}
