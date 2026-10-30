"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Card, Spinner } from "@/components/ui";
import { StatusPill } from "@/components/SubmissionView";
import AnnotatedCode from "@/components/AnnotatedCode";
import { LatePill } from "@/components/Deadline";
import { AnnotationsPill, MonitorCountPill, ReviewStatusPill, SubmissionFeedback } from "@/components/StudentFeedback";
import { getMyHistory, getMySubmission, StudentHistory, StudentSubmission, Submission } from "@/lib/api";
import TestResults, { TestsPill } from "@/components/TestResults";
import { formatDateTime, formatFeedback } from "@/lib/format";

type Tab = "entregas" | "intentos";

function HistoryDetail({ item }: { item: StudentSubmission }) {
  const [sub, setSub] = useState<Submission | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getMySubmission(item.id)
      .then(setSub)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar el envío"));
  }, [item.id]);

  if (error) return <p className="rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>;
  if (!sub)
    return (
      <p className="flex items-center gap-2 text-sm text-ink-soft">
        <Spinner className="h-4 w-4 border-lavender-500" /> Cargando...
      </p>
    );

  return (
    <div className="space-y-4">
      <AnnotatedCode key={sub.id} code={sub.code} submissionId={sub.id} mode="student" initial={sub.annotations ?? []} canAnnotate={false} />
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
      {(sub.tests?.length ?? 0) > 0 && <TestResults results={sub.tests!} />}

      {item.kind === "official" && (
        <div className="space-y-3">
          <h4 className="text-xs font-bold uppercase tracking-wider text-ink-soft">Revisión</h4>
          <SubmissionFeedback item={item} />
          {(sub.ai_feedback || sub.ai_feedback_informal) && (
            <details className="rounded-2xl bg-white p-3">
              <summary className="cursor-pointer text-sm font-bold text-ink-soft">Orientación de la IA que recibiste al enviarla</summary>
              <div className="mt-3 space-y-3">
                {[sub.ai_feedback, sub.ai_feedback_informal].filter(Boolean).map((text, i) => (
                  <div
                    key={i}
                    className={`rounded-2xl p-3 text-sm leading-relaxed ${i === 0 && sub.ai_feedback ? "bg-mint-50" : "bg-peach-50"}`}
                    dangerouslySetInnerHTML={{ __html: formatFeedback(text) }}
                  />
                ))}
              </div>
            </details>
          )}
        </div>
      )}
    </div>
  );
}

function Row({
  item,
  open,
  onToggle,
  children,
}: {
  item: StudentSubmission;
  open: boolean;
  onToggle: () => void;
  children: React.ReactNode;
}) {
  return (
    <li className="rounded-2xl bg-lavender-50/60">
      <button
        onClick={onToggle}
        aria-expanded={open}
        className="flex w-full flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-left"
      >
        {children}
      </button>
      {open && (
        <div className="px-4 pb-4">
          <HistoryDetail item={item} />
        </div>
      )}
    </li>
  );
}

function Tile({ value, label, tone }: { value: string | number; label: string; tone: string }) {
  return (
    <div className={`rounded-2xl px-4 py-3 text-center ${tone}`}>
      <p className="text-2xl font-extrabold">{value}</p>
      <p className="mt-0.5 text-[11px] font-semibold">{label}</p>
    </div>
  );
}

function HistoryView() {
  const [history, setHistory] = useState<StudentHistory | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [groupId, setGroupId] = useState<string>("all");
  const [tab, setTab] = useState<Tab>("entregas");
  const [openId, setOpenId] = useState<string | null>(null);

  useEffect(() => {
    getMyHistory()
      .then(setHistory)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar tu historial"));
  }, []);

  const groups = useMemo(() => {
    const seen = new Map<string, string>();
    for (const s of [...(history?.submissions ?? []), ...(history?.attempts ?? [])]) {
      if (!seen.has(s.group_id)) seen.set(s.group_id, `${s.course_name} · ${s.group_name}`);
    }
    return Array.from(seen.entries());
  }, [history]);

  const submissions = useMemo(
    () => history?.submissions.filter((s) => groupId === "all" || s.group_id === groupId) ?? [],
    [history, groupId]
  );
  const attempts = useMemo(
    () => history?.attempts.filter((s) => groupId === "all" || s.group_id === groupId) ?? [],
    [history, groupId]
  );

  const byAssignment = useMemo(() => {
    const map = new Map<string, { title: string; place: string; items: StudentSubmission[] }>();
    for (const s of submissions) {
      const key = `${s.group_id}:${s.assignment_id ?? ""}`;
      if (!map.has(key)) {
        map.set(key, { title: s.assignment_title || "Práctica libre", place: `${s.course_name} · ${s.group_name}`, items: [] });
      }
      map.get(key)!.items.push(s);
    }
    return Array.from(map.entries());
  }, [submissions]);

  const graded = submissions.filter((s) => s.review?.grade !== undefined);
  const average = graded.length ? graded.reduce((n, s) => n + (s.review!.grade ?? 0), 0) / graded.length : null;
  const compiled = attempts.filter((s) => s.success).length;

  if (error) return <p className="mx-auto max-w-5xl rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>;
  if (!history) return <p className="mx-auto max-w-5xl text-sm text-ink-soft">Cargando...</p>;

  const toggle = (id: string) => setOpenId((current) => (current === id ? null : id));

  return (
    <div className="mx-auto max-w-5xl space-y-5">
      <section className="rise-in rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">Mi historial</h1>
        <p className="mt-0.5 text-sm text-ink-soft">
          Todo lo que has enviado: tus entregas oficiales con la revisión de tu profesor y tus intentos de compilación.
        </p>
      </section>

      {history.submissions.length === 0 && history.attempts.length === 0 ? (
        <Card>
          <p className="font-bold">Aún no has enviado nada</p>
          <p className="mt-1 text-sm text-ink-soft">
            Elige tu clase en el{" "}
            <Link href="/" className="font-bold text-lavender-600 hover:text-lavender-700">
              compilador
            </Link>
            , compila tu código y envíalo: cada intento y cada entrega aparecerán aquí.
          </p>
        </Card>
      ) : (
        <>
          {groups.length > 1 && (
            <div role="radiogroup" aria-label="Clase" className="flex flex-wrap gap-1 rounded-full bg-white p-1 shadow-soft">
              {[["all", "Todas las clases"] as [string, string], ...groups].map(([id, label]) => (
                <button
                  key={id}
                  type="button"
                  role="radio"
                  aria-checked={groupId === id}
                  onClick={() => {
                    setGroupId(id);
                    setOpenId(null);
                  }}
                  className={`rounded-full px-4 py-1.5 text-sm font-semibold transition-colors ${
                    groupId === id ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          )}

          <div className="grid grid-cols-2 gap-2 sm:grid-cols-5">
            <Tile value={submissions.length} label="entregas oficiales" tone="bg-white text-lavender-700 shadow-soft" />
            <Tile value={graded.length} label="con nota" tone="bg-white text-mint-700 shadow-soft" />
            <Tile value={average === null ? "—" : average.toFixed(1)} label="nota promedio" tone="bg-white text-ink shadow-soft" />
            <Tile value={attempts.length} label="intentos de compilación" tone="bg-white text-peach-700 shadow-soft" />
            <Tile
              value={attempts.length ? `${Math.round((compiled / attempts.length) * 100)}%` : "—"}
              label="intentos que compilan"
              tone="bg-white text-amber-800 shadow-soft"
            />
          </div>

          <div role="tablist" className="flex gap-1 rounded-full bg-white p-1 shadow-soft w-fit">
            {(
              [
                ["entregas", `Entregas por taller (${submissions.length})`],
                ["intentos", `Intentos (${attempts.length})`],
              ] as [Tab, string][]
            ).map(([id, label]) => (
              <button
                key={id}
                role="tab"
                aria-selected={tab === id}
                onClick={() => {
                  setTab(id);
                  setOpenId(null);
                }}
                className={`rounded-full px-4 py-1.5 text-sm font-semibold transition-colors ${
                  tab === id ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          {tab === "entregas" &&
            (byAssignment.length === 0 ? (
              <Card>
                <p className="text-sm text-ink-soft">Aún no tienes entregas oficiales en esta clase.</p>
              </Card>
            ) : (
              byAssignment.map(([key, group]) => (
                <Card
                  key={key}
                  className="rise-in"
                  title={group.title}
                  action={
                    <span className="shrink-0 text-right text-xs font-semibold text-ink-faint">
                      <span className="hidden sm:inline">{group.place} · </span>
                      {group.items.length} {group.items.length === 1 ? "entrega" : "entregas"}
                    </span>
                  }
                >
                  <ul className="space-y-2">
                    {group.items.map((s, i) => (
                      <Row key={s.id} item={s} open={openId === s.id} onToggle={() => toggle(s.id)}>
                        <span className="font-bold">Entrega {group.items.length - i}</span>
                        <span className="text-xs text-ink-soft">{formatDateTime(s.created_at)}</span>
                        <span className="text-xs text-ink-faint">{s.lines} líneas</span>
                        <span className="flex-1" />
                        <StatusPill meta={s} />
                        <TestsPill meta={s} />
                        <LatePill meta={s} />
                        <AnnotationsPill item={s} />
                        <MonitorCountPill item={s} />
                        <ReviewStatusPill item={s} />
                      </Row>
                    ))}
                  </ul>
                </Card>
              ))
            ))}

          {tab === "intentos" && (
            <Card className="rise-in" title="Intentos de compilación">
              {attempts.length === 0 ? (
                <p className="text-sm text-ink-soft">
                  Aún no hay intentos guardados. Se guardan cuando compilas con una clase elegida en el compilador.
                </p>
              ) : (
                <ul className="space-y-2">
                  {attempts.map((s) => (
                    <Row key={s.id} item={s} open={openId === s.id} onToggle={() => toggle(s.id)}>
                      <span className="text-sm font-bold">{s.assignment_title || "Práctica libre"}</span>
                      <span className="text-xs text-ink-soft">{formatDateTime(s.created_at)}</span>
                      {s.error_summary && (
                        <span className="min-w-0 max-w-full truncate font-mono text-[11px] text-peach-700">{s.error_summary}</span>
                      )}
                      <span className="flex-1" />
                      <StatusPill meta={s} />
                    </Row>
                  ))}
                </ul>
              )}
            </Card>
          )}
        </>
      )}
    </div>
  );
}

export default function HistoryPage() {
  return (
    <RequireAuth role="student">
      <AppShell>
        <HistoryView />
      </AppShell>
    </RequireAuth>
  );
}
