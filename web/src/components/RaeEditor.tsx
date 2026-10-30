"use client";

import { useState } from "react";
import { createRAE, deleteRAE, RAE, updateRAE } from "@/lib/api";
import { SmallButton } from "@/components/ui";
import { OFFICIAL_RAES } from "@/lib/rae";

const inputClass =
  "rounded-full border border-lavender-100 bg-lavender-50/60 px-4 py-2 text-sm placeholder-ink-faint " +
  "focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100";

function RaeForm({
  initialCode,
  initialDescription = "",
  submitLabel,
  onSubmit,
  onCancel,
}: {
  initialCode: string;
  initialDescription?: string;
  submitLabel: string;
  onSubmit: (code: string, description: string) => Promise<void>;
  onCancel?: () => void;
}) {
  const [code, setCode] = useState(initialCode);
  const [description, setDescription] = useState(initialDescription);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      await onSubmit(code.trim(), description.trim());
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo guardar el RAE");
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-1.5">
      <div className="flex flex-wrap gap-2">
        <input
          value={code}
          onChange={(e) => setCode(e.target.value)}
          maxLength={20}
          placeholder="RAE 1"
          aria-label="Código del RAE"
          className={`${inputClass} w-24`}
        />
        <input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          maxLength={500}
          placeholder="Qué debe lograr el estudiante (p. ej. Diseña algoritmos con condicionales y ciclos)"
          aria-label="Descripción del RAE"
          className={`${inputClass} min-w-0 flex-1`}
        />
        <button
          type="submit"
          disabled={busy || !code.trim() || description.trim().length < 3}
          className="shrink-0 rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
        >
          {busy ? "Guardando..." : submitLabel}
        </button>
        {onCancel && (
          <button type="button" onClick={onCancel} className="rounded-full px-3 py-2 text-sm font-bold text-ink-soft hover:bg-lavender-50">
            Cancelar
          </button>
        )}
      </div>
      {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
    </form>
  );
}

export default function RaeEditor({
  courseId,
  raes,
  onChanged,
}: {
  courseId: string;
  raes: RAE[];
  onChanged: () => Promise<void>;
}) {
  const [editing, setEditing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const loadOfficial = async () => {
    setLoading(true);
    setError(null);
    try {
      for (const r of OFFICIAL_RAES) await createRAE(courseId, r.code, r.description);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudieron cargar los RAE oficiales");
    } finally {
      setLoading(false);
    }
  };

  const remove = async (r: RAE) => {
    if (!window.confirm(`¿Eliminar «${r.code}»? Se borrarán los niveles que ya se marcaron para este RAE en las revisiones.`)) return;
    setError(null);
    try {
      await deleteRAE(r.id);
      await onChanged();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo eliminar el RAE");
    }
  };

  return (
    <details className="group mb-4 rounded-2xl border border-lavender-100 px-4 py-3">
      <summary className="flex cursor-pointer list-none items-center gap-2">
        <span className="flex-1 text-sm font-bold">Resultados de Aprendizaje Esperados (RAE)</span>
        <span className="text-xs font-semibold text-ink-faint">{raes.length === 0 ? "Sin definir" : raes.length}</span>
        <svg className="h-4 w-4 text-ink-faint transition-transform group-open:rotate-180" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
          <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clipRule="evenodd" />
        </svg>
      </summary>
      <div className="mt-3 space-y-3">
        <p className="text-xs text-ink-soft">
          Al revisar cada entrega marcas el nivel de logro de cada RAE (bajo, medio o alto). Con eso se arma el diagrama de
          rendimiento de cada clase de esta materia.
        </p>
        {raes.length === 0 && (
          <button
            type="button"
            disabled={loading}
            onClick={loadOfficial}
            className="w-full rounded-2xl border border-dashed border-lavender-300 px-4 py-3 text-left text-sm transition hover:bg-lavender-50 disabled:opacity-50"
          >
            <span className="block font-bold text-lavender-700">
              {loading ? "Cargando..." : "Cargar los 4 RAE oficiales de Introducción a la Programación (33698)"}
            </span>
            <span className="block text-xs text-ink-soft">
              Los del programa del curso: caracterizar el problema, diseñar con construcciones básicas, organizar datos con colecciones e
              implementar en C++. Luego puedes editarlos.
            </span>
          </button>
        )}
        {raes.length > 0 && (
          <ul className="space-y-1.5">
            {raes.map((r) =>
              editing === r.id ? (
                <li key={r.id}>
                  <RaeForm
                    initialCode={r.code}
                    initialDescription={r.description}
                    submitLabel="Guardar"
                    onCancel={() => setEditing(null)}
                    onSubmit={async (code, description) => {
                      await updateRAE(r.id, code, description);
                      setEditing(null);
                      await onChanged();
                    }}
                  />
                </li>
              ) : (
                <li key={r.id} className="flex items-start gap-2 rounded-xl bg-lavender-50/70 px-3 py-2 text-sm">
                  <span className="shrink-0 font-bold text-lavender-700">{r.code}</span>
                  <span className="min-w-0 flex-1 text-ink-soft">{r.description}</span>
                  <SmallButton onClick={() => setEditing(r.id)}>Editar</SmallButton>
                  <SmallButton tone="danger" onClick={() => remove(r)}>
                    Eliminar
                  </SmallButton>
                </li>
              )
            )}
          </ul>
        )}
        <RaeForm
          key={raes.length}
          initialCode={`RAE ${raes.length + 1}`}
          submitLabel="Agregar RAE"
          onSubmit={async (code, description) => {
            await createRAE(courseId, code, description);
            await onChanged();
          }}
        />
        {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
      </div>
    </details>
  );
}
