"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card, InlineCreate, RenameForm, SmallButton } from "@/components/ui";
import RaeEditor from "@/components/RaeEditor";
import { useAuth } from "@/components/AuthProvider";
import {
  addCoTeacher,
  AIStatus,
  cloneCourse,
  closeCourse,
  removeCoTeacher,
  createCourse,
  createGroup,
  CourseSummary,
  deleteCourse,
  getAIStatus,
  listCourses,
  listOpenQuestions,
  OpenQuestion,
  renameCourse,
  reopenCourse,
} from "@/lib/api";
import { formatDateTime, relativeTime } from "@/lib/format";

function QuestionsInbox() {
  const [questions, setQuestions] = useState<OpenQuestion[]>([]);
  useEffect(() => {
    listOpenQuestions()
      .then(setQuestions)
      .catch(() => setQuestions([]));
  }, []);
  if (questions.length === 0) return null;
  return (
    <Card title={`Preguntas de tus estudiantes (${questions.length})`} className="rise-in" surface="bg-amber-50">
      <ul className="space-y-2">
        {questions.slice(0, 8).map((q) => (
          <li key={q.id}>
            <Link href={`/docente/clases/${q.group_id}`} className="block rounded-2xl bg-white px-3 py-2 transition hover:bg-lavender-50">
              <span className="block text-xs font-bold text-ink-soft">
                {q.student_name} · {q.group_name}
                {q.assignment_title ? ` · ${q.assignment_title}` : ""} · {relativeTime(q.created_at).toLowerCase()}
              </span>
              <span className="block truncate text-sm">{q.body}</span>
            </Link>
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs text-ink-faint">Respóndelas desde la entrega del estudiante, debajo del código.</p>
    </Card>
  );
}

function AIAlert() {
  const [status, setStatus] = useState<AIStatus | null>(null);
  useEffect(() => {
    getAIStatus()
      .then(setStatus)
      .catch(() => setStatus(null));
  }, []);
  if (!status || (status.available && status.failed.length === 0)) return null;
  return (
    <section role="alert" className="rise-in space-y-2 rounded-3xl bg-peach-50 p-5 text-sm text-peach-700">
      {!status.available && (
        <p>
          <span className="font-bold">La IA no está disponible.</span> Las entregas se guardan intactas
          {status.pending > 0 && ` y ${status.pending} ${status.pending === 1 ? "propuesta espera" : "propuestas esperan"} en la cola`}. Puedes
          revisar sin IA mientras se restablece.
        </p>
      )}
      {status.failed.map((f) => (
        <p key={f.group_id}>
          <span className="font-bold">
            La IA no pudo preparar {f.failed} {f.failed === 1 ? "propuesta" : "propuestas"} en {f.course_name} · {f.group_name}.
          </span>{" "}
          {f.last_error}{" "}
          <Link href={`/docente/clases/${f.group_id}`} className="font-bold underline">
            Ver la clase
          </Link>
        </p>
      ))}
    </section>
  );
}

function CourseTeachers({ course, onChanged }: { course: CourseSummary; onChanged: () => Promise<void> }) {
  const { user } = useAuth();
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const run = async (action: () => Promise<unknown>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo cambiar los profesores");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="mb-4 rounded-2xl bg-lavender-50/60 px-4 py-3 text-sm">
      <p className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Profesores</p>
      <ul className="flex flex-wrap gap-1.5">
        {(course.teachers ?? []).map((t) => (
          <li key={t.id} className="flex items-center gap-1 rounded-full bg-white py-0.5 pl-3 pr-1 ring-1 ring-lavender-100">
            <span className="font-semibold">{t.name}</span>
            {t.owner && <span className="text-xs text-ink-soft">(dueño)</span>}
            {!t.owner && (course.owner || t.id === user?.id) && (
              <SmallButton
                tone="danger"
                disabled={busy}
                onClick={() => {
                  const self = t.id === user?.id;
                  if (!window.confirm(self ? `¿Salir de «${course.name}»? Tus revisiones pasan al dueño de la materia.` : `¿Retirar a ${t.name} de «${course.name}»?`)) return;
                  run(() => removeCoTeacher(course.id, t.id));
                }}
              >
                {t.id === user?.id ? "Salir" : "Quitar"}
              </SmallButton>
            )}
          </li>
        ))}
      </ul>
      {course.owner && (
        <form
          className="mt-2 flex flex-wrap gap-2"
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              await addCoTeacher(course.id, email.trim());
              setEmail("");
            });
          }}
        >
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email de otro profesor"
            aria-label="Email del profesor que compartirá la materia"
            className="min-w-0 flex-1 rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
          <button
            type="submit"
            disabled={busy || !email.trim()}
            className="rounded-full bg-white px-4 py-1.5 text-sm font-bold text-lavender-700 ring-1 ring-lavender-200 hover:bg-lavender-50 disabled:opacity-50"
          >
            Compartir materia
          </button>
        </form>
      )}
      {error && <p className="mt-2 rounded-2xl bg-peach-50 px-3 py-1.5 text-sm font-semibold text-peach-700">{error}</p>}
    </div>
  );
}

function CourseCard({ course, onChanged }: { course: CourseSummary; onChanged: () => Promise<void> }) {
  const students = course.groups.reduce((n, g) => n + g.student_count, 0);
  const [renaming, setRenaming] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const remove = async () => {
    const n = course.groups.length;
    const detail = n > 0 ? ` Se borrarán sus ${n} ${n === 1 ? "clase" : "clases"} con todos sus talleres, entregas y revisiones.` : "";
    if (!window.confirm(`¿Eliminar la materia «${course.name}»?${detail} No se puede deshacer.`)) return;
    setError(null);
    try {
      await deleteCourse(course.id);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo eliminar la materia");
    }
  };

  const copy = async () => {
    const name = window.prompt(
      `Nombre de la copia de «${course.name}». Se copian las clases (sin estudiantes y con códigos nuevos), los talleres con sus archivos, casos y rúbricas, los RAE y los profesores. Las fechas límite quedan vacías.`,
      `${course.name} (copia)`
    );
    if (name === null) return;
    const closeOld = !course.closed_at && window.confirm(`¿Cerrar también el periodo de «${course.name}»?`);
    setError(null);
    try {
      await cloneCourse(course.id, name.trim(), closeOld);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo copiar la materia");
    }
  };

  const togglePeriod = async () => {
    const question = course.closed_at
      ? `¿Reabrir el periodo de «${course.name}»? Los estudiantes podrán volver a enviar y los datos ya no se borrarán.`
      : `¿Cerrar el periodo de «${course.name}»? Los estudiantes ya no podrán enviar ni unirse, y recibirán un aviso con la fecha en que se borrarán sus entregas. Tú podrás seguir revisando y exportando hasta esa fecha.`;
    if (!window.confirm(question)) return;
    setError(null);
    try {
      if (course.closed_at) await reopenCourse(course.id);
      else await closeCourse(course.id);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo cambiar el periodo");
    }
  };

  return (
    <Card
      className="rise-in"
      title={course.name}
      action={
        <span className="flex flex-wrap items-center justify-end gap-1">
          <span className="mr-1 text-xs font-semibold text-ink-soft">
            {course.groups.length} {course.groups.length === 1 ? "clase" : "clases"} · {students}{" "}
            {students === 1 ? "estudiante" : "estudiantes"}
          </span>
          <SmallButton onClick={() => setRenaming(true)}>Renombrar</SmallButton>
          <SmallButton onClick={copy}>Copiar para otro periodo</SmallButton>
          <SmallButton onClick={togglePeriod}>{course.closed_at ? "Reabrir periodo" : "Cerrar periodo"}</SmallButton>
          {course.owner && (
            <SmallButton tone="danger" onClick={remove}>
              Eliminar
            </SmallButton>
          )}
        </span>
      }
    >
      {renaming && (
        <div className="mb-4 flex">
          <RenameForm
            initial={course.name}
            onCancel={() => setRenaming(false)}
            onSave={async (name) => {
              await renameCourse(course.id, name);
              setRenaming(false);
              await onChanged();
            }}
          />
        </div>
      )}
      {error && <p className="mb-3 rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}
      {course.closed_at && (
        <p className="mb-3 rounded-2xl bg-amber-50 px-4 py-2 text-sm font-semibold text-amber-900">
          Periodo cerrado el {formatDateTime(course.closed_at)}.{" "}
          {course.purge_at ? `Sus datos se borrarán el ${formatDateTime(course.purge_at)}.` : "Sus datos no se borran automáticamente."}
        </p>
      )}
      <CourseTeachers course={course} onChanged={onChanged} />
      <RaeEditor courseId={course.id} raes={course.raes ?? []} onChanged={onChanged} />
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
                  {g.student_count} {g.student_count === 1 ? "estudiante" : "estudiantes"} · {g.submission_count}{" "}
                  {g.submission_count === 1 ? "entrega" : "entregas"}
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

      <AIAlert />

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
          <QuestionsInbox />
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
              <li>Dentro de cada clase, publica talleres con su enunciado, archivos de apoyo y casos de prueba.</li>
              <li>Tus estudiantes compilan, ven qué casos superan y te envían su código. Cada entrega entra sola a la cola de la IA.</li>
              <li>Entra a la clase para revisar cada propuesta de la IA: edítala, apruébala o descártala. En modo supervisado el estudiante solo ve lo que apruebas.</li>
              <li>Desde la clase administras todo: renombrarla, cambiar su código, quitar estudiantes, editar talleres y agregar monitores que retroalimentan sin calificar.</li>
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
