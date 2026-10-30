"use client";

import { useRef, useState } from "react";
import { Assignment, assignmentFileUrl, createAssignment, deleteAssignment, updateAssignment } from "@/lib/api";
import { Card, SmallButton } from "@/components/ui";
import { formatDeadline, formatSize } from "@/lib/format";
import { DeadlineBadge } from "@/components/Deadline";

const MAX_FILES = 10;
const MAX_FILE_MB = 10;

const fieldClass =
  "w-full rounded-2xl border border-lavender-100 bg-lavender-50/60 px-4 py-2.5 text-sm placeholder-ink-faint " +
  "focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100";

function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const d = new Date(iso);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function AssignmentForm({
  groupId,
  initial,
  onSaved,
  onCancel,
}: {
  groupId: string;
  initial?: Assignment;
  onSaved: (a: Assignment) => void;
  onCancel?: () => void;
}) {
  const [title, setTitle] = useState(initial?.title ?? "");
  const [statement, setStatement] = useState(initial?.statement ?? "");
  const [rubric, setRubric] = useState(initial?.rubric ?? "");
  const [due, setDue] = useState(toLocalInput(initial?.due_at));
  const [lateMode, setLateMode] = useState<"accept" | "until" | "closed">(
    !initial || initial.allow_late ? (initial?.late_until ? "until" : "accept") : "closed"
  );
  const [lateUntil, setLateUntil] = useState(toLocalInput(initial?.late_until));
  const [penalty, setPenalty] = useState(initial?.late_penalty !== undefined ? String(initial.late_penalty) : "");
  const dueMs = due ? new Date(due).getTime() : null;
  const lateUntilMs = lateMode === "until" && lateUntil ? new Date(lateUntil).getTime() : null;
  const policyError =
    lateMode === "until" && !lateUntil
      ? "Indica hasta cuándo se reciben entregas tardías."
      : dueMs !== null && lateUntilMs !== null && lateUntilMs <= dueMs
        ? "El cierre de entregas tardías debe ser posterior a la fecha límite."
        : penalty.trim() && !(Number(penalty.replace(",", ".")) >= 0 && Number(penalty.replace(",", ".")) <= 5)
          ? "La penalización por día debe estar entre 0.0 y 5.0."
          : null;
  const [files, setFiles] = useState<File[]>([]);
  const [removed, setRemoved] = useState<Set<string>>(new Set());
  const kept = (initial?.files ?? []).filter((f) => !removed.has(f.id));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const addFiles = (list: FileList | null) => {
    if (!list) return;
    const next = [...files, ...Array.from(list)];
    const big = next.find((f) => f.size > MAX_FILE_MB * 1024 * 1024);
    if (big) setError(`«${big.name}» supera el máximo de ${MAX_FILE_MB} MB`);
    else if (kept.length + next.length > MAX_FILES) setError(`Máximo ${MAX_FILES} archivos por taller`);
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
      if (rubric.trim()) form.append("rubric", rubric.trim());
      if (due) {
        form.append("due_at", new Date(due).toISOString());
        form.append("allow_late", lateMode === "closed" ? "false" : "true");
        if (lateMode === "until" && lateUntil) form.append("late_until", new Date(lateUntil).toISOString());
        if (lateMode !== "closed" && penalty.trim()) form.append("late_penalty", penalty.trim().replace(",", "."));
      }
      files.forEach((f) => form.append("files", f, f.name));
      removed.forEach((id) => form.append("remove_files", id));
      const a = initial ? await updateAssignment(initial.id, form) : await createAssignment(groupId, form);
      if (!initial) {
        setTitle("");
        setStatement("");
        setRubric("");
        setDue("");
        setLateMode("accept");
        setLateUntil("");
        setPenalty("");
        setFiles([]);
      }
      onSaved(a);
    } catch (err) {
      setError(err instanceof Error ? err.message : initial ? "No se pudo guardar el taller" : "No se pudo crear el taller");
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
      <textarea
        value={rubric}
        onChange={(e) => setRubric(e.target.value)}
        placeholder="Criterios de evaluación (opcional): p. ej. Compila (1.0) · Lee bien la entrada (1.0) · Resultado correcto (2.0) · Legibilidad (1.0)"
        maxLength={5000}
        className={`${fieldClass} h-20 resize-y`}
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
        <span className="text-xs text-ink-faint">(opcional, en tu hora local)</span>
        {dueMs !== null && dueMs < Date.now() && (
          <span className="text-xs font-semibold text-peach-700">Esta fecha ya pasó.</span>
        )}
      </div>

      {due && (
        <div className="space-y-3 rounded-2xl bg-lavender-50/60 p-3">
          <label className="flex flex-wrap items-center gap-2 text-sm font-semibold text-ink-soft">
            Entregas tardías
            <select
              value={lateMode}
              onChange={(e) => setLateMode(e.target.value as "accept" | "until" | "closed")}
              className="rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
            >
              <option value="accept">Se aceptan y quedan marcadas como tardías</option>
              <option value="until">Se aceptan hasta una fecha de cierre</option>
              <option value="closed">No se aceptan después de la fecha límite</option>
            </select>
          </label>
          {lateMode === "until" && (
            <label className="flex flex-wrap items-center gap-2 text-sm font-semibold text-ink-soft">
              Cierre definitivo
              <input
                type="datetime-local"
                value={lateUntil}
                min={due}
                onChange={(e) => setLateUntil(e.target.value)}
                className="rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
              />
            </label>
          )}
          {lateMode !== "closed" && (
            <label className="flex flex-wrap items-center gap-2 text-sm font-semibold text-ink-soft">
              Penalización sugerida por día de retraso
              <input
                value={penalty}
                onChange={(e) => setPenalty(e.target.value)}
                inputMode="decimal"
                placeholder="p. ej. 0.5"
                className="w-24 rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-center text-sm text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
              />
              <span className="text-xs font-normal text-ink-faint">
                (opcional; cada día iniciado de retraso cuenta como uno. Al revisar verás la sugerencia y decides tú.)
              </span>
            </label>
          )}
          {policyError && <p className="text-xs font-semibold text-peach-700">{policyError}</p>}
        </div>
      )}

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
        {initial && initial.files.length > 0 && (
          <ul className="mt-2 space-y-1">
            {initial.files.map((f) => (
              <li key={f.id} className="flex items-center gap-2 rounded-xl bg-lavender-50 px-3 py-1.5 text-sm">
                <span className={`min-w-0 flex-1 truncate ${removed.has(f.id) ? "text-ink-faint line-through" : ""}`}>{f.name}</span>
                <span className="text-xs text-ink-faint">{formatSize(f.size)}</span>
                <button
                  type="button"
                  onClick={() =>
                    setRemoved((prev) => {
                      const next = new Set(prev);
                      if (next.has(f.id)) next.delete(f.id);
                      else next.add(f.id);
                      return next;
                    })
                  }
                  className="text-xs font-bold text-peach-700 hover:underline"
                >
                  {removed.has(f.id) ? "Conservar" : "Quitar"}
                </button>
              </li>
            ))}
          </ul>
        )}
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

      <div className="flex flex-wrap gap-2">
        <button
          type="submit"
          disabled={busy || title.trim().length < 2 || !statement.trim() || (!!due && !!policyError)}
          className="rounded-full bg-lavender-600 px-5 py-2.5 text-sm font-bold text-white shadow-soft transition
            hover:bg-lavender-700 disabled:opacity-40"
        >
          {busy ? (initial ? "Guardando..." : "Publicando...") : initial ? "Guardar cambios" : "Publicar taller"}
        </button>
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            className="rounded-full px-4 py-2.5 text-sm font-bold text-ink-soft transition hover:bg-lavender-50"
          >
            Cancelar
          </button>
        )}
      </div>
    </form>
  );
}

/** Talleres de una clase: crearlos y ver los publicados. */
export default function TeacherAssignments({
  groupId,
  assignments,
  submissionCounts,
  onCreated,
  onUpdated,
  onDeleted,
}: {
  groupId: string;
  assignments: Assignment[];
  submissionCounts: Map<string, number>;
  onCreated: (a: Assignment) => void;
  onUpdated: (a: Assignment) => void;
  onDeleted: (id: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const remove = async (a: Assignment) => {
    if (
      !window.confirm(
        `¿Eliminar el taller «${a.title}»? Se borrarán sus archivos. Las entregas que respondían a él se conservan como práctica libre.`
      )
    )
      return;
    setError(null);
    try {
      await deleteAssignment(a.id);
      onDeleted(a.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo eliminar el taller");
    }
  };

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
          <AssignmentForm
            groupId={groupId}
            onSaved={(a) => {
              setOpen(false);
              onCreated(a);
            }}
          />
        </div>
      )}

      {error && <p className="mb-3 rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}

      {assignments.length === 0 ? (
        <p className="text-sm text-ink-soft">
          Aún no hay talleres en esta clase. Crea uno con su enunciado y, si quieres, archivos de apoyo: tus
          estudiantes lo verán al entrar al compilador y la IA usará el enunciado al darles retroalimentación.
        </p>
      ) : (
        <ul className="space-y-3">
          {assignments.map((a) =>
            editing === a.id ? (
              <li key={a.id} className="rounded-3xl border border-lavender-100 p-4">
                <AssignmentForm
                  groupId={groupId}
                  initial={a}
                  onCancel={() => setEditing(null)}
                  onSaved={(updated) => {
                    setEditing(null);
                    onUpdated(updated);
                  }}
                />
              </li>
            ) : (
            <li key={a.id}>
              <details className="group rounded-2xl bg-lavender-50/60 px-4 py-3 open:bg-lavender-50">
                <summary className="flex cursor-pointer list-none flex-wrap items-center gap-x-3 gap-y-1">
                  <span className="min-w-0 flex-1 font-bold">{a.title}</span>
                  <DeadlineBadge assignment={a} />
                  <span className="text-xs text-ink-soft">
                    {a.files.length} {a.files.length === 1 ? "archivo" : "archivos"} · {submissionCounts.get(a.id) ?? 0} entregas
                  </span>
                  <svg className="h-4 w-4 text-ink-faint transition-transform group-open:rotate-180" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
                    <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clipRule="evenodd" />
                  </svg>
                </summary>
                <div className="mt-3 space-y-3">
                  {a.due_at && (
                    <p className="text-xs text-ink-soft">
                      Fecha límite: <span className="font-semibold text-ink">{formatDeadline(a.due_at)}</span>
                      {" · "}
                      {!a.allow_late
                        ? "no se aceptan tardías"
                        : a.late_until
                          ? `tardías hasta el ${formatDeadline(a.late_until)}`
                          : "se aceptan tardías"}
                      {a.late_penalty !== undefined && a.allow_late && ` · penalización sugerida −${a.late_penalty.toFixed(1)} por día`}
                    </p>
                  )}
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
                  <div className="flex flex-wrap gap-1 border-t border-lavender-100 pt-2">
                    <SmallButton onClick={() => setEditing(a.id)}>Editar taller</SmallButton>
                    <SmallButton tone="danger" onClick={() => remove(a)}>
                      Eliminar taller
                    </SmallButton>
                  </div>
                </div>
              </details>
            </li>
            )
          )}
        </ul>
      )}
    </Card>
  );
}
