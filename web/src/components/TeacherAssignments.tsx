"use client";

import { useRef, useState } from "react";
import { Assignment, assignmentFileUrl, createAssignment, Criterion, deleteAssignment, TestCase, TestMatch, updateAssignment } from "@/lib/api";
import { Card, SmallButton } from "@/components/ui";
import { formatDeadline, formatSize } from "@/lib/format";
import { DeadlineBadge } from "@/components/Deadline";

const MAX_FILES = 10;
const MAX_FILE_MB = 10;
const MAX_TESTS = 10;

const emptyTest = (): TestCase => ({ name: "", stdin: "", expected: "", match: "contiene", hidden: false });

export function TestsEditor({
  tests,
  onChange,
  student = false,
}: {
  tests: TestCase[];
  onChange: (tests: TestCase[]) => void;
  student?: boolean;
}) {
  const update = (i: number, patch: Partial<TestCase>) => onChange(tests.map((t, j) => (j === i ? { ...t, ...patch } : t)));
  const small =
    "w-full rounded-xl border border-lavender-100 bg-white px-3 py-1.5 font-mono text-xs focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100";
  return (
    <div className="space-y-2 rounded-2xl bg-lavender-50/60 p-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold text-ink-soft">{student ? "Mis casos de prueba" : "Casos de prueba"}</span>
        <span className="text-xs text-ink-faint">
          {student
            ? "Cada vez que compilas, tu programa también se prueba con estas entradas. Se guardan solo en este navegador."
            : "(opcional) Cada envío y cada compilación del estudiante se ejecuta con estas entradas y se compara con la salida esperada."}
        </span>
      </div>
      {tests.map((t, i) => (
        <div key={i} className="space-y-2 rounded-2xl bg-white p-3 ring-1 ring-lavender-100">
          <div className="flex flex-wrap items-center gap-2">
            <input
              value={t.name}
              onChange={(e) => update(i, { name: e.target.value })}
              maxLength={80}
              placeholder={`Caso ${i + 1}`}
              aria-label={`Nombre del caso ${i + 1}`}
              className="min-w-[9rem] flex-1 rounded-full border border-lavender-100 bg-lavender-50/60 px-3 py-1.5 text-sm focus:outline-none focus:ring-4 focus:ring-lavender-100"
            />
            <select
              value={t.match}
              onChange={(e) => update(i, { match: e.target.value as TestMatch })}
              aria-label={`Comparación del caso ${i + 1}`}
              className="rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-xs font-semibold text-ink focus:outline-none focus:ring-4 focus:ring-lavender-100"
            >
              <option value="contiene">La salida contiene lo esperado</option>
              <option value="exacta">La salida es exactamente lo esperado</option>
            </select>
            {!student && (
              <label className="flex items-center gap-1.5 text-xs font-semibold text-ink-soft">
                <input type="checkbox" checked={t.hidden} onChange={(e) => update(i, { hidden: e.target.checked })} />
                Oculto al estudiante
              </label>
            )}
            <button
              type="button"
              onClick={() => onChange(tests.filter((_, j) => j !== i))}
              className="text-xs font-bold text-peach-700 hover:underline"
            >
              Quitar
            </button>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            <textarea
              value={t.stdin}
              onChange={(e) => update(i, { stdin: e.target.value })}
              maxLength={4000}
              placeholder="Entrada (lo que se escribe en cin), p. ej. 5"
              aria-label={`Entrada del caso ${i + 1}`}
              className={`${small} h-16 resize-y`}
            />
            <textarea
              value={t.expected}
              onChange={(e) => update(i, { expected: e.target.value })}
              maxLength={4000}
              placeholder="Salida esperada, p. ej. Suma: 15"
              aria-label={`Salida esperada del caso ${i + 1}`}
              className={`${small} h-16 resize-y`}
            />
          </div>
        </div>
      ))}
      {tests.length < MAX_TESTS && (
        <button
          type="button"
          onClick={() => onChange([...tests, emptyTest()])}
          className="rounded-full bg-white px-4 py-1.5 text-sm font-bold text-lavender-700 ring-1 ring-lavender-200 transition hover:bg-lavender-50"
        >
          Agregar caso de prueba
        </button>
      )}
      <p className="text-[11px] text-ink-faint">
        Se ignoran los espacios repetidos y las líneas vacías. «Contiene» acepta mensajes extra como «Ingrese un número:».
        {!student && " Los casos ocultos solo le muestran al estudiante si pasa o no."}
      </p>
    </div>
  );
}

function CriteriaEditor({ criteria, onChange }: { criteria: Criterion[]; onChange: (c: Criterion[]) => void }) {
  const update = (i: number, patch: Partial<Criterion>) => onChange(criteria.map((c, j) => (j === i ? { ...c, ...patch } : c)));
  const total = criteria.reduce((n, c) => n + (Number(c.points) || 0), 0);
  return (
    <div className="space-y-2 rounded-2xl bg-mint-50/70 p-3">
      <div className="flex flex-wrap items-center gap-2">
        <span className="text-sm font-semibold text-ink-soft">Rúbrica con puntaje</span>
        <span className="text-xs text-ink-faint">
          (opcional) Al revisar asignas un puntaje por criterio y EduTrace calcula la nota sobre 5.0. Total: {total.toFixed(1)} puntos.
        </span>
      </div>
      {criteria.map((c, i) => (
        <div key={i} className="flex flex-wrap items-center gap-2">
          <input
            value={c.name}
            onChange={(e) => update(i, { name: e.target.value })}
            maxLength={120}
            placeholder="Criterio, p. ej. Lee bien la entrada"
            aria-label={`Nombre del criterio ${i + 1}`}
            className="min-w-0 flex-1 rounded-full border border-mint-200 bg-white px-3 py-1.5 text-sm focus:outline-none focus:ring-4 focus:ring-mint-100"
          />
          <input
            type="number"
            min={0.1}
            max={100}
            step={0.1}
            value={Number.isFinite(c.points) ? c.points : ""}
            onChange={(e) => update(i, { points: e.target.valueAsNumber })}
            aria-label={`Puntos del criterio ${i + 1}`}
            className="w-24 rounded-full border border-mint-200 bg-white px-3 py-1.5 text-center text-sm focus:outline-none focus:ring-4 focus:ring-mint-100"
          />
          <span className="text-xs text-ink-soft">puntos</span>
          <button type="button" onClick={() => onChange(criteria.filter((_, j) => j !== i))} className="text-xs font-bold text-peach-700 hover:underline">
            Quitar
          </button>
        </div>
      ))}
      {criteria.length < 10 && (
        <button
          type="button"
          onClick={() => onChange([...criteria, { id: "", name: "", points: 1 }])}
          className="rounded-full bg-white px-4 py-1.5 text-sm font-bold text-mint-700 ring-1 ring-mint-200 transition hover:bg-mint-50"
        >
          Agregar criterio
        </button>
      )}
    </div>
  );
}

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
  const [evaluation, setEvaluation] = useState(initial?.evaluation ?? false);
  const [tests, setTests] = useState<TestCase[]>(initial?.tests ?? []);
  const [criteria, setCriteria] = useState<Criterion[]>(initial?.criteria ?? []);
  const criteriaError = criteria.some((c) => !c.name.trim() || !(c.points > 0 && c.points <= 100))
    ? "Cada criterio necesita un nombre y un puntaje entre 0.1 y 100."
    : null;
  const testsError = tests.some((t) => !t.expected.trim()) ? "Cada caso de prueba necesita su salida esperada." : null;
  const dueMs = due ? new Date(due).getTime() : null;
  const lateUntilMs = lateMode === "until" && lateUntil ? new Date(lateUntil).getTime() : null;
  const policyError =
    evaluation && !due
      ? "Un taller en modo evaluación necesita fecha límite: la IA queda apagada para la clase hasta esa fecha."
      : lateMode === "until" && !lateUntil
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
      form.append("evaluation", evaluation ? "true" : "false");
      form.append(
        "tests",
        JSON.stringify(tests.map((t) => ({ name: t.name.trim(), stdin: t.stdin, expected: t.expected, match: t.match, hidden: t.hidden })))
      );
      form.append("criteria", JSON.stringify(criteria.map((c) => ({ name: c.name.trim(), points: c.points }))));
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
        setEvaluation(false);
        setTests([]);
        setCriteria([]);
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

      <label className="flex items-start gap-3 rounded-2xl bg-amber-50 px-4 py-3 text-sm">
        <input
          type="checkbox"
          checked={evaluation}
          onChange={(e) => setEvaluation(e.target.checked)}
          className="mt-0.5 h-4 w-4 accent-lavender-600"
        />
        <span>
          <span className="font-bold text-ink">Modo evaluación (parcial o quiz)</span>
          <span className="block text-ink-soft">
            Desde que guardas el taller hasta su fecha límite (o el cierre de entregas tardías), los estudiantes de la clase no
            pueden usar la IA ni el chat en ninguna parte de EduTrace. Tú sigues recibiendo la propuesta de la IA para revisar.
          </span>
        </span>
      </label>
      {evaluation && !due && <p className="text-xs font-semibold text-peach-700">{policyError}</p>}

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

      <TestsEditor tests={tests} onChange={setTests} />
      {testsError && <p className="text-xs font-semibold text-peach-700">{testsError}</p>}
      <CriteriaEditor criteria={criteria} onChange={setCriteria} />
      {criteriaError && <p className="text-xs font-semibold text-peach-700">{criteriaError}</p>}

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
          disabled={busy || title.trim().length < 2 || !statement.trim() || (!!due && !!policyError) || (evaluation && !due) || !!testsError || !!criteriaError}
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
                  {a.evaluation && (
                    <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-bold text-amber-900">Evaluación · sin IA</span>
                  )}
                  <DeadlineBadge assignment={a} />
                  <span className="text-xs text-ink-soft">
                    {a.files.length} {a.files.length === 1 ? "archivo" : "archivos"} · {(a.tests ?? []).length}{" "}
                    {(a.tests ?? []).length === 1 ? "caso de prueba" : "casos de prueba"} · {submissionCounts.get(a.id) ?? 0} entregas
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
                  {(a.tests ?? []).length > 0 && (
                    <ul className="space-y-1">
                      {a.tests.map((t, i) => (
                        <li key={t.id ?? i} className="flex flex-wrap items-center gap-2 rounded-xl bg-white px-3 py-1.5 text-xs">
                          <span className="font-bold">{t.name}</span>
                          <span className="font-mono text-ink-soft">
                            {JSON.stringify(t.stdin)} → {JSON.stringify(t.expected)}
                          </span>
                          <span className="text-ink-faint">{t.match === "exacta" ? "exacta" : "contiene"}</span>
                          {t.hidden && <span className="rounded-full bg-lavender-100 px-2 py-0.5 font-bold text-lavender-700">oculto</span>}
                        </li>
                      ))}
                    </ul>
                  )}
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
