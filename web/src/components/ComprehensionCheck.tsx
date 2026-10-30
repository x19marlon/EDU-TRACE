"use client";

import { useState } from "react";
import { rateFeedback, RatingKind, Understood } from "@/lib/api";

const OPTIONS: [Understood, string][] = [
  ["si", "Sí"],
  ["parcial", "En parte"],
  ["no", "No"],
];

export default function ComprehensionCheck({
  kind,
  submissionId,
  assignmentId,
  initial,
}: {
  kind: RatingKind;
  submissionId?: string;
  assignmentId?: string;
  initial?: Understood;
}) {
  const [choice, setChoice] = useState<Understood | undefined>(initial);
  const [comment, setComment] = useState("");
  const [saved, setSaved] = useState<"choice" | "comment" | null>(initial ? "choice" : null);
  const [error, setError] = useState<string | null>(null);

  const send = async (understood: Understood, text = "") => {
    setError(null);
    try {
      await rateFeedback({ kind, understood, comment: text, submission_id: submissionId, assignment_id: assignmentId });
      setChoice(understood);
      setSaved(text ? "comment" : "choice");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo guardar tu respuesta");
    }
  };

  return (
    <div className="space-y-2 rounded-2xl bg-white/70 p-3 text-xs">
      <div className="flex flex-wrap items-center gap-2">
        <span className="font-bold text-ink-soft">¿Entendiste esta retroalimentación?</span>
        <div role="radiogroup" aria-label="¿Entendiste esta retroalimentación?" className="flex gap-1">
          {OPTIONS.map(([value, label]) => (
            <button
              key={value}
              type="button"
              role="radio"
              aria-checked={choice === value}
              onClick={() => send(value)}
              className={`rounded-full px-3 py-1 font-bold transition ${
                choice === value ? "bg-lavender-600 text-white" : "bg-lavender-50 text-ink-soft hover:bg-lavender-100"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
        {saved && <span className="text-ink-faint">Gracias, tu profesor verá el resumen sin tu nombre.</span>}
      </div>
      {choice && choice !== "si" && saved !== "comment" && (
        <form
          onSubmit={(e) => {
            e.preventDefault();
            send(choice, comment.trim());
          }}
          className="flex flex-wrap gap-2"
        >
          <input
            value={comment}
            onChange={(e) => setComment(e.target.value)}
            maxLength={500}
            placeholder="¿Qué parte no quedó clara? (opcional)"
            aria-label="Qué parte no quedó clara"
            className="min-w-0 flex-1 rounded-full border border-lavender-100 bg-white px-3 py-1.5 focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
          <button
            type="submit"
            disabled={!comment.trim()}
            className="rounded-full bg-lavender-100 px-3 py-1.5 font-bold text-lavender-700 transition hover:bg-lavender-200 disabled:opacity-40"
          >
            Enviar
          </button>
        </form>
      )}
      {error && <p className="font-semibold text-peach-700">{error}</p>}
    </div>
  );
}
