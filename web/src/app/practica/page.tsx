"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import { LEARNER_ROLES } from "@/lib/role";
import AppShell from "@/components/AppShell";
import { Card } from "@/components/ui";
import { Exercise, listExercises } from "@/lib/api";

const LEVELS = [
  null,
  { label: "Básico", tone: "bg-green-100 text-green-800" },
  { label: "Medio", tone: "bg-yellow-100 text-yellow-900" },
  { label: "Difícil", tone: "bg-red-100 text-red-800" },
];

function status(e: Exercise) {
  if (e.result?.solved_at) return { label: "Resuelto", tone: "bg-mint-100 text-mint-700" };
  if (e.result) return { label: `Intentado: ${e.result.best_passed} de ${e.result.total}`, tone: "bg-lavender-100 text-lavender-700" };
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
          Aquí encontrarás ejercicios para practicar a tu propio ritmo y reforzar lo que has aprendido. También te sirven para prepararte para
          el parcial. Al compilar, cada ejercicio se revisa automáticamente con sus propios casos de prueba. No tienes que enviarlos a tu
          profesor: son solo para que practiques y veas cómo vas avanzando.
        </p>
        {exercises && (
          <p className="mt-2 text-sm font-bold text-ink">
            Has resuelto {solved} de {exercises.length} ejercicios.
          </p>
        )}
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
                      <span className="flex flex-wrap items-center gap-2 text-xs text-ink-soft">
                        <span className={`rounded-full px-2.5 py-0.5 font-bold ${LEVELS[e.level]?.tone ?? ""}`}>{LEVELS[e.level]?.label}</span>
                        {e.tests.length} casos de prueba
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
    <RequireAuth role={LEARNER_ROLES}>
      <AppShell>
        <PracticeView />
      </AppShell>
    </RequireAuth>
  );
}
