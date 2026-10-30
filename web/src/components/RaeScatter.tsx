"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getRAEMatrix, RAELevel, RAEMatrix, StudentPoint } from "@/lib/api";
import { Card, SmallButton } from "@/components/ui";

const W = 640;
const H = 360;
const M = { top: 14, right: 18, bottom: 64, left: 78 };
const PW = W - M.left - M.right;
const PH = H - M.top - M.bottom;
const RH = PH / 3;
const ROWS: RAELevel[] = ["alto", "medio", "bajo"];
const BANDS = [
  { label: "Bajo", range: "0.0 – 2.9", from: 0, to: 3 },
  { label: "Medio", range: "3.0 – 3.9", from: 3, to: 4 },
  { label: "Alto", range: "4.0 – 5.0", from: 4, to: 5 },
];
const INK = "#2E2A3F";
const INK_SOFT = "#6B6680";
const INK_FAINT = "#A29DB3";
const GRID = "#EDE7FC";
const DIVIDER = "#DCD2F8";
const POINT = "#7154CC";
const POINT_STRONG = "#5C42AE";

const xOf = (grade: number) => M.left + (grade / 5) * PW;
const rowOf = (level: RAELevel) => ROWS.indexOf(level);
const bandOf = (grade: number) => (grade < 3 ? 0 : grade < 4 ? 1 : 2);
const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

interface Plotted {
  student: StudentPoint;
  grade: number;
  level: RAELevel;
  x: number;
  y: number;
}

export default function RaeScatter({
  groupId,
  refreshKey,
  selectedId,
  onSelect,
}: {
  groupId: string;
  refreshKey: number;
  selectedId: string | null;
  onSelect: (studentId: string) => void;
}) {
  const [matrix, setMatrix] = useState<RAEMatrix | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [raeId, setRaeId] = useState<string | null>(null);
  const [hover, setHover] = useState<string | null>(null);
  const [table, setTable] = useState(false);

  useEffect(() => {
    getRAEMatrix(groupId)
      .then((m) => {
        setMatrix(m);
        setError(null);
        setRaeId((current) => (current && m.raes.some((r) => r.id === current) ? current : m.raes[0]?.id ?? null));
      })
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar el rendimiento"));
  }, [groupId, refreshKey]);

  const rae = matrix?.raes.find((r) => r.id === raeId);

  const { plotted, missing, counts } = useMemo(() => {
    const plotted: Plotted[] = [];
    const missing: { student: StudentPoint; reason: string }[] = [];
    const counts = ROWS.map(() => [0, 0, 0]);
    if (!matrix || !raeId) return { plotted, missing, counts };
    for (const s of matrix.students) {
      const level = s.levels[raeId];
      if (s.average_grade === undefined) missing.push({ student: s, reason: "sin notas devueltas" });
      else if (!level) missing.push({ student: s, reason: "sin nivel marcado en este RAE" });
      else {
        counts[rowOf(level)][bandOf(s.average_grade)]++;
        plotted.push({ student: s, grade: s.average_grade, level, x: xOf(s.average_grade), y: 0 });
      }
    }
    for (const level of ROWS) {
      const row = plotted.filter((p) => p.level === level).sort((a, b) => a.grade - b.grade);
      row.forEach((p, i) => {
        p.y = M.top + rowOf(level) * RH + RH / 2 + ((i % 5) - 2) * RH * 0.11;
      });
    }
    return { plotted, missing, counts };
  }, [matrix, raeId]);

  const hovered = plotted.find((p) => p.student.id === hover);
  const labelNames = plotted.length <= 8;

  return (
    <Card
      title="Rendimiento por RAE"
      className="rise-in"
      action={
        matrix && matrix.raes.length > 0 ? (
          <SmallButton onClick={() => setTable((t) => !t)}>{table ? "Ver diagrama" : "Ver como tabla"}</SmallButton>
        ) : undefined
      }
    >
      {error && <p className="rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>}
      {!matrix && !error && <p className="text-sm text-ink-soft">Cargando...</p>}
      {matrix && matrix.raes.length === 0 && (
        <p className="text-sm text-ink-soft">
          Esta materia aún no tiene RAE. Defínelos en{" "}
          <Link href="/docente" className="font-bold text-lavender-600 hover:text-lavender-700">
            Mis materias
          </Link>{" "}
          (sección «Resultados de Aprendizaje Esperados») y marca el nivel de cada RAE al revisar las entregas.
        </p>
      )}

      {matrix && rae && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <div role="radiogroup" aria-label="RAE" className="flex flex-wrap gap-1 rounded-full bg-lavender-50 p-1">
              {matrix.raes.map((r) => (
                <button
                  key={r.id}
                  type="button"
                  role="radio"
                  aria-checked={r.id === raeId}
                  onClick={() => setRaeId(r.id)}
                  className={`rounded-full px-3 py-1 text-xs font-bold transition ${
                    r.id === raeId ? "bg-white text-lavender-700 shadow-soft" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {r.code}
                </button>
              ))}
            </div>
            <p className="min-w-0 flex-1 text-sm text-ink-soft">{rae.description}</p>
          </div>

          {table ? (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead className="text-xs uppercase tracking-wider text-ink-faint">
                  <tr>
                    <th className="py-2 pr-3 font-bold">Estudiante</th>
                    <th className="py-2 pr-3 font-bold">Nota promedio</th>
                    <th className="py-2 pr-3 font-bold">Nivel {rae.code}</th>
                    <th className="py-2 pr-3 font-bold">Entregas calificadas</th>
                  </tr>
                </thead>
                <tbody>
                  {matrix.students.map((s) => (
                    <tr key={s.id} className="border-t border-lavender-100">
                      <td className="py-2 pr-3 font-semibold">{s.name}</td>
                      <td className="py-2 pr-3 tabular-nums">{s.average_grade === undefined ? "—" : s.average_grade.toFixed(1)}</td>
                      <td className="py-2 pr-3">{s.levels[rae.id] ? capitalize(s.levels[rae.id]) : "—"}</td>
                      <td className="py-2 pr-3 tabular-nums">{s.graded}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <div className="relative min-w-[520px]">
                <svg viewBox={`0 0 ${W} ${H}`} className="block h-auto w-full" role="img" aria-label={`Diagrama de dispersión: nota promedio contra nivel en ${rae.code}`}>
                  <rect x={M.left} y={M.top + 2 * RH} width={xOf(3) - M.left} height={RH} fill="#FFF6EF" />
                  <rect x={xOf(4)} y={M.top} width={M.left + PW - xOf(4)} height={RH} fill="#F0FAF5" />
                  <text x={M.left + 8} y={M.top + 2 * RH + 15} fontSize={11} fill={INK_FAINT}>Requiere acompañamiento</text>
                  <text x={xOf(4) + 8} y={M.top + 15} fontSize={11} fill={INK_FAINT}>Logro consolidado</text>

                  {[0, 1, 2, 3, 4, 5].map((t) => (
                    <g key={t}>
                      <line x1={xOf(t)} x2={xOf(t)} y1={M.top} y2={M.top + PH} stroke={t === 3 || t === 4 ? DIVIDER : GRID} strokeWidth={1} />
                      <text x={xOf(t)} y={M.top + PH + 16} fontSize={11} fill={INK_SOFT} textAnchor="middle" className="tabular-nums">
                        {t.toFixed(1)}
                      </text>
                    </g>
                  ))}
                  {[0, 1, 2, 3].map((i) => (
                    <line key={i} x1={M.left} x2={M.left + PW} y1={M.top + i * RH} y2={M.top + i * RH} stroke={i === 0 || i === 3 ? GRID : DIVIDER} strokeWidth={1} />
                  ))}
                  {ROWS.map((level, i) => (
                    <text key={level} x={M.left - 12} y={M.top + i * RH + RH / 2 + 4} fontSize={12} fontWeight={700} fill={INK} textAnchor="end">
                      {capitalize(level)}
                    </text>
                  ))}
                  {BANDS.map((b) => (
                    <g key={b.label}>
                      <text x={(xOf(b.from) + xOf(b.to)) / 2} y={M.top + PH + 36} fontSize={12} fontWeight={700} fill={INK} textAnchor="middle">
                        {b.label}
                      </text>
                      <text x={(xOf(b.from) + xOf(b.to)) / 2} y={M.top + PH + 51} fontSize={10} fill={INK_FAINT} textAnchor="middle">
                        {b.range}
                      </text>
                    </g>
                  ))}
                  <text x={14} y={M.top + PH / 2} fontSize={11} fill={INK_SOFT} textAnchor="middle" transform={`rotate(-90 14 ${M.top + PH / 2})`}>
                    Nivel en {rae.code}
                  </text>

                  {ROWS.map((level, r) =>
                    BANDS.map((b, c) =>
                      counts[r][c] > 0 ? (
                        <text key={`${level}-${b.label}`} x={xOf(b.to) - 6} y={M.top + (r + 1) * RH - 7} fontSize={10} fill={INK_FAINT} textAnchor="end">
                          {counts[r][c]} est.
                        </text>
                      ) : null
                    )
                  )}

                  {plotted.map((p) => {
                    const active = p.student.id === hover || p.student.id === selectedId;
                    const right = p.x > M.left + PW - 90;
                    return (
                      <g
                        key={p.student.id}
                        role="button"
                        tabIndex={0}
                        aria-label={`${p.student.name}: nota promedio ${p.grade.toFixed(1)}, nivel ${p.level}`}
                        onMouseEnter={() => setHover(p.student.id)}
                        onMouseLeave={() => setHover(null)}
                        onFocus={() => setHover(p.student.id)}
                        onBlur={() => setHover(null)}
                        onClick={() => onSelect(p.student.id)}
                        onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && onSelect(p.student.id)}
                        className="cursor-pointer outline-none"
                      >
                        <circle cx={p.x} cy={p.y} r={14} fill="transparent" />
                        <circle cx={p.x} cy={p.y} r={active ? 8 : 6} fill={active ? POINT_STRONG : POINT} stroke="#FFFFFF" strokeWidth={2} />
                        {labelNames && (
                          <text x={right ? p.x - 12 : p.x + 12} y={p.y + 4} fontSize={11} fill={INK_SOFT} textAnchor={right ? "end" : "start"}>
                            {p.student.name.split(" ")[0]}
                          </text>
                        )}
                      </g>
                    );
                  })}
                  {plotted.length === 0 && (
                    <text x={M.left + PW / 2} y={M.top + PH / 2} fontSize={13} fill={INK_SOFT} textAnchor="middle">
                      Aún no hay entregas devueltas con nivel en {rae.code}
                    </text>
                  )}
                </svg>

                {hovered && (
                  <div
                    className="pointer-events-none absolute z-10 w-48 -translate-x-1/2 -translate-y-full rounded-2xl bg-white px-3 py-2 shadow-lift ring-1 ring-lavender-100"
                    style={{ left: `${(hovered.x / W) * 100}%`, top: `calc(${(hovered.y / H) * 100}% - 14px)` }}
                  >
                    <p className="text-lg font-extrabold leading-tight text-ink">
                      {hovered.grade.toFixed(1)} <span className="text-xs font-semibold text-ink-faint">nota promedio</span>
                    </p>
                    <p className="text-sm font-bold text-ink">{hovered.student.name}</p>
                    <p className="text-xs text-ink-soft">
                      Nivel {hovered.level} en {rae.code} · {hovered.student.graded}{" "}
                      {hovered.student.graded === 1 ? "entrega calificada" : "entregas calificadas"}
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}

          <p className="text-xs text-ink-faint">
            Cada punto es un estudiante: su nota promedio en las entregas devueltas (eje X) y su nivel de logro del RAE según tus
            revisiones (eje Y). Haz clic en un punto para ver su proceso.
          </p>
          {missing.length > 0 && (
            <p className="text-xs text-ink-soft">
              <span className="font-bold">Sin datos para {rae.code}: </span>
              {missing.map((m) => `${m.student.name} (${m.reason})`).join(", ")}.
            </p>
          )}
        </div>
      )}
    </Card>
  );
}
