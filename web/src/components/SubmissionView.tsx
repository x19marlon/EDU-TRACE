"use client";

import { useEffect, useState } from "react";
import {
  deleteMonitorFeedback,
  deleteMonitorFeedbackAsTeacher,
  getMonitorSubmission,
  getSubmission,
  MonitorFeedback,
  Submission,
  SubmissionMeta,
} from "@/lib/api";
import { Spinner } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import { MonitorFeedbackForm, MonitorNotes } from "@/components/MonitorFeedback";
import { formatDateTime, formatFeedback } from "@/lib/format";
import ReviewPanel from "@/components/ReviewPanel"
import AnnotatedCode from "@/components/AnnotatedCode";

/** Estado de compilación/ejecución de un envío o intento. */
export function StatusPill({ meta }: { meta: SubmissionMeta }) {
  if (!meta.success) return <span className="rounded-full bg-peach-100 px-2.5 py-0.5 text-[11px] font-bold text-peach-700">No compila</span>;
  if (meta.run_error || meta.exit_code !== 0)
    return <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-[11px] font-bold text-amber-800">Error al ejecutar</span>;
  return <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-[11px] font-bold text-mint-700">Compila y ejecuta</span>;
}

export function ReviewPill({ meta }: { meta: SubmissionMeta }) {
  if (meta.kind === "attempt") return null;
  const r = meta.review_state;
  if (!r) return <span className="rounded-full bg-amber-50 px-2.5 py-0.5 text-[11px] font-bold text-amber-800">Sin revisar</span>;
  if (r.status === "draft")
    return <span className="rounded-full bg-lavender-50 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">Borrador</span>;
  return (
    <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-[11px] font-bold text-mint-700">
      Devuelta{r.grade !== undefined ? ` · ${r.grade.toFixed(1)}` : ""}
    </span>
  );
}


export function KindPill({ meta }: { meta: SubmissionMeta }) {
  return meta.kind === "attempt" ? (
    <span className="rounded-full border border-lavender-200 px-2.5 py-0.5 text-[11px] font-bold text-ink-soft">Intento</span>
  ) : (
    <span className="rounded-full bg-lavender-600 px-2.5 py-0.5 text-[11px] font-bold text-white">Entrega oficial</span>
  );
}

export function CodeView({ code }: { code: string }) {
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

/** Una retroalimentación de la IA guardada con el envío. */
function SavedFeedback({ label, text, box, rule, note }: { label: string; text: string; box: string; rule: string; note: string }) {
  return (
    <div className={`rounded-2xl p-4 ${box}`}>
      <p className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-soft">{label}</p>
      <div className="text-sm leading-relaxed" dangerouslySetInnerHTML={{ __html: formatFeedback(text) }} />
      <p className={`mt-3 border-t pt-2 text-xs text-ink-faint ${rule}`}>{note}</p>
    </div>
  );
}

export function SubmissionDetail({
  id,
  studentName,
  assignmentTitle,
  onReviewSaved,
  mode = "teacher",
  onNotesChanged,
}: {
  id: string;
  studentName: string;
  assignmentTitle?: string;
  onReviewSaved?: () => void;
  mode?: "teacher" | "monitor";
  onNotesChanged?: () => void;
}) {
  const { user } = useAuth();
  const [sub, setSub] = useState<Submission | null>(null);
  const [notes, setNotes] = useState<MonitorFeedback[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setSub(null);
    setError(null);
    (mode === "monitor" ? getMonitorSubmission(id) : getSubmission(id))
      .then((s) => {
        setSub(s);
        setNotes(s.monitor_feedback ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Error al cargar el envío"));
  }, [id, mode]);

  const removeNote = async (note: MonitorFeedback) => {
    await (mode === "monitor" ? deleteMonitorFeedback(note.id) : deleteMonitorFeedbackAsTeacher(note.id));
    setNotes((list) => list.filter((n) => n.id !== note.id));
    onNotesChanged?.();
  };

  const aiNote =
    mode === "monitor"
      ? "Generada por IA. Es la orientación automática que vio el estudiante."
      : "Generada por IA. Úsala como apoyo: la valoración y la retroalimentación final son tuyas.";

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
        <KindPill meta={sub} />
        <StatusPill meta={sub} />
        <span className="rounded-full bg-lavender-50 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
          {assignmentTitle ?? "Práctica libre"}
        </span>
      </div>

      <AnnotatedCode
        key={sub.id}
        code={sub.code}
        submissionId={sub.id}
        mode={mode}
        initial={sub.annotations ?? []}
        canAnnotate={mode === "teacher" || sub.kind === "official"}
      />

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

      
      {sub.kind !== "attempt" && (
        <div className="space-y-3">
          <h4 className="text-xs font-bold uppercase tracking-wider text-ink-soft">Retroalimentación de la IA que vio el estudiante</h4>
          {sub.ai_feedback && (
            <SavedFeedback label="Formal" text={sub.ai_feedback} box="bg-mint-50" rule="border-mint-200" note={aiNote} />
          )}
          {sub.ai_feedback_informal && (
            <SavedFeedback label="Informal" text={sub.ai_feedback_informal} box="bg-peach-50" rule="border-peach-200" note={aiNote} />
          )}
          {!sub.ai_feedback && !sub.ai_feedback_informal && (
            <p className="text-sm text-ink-soft">El estudiante envió este código sin pedir retroalimentación a la IA.</p>
          )}
        </div>
      )}

      {sub.kind !== "attempt" && mode === "teacher" && notes.length > 0 && (
        <div className="space-y-2">
          <h4 className="text-xs font-bold uppercase tracking-wider text-ink-soft">Retroalimentación de monitores</h4>
          <MonitorNotes notes={notes} onDelete={removeNote} />
        </div>
      )}

      {sub.kind !== "attempt" && mode === "monitor" && (
        <div className="space-y-3">
          <h4 className="text-xs font-bold uppercase tracking-wider text-ink-soft">Retroalimentación de monitores</h4>
          <MonitorNotes notes={notes} canDelete={(n) => n.monitor_id === user?.id} onDelete={removeNote} />
          <MonitorFeedbackForm
            submissionId={sub.id}
            onAdded={(n) => {
              setNotes((list) => [...list, n]);
              onNotesChanged?.();
            }}
          />
        </div>
      )}

      {/* Revisión del profesor: solo para entregas oficiales */}
      {sub.kind !== "attempt" && mode === "teacher" && <ReviewPanel submissionId={sub.id} onSaved={onReviewSaved} />}
    </div>
  );
}
