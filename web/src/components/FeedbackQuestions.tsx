"use client";

import { useState } from "react";
import { answerQuestion, askQuestion, Question } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

const ROLE_NAME: Record<string, string> = { teacher: "profesor", monitor: "monitor" };

function Thread({ q, children }: { q: Question; children?: React.ReactNode }) {
  return (
    <li className="space-y-1.5 rounded-2xl bg-white p-3 text-sm ring-1 ring-lavender-100">
      <p>
        <span className="font-bold">Pregunta</span>{" "}
        <span className="text-xs text-ink-faint">· {formatDateTime(q.created_at)}</span>
      </p>
      <p className="whitespace-pre-wrap text-ink-soft">{q.body}</p>
      {q.answer && q.answered_at && (
        <div className="rounded-xl bg-mint-50 p-2.5">
          <p className="text-xs font-bold text-mint-700">
            {q.answered_by} ({ROLE_NAME[q.answered_role ?? ""] ?? "docente"}) · {formatDateTime(q.answered_at)}
          </p>
          <p className="whitespace-pre-wrap">{q.answer}</p>
        </div>
      )}
      {children}
    </li>
  );
}

export function StudentQuestions({ submissionId, initial }: { submissionId: string; initial: Question[] }) {
  const [questions, setQuestions] = useState(initial);
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const q = await askQuestion(submissionId, body.trim());
      setQuestions((list) => [...list, q]);
      setBody("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo enviar la pregunta");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-2">
      <p className="text-xs font-bold uppercase tracking-wider text-ink-soft">¿Tienes una duda sobre esta retroalimentación?</p>
      {questions.length > 0 && (
        <ul className="space-y-2">
          {questions.map((q) => (
            <Thread key={q.id} q={q}>
              {!q.answered_at && <p className="text-xs text-ink-faint">Esperando respuesta de tu profesor o tu monitor.</p>}
            </Thread>
          ))}
        </ul>
      )}
      <form onSubmit={send} className="flex flex-wrap gap-2">
        <textarea
          value={body}
          onChange={(e) => setBody(e.target.value)}
          maxLength={1000}
          rows={2}
          placeholder="Escribe tu pregunta: por ejemplo, ¿qué quiere decir que la división es entera?"
          aria-label="Pregunta sobre la retroalimentación"
          className="min-w-0 flex-1 resize-y rounded-2xl border border-lavender-100 bg-white px-3 py-2 text-sm focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100"
        />
        <button
          type="submit"
          disabled={busy || body.trim().length < 2}
          className="self-end rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
        >
          {busy ? "Enviando..." : "Preguntar"}
        </button>
      </form>
      {error && <p className="text-xs font-semibold text-peach-700">{error}</p>}
    </div>
  );
}

function AnswerForm({ q, role, onAnswered }: { q: Question; role: "teacher" | "monitor"; onAnswered: (q: Question) => void }) {
  const [answer, setAnswer] = useState(q.answer ?? "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onAnswered(await answerQuestion(role, q.id, answer.trim()));
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo responder");
    } finally {
      setBusy(false);
    }
  };
  return (
    <form onSubmit={send} className="space-y-1.5">
      <textarea
        value={answer}
        onChange={(e) => setAnswer(e.target.value)}
        maxLength={5000}
        rows={2}
        placeholder="Tu respuesta al estudiante"
        aria-label="Respuesta a la pregunta"
        className="w-full resize-y rounded-xl border border-lavender-100 bg-lavender-50/60 px-3 py-2 text-sm focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100"
      />
      <div className="flex items-center gap-2">
        <button
          type="submit"
          disabled={busy || answer.trim().length < 2}
          className="rounded-full bg-lavender-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
        >
          {busy ? "Enviando..." : q.answered_at ? "Actualizar respuesta" : "Responder"}
        </button>
        {error && <span className="text-xs font-semibold text-peach-700">{error}</span>}
      </div>
    </form>
  );
}

export function StaffQuestions({
  role,
  initial,
  onChanged,
}: {
  role: "teacher" | "monitor";
  initial: Question[];
  onChanged?: () => void;
}) {
  const [questions, setQuestions] = useState(initial);
  const [editing, setEditing] = useState<string | null>(null);
  if (questions.length === 0) return null;
  const open = questions.filter((q) => !q.answered_at).length;
  return (
    <div className="space-y-2">
      <h4 className="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-ink-soft">
        Preguntas del estudiante sobre su retroalimentación
        {open > 0 && <span className="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] text-amber-800">{open} sin responder</span>}
      </h4>
      <ul className="space-y-2">
        {questions.map((q) => (
          <Thread key={q.id} q={q}>
            {!q.answered_at || editing === q.id ? (
              <AnswerForm
                q={q}
                role={role}
                onAnswered={(next) => {
                  setQuestions((list) => list.map((x) => (x.id === next.id ? next : x)));
                  setEditing(null);
                  onChanged?.();
                }}
              />
            ) : (
              <button type="button" onClick={() => setEditing(q.id)} className="text-xs font-bold text-lavender-600 hover:underline">
                Editar respuesta
              </button>
            )}
          </Thread>
        ))}
      </ul>
    </div>
  );
}
