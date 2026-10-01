"use client";

import { useState } from "react";
import { joinGroup, StudentGroup } from "@/lib/api";

/** Clases del estudiante: elegir a cuál enviar y unirse con un código. */
export default function StudentClasses({
  groups,
  selectedId,
  onSelect,
  onJoined,
}: {
  groups: StudentGroup[];
  selectedId: string | null;
  onSelect: (id: string) => void;
  onJoined: (g: StudentGroup) => void;
}) {
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const join = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const g = await joinGroup(code);
      setCode("");
      onJoined(g);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo unir a la clase");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-wrap items-start justify-between gap-4 border-t border-lavender-100 pt-4">
      <div className="min-w-0 flex-1">
        <p className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-faint">
          {groups.length ? "Enviar a la clase" : "Mis clases"}
        </p>
        {groups.length === 0 ? (
          <p className="text-sm text-ink-soft">
            Aún no estás en ninguna clase. Pídele el código a tu profesor para poder enviarle tu código.
          </p>
        ) : (
          <div role="radiogroup" aria-label="Clase a la que envías" className="flex flex-wrap gap-2">
            {groups.map((g) => (
              <button
                key={g.id}
                type="button"
                role="radio"
                aria-checked={selectedId === g.id}
                onClick={() => onSelect(g.id)}
                className={`rounded-2xl px-3.5 py-2 text-left transition ${
                  selectedId === g.id
                    ? "bg-lavender-600 text-white shadow-soft"
                    : "bg-lavender-50 text-ink hover:bg-lavender-100"
                }`}
              >
                <span className="block text-sm font-bold">{g.course_name}</span>
                <span className={`block text-xs ${selectedId === g.id ? "text-lavender-100" : "text-ink-soft"}`}>
                  {g.name} · {g.teacher_name}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>

      <form onSubmit={join} className="w-full space-y-1 sm:w-auto">
        <p className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-faint">Unirme a una clase</p>
        <div className="flex gap-2">
          <input
            value={code}
            onChange={(e) => setCode(e.target.value.toUpperCase())}
            placeholder="Código (p. ej. PQ54X8)"
            maxLength={12}
            aria-label="Código de la clase"
            className="w-full min-w-0 rounded-full border border-lavender-100 bg-lavender-50/60 px-4 py-2 font-mono text-sm
              tracking-widest placeholder:font-sans placeholder:tracking-normal placeholder-ink-faint
              focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100 sm:w-48"
          />
          <button
            type="submit"
            disabled={busy || code.trim().length < 4}
            className="shrink-0 rounded-full bg-lavender-100 px-4 py-2 text-sm font-bold text-lavender-700 transition
              hover:bg-lavender-200 disabled:opacity-40"
          >
            {busy ? "Uniendo..." : "Unirme"}
          </button>
        </div>
        {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
      </form>
    </div>
  );
}
