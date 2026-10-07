"use client";

import { assignmentFileUrl, StudentAssignment } from "@/lib/api";
import { dueLabel, formatSize } from "@/lib/format";

/** Talleres de la clase elegida: el estudiante elige en cuál trabaja y lee su enunciado. */
export default function StudentAssignments({
  assignments,
  selectedId,
  onSelect,
}: {
  assignments: StudentAssignment[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
}) {
  const selected = assignments.find((a) => a.id === selectedId) ?? null;
  const chip = (active: boolean) =>
    `rounded-full px-3.5 py-1.5 text-sm font-bold transition ${
      active ? "bg-lavender-600 text-white shadow-soft" : "bg-lavender-50 text-ink-soft hover:bg-lavender-100 hover:text-ink"
    }`;

  return (
    <section className="rise-in rounded-3xl bg-white p-5 shadow-soft">
      <div className="flex flex-wrap items-center gap-2">
        <h2 className="mr-2 font-bold">Taller</h2>
        <div role="radiogroup" aria-label="Taller en el que trabajas" className="flex flex-wrap gap-2">
          <button type="button" role="radio" aria-checked={!selected} onClick={() => onSelect(null)} className={chip(!selected)}>
            Práctica libre
          </button>
          {assignments.map((a) => (
            <button
              key={a.id}
              type="button"
              role="radio"
              aria-checked={selected?.id === a.id}
              onClick={() => onSelect(a.id)}
              className={chip(selected?.id === a.id)}
            >
              {a.title}
            </button>
          ))}
        </div>
      </div>

      {selected ? (
        <div className="mt-4 grid gap-4 lg:grid-cols-3">
          <div className="lg:col-span-2">
            {dueLabel(selected.due_at) && (
              <p className="mb-2 text-xs font-bold uppercase tracking-wider text-peach-700">{dueLabel(selected.due_at)}</p>
            )}
            <div className="max-h-56 overflow-auto whitespace-pre-wrap rounded-2xl bg-lavender-50/70 p-4 text-sm leading-relaxed">
              {selected.statement}
            </div>
            <p className="mt-2 text-xs text-ink-faint">
              La retroalimentación de la IA tendrá en cuenta este enunciado, y tu envío quedará asociado a este taller.
            </p>
          </div>
          <div>
            <h3 className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-faint">Archivos del taller</h3>
            {selected.files.length === 0 ? (
              <p className="text-sm text-ink-soft">Este taller no tiene archivos adjuntos.</p>
            ) : (
              <ul className="space-y-1.5">
                {selected.files.map((f) => (
                  <li key={f.id}>
                    <a
                      href={assignmentFileUrl(selected.id, f.id)}
                      className="flex items-center gap-2 rounded-2xl bg-mint-50 px-3 py-2 text-sm transition hover:bg-mint-100"
                    >
                      <svg className="h-4 w-4 shrink-0 text-mint-700" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24" aria-hidden>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M3 16.5v2.25A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75V16.5M16.5 12 12 16.5m0 0L7.5 12m4.5 4.5V3" />
                      </svg>
                      <span className="min-w-0 flex-1 truncate font-semibold">{f.name}</span>
                      <span className="text-xs text-ink-faint">{formatSize(f.size)}</span>
                    </a>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ) : (
        <p className="mt-3 text-sm text-ink-soft">
          Elige un taller para ver su enunciado y sus archivos, o sigue en práctica libre.
        </p>
      )}
    </section>
  );
}
