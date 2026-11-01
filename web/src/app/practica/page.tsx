"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Card } from "@/components/ui";
import { Exercise, listExercises } from "@/lib/api";

const LEVELS = ["", "Básico", "Intermedio", "Avanzado"];

function status(e: Exercise) {
  if (e.result?.solved_at) return { label: "Resuelto", tone: "bg-mint-100 text-mint-700" };
  if (e.result) return { label: `Intentado · ${e.result.best_passed}/${e.result.total}`, tone: "bg-amber-100 text-amber-900" };
  return { label: "Sin empezar", tone: "bg-lavender-50 text-ink-soft" };
}

function PracticeView() {
  const [exercises, setExercises] = useState<Exercise[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    listExercises()
      .then(setExercises)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudieron cargar los ejercicios"));
  }, []);
  const topics = useMemo(() => [...new Set((exercises ?? []).map((e) => e.topic))], [exercises]);
  const solved = (exercises ?? []).filter((e) => e.result?.solved_at).length;

  return (
    <div className="mx-auto max-w-5xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Práctica</h1>
        <p className="mt-1 text-sm text-ink-soft">
          Ejercicios para practicar por tu cuenta. Cada uno se corrige solo con sus casos de prueba al compilar, y no se envía a tu
          profesor.
          {exercises && ` Llevas ${solved} de ${exercises.length} resueltos.`}
        </p>
      </section>
      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}
      {topics.map((topic) => (
        <Card key={topic} title={topic} className="rise-in">
          <ul className="grid gap-2 sm:grid-cols-2">
            {exercises!
              .filter((e) => e.topic === topic)
              .map((e) => {
                const st = status(e);
                return (
                  <li key={e.id}>
                    <Link
                      href={`/?ejercicio=${e.id}`}
                      className="flex h-full flex-col gap-1 rounded-2xl bg-lavender-50/70 px-4 py-3 transition hover:bg-lavender-100"
                    >
                      <span className="flex flex-wrap items-center gap-2">
                        <span className="min-w-0 flex-1 font-bold">{e.title}</span>
                        <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${st.tone}`}>{st.label}</span>
                      </span>
                      <span className="text-xs text-ink-soft">
                        {LEVELS[e.level]} · {e.tests.length} casos de prueba
                      </span>
                    </Link>
                  </li>
                );
              })}
          </ul>
        </Card>
      ))}
    </div>
  );
}

export default function PracticePage() {
  return (
    <RequireAuth role="student">
      <AppShell>
        <PracticeView />
      </AppShell>
    </RequireAuth>
  );
}
