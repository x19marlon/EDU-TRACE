"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Card, Spinner } from "@/components/ui";
import LevelBar from "@/components/LevelBar";
import { ErrorList } from "@/components/ClassErrors";
import {
  CourseSummary,
  ErrorReport,
  getFrequentErrors,
  getGroup,
  getRAEMatrix,
  getStudentAlerts,
  GroupDetail,
  listCourses,
  listGroupSubmissions,
  RAEMatrix,
  StudentAlert,
  SubmissionMeta,
} from "@/lib/api";
import { isActive } from "@/lib/format";
import { gradeLevel, LEVEL_COLOR, raeBucket, raeSegments } from "@/lib/rae";

interface ClassData {
  id: string;
  name: string;
  detail: GroupDetail;
  matrix: RAEMatrix;
  alerts: StudentAlert[];
  submissions: SubmissionMeta[];
}

const counted = (n: number, one: string, many: string) => `${n} ${n === 1 ? one : many}`;

function classAverage(grades: number[]): number | undefined {
  return grades.length ? grades.reduce((a, b) => a + b, 0) / grades.length : undefined;
}

const pendingOf = (c: ClassData) => c.submissions.filter((m) => m.review_state?.status !== "returned").length;
const activeOf = (c: ClassData) => c.detail.students.filter(isActive).length;
const gradesOf = (c: ClassData) => c.matrix.students.flatMap((s) => (s.average_grade === undefined ? [] : [s.average_grade]));

function Overview({ classes }: { classes: ClassData[] }) {
  const students = classes.reduce((n, c) => n + c.detail.students.length, 0);
  const active = classes.reduce((n, c) => n + activeOf(c), 0);
  const submissions = classes.reduce((n, c) => n + c.submissions.length, 0);
  const pending = classes.reduce((n, c) => n + pendingOf(c), 0);
  const alerts = classes.reduce((n, c) => n + c.alerts.length, 0);
  const grades = classes.flatMap(gradesOf);
  const average = classAverage(grades);
  const raes = classes[0]?.matrix.raes ?? [];
  const everyone = classes.flatMap((c) => c.matrix.students);

  return (
    <>
      <p className="rise-in rounded-3xl bg-white p-5 text-sm leading-relaxed shadow-soft">
        Tienes {counted(students, "estudiante", "estudiantes")} en {counted(classes.length, "clase", "clases")}.{" "}
        {pending > 0 ? `${counted(pending, "entrega espera", "entregas esperan")} tu revisión` : "No tienes entregas por revisar"}
        {alerts > 0 ? ` y ${counted(alerts, "estudiante podría", "estudiantes podrían")} necesitar más acompañamiento.` : " y nadie necesita atención especial por ahora."}
      </p>

      <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
        {[
          { v: students, l: "estudiantes", c: "bg-lavender-50 text-lavender-700" },
          { v: `${active} de ${students}`, l: "activos esta semana", c: "bg-mint-50 text-mint-700" },
          { v: submissions, l: "entregas", c: "bg-sky-50 text-sky-800" },
          { v: pending, l: "por revisar", c: "bg-amber-50 text-amber-800" },
          { v: average === undefined ? "—" : average.toFixed(1), l: "nota promedio", c: "bg-peach-50 text-peach-700" },
        ].map((x) => (
          <div key={x.l} className={`rise-in rounded-2xl py-4 text-center shadow-soft ${x.c}`}>
            <p className="text-2xl font-extrabold tabular-nums">{x.v}</p>
            <p className="mt-0.5 text-xs font-semibold">{x.l}</p>
          </div>
        ))}
      </div>

      <Card title="Clases" className="rise-in">
        <div role="region" aria-label="Comparación de las clases" tabIndex={0} className="overflow-x-auto rounded-xl focus:outline-none focus-visible:ring-4 focus-visible:ring-lavender-100">
          <table className="w-full min-w-[34rem] text-left text-sm">
            <thead className="text-xs uppercase tracking-wider text-ink-faint">
              <tr>
                <th scope="col" className="py-2 pr-3 font-bold">Clase</th>
                <th scope="col" className="py-2 pr-3 text-right font-bold">Estudiantes</th>
                <th scope="col" className="py-2 pr-3 text-right font-bold">Activos</th>
                <th scope="col" className="py-2 pr-3 text-right font-bold">Entregas</th>
                <th scope="col" className="py-2 pr-3 text-right font-bold">Por revisar</th>
                <th scope="col" className="py-2 pr-3 text-right font-bold">Nota promedio</th>
                <th scope="col" className="py-2 text-right font-bold">Con alertas</th>
              </tr>
            </thead>
            <tbody>
              {classes.map((c) => {
                const avg = classAverage(gradesOf(c));
                return (
                  <tr key={c.id} className="border-t border-lavender-100">
                    <td className="min-w-[7rem] py-2 pr-3 font-bold">
                      <Link href={`/docente/clases/${c.id}`} className="text-lavender-700 hover:underline">
                        {c.name}
                      </Link>
                    </td>
                    <td className="py-2 pr-3 text-right tabular-nums">{c.detail.students.length}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{activeOf(c)}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{c.submissions.length}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{pendingOf(c)}</td>
                    <td className="py-2 pr-3 text-right tabular-nums">{avg === undefined ? "—" : avg.toFixed(1)}</td>
                    <td className="py-2 text-right tabular-nums">{c.alerts.length}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </Card>

      <Card title="Notas de la materia" className="rise-in">
        <p className="mb-3 text-sm text-ink-soft">Cuántos estudiantes hay en cada rango, según el promedio de sus entregas devueltas.</p>
        <LevelBar
          label="Estudiantes por rango de nota"
          unit={["estudiante", "estudiantes"]}
          segments={[
            { key: "bajo", label: "Menos de 3.0", color: LEVEL_COLOR.bajo, count: grades.filter((g) => gradeLevel(g) === "bajo").length },
            { key: "medio", label: "De 3.0 a 3.74", color: LEVEL_COLOR.medio, count: grades.filter((g) => gradeLevel(g) === "medio").length },
            { key: "alto", label: "De 3.75 a 5.0", color: LEVEL_COLOR.alto, count: grades.filter((g) => gradeLevel(g) === "alto").length },
            { key: "sin", label: "Sin nota", color: LEVEL_COLOR.sin_evaluar, count: everyone.length - grades.length },
          ]}
        />
      </Card>

      {raes.length > 0 && (
        <Card title="Resultados de aprendizaje (RAE)" className="rise-in">
          <p className="mb-3 text-sm text-ink-soft">En qué nivel va cada estudiante en cada RAE, según su última entrega devuelta que lo evaluó. Quien tiene una nota muy alejada de su nivel aparece como «para revisar».</p>
          <ul className="space-y-4">
            {raes.map((r) => (
              <li key={r.id}>
                <p className="mb-1 text-sm">
                  <span className="font-bold text-lavender-700">{r.code}</span> <span className="text-ink-soft">{r.description}</span>
                </p>
                <LevelBar
                  label={`Estudiantes por nivel en ${r.code}`}
                  unit={["estudiante", "estudiantes"]}
                  segments={raeSegments(everyone.map((s) => raeBucket(s, r.id)))}
                />
              </li>
            ))}
          </ul>
        </Card>
      )}
    </>
  );
}

function CommonErrors({ report }: { report: ErrorReport }) {
  return (
    <Card title="Errores más frecuentes en la materia" className="rise-in">
      <div className="grid gap-4 lg:grid-cols-2">
        <ErrorList title="Errores de compilación" items={report.compile.slice(0, 3)} empty="Nadie ha tenido errores de compilación." />
        <ErrorList title="Problemas al ejecutar" items={report.runtime.slice(0, 3)} empty="Ningún programa falló al ejecutarse." />
      </div>
    </Card>
  );
}

function CourseProgress() {
  const wanted = useSearchParams().get("materia");
  const [courses, setCourses] = useState<CourseSummary[] | null>(null);
  const [courseId, setCourseId] = useState<string | null>(null);
  const [classes, setClasses] = useState<ClassData[] | null>(null);
  const [errors, setErrors] = useState<ErrorReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listCourses()
      .then((list) => {
        setCourses(list);
        const preferred = list.find((c) => c.id === wanted) ?? list.find((c) => !c.closed_at) ?? list[list.length - 1];
        setCourseId(preferred?.id ?? null);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudieron cargar tus materias"));
  }, [wanted]);

  const course = useMemo(() => courses?.find((c) => c.id === courseId), [courses, courseId]);

  useEffect(() => {
    if (!course) return;
    let stale = false;
    setClasses(null);
    setErrors(null);
    setError(null);
    Promise.all(
      course.groups.map(async (g) => {
        const [detail, matrix, alerts, submissions] = await Promise.all([
          getGroup(g.id),
          getRAEMatrix(g.id),
          getStudentAlerts(g.id),
          listGroupSubmissions(g.id),
        ]);
        return { id: g.id, name: g.name, detail, matrix, alerts, submissions };
      })
    )
      .then((list) => !stale && setClasses(list))
      .catch((err) => !stale && setError(err instanceof Error ? err.message : "No se pudo calcular cómo va el curso"));
    if (course.groups.length > 0) {
      getFrequentErrors(course.groups[0].id, true)
        .then((report) => !stale && setErrors(report))
        .catch(() => !stale && setErrors(null));
    }
    return () => {
      stale = true;
    };
  }, [course]);

  return (
    <div className="mx-auto max-w-6xl space-y-5">
      <section className="rise-in flex flex-wrap items-end justify-between gap-4 rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <div className="min-w-0 flex-1 basis-72">
          <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Cómo va el curso</h1>
          <p className="mt-0.5 text-sm text-ink-soft">
            Una mirada general a tu materia: participación, revisiones pendientes, notas y resultados de aprendizaje de todas sus clases.
          </p>
        </div>
        {courses && courses.length > 1 && (
          <label className="text-sm font-semibold text-ink-soft">
            <span className="mb-1 block">Materia</span>
            <select
              value={courseId ?? ""}
              onChange={(e) => setCourseId(e.target.value)}
              className="max-w-full rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
            >
              {courses.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                  {c.closed_at ? " (periodo cerrado)" : ""}
                </option>
              ))}
            </select>
          </label>
        )}
      </section>

      {error && <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>}
      {courses && courses.length === 0 && (
        <Card>
          <p className="font-bold">Aún no tienes materias</p>
          <p className="mt-1 text-sm text-ink-soft">
            Crea tu primera materia en{" "}
            <Link href="/docente" className="font-bold text-lavender-600 hover:text-lavender-700">
              Mis materias
            </Link>{" "}
            y aquí verás cómo va.
          </p>
        </Card>
      )}
      {course && course.groups.length === 0 && (
        <Card>
          <p className="text-sm text-ink-soft">
            «{course.name}» aún no tiene clases. Cuando agregues una y tus estudiantes empiecen a trabajar, aquí verás cómo va.
          </p>
        </Card>
      )}
      {course && course.groups.length > 0 && !classes && !error && (
        <p className="flex items-center gap-2 text-sm text-ink-soft">
          <Spinner className="h-4 w-4 border-lavender-500" /> Calculando...
        </p>
      )}
      {classes && classes.length > 0 && <Overview classes={classes} />}
      {classes && classes.length > 0 && errors && <CommonErrors report={errors} />}
    </div>
  );
}

export default function CourseProgressPage() {
  return (
    <RequireAuth role="teacher">
      <AppShell>
        <Suspense fallback={<p className="text-sm text-ink-soft">Cargando...</p>}>
          <CourseProgress />
        </Suspense>
      </AppShell>
    </RequireAuth>
  );
}
