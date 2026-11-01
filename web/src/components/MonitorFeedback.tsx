"use client";

import { useState } from "react";
import { addMonitorFeedback, MonitorFeedback } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { SmallButton } from "@/components/ui";

export function MonitorNotes({
  notes,
  canDelete,
  onDelete,
}: {
  notes: MonitorFeedback[];
  canDelete?: (note: MonitorFeedback) => boolean;
  onDelete?: (note: MonitorFeedback) => Promise<void>;
}) {
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (notes.length === 0) return null;

  const remove = async (note: MonitorFeedback) => {
    if (!onDelete || !window.confirm("¿Eliminar esta retroalimentación? El estudiante dejará de verla.")) return;
    setBusy(note.id);
    setError(null);
    try {
      await onDelete(note);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo eliminar");
    } finally {
      setBusy(null);
    }
  };

  return (
    <div className="space-y-2">
      <ul className="space-y-2">
        {notes.map((n) => (
          <li key={n.id} className="rounded-2xl bg-mint-50 p-4">
            <div className="mb-1.5 flex flex-wrap items-center gap-2 text-xs">
              <span className="font-bold text-mint-700">{n.monitor_name}</span>
              <span className="text-ink-faint">Monitor · {formatDateTime(n.created_at)}</span>
              <span className="flex-1" />
              {onDelete && (!canDelete || canDelete(n)) && (
                <SmallButton tone="danger" disabled={busy === n.id} onClick={() => remove(n)}>
                  {busy === n.id ? "Eliminando..." : "Eliminar"}
                </SmallButton>
              )}
            </div>
            <p className="whitespace-pre-wrap text-sm leading-relaxed">{n.body}</p>
          </li>
        ))}
      </ul>
      {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
    </div>
  );
}

export function MonitorFeedbackForm({
  submissionId,
  onAdded,
}: {
  submissionId: string;
  onAdded: (note: MonitorFeedback) => void;
}) {
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      onAdded(await addMonitorFeedback(submissionId, body.trim()));
      setBody("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo enviar");
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-2 rounded-2xl border border-mint-200 p-4">
      <textarea
        value={body}
        onChange={(e) => setBody(e.target.value)}
        maxLength={5000}
        placeholder="Escribe tu retroalimentación: qué hizo bien, qué línea revisar y por qué. Da pistas y preguntas guía, no la solución."
        className="h-32 w-full resize-y rounded-2xl border border-lavender-100 bg-lavender-50/60 px-4 py-3 text-sm placeholder-ink-faint
          focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100"
      />
      <div className="flex flex-wrap items-center gap-3">
        <button
          type="submit"
          disabled={busy || !body.trim()}
          className="rounded-full bg-mint-700 px-5 py-2 text-sm font-bold text-white shadow-soft transition hover:bg-ink disabled:opacity-40"
        >
          {busy ? "Enviando..." : "Enviar al estudiante"}
        </button>
        <p className="flex-1 text-xs text-ink-faint">
          El estudiante la verá en «Mis entregas». Como monitor no calificas: la nota la decide el profesor.
        </p>
      </div>
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}
    </form>
  );
}
