"use client";

import { useState } from "react";
import { Card } from "@/components/ui";
import { Assignment, getSimilarity, SimilarPair } from "@/lib/api";

export default function SimilarityCard({
  groupId,
  assignments,
  onOpen,
}: {
  groupId: string;
  assignments: Assignment[];
  onOpen: (submissionId: string) => void;
}) {
  const [assignmentId, setAssignmentId] = useState("");
  const [result, setResult] = useState<{ compared: number; pairs: SimilarPair[] } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const compare = async () => {
    if (!assignmentId) return;
    setBusy(true);
    setError(null);
    try {
      setResult(await getSimilarity(groupId, assignmentId));
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudieron comparar las entregas");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card title="Código similar" className="rise-in">
      <p className="mb-3 text-xs text-ink-soft">
        Compara la última entrega de cada estudiante en un taller. Cambiar nombres de variables, comentarios o espacios no oculta el
        parecido, y lo que comparte casi toda la clase (como la plantilla del taller) no cuenta. Un parecido alto es una razón para
        revisar las dos entregas, no una prueba de copia.
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <label className="flex flex-wrap items-center gap-2 text-sm font-semibold text-ink-soft">
          Taller
          <select
            value={assignmentId}
            onChange={(e) => {
              setAssignmentId(e.target.value);
              setResult(null);
            }}
            className="rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
          >
            <option value="">Elige un taller</option>
            {assignments.map((a) => (
              <option key={a.id} value={a.id}>
                {a.title}
              </option>
            ))}
          </select>
        </label>
        <button
          type="button"
          onClick={compare}
          disabled={!assignmentId || busy}
          className="rounded-full bg-lavender-600 px-4 py-1.5 text-sm font-bold text-white transition hover:bg-lavender-700 disabled:opacity-50"
        >
          {busy ? "Comparando..." : "Comparar"}
        </button>
      </div>
      {error && <p className="mt-3 rounded-2xl bg-peach-50 px-4 py-2 text-sm text-peach-700">{error}</p>}
      {result && (
        <div className="mt-3 text-sm" role="status">
          <p className="text-ink-soft">
            {result.compared} {result.compared === 1 ? "entrega comparada" : "entregas comparadas"}.{" "}
            {result.pairs.length === 0 ? "No hay pares con un parecido de 50 % o más." : `${result.pairs.length} con un parecido de 50 % o más:`}
          </p>
          {result.pairs.length > 0 && (
            <ul className="mt-2 space-y-1.5">
              {result.pairs.map((p) => (
                <li key={p.a.submission_id + p.b.submission_id} className="flex flex-wrap items-center gap-2 rounded-2xl bg-lavender-50/70 px-3 py-2">
                  <span className="w-14 font-extrabold tabular-nums text-lavender-700">{Math.round(p.score * 100)} %</span>
                  <button type="button" onClick={() => onOpen(p.a.submission_id)} className="font-bold underline-offset-2 hover:underline">
                    {p.a.name}
                  </button>
                  <span aria-hidden className="text-ink-soft">↔</span>
                  <button type="button" onClick={() => onOpen(p.b.submission_id)} className="font-bold underline-offset-2 hover:underline">
                    {p.b.name}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Card>
  );
}
