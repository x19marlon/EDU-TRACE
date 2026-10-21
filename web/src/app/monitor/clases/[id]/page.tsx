"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card } from "@/components/ui";
import { StatusPill, SubmissionDetail } from "@/components/SubmissionView";
import {
  Assignment,
  assignmentFileUrl,
  getMonitorGroup,
  GroupDetail,
  listMonitorAssignments,
  listMonitorSubmissions,
  SubmissionMeta,
} from "@/lib/api";
import { dueLabel, formatDateTime, formatSize } from "@/lib/format";

function MonitorGroupView() {
  const { id } = useParams<{ id: string }>();
  const [group, setGroup] = useState<GroupDetail | null>(null);
  const [submissions, setSubmissions] = useState<SubmissionMeta[] | null>(null);
  const [assignments, setAssignments] = useState<Assignment[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [studentId, setStudentId] = useState<string | null>(null);
  const [submissionId, setSubmissionId] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([getMonitorGroup(id), listMonitorSubmissions(id), listMonitorAssignments(id)])
      .then(([g, s, a]) => {
        setGroup(g);
        setSubmissions(s);
        setAssignments(a);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar la clase"));
  }, [id]);

  const reloadSubmissions = useCallback(() => {
    listMonitorSubmissions(id).then(setSubmissions).catch(() => {});
  }, [id]);

  const names = useMemo(() => new Map(group?.students.map((s) => [s.id, s.name]) ?? []), [group]);
  const titles = useMemo(() => new Map(assignments.map((a) => [a.id, a.title])), [assignments]);
  const visible = useMemo(
    () => submissions?.filter((s) => !studentId || s.student_id === studentId) ?? [],
    [submissions, studentId]
  );

  if (error)
    return (
      <div className="mx-auto max-w-7xl">
        <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>
        <Link href="/monitor" className="mt-3 inline-block text-sm font-bold text-lavender-600">← Volver a mis clases</Link>
      </div>
    );
  if (!group || !submissions) return <p className="mx-auto max-w-7xl text-sm text-ink-soft">Cargando...</p>;

  const pending = submissions.filter((m) => !m.monitor_feedback_count).length;
  const selected = visible.find((m) => m.id === submissionId);

  return (
    <div className="mx-auto max-w-7xl space-y-5">
      <nav className="text-sm font-semibold text-ink-soft">
        <Link href="/monitor" className="hover:text-lavender-700">Mis clases</Link>
        <span className="mx-2 text-ink-faint">/</span>
        {group.course_name}
        <span className="mx-2 text-ink-faint">/</span>
        <span className="text-ink">{group.name}</span>
      </nav>

      <section className="rise-in flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <div>
          <p className="text-xs font-bold uppercase tracking-wider text-lavender-600">{group.course_name}</p>
          <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">{group.name}</h1>
          <p className="mt-0.5 text-sm text-ink-soft">Eres monitor de esta clase: retroalimentas el código, no calificas.</p>
        </div>
        <div className="grid grid-cols-3 gap-2 text-center">
          {[
            { v: group.students.length, l: "estudiantes", c: "bg-lavender-50 text-lavender-700" },
            { v: submissions.length, l: "entregas", c: "bg-mint-50 text-mint-700" },
            { v: pending, l: "sin retroalimentar", c: "bg-amber-50 text-amber-800" },
          ].map((x) => (
            <div key={x.l} className={`rounded-2xl px-4 py-3 ${x.c}`}>
              <p className="text-xl font-extrabold tabular-nums">{x.v}</p>
              <p className="text-[11px] font-semibold">{x.l}</p>
            </div>
          ))}
        </div>
      </section>

      {assignments.length > 0 && (
        <Card title="Talleres" className="rise-in">
          <ul className="space-y-2">
            {assignments.map((a) => (
              <li key={a.id}>
                <details className="rounded-2xl bg-lavender-50/60 px-4 py-3 open:bg-lavender-50">
                  <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1">
                    <span className="min-w-0 flex-1 font-bold">{a.title}</span>
                    {dueLabel(a.due_at) && <span className="text-xs font-semibold text-peach-700">{dueLabel(a.due_at)}</span>}
                  </summary>
                  <div className="mt-3 space-y-3">
                    <div className="max-h-60 overflow-auto whitespace-pre-wrap rounded-2xl bg-white p-3 text-sm leading-relaxed">
                      {a.statement}
                    </div>
                    {a.files.length > 0 && (
                      <ul className="flex flex-wrap gap-2">
                        {a.files.map((f) => (
                          <li key={f.id}>
                            <a
                              href={assignmentFileUrl(a.id, f.id)}
                              className="inline-flex items-center gap-2 rounded-full bg-white px-3 py-1.5 text-sm font-semibold transition hover:bg-mint-50"
                            >
                              {f.name}
                              <span className="text-xs font-normal text-ink-faint">{formatSize(f.size)}</span>
                            </a>
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                </details>
              </li>
            ))}
          </ul>
        </Card>
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-5">
        <Card
          title="Estudiantes"
          className="rise-in lg:col-span-2 self-start"
          bodyClassName="p-2"
          action={<span className="text-xs font-semibold text-ink-faint">{group.students.length}</span>}
        >
          {group.students.length === 0 ? (
            <p className="p-3 text-sm text-ink-soft">Todavía no hay estudiantes en esta clase.</p>
          ) : (
            <ul className="space-y-1">
              <li>
                <button
                  onClick={() => {
                    setStudentId(null);
                    setSubmissionId(null);
                  }}
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
                    onClick={() => {
                      setStudentId(s.id);
                      setSubmissionId(null);
                    }}
                    className={`flex w-full items-center gap-3 rounded-2xl px-3 py-2.5 text-left transition-colors ${
                      studentId === s.id ? "bg-lavender-100" : "hover:bg-lavender-50"
                    }`}
                  >
                    <Avatar name={s.name} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-bold">{s.name}</span>
                      <span className="block truncate text-xs text-ink-soft">
                        {s.submission_count} {s.submission_count === 1 ? "entrega" : "entregas"}
                      </span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card
          title="Entregas oficiales"
          className="rise-in lg:col-span-3"
          action={<span className="text-xs font-semibold text-ink-faint">{visible.length}</span>}
        >
          {visible.length === 0 ? (
            <p className="text-sm text-ink-soft">No hay entregas oficiales para mostrar.</p>
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
                      <span className="text-sm font-bold">{names.get(m.student_id) ?? "Estudiante"}</span>
                      <span className="text-xs text-ink-soft">{formatDateTime(m.created_at)}</span>
                      <span className="text-xs text-ink-faint">{m.lines} líneas</span>
                      {m.assignment_id && (
                        <span className="rounded-full bg-lavender-100 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
                          {titles.get(m.assignment_id) ?? "Taller"}
                        </span>
                      )}
                      <span className="flex-1" />
                      {m.monitor_feedback_count ? (
                        <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-[11px] font-bold text-mint-700">
                          {m.monitor_feedback_count === 1 ? "1 retroalimentación" : `${m.monitor_feedback_count} retroalimentaciones`}
                        </span>
                      ) : (
                        <span className="rounded-full bg-amber-50 px-2.5 py-0.5 text-[11px] font-bold text-amber-800">Sin retroalimentar</span>
                      )}
                      <StatusPill meta={m} />
                    </button>
                  </li>
                ))}
              </ul>
              {selected && (
                <div className="border-t border-lavender-100 pt-4">
                  <SubmissionDetail
                    id={selected.id}
                    mode="monitor"
                    onNotesChanged={reloadSubmissions}
                    studentName={names.get(selected.student_id) ?? "Estudiante"}
                    assignmentTitle={selected.assignment_id ? titles.get(selected.assignment_id) ?? "Taller" : undefined}
                  />
                </div>
              )}
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

export default function MonitorGroupPage() {
  return (
    <RequireAuth role="monitor">
      <AppShell>
        <MonitorGroupView />
      </AppShell>
    </RequireAuth>
  );
}
