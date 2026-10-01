"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card, InlineCreate } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import { createCourse, createGroup, CourseSummary, listCourses } from "@/lib/api";

function CourseCard({ course, onChanged }: { course: CourseSummary; onChanged: () => Promise<void> }) {
  const students = course.groups.reduce((n, g) => n + g.student_count, 0);
  return (
    <Card
      className="rise-in"
      title={course.name}
      action={
        <span className="text-xs font-semibold text-ink-faint">
          {course.groups.length} {course.groups.length === 1 ? "clase" : "clases"} · {students}{" "}
          {students === 1 ? "estudiante" : "estudiantes"}
        </span>
      }
    >
      {course.groups.length === 0 ? (
        <p className="mb-3 text-sm text-ink-soft">
          Esta materia aún no tiene clases. Crea una y comparte su código con tus estudiantes.
        </p>
      ) : (
        <ul className="mb-4 space-y-2">
          {course.groups.map((g) => (
            <li key={g.id}>
              <Link
                href={`/docente/clases/${g.id}`}
                className="flex flex-wrap items-center gap-x-4 gap-y-1 rounded-2xl bg-lavender-50/70 px-4 py-3 transition
                  hover:bg-lavender-100"
              >
                <span className="min-w-0 flex-1 font-bold">{g.name}</span>
                <span className="rounded-full bg-white px-2.5 py-0.5 font-mono text-xs font-bold tracking-widest text-lavender-700">
                  {g.join_code}
                </span>
                <span className="text-xs font-semibold text-ink-soft">
                  {g.student_count} estudiantes · {g.submission_count} envíos
                </span>
                <svg className="h-4 w-4 text-ink-faint" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                  <path fillRule="evenodd" d="M7.21 14.77a.75.75 0 0 1 .02-1.06L11.17 10 7.23 6.29a.75.75 0 1 1 1.04-1.08l4.5 4.25a.75.75 0 0 1 0 1.08l-4.5 4.25a.75.75 0 0 1-1.06-.02Z" clipRule="evenodd" />
                </svg>
              </Link>
            </li>
          ))}
        </ul>
      )}
      <InlineCreate
        placeholder="Nueva clase (p. ej. Grupo 2)"
        button="Agregar clase"
        onCreate={async (name) => {
          await createGroup(course.id, name);
          await onChanged();
        }}
      />
    </Card>
  );
}

function CoursesDashboard() {
  const { user } = useAuth();
  const [courses, setCourses] = useState<CourseSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    try {
      setCourses(await listCourses());
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al cargar tus materias");
    }
  }, []);

  useEffect(() => {
    reload();
  }, [reload]);

  if (!user) return null;
  const firstName = user.name.split(" ")[0];

  return (
    <div className="mx-auto max-w-7xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <div className="flex items-center gap-4">
          <Avatar name={user.name} className="h-14 w-14 text-lg ring-4 ring-lavender-50" />
          <div>
            <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">¡Hola, {firstName}!</h1>
            <p className="text-ink-soft text-sm mt-0.5">
              Organiza tus materias y clases. EduTrace te muestra quién necesita acompañamiento; tú decides.
            </p>
          </div>
        </div>
      </section>

      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-3">
        <div className="space-y-5 lg:col-span-2">
          {!courses ? (
            <p className="text-sm text-ink-soft">Cargando...</p>
          ) : courses.length === 0 ? (
            <Card>
              <p className="font-bold">Aún no tienes materias</p>
              <p className="mt-1 text-sm text-ink-soft">
                Crea tu primera materia en el panel de la derecha. Luego agrégale una o varias clases y comparte
                el código de cada clase con tus estudiantes para que se unan y te envíen su código.
              </p>
            </Card>
          ) : (
            courses.map((c) => <CourseCard key={c.id} course={c} onChanged={reload} />)
          )}
        </div>

        <div className="space-y-5">
          <Card title="Nueva materia" className="rise-in" surface="bg-mint-50">
            <InlineCreate
              placeholder="Nombre (p. ej. Programación I)"
              button="Crear"
              onCreate={async (name) => {
                await createCourse(name);
                await reload();
              }}
            />
          </Card>
          <Card title="¿Cómo funciona?" className="rise-in" surface="bg-peach-50">
            <ol className="list-decimal space-y-2 pl-5 text-sm text-ink-soft">
              <li>Crea una materia y agrégale sus clases (grupos).</li>
              <li>Comparte el código de cada clase con tus estudiantes.</li>
              <li>Ellos se unen y te envían su código desde el compilador.</li>
              <li>Entra a una clase para ver a cada estudiante, sus envíos y la orientación que le dio la IA.</li>
            </ol>
          </Card>
        </div>
      </div>
    </div>
  );
}

export default function TeacherPage() {
  return (
    <RequireAuth role="teacher">
      <AppShell>
        <CoursesDashboard />
      </AppShell>
    </RequireAuth>
  );
}
