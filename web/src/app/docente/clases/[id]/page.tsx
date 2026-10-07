"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card, ProgressRing } from "@/components/ui";
import TeacherAssignments from "@/components/TeacherAssignments";
import StudentProcess from "@/components/StudentProcess";
import { StatusPill, SubmissionDetail } from "@/components/SubmissionView";
import {
  Assignment,
  getGroup,
  listGroupAssignments,
  GroupDetail,
  GroupStudent,
  listGroupSubmissions,
  SubmissionMeta,
} from "@/lib/api";
import { formatDateTime, relativeTime } from "@/lib/format";

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

/** Última señal de actividad del estudiante: compilar, pedir orientación o enviar. */
function lastSeen(s: GroupStudent): string | undefined {
  const a = s.activity.last_active_at;
  const candidates = [a && !a.startsWith("0001") ? a : undefined, s.last_submission_at, s.last_attempt_at].filter(
    Boolean
  ) as string[];
  return candidates.sort().at(-1);
}

function isActive(s: GroupStudent): boolean {
  const seen = lastSeen(s);
  return !!seen && Date.now() - new Date(seen).getTime() < WEEK_MS;
}

function CopyCode({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(code);
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } catch {
          /* portapapeles bloqueado: el código sigue visible */
        }
      }}
      className="flex items-center gap-2 rounded-2xl bg-lavender-50 px-4 py-2 transition hover:bg-lavender-100"
      title="Copiar código de la clase"
    >
      <span className="font-mono text-xl font-extrabold tracking-[0.25em] text-lavender-700">{code}</span>
      <span className="text-xs font-bold text-lavender-600">{copied ? "¡Copiado!" : "Copiar"}</span>
    </button>
  );
}

function GroupView() {
  const { id } = useParams<{ id: string }>();
  const [group, setGroup] = useState<GroupDetail | null>(null);
  const [submissions, setSubmissions] = useState<SubmissionMeta[] | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [studentId, setStudentId] = useState<string | null>(null);
  const [submissionId, setSubmissionId] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([getGroup(id), listGroupSubmissions(id), listGroupAssignments(id)])
      .then(([g, s, a]) => {
        setGroup(g);
        setSubmissions(s);
        setAssignments(a);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar la clase"));
  }, [id]);

  const names = useMemo(() => new Map(group?.students.map((s) => [s.id, s.name]) ?? []), [group]);
  const assignmentTitles = useMemo(() => new Map(assignments.map((a) => [a.id, a.title])), [assignments]);
  const perAssignment = useMemo(() => {
    const counts = new Map<string, number>();
    submissions?.forEach((m) => m.assignment_id && counts.set(m.assignment_id, (counts.get(m.assignment_id) ?? 0) + 1));
    return counts;
  }, [submissions]);
  const visible = useMemo(
    () => submissions?.filter((s) => !studentId || s.student_id === studentId) ?? [],
    [submissions, studentId]
  );

  if (error)
    return (
      <div className="mx-auto max-w-7xl">
        <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>
        <Link href="/docente" className="mt-3 inline-block text-sm font-bold text-lavender-600">← Volver a mis materias</Link>
      </div>
    );
  if (!group || !submissions) return <p className="mx-auto max-w-7xl text-sm text-ink-soft">Cargando...</p>;

  const total = group.students.length;
  const active = group.students.filter(isActive).length;
  const withSubmission = group.students.filter((s) => s.submission_count > 0).length;
  const selected = group.students.find((s) => s.id === studentId);

  return (
    <div className="mx-auto max-w-7xl space-y-5">
      <nav className="text-sm font-semibold text-ink-soft">
        <Link href="/docente" className="hover:text-lavender-700">Mis materias</Link>
        <span className="mx-2 text-ink-faint">/</span>
        {group.course_name}
        <span className="mx-2 text-ink-faint">/</span>
        <span className="text-ink">{group.name}</span>
      </nav>

      <section className="rise-in flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <div>
          <p className="text-xs font-bold uppercase tracking-wider text-lavender-600">{group.course_name}</p>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">{group.name}</h1>
          <p className="mt-0.5 text-sm text-ink-soft">Comparte este código para que tus estudiantes se unan a la clase.</p>
        </div>
        <CopyCode code={group.join_code} />
      </section>

      <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
        <Card title="Progreso semanal" className="rise-in">
          <div className="flex items-center gap-5">
            <ProgressRing value={total ? Math.round((active / total) * 100) : 0} />
            <div>
              <p className="font-bold">Estudiantes activos</p>
              <p className="text-sm text-ink-soft">{active} de {total} en los últimos 7 días</p>
            </div>
          </div>
        </Card>
        <Card title="Resumen de la clase" className="rise-in">
          <div className="grid grid-cols-3 gap-2 text-center">
            {[
              { v: total, l: "estudiantes", c: "bg-lavender-50 text-lavender-700" },
              { v: submissions.length, l: "entregas", c: "bg-mint-50 text-mint-700" },
              { v: total - withSubmission, l: "sin enviar", c: "bg-peach-50 text-peach-700" },
            ].map((x) => (
              <div key={x.l} className={`rounded-2xl py-4 ${x.c}`}>
                <p className="text-2xl font-extrabold tabular-nums">{x.v}</p>
                <p className="mt-0.5 text-[11px] font-semibold">{x.l}</p>
              </div>
            ))}
          </div>
        </Card>
      </div>

      <TeacherAssignments
        groupId={group.id}
        assignments={assignments}
        submissionCounts={perAssignment}
        onCreated={(a) => setAssignments((list) => [a, ...list])}
      />

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-5">
        {/* Estudiantes */}
        <Card
          title="Estudiantes"
          className="rise-in lg:col-span-2 self-start"
          bodyClassName="p-2"
          action={<span className="text-xs font-semibold text-ink-faint">{total}</span>}
        >
          {total === 0 ? (
            <p className="p-3 text-sm text-ink-soft">Nadie se ha unido todavía. Comparte el código {group.join_code}.</p>
          ) : (
            <ul className="space-y-1">
              <li>
                <button
                  onClick={() => { setStudentId(null); setSubmissionId(null); }}
                  className={`w-full rounded-2xl px-3 py-2.5 text-left text-sm font-bold transition-colors ${
                    !studentId ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:bg-lavender-50"
                  }`}
                >
                  Todas las entregas
                </button>
              </li>
              {group.students.map((s) => (
                <li key={s.id}>
                  <button
                    onClick={() => { setStudentId(s.id); setSubmissionId(null); }}
                    className={`flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors ${
                      studentId === s.id ? "bg-lavender-100" : "hover:bg-lavender-50"
                    }`}
                  >
                    <Avatar name={s.name} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-bold">{s.name}</span>
                      <span className="block truncate text-xs text-ink-soft">
                        {s.submission_count} {s.submission_count === 1 ? "entrega" : "entregas"} · {s.attempt_count}{" "}
                        {s.attempt_count === 1 ? "intento" : "intentos"} · {relativeTime(lastSeen(s))}
                      </span>
                    </span>
                    <span
                      className={`h-2.5 w-2.5 shrink-0 rounded-full ${
                        isActive(s) ? "bg-mint-400" : lastSeen(s) ? "bg-peach-400" : "bg-lavender-200"
                      }`}
                      title={isActive(s) ? "Activo esta semana" : lastSeen(s) ? "Sin actividad reciente" : "Aún no empieza"}
                    />
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Card>

        {/* Proceso del estudiante elegido, o entregas oficiales recientes de la clase */}
        {selected ? (
          <StudentProcess key={selected.id} groupId={group.id} student={selected} assignmentTitles={assignmentTitles} />
        ) : (
        <Card
          title="Entregas oficiales recientes"
          className="rise-in lg:col-span-3"
          action={<span className="text-xs font-semibold text-ink-faint">{visible.length}</span>}
        >
          {visible.length === 0 ? (
            <p className="text-sm text-ink-soft">
              Todavía no hay entregas oficiales en esta clase. Elige un estudiante para ver también sus intentos.
            </p>
          ) : (
            <div className="space-y-4">
              <ul className="max-h-72 space-y-1 overflow-auto pr-1">
                {visible.map((m) => (
                  <li key={m.id}>
                    <button
                      onClick={() => setSubmissionId(m.id)}
                      className={`flex w-full flex-wrap items-center gap-x-3 gap-y-1 rounded-2xl px-3 py-2.5 text-left transition-colors ${
                        submissionId === m.id ? "bg-lavender-100" : "bg-lavender-50/50 hover:bg-lavender-50"
                      }`}
                    >
                      {!selected && <span className="text-sm font-bold">{names.get(m.student_id) ?? "Estudiante"}</span>}
                      <span className="text-xs text-ink-soft">{formatDateTime(m.created_at)}</span>
                      <span className="text-xs text-ink-faint">{m.lines} líneas</span>
                      {m.assignment_id && (
                        <span className="rounded-full bg-lavender-100 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
                          {assignmentTitles.get(m.assignment_id) ?? "Taller"}
                        </span>
                      )}
                      <span className="flex-1" />
                      {m.has_ai_feedback && (
                        <span className="rounded-full bg-white px-2.5 py-0.5 text-[11px] font-bold text-lavender-600">Con retroalimentación IA</span>
                      )}
                      <StatusPill meta={m} />
                    </button>
                  </li>
                ))}
              </ul>
              {submissionId && (
                <div className="border-t border-lavender-100 pt-4">
                  <SubmissionDetail
                    id={submissionId}
                    studentName={names.get(visible.find((m) => m.id === submissionId)?.student_id ?? "") ?? "Estudiante"}
                    assignmentTitle={(() => {
                      const aid = visible.find((m) => m.id === submissionId)?.assignment_id;
                      return aid ? assignmentTitles.get(aid) ?? "Taller" : undefined;
                    })()}
                  />
                </div>
              )}
            </div>
          )}
        </Card>
        )}
      </div>
    </div>
  );
}

export default function GroupPage() {
  return (
    <RequireAuth role="teacher">
      <AppShell>
        <GroupView />
      </AppShell>
    </RequireAuth>
  );
}
