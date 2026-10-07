"use client";

import { useRef, useState } from "react";
import { Assignment, assignmentFileUrl, createAssignment } from "@/lib/api";
import { Card } from "@/components/ui";
import { dueLabel, formatSize } from "@/lib/format";

const MAX_FILES = 10;
const MAX_FILE_MB = 10;

const fieldClass =
  "w-full rounded-2xl border border-lavender-100 bg-lavender-50/60 px-4 py-2.5 text-sm placeholder-ink-faint " +
  "focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100";

function NewAssignmentForm({ groupId, onCreated }: { groupId: string; onCreated: (a: Assignment) => void }) {
  const [title, setTitle] = useState("");
  const [statement, setStatement] = useState("");
  const [due, setDue] = useState("");
  const [files, setFiles] = useState<File[]>([]);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const addFiles = (list: FileList | null) => {
    if (!list) return;
    const next = [...files, ...Array.from(list)];
    const big = next.find((f) => f.size > MAX_FILE_MB * 1024 * 1024);
    if (big) setError(`«${big.name}» supera el máximo de ${MAX_FILE_MB} MB`);
    else if (next.length > MAX_FILES) setError(`Máximo ${MAX_FILES} archivos por taller`);
    else {
      setError(null);
      setFiles(next);
    }
    if (fileInput.current) fileInput.current.value = "";
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("title", title.trim());
      form.append("statement", statement.trim());
      // datetime-local no trae zona horaria: se interpreta en la hora local del profesor.
      if (due) form.append("due_at", new Date(due).toISOString());
      files.forEach((f) => form.append("files", f, f.name));
      const a = await createAssignment(groupId, form);
      setTitle("");
      setStatement("");
      setDue("");
      setFiles([]);
      onCreated(a);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo crear el taller");
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-3">
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Título (p. ej. Taller 1: ciclos)"
        maxLength={120}
        required
        className={fieldClass}
      />
      <textarea
        value={statement}
        onChange={(e) => setStatement(e.target.value)}
        placeholder="Enunciado: qué debe hacer el programa, entradas y salidas esperadas, ejemplos…"
        required
        maxLength={20000}
        className={`${fieldClass} h-36 resize-y`}
      />
      <div className="flex flex-wrap items-center gap-3">
        <label className="flex items-center gap-2 text-sm font-semibold text-ink-soft">
          Fecha límite
          <input
            type="datetime-local"
            value={due}
            onChange={(e) => setDue(e.target.value)}
            className="rounded-full border border-lavender-100 bg-lavender-50/60 px-3 py-1.5 text-sm text-ink
              focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
        </label>
        <span className="text-xs text-ink-faint">(opcional)</span>
      </div>

      <div>
        <input ref={fileInput} type="file" multiple className="hidden" onChange={(e) => addFiles(e.target.files)} />
        <button
          type="button"
          onClick={() => fileInput.current?.click()}
          className="rounded-full bg-mint-100 px-4 py-2 text-sm font-bold text-mint-700 transition hover:bg-mint-200"
        >
          Adjuntar archivos
        </button>
        <span className="ml-2 text-xs text-ink-faint">Hasta {MAX_FILES} archivos de {MAX_FILE_MB} MB</span>
        {files.length > 0 && (
          <ul className="mt-2 space-y-1">
            {files.map((f, i) => (
              <li key={`${f.name}-${i}`} className="flex items-center gap-2 rounded-xl bg-lavender-50 px-3 py-1.5 text-sm">
                <span className="min-w-0 flex-1 truncate">{f.name}</span>
                <span className="text-xs text-ink-faint">{formatSize(f.size)}</span>
                <button
                  type="button"
                  onClick={() => setFiles(files.filter((_, j) => j !== i))}
                  className="text-xs font-bold text-peach-700 hover:underline"
                  aria-label={`Quitar ${f.name}`}
                >
                  Quitar
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>

      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}

      <button
        type="submit"
        disabled={busy || title.trim().length < 2 || !statement.trim()}
        className="rounded-full bg-lavender-600 px-5 py-2.5 text-sm font-bold text-white shadow-soft transition
          hover:bg-lavender-700 disabled:opacity-40"
      >
        {busy ? "Publicando..." : "Publicar taller"}
      </button>
    </form>
  );
}

/** Talleres de una clase: crearlos y ver los publicados. */
export default function TeacherAssignments({
  groupId,
  assignments,
  submissionCounts,
  onCreated,
}: {
  groupId: string;
  assignments: Assignment[];
  submissionCounts: Map<string, number>;
  onCreated: (a: Assignment) => void;
}) {
  const [open, setOpen] = useState(false);

  return (
    <Card
      title="Talleres"
      className="rise-in"
      action={
        <button
          onClick={() => setOpen((o) => !o)}
          className="rounded-full bg-lavender-100 px-4 py-1.5 text-sm font-bold text-lavender-700 transition hover:bg-lavender-200"
        >
          {open ? "Cancelar" : "Nuevo taller"}
        </button>
      }
    >
      {open && (
        <div className="mb-5 rounded-3xl border border-lavender-100 p-4">
          <NewAssignmentForm
            groupId={groupId}
            onCreated={(a) => {
              setOpen(false);
              onCreated(a);
            }}
          />
        </div>
      )}

      {assignments.length === 0 ? (
        <p className="text-sm text-ink-soft">
          Aún no hay talleres en esta clase. Crea uno con su enunciado y, si quieres, archivos de apoyo: tus
          estudiantes lo verán al entrar al compilador y la IA usará el enunciado al darles retroalimentación.
        </p>
      ) : (
        <ul className="space-y-3">
          {assignments.map((a) => (
            <li key={a.id}>
              <details className="group rounded-2xl bg-lavender-50/60 px-4 py-3 open:bg-lavender-50">
                <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1">
                  <span className="min-w-0 flex-1 font-bold">{a.title}</span>
                  {dueLabel(a.due_at) && <span className="text-xs font-semibold text-peach-700">{dueLabel(a.due_at)}</span>}
                  <span className="text-xs text-ink-soft">
                    {a.files.length} {a.files.length === 1 ? "archivo" : "archivos"} · {submissionCounts.get(a.id) ?? 0} entregas
                  </span>
                  <svg className="h-4 w-4 text-ink-faint transition-transform group-open:rotate-180" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                    <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clipRule="evenodd" />
                  </svg>
                </summary>
                <div className="mt-3 space-y-3">
                  <div className="max-h-60 overflow-auto whitespace-pre-wrap rounded-2xl bg-white p-3 text-sm leading-relaxed">
                    {a.statement}
                  </div>
                  {a.files.length > 0 && (
                    <ul className="flex flex-wrap gap-2">
                      {a.files.map((f) => (
                        <li key={f.id}>
                          <a
                            href={assignmentFileUrl(a.id, f.id)}
                            className="inline-flex items-center gap-2 rounded-full bg-white px-3 py-1.5 text-sm font-semibold transition hover:bg-mint-50"
                          >
                            {f.name}
                            <span className="text-xs font-normal text-ink-faint">{formatSize(f.size)}</span>
                          </a>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </details>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
