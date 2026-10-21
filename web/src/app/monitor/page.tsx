"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import { listMonitorGroups, MonitorGroup } from "@/lib/api";

function MonitorDashboard() {
  const { user } = useAuth();
  const [groups, setGroups] = useState<MonitorGroup[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listMonitorGroups()
      .then(setGroups)
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar tus clases"));
  }, []);

  if (!user) return null;

  return (
    <div className="mx-auto max-w-7xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <div className="flex items-center gap-4">
          <Avatar name={user.name} className="h-14 w-14 text-lg ring-4 ring-mint-50" />
          <div>
            <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">¡Hola, {user.name.split(" ")[0]}!</h1>
            <p className="mt-0.5 text-sm text-ink-soft">
              Acompaña a los estudiantes de tus clases con retroalimentación sobre su código. El profesor decide las notas.
            </p>
          </div>
        </div>
      </section>

      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        <div className="space-y-3 lg:col-span-2">
          {!groups ? (
            <p className="text-sm text-ink-soft">Cargando...</p>
          ) : groups.length === 0 ? (
            <Card>
              <p className="font-bold">Aún no eres monitor de ninguna clase</p>
              <p className="mt-1 text-sm text-ink-soft">
                Pídele a tu profesor que te agregue como monitor en su clase con tu email:{" "}
                <span className="font-bold text-ink">{user.email}</span>.
              </p>
            </Card>
          ) : (
            groups.map((g) => (
              <Link
                key={g.id}
                href={`/monitor/clases/${g.id}`}
                className="rise-in flex flex-wrap items-center gap-x-4 gap-y-1 rounded-3xl border border-lavender-100/70 bg-white px-5 py-4 shadow-soft transition hover:bg-lavender-50"
              >
                <span className="min-w-0 flex-1">
                  <span className="block text-xs font-bold uppercase tracking-wider text-lavender-600">{g.course_name}</span>
                  <span className="block font-bold">{g.name}</span>
                  <span className="block text-xs text-ink-soft">Profesor: {g.teacher_name}</span>
                </span>
                <span className="text-xs font-semibold text-ink-soft">
                  {g.student_count} {g.student_count === 1 ? "estudiante" : "estudiantes"} · {g.submission_count}{" "}
                  {g.submission_count === 1 ? "entrega" : "entregas"}
                </span>
                <svg className="h-4 w-4 text-ink-faint" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                  <path fillRule="evenodd" d="M7.21 14.77a.75.75 0 0 1 .02-1.06L11.17 10 7.23 6.29a.75.75 0 1 1 1.04-1.08l4.5 4.25a.75.75 0 0 1 0 1.08l-4.5 4.25a.75.75 0 0 1-1.06-.02Z" clipRule="evenodd" />
                </svg>
              </Link>
            ))
          )}
        </div>

        <Card title="Tu papel como monitor" className="rise-in self-start" surface="bg-mint-50">
          <ul className="list-disc space-y-2 pl-5 text-sm text-ink-soft">
            <li>Ves las entregas oficiales de las clases donde el profesor te agregó.</li>
            <li>Escribes retroalimentación sobre el código: qué está bien, qué revisar y por qué.</li>
            <li>Das pistas y preguntas guía; no entregues la solución.</li>
            <li>No calificas ni ves las notas: eso lo decide el profesor.</li>
          </ul>
        </Card>
      </div>
    </div>
  );
}

export default function MonitorPage() {
  return (
    <RequireAuth role="monitor">
      <AppShell>
        <MonitorDashboard />
      </AppShell>
    </RequireAuth>
  );
}
