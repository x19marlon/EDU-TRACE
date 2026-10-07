"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import dynamic from "next/dynamic";
import OutputPanel from "@/components/OutputPanel";
import FeedbackPanel from "@/components/FeedbackPanel";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import { Avatar, Card, Spinner } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import StudentClasses from "@/components/StudentClasses";
import StudentAssignments from "@/components/StudentAssignments";
import {
  compileCode,
  streamFeedback,
  CompileResult,
  FeedbackContext,
  FeedbackMeta,
  FeedbackMode,
  listMyAssignments,
  listMyGroups,
  StudentAssignment,
  StudentGroup,
  submitCode,
  UnauthorizedError,
} from "@/lib/api";

// CodeMirror no es SSR-safe, se carga solo en el cliente.
const CodeEditor = dynamic(() => import("@/components/CodeEditor"), {
  ssr: false,
  loading: () => (
    <div className="w-full min-h-[400px] flex items-center justify-center">
      <p className="text-ink-faint text-sm">Cargando editor...</p>
    </div>
  ),
});

const DEFAULT_CODE = `#include <iostream>
using namespace std;

int main() {
    cout << "¡Hola, mundo!" << endl;
    return 0;
}
`;

export default function Home() {
  return (
    <RequireAuth>
      <AppShell>
        <CompilerPage />
      </AppShell>
    </RequireAuth>
  );
}

function StatChip({ value, label, tone }: { value: number; label: string; tone: "mint" | "peach" }) {
  const tones = { mint: "bg-mint-100 text-mint-700", peach: "bg-peach-100 text-peach-700" };
  return (
    <div className={`rounded-2xl px-4 py-2.5 ${tones[tone]}`}>
      <p className="text-2xl font-extrabold leading-none tabular-nums">{value}</p>
      <p className="mt-1 text-xs font-semibold">{label}</p>
    </div>
  );
}

type FeedbackState = {
  text: string;
  meta: FeedbackMeta | null;
  loading: boolean;
  error: string | null;
  code: string | null; // código sobre el que se generó (se adjunta al envío solo si coincide)
};

const EMPTY_FEEDBACK: FeedbackState = { text: "", meta: null, loading: false, error: null, code: null };

const MODE_LABEL: Record<FeedbackMode, string> = { formal: "Formal", informal: "Informal" };

function CompilerPage() {
  const { user, setUser } = useAuth();
  const [code, setCode] = useState(DEFAULT_CODE);
  const [stdin, setStdin] = useState("");
  const [compileResult, setCompileResult] = useState<CompileResult | null>(null);
  // Código y entrada exactos de la última compilación, para no enviar a la IA
  // resultados de una versión anterior del código.
  const [compiledRun, setCompiledRun] = useState<{ code: string; stdin: string } | null>(null);
  const [isCompiling, setIsCompiling] = useState(false);
  const [compileError, setCompileError] = useState<string | null>(null);

  // Una retroalimentación por modalidad; las dos están siempre disponibles.
  const [feedback, setFeedback] = useState<Record<FeedbackMode, FeedbackState>>({
    formal: EMPTY_FEEDBACK,
    informal: EMPTY_FEEDBACK,
  });
  const feedbackText = useRef<Record<FeedbackMode, string>>({ formal: "", informal: "" });
  const updateFeedback = (mode: FeedbackMode, patch: Partial<FeedbackState>) =>
    setFeedback((f) => ({ ...f, [mode]: { ...f[mode], ...patch } }));
  // El servidor atiende una retroalimentación a la vez por usuario.
  const anyFeedbackLoading = feedback.formal.loading || feedback.informal.loading;

  // Clases, talleres y envío al profesor.
  const isStudent = user?.role === "student";
  const [groups, setGroups] = useState<StudentGroup[]>([]);
  const [groupId, setGroupId] = useState<string | null>(null);
  const [assignments, setAssignments] = useState<StudentAssignment[]>([]);
  const [assignmentId, setAssignmentId] = useState<string | null>(null);
  const [sending, setSending] = useState(false);
  const [sendStatus, setSendStatus] = useState<{ ok: boolean; text: string } | null>(null);

  const loadAssignments = useCallback(() => {
    listMyAssignments()
      .then(setAssignments)
      .catch(() => setAssignments([]));
  }, []);

  useEffect(() => {
    if (!isStudent) return;
    listMyGroups()
      .then((gs) => {
        setGroups(gs);
        setGroupId((cur) => cur ?? gs[0]?.id ?? null);
      })
      .catch(() => setGroups([]));
    loadAssignments();
  }, [isStudent, loadAssignments]);

  const groupAssignments = assignments.filter((a) => a.group_id === groupId);
  // Al cambiar de clase, el taller elegido deja de valer si no es de esa clase.
  useEffect(() => {
    if (assignmentId && !assignments.some((a) => a.id === assignmentId && a.group_id === groupId)) {
      setAssignmentId(null);
    }
  }, [groupId, assignments, assignmentId]);

  const [activeTab, setActiveTab] = useState<"output" | FeedbackMode>("output");

  // Contadores del saludo: parten de la sesión y se actualizan localmente.
  const [stats, setStats] = useState({
    compiles: user?.activity?.compiles ?? 0,
    feedbacks: user?.activity?.feedbacks ?? 0,
  });

  const handleCompile = useCallback(async () => {
    setIsCompiling(true);
    setCompileError(null);
    setCompileResult(null);
    setCompiledRun(null);
    setActiveTab("output");

    try {
      // Con una clase elegida, la compilación queda como intento visible para el profesor.
      const context = isStudent && groupId ? { group_id: groupId, assignment_id: assignmentId ?? undefined } : undefined;
      const result = await compileCode(code, stdin, context);
      setCompileResult(result);
      setCompiledRun({ code, stdin });
      setStats((s) => ({ ...s, compiles: s.compiles + 1 }));
    } catch (err) {
      // Sesión expirada: RequireAuth redirige al login.
      if (err instanceof UnauthorizedError) setUser(null);
      setCompileError(
        err instanceof Error ? err.message : "Error de conexión con el servidor"
      );
    } finally {
      setIsCompiling(false);
    }
  }, [code, stdin, setUser, isStudent, groupId, assignmentId]);

  const handleFeedback = async (mode: FeedbackMode) => {
    feedbackText.current[mode] = "";
    updateFeedback(mode, { text: "", meta: null, error: null, loading: true, code });
    setActiveTab(mode);

    try {
      const upToDate = compileResult !== null && compiledRun?.code === code;
      const base = { mode, code, assignment_id: (isStudent && assignmentId) || undefined };
      const context: FeedbackContext =
        upToDate && compileResult
          ? {
              ...base,
              compiled: true,
              success: compileResult.success,
              compiler_output: compileResult.compiler_output,
              stdin: compiledRun.stdin,
              program_output: compileResult.program_output,
              exit_code: compileResult.exit_code,
              run_error: compileResult.error ?? "",
            }
          : { ...base, compiled: false };

      const meta = await streamFeedback(context, (token) => {
        feedbackText.current[mode] += token;
        updateFeedback(mode, { text: feedbackText.current[mode] });
      });
      updateFeedback(mode, { meta });
      setStats((s) => ({ ...s, feedbacks: s.feedbacks + 1 }));
    } catch (err) {
      if (err instanceof UnauthorizedError) setUser(null);
      updateFeedback(mode, { error: err instanceof Error ? err.message : "Error de conexión con Ollama" });
    } finally {
      updateFeedback(mode, { loading: false });
    }
  };

  /** Texto de la retroalimentación si corresponde exactamente al código actual. */
  const usableFeedback = (mode: FeedbackMode) => {
    const f = feedback[mode];
    return !f.loading && !f.error && f.code === code && f.text.trim() !== "" ? f.text : "";
  };

  const handleSend = async () => {
    const group = groups.find((g) => g.id === groupId);
    if (!group) return;
    const assignment = groupAssignments.find((a) => a.id === assignmentId);
    setSending(true);
    setSendStatus(null);
    try {
      const formal = usableFeedback("formal");
      const informal = usableFeedback("informal");
      await submitCode({
        group_id: group.id,
        assignment_id: assignment?.id,
        code,
        stdin,
        ai_feedback: formal,
        ai_feedback_informal: informal,
      });
      setStats((s) => ({ ...s, compiles: s.compiles + 1 }));
      const extras = [formal && "la retroalimentación formal", informal && "la informal"].filter(Boolean);
      setSendStatus({
        ok: true,
        text:
          `Enviado a ${group.course_name} · ${group.name}${assignment ? ` (${assignment.title})` : ""}. ` +
          `Tu profesor ya puede verlo${extras.length ? ` junto con ${extras.join(" y ")}` : ""}.`,
      });
    } catch (err) {
      if (err instanceof UnauthorizedError) setUser(null);
      setSendStatus({ ok: false, text: err instanceof Error ? err.message : "No se pudo enviar" });
    } finally {
      setSending(false);
    }
  };

  if (!user) return null;
  const firstName = user.name.split(" ")[0];

  return (
    <div className="mx-auto max-w-7xl space-y-5">
      {/* Saludo */}
      <section className="rise-in rounded-3xl bg-white p-5 sm:p-6 shadow-soft">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div className="flex items-center gap-4">
            <Avatar name={user.name} className="h-14 w-14 text-lg ring-4 ring-lavender-50" />
            <div>
              <h1 className="text-2xl sm:text-3xl font-extrabold tracking-tight">¡Hola, {firstName}!</h1>
              <p className="text-ink-soft text-sm mt-0.5">
                Escribe tu solución, compílala y pide retroalimentación formal o informal cuando la necesites.
              </p>
            </div>
          </div>
          <div className="flex gap-3">
            <StatChip value={stats.compiles} label="compilaciones" tone="mint" />
            <StatChip value={stats.feedbacks} label="retroalimentaciones" tone="peach" />
          </div>
        </div>
        {isStudent && (
          <div className="mt-4">
            <StudentClasses
              groups={groups}
              selectedId={groupId}
              onSelect={setGroupId}
              onJoined={(g) => {
                setGroups((gs) => (gs.some((x) => x.id === g.id) ? gs : [...gs, g]));
                setGroupId(g.id);
                loadAssignments();
              }}
            />
            {sendStatus && (
              <p
                role="status"
                className={`mt-3 rounded-2xl px-4 py-2.5 text-sm font-semibold ${
                  sendStatus.ok ? "bg-mint-50 text-mint-700" : "bg-peach-50 text-peach-700"
                }`}
              >
                {sendStatus.text}
              </p>
            )}
          </div>
        )}
      </section>

      {isStudent && groupAssignments.length > 0 && (
        <StudentAssignments assignments={groupAssignments} selectedId={assignmentId} onSelect={setAssignmentId} />
      )}

      <div className="grid grid-cols-1 gap-5 lg:grid-cols-5">
        {/* Editor */}
        <Card
          className="rise-in lg:col-span-3 min-h-[520px] overflow-hidden"
          bodyClassName="pt-3"
          title={
            <span className="flex items-center gap-2 whitespace-nowrap">
              Código fuente
              <span className="rounded-full bg-lavender-50 px-2 py-0.5 text-[11px] font-semibold text-lavender-600">
                C++17
              </span>
            </span>
          }
          action={
            <div className="flex flex-wrap items-center justify-end gap-2">
              <div className="flex items-center gap-1 rounded-full bg-lavender-50 p-1" role="group" aria-label="Pedir retroalimentación">
                <span className="pl-2 pr-0.5 text-xs font-bold text-ink-soft" title="Retroalimentación con IA">IA</span>
                {(["formal", "informal"] as const).map((mode) => (
                  <button
                    key={mode}
                    onClick={() => handleFeedback(mode)}
                    disabled={anyFeedbackLoading || !code.trim()}
                    title={`Pedir retroalimentación ${mode}`}
                    aria-label={`Pedir retroalimentación ${mode}`}
                    className={`flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm font-bold transition disabled:opacity-50 ${
                      mode === "formal"
                        ? "bg-mint-100 text-mint-700 hover:bg-mint-200"
                        : "bg-peach-100 text-peach-700 hover:bg-peach-200"
                    }`}
                  >
                    {feedback[mode].loading && <Spinner className="h-3.5 w-3.5 border-current" />}
                    {MODE_LABEL[mode]}
                  </button>
                ))}
              </div>
              <button
                onClick={handleCompile}
                disabled={isCompiling || !code.trim()}
                className="flex items-center gap-2 rounded-full bg-lavender-600 px-5 py-2 text-sm font-bold text-white
                  shadow-soft transition hover:bg-lavender-700 disabled:opacity-50"
              >
                {isCompiling ? (
                  <Spinner />
                ) : (
                  <svg className="h-4 w-4" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                    <path d="M6.3 2.84A1.5 1.5 0 0 0 4 4.11v11.78a1.5 1.5 0 0 0 2.3 1.27l9.34-5.89a1.5 1.5 0 0 0 0-2.54L6.3 2.84Z" />
                  </svg>
                )}
                {isCompiling ? "Compilando..." : "Compilar"}
              </button>
              {isStudent && (
                <button
                  onClick={handleSend}
                  disabled={sending || !groupId || !code.trim()}
                  title={groupId ? "Enviar este código a tu profesor" : "Únete a una clase para poder enviar"}
                  className="flex items-center gap-2 rounded-full bg-white px-4 py-2 text-sm font-bold text-lavender-700
                    ring-1 ring-lavender-200 transition hover:bg-lavender-50 disabled:opacity-50"
                >
                  {sending ? (
                    <Spinner className="h-4 w-4 border-lavender-700" />
                  ) : (
                    <svg className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24" aria-hidden>
                      <path strokeLinecap="round" strokeLinejoin="round" d="M6 12 3.269 3.125A59.769 59.769 0 0 1 21.485 12 59.768 59.768 0 0 1 3.27 20.875L5.999 12Zm0 0h7.5" />
                    </svg>
                  )}
                  {sending ? "Enviando..." : "Enviar"}
                </button>
              )}
            </div>
          }
        >
          <div className="h-full border-t border-lavender-100">
            <CodeEditor value={code} onChange={setCode} />
          </div>
        </Card>

        <div className="lg:col-span-2 flex flex-col gap-5">
          {/* Entrada */}
          <section className="rise-in rounded-3xl bg-peach-100 p-5 shadow-soft">
            <div className="mb-2 flex items-center justify-between">
              <h2 className="font-bold">Entrada</h2>
              <span className="text-xs font-semibold text-peach-700">datos para cin / scanf</span>
            </div>
            <textarea
              value={stdin}
              onChange={(e) => setStdin(e.target.value)}
              placeholder="Escribe aquí la entrada para tu programa..."
              className="h-24 w-full resize-y rounded-2xl border border-peach-200 bg-white/80 p-3 font-mono text-sm
                text-ink placeholder-ink-faint focus:border-peach-400 focus:outline-none focus:ring-2 focus:ring-peach-200"
              spellCheck={false}
            />
          </section>

          {/* Resultado / Retroalimentación formal / informal */}
          <Card className="rise-in flex-1 min-h-[360px] overflow-hidden" bodyClassName="">
            <div className="flex flex-wrap gap-1 p-2 pb-0">
              {(
                [
                  ["output", "Resultado"],
                  ["formal", "Formal"],
                  ["informal", "Informal"],
                ] as const
              ).map(([tab, label]) => (
                <button
                  key={tab}
                  onClick={() => setActiveTab(tab)}
                  className={`flex items-center gap-1.5 rounded-full px-4 py-1.5 text-sm font-bold transition-colors ${
                    activeTab === tab ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {label}
                  {tab !== "output" && feedback[tab].loading && <Spinner className="h-3 w-3 border-lavender-500" />}
                  {tab !== "output" && !feedback[tab].loading && feedback[tab].text && (
                    <span className="h-1.5 w-1.5 rounded-full bg-current" aria-label="con respuesta" />
                  )}
                </button>
              ))}
            </div>
            {activeTab === "output" ? (
              <OutputPanel result={compileResult} isLoading={isCompiling} error={compileError} />
            ) : (
              <FeedbackPanel
                mode={activeTab}
                feedbackText={feedback[activeTab].text}
                meta={feedback[activeTab].meta}
                isLoading={feedback[activeTab].loading}
                error={feedback[activeTab].error}
                onRequestFeedback={() => handleFeedback(activeTab)}
                canRequestFeedback={!anyFeedbackLoading && !!code.trim()}
              />
            )}
          </Card>
        </div>
      </div>
    </div>
  );
}
