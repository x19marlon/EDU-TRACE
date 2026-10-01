"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card, ProgressRing, Spinner } from "@/components/ui";
import {
  getGroup,
  getSubmission,
  GroupDetail,
  GroupStudent,
  listGroupSubmissions,
  Submission,
  SubmissionMeta,
} from "@/lib/api";
import { formatDateTime, formatFeedback, relativeTime } from "@/lib/format";

const WEEK_MS = 7 * 24 * 60 * 60 * 1000;

/** Última señal de actividad del estudiante: compilar, pedir orientación o enviar. */
function lastSeen(s: GroupStudent): string | undefined {
  const a = s.activity.last_active_at;
  const candidates = [a && !a.startsWith("0001") ? a : undefined, s.last_submission_at].filter(Boolean) as string[];
  return candidates.sort().at(-1);
}

function isActive(s: GroupStudent): boolean {
  const seen = lastSeen(s);
  return !!seen && Date.now() - new Date(seen).getTime() < WEEK_MS;
}

function StatusPill({ meta }: { meta: SubmissionMeta }) {
  if (!meta.success) return <span className="rounded-full bg-peach-100 px-2.5 py-0.5 text-[11px] font-bold text-peach-700">No compila</span>;
  if (meta.run_error || meta.exit_code !== 0)
    return <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-[11px] font-bold text-amber-800">Error al ejecutar</span>;
  return <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-[11px] font-bold text-mint-700">Compila y ejecuta</span>;
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

function CodeView({ code }: { code: string }) {
  const lines = code.replace(/\n$/, "").split("\n");
  return (
    <pre className="overflow-x-auto rounded-2xl border border-lavender-100 bg-white py-3 font-mono text-[13px] leading-relaxed">
      {lines.map((line, i) => (
        <div key={i} className="flex">
          <span className="w-10 shrink-0 select-none pr-3 text-right text-ink-faint">{i + 1}</span>
          <span className="whitespace-pre pr-4">{line || " "}</span>
        </div>
      ))}
    </pre>
  );
}

function SubmissionDetail({ id, studentName }: { id: string; studentName: string }) {
  const [sub, setSub] = useState<Submission | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setSub(null);
    setError(null);
    getSubmission(id)
      .then(setSub)
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar el envío"));
  }, [id]);

  if (error) return <p className="rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>;
  if (!sub)
    return (
      <p className="flex items-center gap-2 p-2 text-sm text-ink-soft">
        <Spinner className="h-4 w-4 border-lavender-500" /> Cargando envío...
      </p>
    );

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-bold">{studentName}</span>
        <span className="text-xs text-ink-soft">{formatDateTime(sub.created_at)}</span>
        <StatusPill meta={sub} />
      </div>

      <CodeView code={sub.code} />

      {sub.compiler_output && (
        <div>
          <h4 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Mensajes del compilador</h4>
          <pre className="whitespace-pre-wrap rounded-2xl bg-peach-50 p-3 font-mono text-xs text-peach-700">{sub.compiler_output}</pre>
        </div>
      )}
      {sub.success && (
        <div className="grid gap-3 sm:grid-cols-2">
          <div>
            <h4 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Entrada</h4>
            <pre className="min-h-[3rem] whitespace-pre-wrap rounded-2xl bg-lavender-50 p-3 font-mono text-xs">{sub.stdin || "(sin entrada)"}</pre>
          </div>
          <div>
            <h4 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Salida</h4>
            <pre className="min-h-[3rem] whitespace-pre-wrap rounded-2xl bg-mint-50 p-3 font-mono text-xs">{sub.program_output || "(sin salida)"}</pre>
          </div>
        </div>
      )}
      {sub.run_error && <p className="rounded-2xl bg-amber-50 p-3 text-xs font-semibold text-amber-800">{sub.run_error}</p>}

      <div>
        <h4 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Orientación de la IA que vio el estudiante</h4>
        {sub.ai_feedback ? (
          <div className="rounded-2xl bg-mint-50 p-4">
            <div className="text-sm leading-relaxed" dangerouslySetInnerHTML={{ __html: formatFeedback(sub.ai_feedback) }} />
            <p className="mt-3 border-t border-mint-200 pt-2 text-xs text-ink-faint">
              Generada por IA. Úsala como apoyo: la valoración y la retroalimentación final son tuyas.
            </p>
          </div>
        ) : (
          <p className="text-sm text-ink-soft">El estudiante envió este código sin pedir orientación a la IA.</p>
        )}
      </div>
    </div>
  );
}

function GroupView() {
  const { id } = useParams<{ id: string }>();
  const [group, setGroup] = useState<GroupDetail | null>(null);
  const [submissions, setSubmissions] = useState<SubmissionMeta[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [studentId, setStudentId] = useState<string | null>(null);
  const [submissionId, setSubmissionId] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([getGroup(id), listGroupSubmissions(id)])
      .then(([g, s]) => {
        setGroup(g);
        setSubmissions(s);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar la clase"));
  }, [id]);

  const names = useMemo(() => new Map(group?.students.map((s) => [s.id, s.name]) ?? []), [group]);
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
              { v: submissions.length, l: "envíos", c: "bg-mint-50 text-mint-700" },
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
                  Todos los envíos
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
                        {s.submission_count} {s.submission_count === 1 ? "envío" : "envíos"} · {relativeTime(lastSeen(s))}
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

        {/* Envíos */}
        <Card
          title={selected ? `Envíos de ${selected.name}` : "Envíos recientes"}
          className="rise-in lg:col-span-3"
          action={<span className="text-xs font-semibold text-ink-faint">{visible.length}</span>}
        >
          {visible.length === 0 ? (
            <p className="text-sm text-ink-soft">
              {selected ? "Este estudiante aún no ha enviado código." : "Todavía no hay envíos en esta clase."}
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
                      <span className="flex-1" />
                      {m.has_ai_feedback && (
                        <span className="rounded-full bg-white px-2.5 py-0.5 text-[11px] font-bold text-lavender-600">Con orientación IA</span>
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

export default function GroupPage() {
  return (
    <RequireAuth role="teacher">
      <AppShell>
        <GroupView />
      </AppShell>
    </RequireAuth>
  );
}
