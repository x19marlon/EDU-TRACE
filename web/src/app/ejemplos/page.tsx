"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import { LEARNER_ROLES } from "@/lib/role";
import AppShell from "@/components/AppShell";
import { Card } from "@/components/ui";
import SyntaxExampleView, { byTopic } from "@/components/SyntaxExampleView";
import { ClassExamples, listMyExamples, SyntaxExample } from "@/lib/api";

function ExampleItem({ example }: { example: SyntaxExample }) {
  const [open, setOpen] = useState(false);
  return (
    <details className="rounded-2xl ring-1 ring-lavender-100" onToggle={(e) => setOpen(e.currentTarget.open)}>
      <summary className="cursor-pointer rounded-2xl px-4 py-3 font-bold hover:bg-lavender-50">{example.title}</summary>
      {open && (
        <div className="px-4 pb-4">
          <SyntaxExampleView
            example={example}
            action={
              <Link
                href={`/?ejemplo=${encodeURIComponent(example.id)}`}
                className="rounded-full bg-lavender-600 px-3 py-1 text-xs font-bold text-white transition hover:bg-lavender-700"
              >
                Probar en el compilador
              </Link>
            }
          />
        </div>
      )}
    </details>
  );
}

function ExamplesView() {
  const [classes, setClasses] = useState<ClassExamples[] | null>(null);
  const [classId, setClassId] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listMyExamples()
      .then((list) => {
        setClasses(list);
        setClassId((list.find((c) => c.examples.length > 0) ?? list[0])?.id ?? "");
      })
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudieron cargar los ejemplos"));
  }, []);

  const current = classes?.find((c) => c.id === classId);
  const topics = useMemo(() => byTopic(current?.examples ?? []), [current]);

  return (
    <div className="mx-auto max-w-5xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Ejemplos</h1>
        <p className="mt-1 text-sm text-ink-soft">
          Ejemplos de sintaxis de C++ que tu profesor dejó para tu clase. Puedes copiarlos o abrirlos en el compilador para ver qué pasa
          al cambiarlos.
        </p>
        {classes && classes.length > 1 && (
          <label className="mt-3 flex flex-wrap items-center gap-2 text-sm font-semibold">
            Clase
            <select
              value={classId}
              onChange={(e) => setClassId(e.target.value)}
              className="max-w-full rounded-xl border border-lavender-200 bg-white px-3 py-1.5 text-sm"
            >
              {classes.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.course_name} · {c.name} ({c.examples.length})
                </option>
              ))}
            </select>
          </label>
        )}
      </section>

      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}
      {!classes && !error && <p className="text-sm text-ink-soft">Cargando...</p>}
      {classes?.length === 0 && (
        <p className="rounded-2xl bg-white p-4 text-sm text-ink-soft shadow-soft">
          Únete a una clase para ver los ejemplos que deja tu profesor.
        </p>
      )}
      {current && current.examples.length === 0 && (
        <p className="rounded-2xl bg-white p-4 text-sm text-ink-soft shadow-soft">Tu profesor todavía no ha agregado ejemplos en esta clase.</p>
      )}

      {topics.map(({ topic, examples }) => (
        <Card key={`${classId}-${topic}`} title={topic} className="rise-in">
          <div className="space-y-2">
            {examples.map((e) => (
              <ExampleItem key={e.id} example={e} />
            ))}
          </div>
        </Card>
      ))}
    </div>
  );
}

export default function ExamplesPage() {
  return (
    <RequireAuth role={LEARNER_ROLES}>
      <AppShell>
        <ExamplesView />
      </AppShell>
    </RequireAuth>
  );
}
