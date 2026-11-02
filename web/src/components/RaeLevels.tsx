"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getRAEMatrix, RAELevel, RAEMatrix, StudentPoint } from "@/lib/api";
import { Card, SmallButton } from "@/components/ui";
import LevelBar from "@/components/LevelBar";
import { gradeLevel, gradeNudge, LEVEL_INFO, NUDGE_NOTE, raeBucket, raeReading, raeSegments } from "@/lib/rae";

const COLUMNS: RAELevel[] = ["bajo", "medio", "alto"];
const COLUMN_SURFACE: Record<RAELevel, string> = {
  bajo: "bg-peach-50",
  medio: "bg-amber-50",
  alto: "bg-mint-50",
};

const NUDGE_MARK = { up: "↑", down: "↓" } as const;

const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1);

export default function RaeLevels({
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

  const groups = useMemo(() => {
    const out: Record<ReturnType<typeof raeBucket>, StudentPoint[]> = { bajo: [], medio: [], alto: [], revisar: [], sin_evaluar: [] };
    if (!matrix || !raeId) return out;
    for (const s of matrix.students) out[raeBucket(s, raeId)].push(s);
    for (const level of COLUMNS) out[level].sort((a, b) => a.standings[raeId].grade - b.standings[raeId].grade);
    return out;
  }, [matrix, raeId]);

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

      {matrix && rae && raeId && (
        <div className="space-y-4">
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

          <p className="text-sm text-ink-soft">
            Aquí ves en qué nivel va cada estudiante en este RAE, según su última entrega devuelta que lo evaluó. El nivel lo marcas tú, y la
            nota también pesa: si se aleja un poco del rango habitual, lo señalamos con una flecha; si se aleja medio punto o más, queda para
            revisar.
          </p>

          <LevelBar
            label={`Estudiantes por nivel en ${rae.code}`}
            unit={["estudiante", "estudiantes"]}
            segments={raeSegments(matrix.students.map((s) => raeBucket(s, raeId)))}
          />

          {table ? (
            <div role="region" aria-label={`Tabla de niveles en ${rae.code}`} tabIndex={0} className="overflow-x-auto rounded-xl focus:outline-none focus-visible:ring-4 focus-visible:ring-lavender-100">
              <table className="w-full text-left text-sm">
                <thead className="text-xs uppercase tracking-wider text-ink-faint">
                  <tr>
                    <th className="py-2 pr-3 font-bold">Estudiante</th>
                    <th className="py-2 pr-3 font-bold">Nivel en {rae.code}</th>
                    <th className="py-2 pr-3 font-bold">Nota de esa entrega</th>
                    <th className="py-2 pr-3 font-bold">Lectura</th>
                    <th className="py-2 pr-3 font-bold">Promedio general</th>
                    <th className="py-2 pr-3 font-bold">Entregas calificadas</th>
                  </tr>
                </thead>
                <tbody>
                  {matrix.students.map((s) => {
                    const standing = s.standings[raeId];
                    return (
                      <tr key={s.id} className="border-t border-lavender-100">
                        <td className="py-2 pr-3 font-semibold">{s.name}</td>
                        <td className="py-2 pr-3">
                          {standing ? capitalize(standing.level) : "Sin evaluar"}
                          {raeBucket(s, raeId) === "revisar" && <span className="font-semibold text-lavender-700"> (para revisar)</span>}
                        </td>
                        <td className="py-2 pr-3 tabular-nums">{standing ? standing.grade.toFixed(1) : "—"}</td>
                        <td className="min-w-[14rem] py-2 pr-3 text-xs text-ink-soft">{standing ? raeReading(standing.level, standing.grade) : "—"}</td>
                        <td className="py-2 pr-3 tabular-nums">{s.average_grade === undefined ? "—" : s.average_grade.toFixed(1)}</td>
                        <td className="py-2 pr-3 tabular-nums">{s.graded}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="grid gap-3 md:grid-cols-3">
              {COLUMNS.map((level) => (
                <section key={level} aria-label={`Nivel ${level}`} className={`rounded-2xl p-3 ${COLUMN_SURFACE[level]}`}>
                  <div className="flex items-baseline justify-between gap-2">
                    <h3 className="font-extrabold">{LEVEL_INFO[level].label}</h3>
                    <span className="text-sm font-bold tabular-nums">{groups[level].length}</span>
                  </div>
                  <p className="text-xs font-semibold text-ink">{LEVEL_INFO[level].range}</p>
                  <p className="text-xs text-ink-soft">{LEVEL_INFO[level].meaning}</p>
                  {groups[level].length === 0 ? (
                    <p className="mt-3 text-xs text-ink-soft">Nadie por ahora.</p>
                  ) : (
                    <ul className="mt-3 flex flex-wrap gap-1.5">
                      {groups[level].map((s) => {
                        const nudge = gradeNudge(s.standings[raeId].grade, level);
                        return (
                          <li key={s.id}>
                            <button
                              type="button"
                              aria-pressed={selectedId === s.id}
                              title={raeReading(level, s.standings[raeId].grade)}
                              onClick={() => onSelect(s.id)}
                              className={`rounded-full bg-white px-3 py-1 text-left text-xs font-semibold transition hover:shadow-soft ${
                                selectedId === s.id ? "ring-2 ring-lavender-600" : "ring-1 ring-lavender-100"
                              }`}
                            >
                              {s.name} <span className="font-extrabold tabular-nums">{s.standings[raeId].grade.toFixed(1)}</span>
                              {nudge && (
                                <>
                                  <span aria-hidden className="ml-1 font-extrabold text-lavender-700">{NUDGE_MARK[nudge]}</span>
                                  <span className="sr-only">. {raeReading(level, s.standings[raeId].grade)}</span>
                                </>
                              )}
                            </button>
                          </li>
                        );
                      })}
                    </ul>
                  )}
                  {(["up", "down"] as const)
                    .filter((nudge) => groups[level].some((s) => gradeNudge(s.standings[raeId].grade, level) === nudge))
                    .map((nudge) => (
                      <p key={nudge} className="mt-2 text-xs text-ink-soft">
                        <span aria-hidden className="mr-1 font-extrabold text-lavender-700">{NUDGE_MARK[nudge]}</span>
                        {NUDGE_NOTE[level][nudge]}
                      </p>
                    ))}
                </section>
              ))}
            </div>
          )}

          {groups.revisar.length > 0 && (
            <div className="rounded-2xl border border-lavender-200 bg-lavender-50/60 p-3">
              <h3 className="text-sm font-bold">Para revisar: la nota y el nivel se alejan ({groups.revisar.length})</h3>
              <p className="mt-0.5 text-xs text-ink-soft">
                La nota de su última entrega está a medio punto o más del rango habitual del nivel que quedó marcado. Puede tener una explicación,
                pero conviene revisar si hay que ajustar alguno de los dos. Mientras tanto, no se cuentan en ninguna columna.
              </p>
              <ul className="mt-2 space-y-1">
                {groups.revisar.map((s) => {
                  const standing = s.standings[raeId];
                  return (
                    <li key={s.id} className="flex flex-wrap items-center justify-between gap-2 rounded-xl bg-white px-3 py-1.5 text-sm">
                      <span>
                        <span className="font-bold">{s.name}:</span> nivel {standing.level} con nota {standing.grade.toFixed(1)}. Con esa nota se
                        esperaría nivel {gradeLevel(standing.grade)}.
                      </span>
                      <SmallButton onClick={() => onSelect(s.id)}>Ver su proceso</SmallButton>
                    </li>
                  );
                })}
              </ul>
            </div>
          )}

          {groups.sin_evaluar.length > 0 && !table && (
            <p className="text-xs text-ink-soft">
              <span className="font-bold">Sin evaluar en {rae.code}: </span>
              {groups.sin_evaluar.map((s) => s.name).join(", ")}. Aún no tienen una entrega devuelta con nota y nivel en este RAE.
            </p>
          )}
        </div>
      )}
    </Card>
  );
}
