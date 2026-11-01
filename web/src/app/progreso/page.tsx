"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Card } from "@/components/ui";
import GradeChart from "@/components/GradeChart";
import { getMyProgress, ProgressGroup } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

const LEVEL_STYLE: Record<string, string> = {
  bajo: "bg-peach-100 text-peach-700",
  medio: "bg-amber-100 text-amber-900",
  alto: "bg-mint-100 text-mint-700",
};

function GroupProgress({ g }: { g: ProgressGroup }) {
  const graded = g.points.filter((p) => p.grade !== undefined).map((p) => ({ title: p.title, at: p.returned_at, grade: p.grade! }));
  return (
    <Card
      className="rise-in"
      title={`${g.course_name} · ${g.group_name}`}
      action={
        g.average !== undefined && (
          <span className="text-sm text-ink-soft">
            Promedio <span className="font-extrabold text-ink">{g.average.toFixed(1)}</span> / 5.0
          </span>
        )
      }
    >
      {g.points.length === 0 ? (
        <p className="text-sm text-ink-soft">Todavía no tienes revisiones devueltas en esta clase.</p>
      ) : (
        <div className="space-y-4">
          {graded.length > 0 && <GradeChart points={graded} label={`Tus notas en ${g.course_name}`} />}
          <details>
            <summary className="cursor-pointer text-sm font-bold text-lavender-700">Ver como tabla</summary>
            <table className="mt-2 w-full text-left text-sm">
              <thead className="text-xs text-ink-soft">
                <tr>
                  <th scope="col" className="py-1 pr-2">Taller</th>
                  <th scope="col" className="py-1 pr-2">Devuelta</th>
                  <th scope="col" className="py-1 pr-2 text-right">Casos</th>
                  <th scope="col" className="py-1 text-right">Nota</th>
                </tr>
              </thead>
              <tbody>
                {g.points.map((p, i) => (
                  <tr key={i} className="border-t border-lavender-100">
                    <td className="py-1 pr-2">{p.title}</td>
                    <td className="py-1 pr-2">{formatDateTime(p.returned_at)}</td>
                    <td className="py-1 pr-2 text-right tabular-nums">{p.tests_total ? `${p.tests_passed}/${p.tests_total}` : "—"}</td>
                    <td className="py-1 text-right font-bold tabular-nums">{p.grade !== undefined ? p.grade.toFixed(1) : "—"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </details>
          {g.raes.length > 0 && (
            <div>
              <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">Tu último nivel en cada resultado de aprendizaje</h3>
              <ul className="space-y-1.5">
                {g.raes.map((r) => (
                  <li key={r.code} className="flex items-start gap-2 text-sm">
                    <span
                      className={`shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-bold capitalize ${
                        r.latest ? LEVEL_STYLE[r.latest] : "bg-lavender-50 text-ink-soft"
                      }`}
                    >
                      {r.latest ?? "sin evaluar"}
                    </span>
                    <span>
                      <span className="font-bold">{r.code}</span> <span className="text-ink-soft">{r.description}</span>
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </Card>
  );
}

function ProgressView() {
  const [data, setData] = useState<{ groups: ProgressGroup[]; practice: { solved: number; total: number } } | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    getMyProgress()
      .then(setData)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar tu progreso"));
  }, []);

  return (
    <div className="mx-auto max-w-4xl space-y-5">
      <section className="rise-in flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <div>
          <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Mi progreso</h1>
          <p className="text-sm text-ink-soft">Cómo van tus notas y tus niveles por resultado de aprendizaje en cada clase.</p>
        </div>
        {data && (
          <Link href="/practica" className="rounded-2xl bg-mint-50 px-4 py-3 text-center hover:bg-mint-100">
            <span className="block text-2xl font-extrabold tabular-nums text-mint-700">
              {data.practice.solved} / {data.practice.total}
            </span>
            <span className="text-xs font-semibold text-mint-700">ejercicios de práctica resueltos</span>
          </Link>
        )}
      </section>
      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}
      {data && data.groups.length === 0 && (
        <p className="rounded-2xl bg-white p-4 text-sm text-ink-soft">Únete a una clase para ver aquí tu progreso.</p>
      )}
      {data?.groups.map((g) => <GroupProgress key={g.group_id} g={g} />)}
    </div>
  );
}

export default function ProgressPage() {
  return (
    <RequireAuth role="student">
      <AppShell>
        <ProgressView />
      </AppShell>
    </RequireAuth>
  );
}
