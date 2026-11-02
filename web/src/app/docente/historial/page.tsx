"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Card } from "@/components/ui";
import { CourseSummary, listCourses } from "@/lib/api";

const formatDate = (iso: string) => new Date(iso).toLocaleDateString("es", { dateStyle: "medium" });

function CourseRow({ course }: { course: CourseSummary }) {
  const students = course.groups.reduce((n, g) => n + g.student_count, 0);
  const submissions = course.groups.reduce((n, g) => n + g.submission_count, 0);
  return (
    <Card
      className="rise-in"
      title={course.name}
      action={
        course.closed_at ? (
          <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-bold text-amber-900">Periodo cerrado</span>
        ) : (
          <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-xs font-bold text-mint-700">En curso</span>
        )
      }
    >
      <p className="text-sm text-ink-soft">
        Creada el {formatDate(course.created_at)}
        {course.closed_at && `, cerrada el ${formatDate(course.closed_at)}`}.
        {course.purge_at && ` Sus datos se borrarán el ${formatDate(course.purge_at)}.`}
      </p>
      <p className="mt-1 text-sm font-semibold">
        {course.groups.length} {course.groups.length === 1 ? "clase" : "clases"}, {students} {students === 1 ? "estudiante" : "estudiantes"},{" "}
        {submissions} {submissions === 1 ? "entrega" : "entregas"}
      </p>
      {(course.teachers ?? []).length > 1 && (
        <p className="mt-1 text-sm text-ink-soft">Profesores: {course.teachers.map((t) => t.name).join(", ")}</p>
      )}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {course.groups.map((g) => (
          <Link
            key={g.id}
            href={`/docente/clases/${g.id}`}
            className="rounded-full bg-lavender-50 px-3 py-1 text-xs font-bold text-lavender-700 transition hover:bg-lavender-100"
          >
            {g.name}
          </Link>
        ))}
        <Link href={`/docente/progreso?materia=${course.id}`} className="ml-auto text-sm font-bold text-lavender-600 hover:text-lavender-700">
          Ver cómo va
        </Link>
      </div>
    </Card>
  );
}

function CourseHistory() {
  const [courses, setCourses] = useState<CourseSummary[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listCourses()
      .then((list) => setCourses([...list].sort((a, b) => b.created_at.localeCompare(a.created_at))))
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar tu historial de cursos"));
  }, []);

  const open = courses?.filter((c) => !c.closed_at) ?? [];
  const closed = courses?.filter((c) => c.closed_at) ?? [];

  return (
    <div className="mx-auto max-w-4xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Historial de cursos</h1>
        <p className="mt-0.5 text-sm text-ink-soft">
          Aquí encuentras todas tus materias, las que siguen en curso y las de periodos anteriores, con sus clases y cifras principales.
        </p>
      </section>

      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}
      {!courses && !error && <p className="text-sm text-ink-soft">Cargando...</p>}
      {courses && courses.length === 0 && (
        <Card>
          <p className="font-bold">Aún no tienes cursos</p>
          <p className="mt-1 text-sm text-ink-soft">
            Cuando crees tu primera materia en{" "}
            <Link href="/docente" className="font-bold text-lavender-600 hover:text-lavender-700">
              Mis materias
            </Link>
            , aparecerá aquí.
          </p>
        </Card>
      )}

      {open.length > 0 && (
        <section aria-labelledby="en-curso" className="space-y-3">
          <h2 id="en-curso" className="text-lg font-extrabold">En curso</h2>
          {open.map((c) => (
            <CourseRow key={c.id} course={c} />
          ))}
        </section>
      )}
      {closed.length > 0 && (
        <section aria-labelledby="anteriores" className="space-y-3">
          <h2 id="anteriores" className="text-lg font-extrabold">Periodos anteriores</h2>
          {closed.map((c) => (
            <CourseRow key={c.id} course={c} />
          ))}
          <p className="text-xs text-ink-soft">
            Cuando se cumple el plazo de conservación de un periodo cerrado, sus datos se borran y deja de aparecer aquí.
          </p>
        </section>
      )}
    </div>
  );
}

export default function CourseHistoryPage() {
  return (
    <RequireAuth role="teacher">
      <AppShell>
        <CourseHistory />
      </AppShell>
    </RequireAuth>
  );
}
