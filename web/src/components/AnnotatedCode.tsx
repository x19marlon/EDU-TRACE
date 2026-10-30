"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { Annotation, AnnotationInput, AnnotationKind, createAnnotation, deleteAnnotation, sendMonitorAnnotations, Stroke } from "@/lib/api";
import { useAuth } from "@/components/AuthProvider";
import { SmallButton, Spinner } from "@/components/ui";
import { formatDateTime } from "@/lib/format";

const LINE_H = 22;
const CODE_X = 56;
const MAX_POINTS = 1999;
const MAX_STROKES = 100;

const PENS = [
  { color: "#D64545", label: "Rojo" },
  { color: "#7154CC", label: "Morado" },
  { color: "#3A8F6F", label: "Verde" },
];

type LineKind = "highlight" | "strike" | "good";

const LINE_KINDS: { kind: LineKind; label: string; chip: string; bg: string; edge: string }[] = [
  { kind: "highlight", label: "Resaltar", chip: "bg-amber-100 text-amber-800", bg: "bg-amber-50", edge: "shadow-[inset_3px_0_0_#E0A800]" },
  { kind: "strike", label: "Tachar", chip: "bg-peach-100 text-peach-700", bg: "bg-peach-50", edge: "shadow-[inset_3px_0_0_#C9683A]" },
  { kind: "good", label: "Bien", chip: "bg-mint-100 text-mint-700", bg: "bg-mint-50", edge: "shadow-[inset_3px_0_0_#3A8F6F]" },
];

const KIND_LABEL: Record<AnnotationKind, string> = {
  highlight: "Resaltado",
  strike: "Tachado",
  good: "Bien hecho",
  note: "Nota general",
  drawing: "Dibujo",
};

const KIND_CHIP: Record<AnnotationKind, string> = {
  highlight: "bg-amber-100 text-amber-800",
  strike: "bg-peach-100 text-peach-700",
  good: "bg-mint-100 text-mint-700",
  note: "bg-lavender-100 text-lavender-700",
  drawing: "bg-lavender-100 text-lavender-700",
};

type Mode = "teacher" | "monitor" | "student";

const round2 = (n: number) => Math.round(n * 100) / 100;

const fieldClass =
  "w-full resize-y rounded-2xl border border-lavender-100 bg-white px-3 py-2 text-sm placeholder-ink-faint " +
  "focus:border-lavender-300 focus:outline-none focus:ring-4 focus:ring-lavender-100";

function linesLabel(a: { line_start?: number; line_end?: number }) {
  if (!a.line_start) return null;
  return a.line_start === a.line_end ? `Línea ${a.line_start}` : `Líneas ${a.line_start}–${a.line_end}`;
}

export default function AnnotatedCode({
  code,
  submissionId,
  mode,
  initial,
  canAnnotate,
}: {
  code: string;
  submissionId: string;
  mode: Mode;
  initial: Annotation[];
  canAnnotate: boolean;
}) {
  const { user } = useAuth();
  const lines = useMemo(() => code.replace(/\n$/, "").split("\n"), [code]);
  const [list, setList] = useState<Annotation[]>(initial);
  const [hovered, setHovered] = useState<string | null>(null);
  const [selection, setSelection] = useState<{ start: number; end: number } | null>(null);
  const anchor = useRef<number | null>(null);
  const [markKind, setMarkKind] = useState<LineKind>("highlight");
  const [markComment, setMarkComment] = useState("");
  const [pen, setPen] = useState(false);
  const [penColor, setPenColor] = useState(PENS[0].color);
  const [pending, setPending] = useState<Stroke[]>([]);
  const [drawComment, setDrawComment] = useState("");
  const drawingNow = useRef(false);
  const [noteOpen, setNoteOpen] = useState(false);
  const [noteText, setNoteText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const innerRef = useRef<HTMLDivElement>(null);
  const measureRef = useRef<HTMLSpanElement>(null);
  const [charW, setCharW] = useState(7.8);

  useEffect(() => {
    const measure = () => {
      if (measureRef.current) setCharW(measureRef.current.getBoundingClientRect().width / 40);
    };
    measure();
    window.addEventListener("resize", measure);
    return () => window.removeEventListener("resize", measure);
  }, []);

  const role = mode === "student" ? null : mode;
  const editable = canAnnotate && role !== null;

  const lineState = useMemo(() => {
    const state = lines.map(() => ({ kinds: new Set<LineKind>(), hovered: false }));
    for (const a of list) {
      if (!a.line_start || !a.line_end || !(a.kind === "highlight" || a.kind === "strike" || a.kind === "good")) continue;
      for (let l = a.line_start; l <= a.line_end && l <= lines.length; l++) {
        state[l - 1].kinds.add(a.kind);
        if (a.id === hovered) state[l - 1].hovered = true;
      }
    }
    return state;
  }, [list, lines, hovered]);

  const sorted = useMemo(
    () => [...list].sort((a, b) => (a.line_start ?? 0) - (b.line_start ?? 0) || a.created_at.localeCompare(b.created_at)),
    [list]
  );
  const myDrafts = list.filter((a) => a.author_id === user?.id && !a.sent_at).length;

  const toPx = (p: [number, number]) => `${CODE_X + p[0] * charW},${p[1] * LINE_H}`;
  const toNorm = (e: React.PointerEvent): [number, number] => {
    const r = innerRef.current!.getBoundingClientRect();
    const col = Math.min(400, Math.max(-2, (e.clientX - r.left - CODE_X) / charW));
    const line = Math.min(lines.length + 1, Math.max(-1, (e.clientY - r.top) / LINE_H));
    return [round2(col), round2(line)];
  };

  const pickLine = (n: number, extend: boolean) => {
    if (!editable) return;
    setPen(false);
    setNoteOpen(false);
    if (extend && anchor.current !== null) {
      setSelection({ start: Math.min(anchor.current, n), end: Math.max(anchor.current, n) });
    } else {
      anchor.current = n;
      setSelection({ start: n, end: n });
    }
  };

  const run = async (action: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await action();
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo guardar");
    } finally {
      setBusy(false);
    }
  };

  const save = (input: AnnotationInput, after: () => void) =>
    run(async () => {
      const a = await createAnnotation(role!, submissionId, input);
      setList((l) => [...l, a]);
      after();
    });

  const remove = (a: Annotation) => {
    if (!window.confirm(`¿Eliminar este ${KIND_LABEL[a.kind].toLowerCase()}?`)) return;
    run(async () => {
      await deleteAnnotation(role!, a.id);
      setList((l) => l.filter((x) => x.id !== a.id));
    });
  };

  const send = () =>
    run(async () => {
      const { sent_at } = await sendMonitorAnnotations(submissionId);
      setList((l) => l.map((a) => (a.author_id === user?.id && !a.sent_at ? { ...a, sent_at } : a)));
    });

  const canDelete = (a: Annotation) => mode === "teacher" || (mode === "monitor" && a.author_id === user?.id);

  const onPointerDown = (e: React.PointerEvent<SVGSVGElement>) => {
    if (!pen || pending.length >= MAX_STROKES) return;
    e.currentTarget.setPointerCapture(e.pointerId);
    drawingNow.current = true;
    const p = toNorm(e);
    setPending((s) => [...s, { color: penColor, width: 3, points: [p] }]);
  };

  const onPointerMove = (e: React.PointerEvent<SVGSVGElement>) => {
    if (!drawingNow.current) return;
    const p = toNorm(e);
    setPending((s) => {
      const last = s[s.length - 1];
      if (!last || last.points.length >= MAX_POINTS) return s;
      const prev = last.points[last.points.length - 1];
      if (Math.hypot(p[0] - prev[0], (p[1] - prev[1]) * 2.5) < 0.3) return s;
      return [...s.slice(0, -1), { ...last, points: [...last.points, p] }];
    });
  };

  const stopDrawing = () => {
    drawingNow.current = false;
  };

  const drawings = list.filter((a) => a.kind === "drawing" && a.data?.strokes);

  if (mode === "student" && list.length === 0) {
    return <CodeLines lines={lines} />;
  }

  return (
    <div className="space-y-3">
      {editable && (
        <div className="flex flex-wrap items-center gap-2">
          <p className="min-w-0 flex-1 text-xs text-ink-faint">
            Haz clic en el número de una línea para marcarla (Shift+clic para varias), dibuja con el lápiz o deja una nota general.
          </p>
          <button
            type="button"
            onClick={() => {
              setPen((p) => !p);
              setSelection(null);
              setNoteOpen(false);
            }}
            aria-pressed={pen}
            className={`rounded-full px-3 py-1.5 text-xs font-bold transition ${
              pen ? "bg-lavender-600 text-white" : "bg-lavender-100 text-lavender-700 hover:bg-lavender-200"
            }`}
          >
            ✎ Lápiz
          </button>
          <button
            type="button"
            onClick={() => {
              setNoteOpen((o) => !o);
              setPen(false);
              setSelection(null);
            }}
            className="rounded-full bg-lavender-100 px-3 py-1.5 text-xs font-bold text-lavender-700 transition hover:bg-lavender-200"
          >
            + Nota general
          </button>
        </div>
      )}

      {pen && (
        <div className="flex flex-wrap items-center gap-2 rounded-2xl bg-lavender-50 px-3 py-2">
          <div role="radiogroup" aria-label="Color del lápiz" className="flex gap-1">
            {PENS.map((p) => (
              <button
                key={p.color}
                type="button"
                role="radio"
                aria-checked={penColor === p.color}
                aria-label={p.label}
                onClick={() => setPenColor(p.color)}
                className={`h-6 w-6 rounded-full ring-2 ring-offset-2 ${penColor === p.color ? "ring-ink" : "ring-transparent"}`}
                style={{ backgroundColor: p.color }}
              />
            ))}
          </div>
          <input
            value={drawComment}
            onChange={(e) => setDrawComment(e.target.value)}
            maxLength={2000}
            placeholder="Comentario del dibujo (opcional)"
            className="min-w-0 flex-1 rounded-full border border-lavender-100 bg-white px-3 py-1.5 text-sm placeholder-ink-faint focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
          <SmallButton disabled={pending.length === 0} onClick={() => setPending((s) => s.slice(0, -1))}>
            Deshacer
          </SmallButton>
          <SmallButton
            disabled={pending.length === 0}
            onClick={() => {
              setPending([]);
              setPen(false);
            }}
            tone="danger"
          >
            Descartar
          </SmallButton>
          <button
            type="button"
            disabled={busy || pending.length === 0}
            onClick={() =>
              save({ kind: "drawing", comment: drawComment, data: { strokes: pending } }, () => {
                setPending([]);
                setDrawComment("");
                setPen(false);
              })
            }
            className="rounded-full bg-lavender-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
          >
            Guardar dibujo
          </button>
        </div>
      )}

      <div className="overflow-x-auto rounded-2xl border border-lavender-100 bg-white py-3 font-mono text-[13px]">
        <div ref={innerRef} className="relative min-w-max">
          <span ref={measureRef} aria-hidden className="invisible absolute whitespace-pre">
            {"0".repeat(40)}
          </span>
          {lines.map((line, i) => {
            const n = i + 1;
            const st = lineState[i];
            const kind = st.kinds.has("strike") ? "strike" : st.kinds.has("highlight") ? "highlight" : st.kinds.has("good") ? "good" : null;
            const style = kind ? LINE_KINDS.find((k) => k.kind === kind)! : null;
            const selected = selection && n >= selection.start && n <= selection.end;
            return (
              <div
                key={i}
                className={`flex items-center ${selected ? "bg-lavender-100" : style ? `${style.bg} ${style.edge}` : ""} ${
                  st.hovered ? "ring-1 ring-inset ring-lavender-400" : ""
                }`}
                style={{ height: LINE_H }}
              >
                <button
                  type="button"
                  tabIndex={editable ? 0 : -1}
                  disabled={!editable}
                  onClick={(e) => pickLine(n, e.shiftKey)}
                  aria-label={editable ? `Marcar la línea ${n}` : undefined}
                  className={`w-12 shrink-0 select-none pr-3 text-right ${
                    selected ? "font-bold text-lavender-700" : "text-ink-faint"
                  } ${editable ? "cursor-pointer hover:text-lavender-700" : "cursor-default"}`}
                >
                  {n}
                </button>
                <span className={`whitespace-pre pl-2 pr-4 ${st.kinds.has("strike") ? "line-through decoration-peach-600 decoration-2" : ""}`}>
                  {line || " "}
                </span>
              </div>
            );
          })}
          <svg
            className={`absolute inset-0 h-full w-full ${pen ? "cursor-crosshair touch-none" : "pointer-events-none"}`}
            onPointerDown={onPointerDown}
            onPointerMove={onPointerMove}
            onPointerUp={stopDrawing}
            onPointerCancel={stopDrawing}
          >
            {drawings.map((a) =>
              a.data!.strokes.map((s, i) => (
                <polyline
                  key={`${a.id}-${i}`}
                  points={s.points.map(toPx).join(" ")}
                  fill="none"
                  stroke={s.color}
                  strokeWidth={a.id === hovered ? s.width + 2 : s.width}
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  opacity={0.85}
                />
              ))
            )}
            {pending.map((s, i) => (
              <polyline
                key={`p-${i}`}
                points={s.points.map(toPx).join(" ")}
                fill="none"
                stroke={s.color}
                strokeWidth={s.width}
                strokeLinecap="round"
                strokeLinejoin="round"
                opacity={0.85}
              />
            ))}
          </svg>
        </div>
      </div>

      {selection && (
        <div className="space-y-2 rounded-2xl border border-lavender-200 p-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-sm font-bold">{linesLabel({ line_start: selection.start, line_end: selection.end })}</span>
            <div role="radiogroup" aria-label="Tipo de marca" className="flex gap-1 rounded-full bg-lavender-50 p-0.5">
              {LINE_KINDS.map((k) => (
                <button
                  key={k.kind}
                  type="button"
                  role="radio"
                  aria-checked={markKind === k.kind}
                  onClick={() => setMarkKind(k.kind)}
                  className={`rounded-full px-3 py-1 text-xs font-bold transition ${markKind === k.kind ? k.chip : "text-ink-soft hover:text-ink"}`}
                >
                  {k.label}
                </button>
              ))}
            </div>
          </div>
          <textarea
            value={markComment}
            onChange={(e) => setMarkComment(e.target.value)}
            maxLength={2000}
            placeholder="Comentario sobre estas líneas (opcional)"
            className={`${fieldClass} h-20`}
          />
          <div className="flex gap-2">
            <button
              type="button"
              disabled={busy}
              onClick={() =>
                save({ kind: markKind, line_start: selection.start, line_end: selection.end, comment: markComment }, () => {
                  setSelection(null);
                  setMarkComment("");
                })
              }
              className="rounded-full bg-lavender-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
            >
              Guardar marca
            </button>
            <SmallButton onClick={() => setSelection(null)}>Cancelar</SmallButton>
          </div>
        </div>
      )}

      {noteOpen && (
        <div className="space-y-2 rounded-2xl border border-lavender-200 p-3">
          <textarea
            value={noteText}
            onChange={(e) => setNoteText(e.target.value)}
            maxLength={2000}
            placeholder="Nota general sobre el código"
            className={`${fieldClass} h-24`}
          />
          <div className="flex gap-2">
            <button
              type="button"
              disabled={busy || !noteText.trim()}
              onClick={() =>
                save({ kind: "note", comment: noteText }, () => {
                  setNoteText("");
                  setNoteOpen(false);
                })
              }
              className="rounded-full bg-lavender-600 px-4 py-1.5 text-xs font-bold text-white transition hover:bg-lavender-700 disabled:opacity-40"
            >
              Guardar nota
            </button>
            <SmallButton onClick={() => setNoteOpen(false)}>Cancelar</SmallButton>
          </div>
        </div>
      )}

      {error && <p className="rounded-2xl bg-peach-50 px-3 py-2 text-sm font-semibold text-peach-700">{error}</p>}

      {(list.length > 0 || mode !== "student") && (
        <div className="space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <h4 className="flex-1 text-xs font-bold uppercase tracking-wider text-ink-soft">
              {mode === "student" ? "Sugerencias de tu monitor sobre tu código" : "Anotaciones sobre el código"}
            </h4>
            {busy && <Spinner className="h-3.5 w-3.5 border-lavender-500" />}
            {mode === "monitor" && myDrafts > 0 && (
              <button
                type="button"
                disabled={busy}
                onClick={send}
                className="rounded-full bg-mint-600 px-4 py-1.5 text-xs font-bold text-white shadow-soft transition hover:bg-mint-700 disabled:opacity-40"
              >
                Enviar al estudiante ({myDrafts})
              </button>
            )}
          </div>
          {mode === "teacher" && (
            <p className="text-xs text-ink-faint">
              Tus anotaciones son privadas: ni el estudiante ni los monitores las ven. Las de un monitor le llegan al estudiante
              solo cuando el monitor las envía.
            </p>
          )}
          {mode === "monitor" && (
            <p className="text-xs text-ink-faint">
              Tus anotaciones quedan en borrador hasta que pulses «Enviar al estudiante». El profesor puede verlas.
            </p>
          )}
          {sorted.length === 0 ? (
            <p className="text-sm text-ink-soft">Aún no hay anotaciones.</p>
          ) : (
            <ul className="space-y-1.5">
              {sorted.map((a) => (
                <li
                  key={a.id}
                  onMouseEnter={() => setHovered(a.id)}
                  onMouseLeave={() => setHovered(null)}
                  className="rounded-2xl bg-lavender-50/60 px-3 py-2"
                >
                  <div className="flex flex-wrap items-center gap-2 text-xs">
                    <span className={`rounded-full px-2.5 py-0.5 font-bold ${KIND_CHIP[a.kind]}`}>
                      {a.kind === "drawing" && (
                        <span
                          aria-hidden
                          className="mr-1 inline-block h-2 w-2 rounded-full align-middle"
                          style={{ backgroundColor: a.data?.strokes[0]?.color }}
                        />
                      )}
                      {KIND_LABEL[a.kind]}
                    </span>
                    {linesLabel(a) && <span className="font-semibold text-ink">{linesLabel(a)}</span>}
                    {mode !== "student" && (
                      <span className="text-ink-soft">
                        {a.author_name} · {a.author_role === "teacher" ? "Profesor" : "Monitor"}
                      </span>
                    )}
                    <span className="text-ink-faint">{formatDateTime(a.sent_at ?? a.created_at)}</span>
                    {mode !== "student" && a.author_role === "monitor" && (
                      <span
                        className={`rounded-full px-2 py-0.5 font-bold ${a.sent_at ? "bg-mint-100 text-mint-700" : "bg-amber-100 text-amber-800"}`}
                      >
                        {a.sent_at ? "Enviada" : "Borrador"}
                      </span>
                    )}
                    <span className="flex-1" />
                    {canDelete(a) && (
                      <SmallButton tone="danger" disabled={busy} onClick={() => remove(a)}>
                        Eliminar
                      </SmallButton>
                    )}
                  </div>
                  {a.comment && <p className="mt-1 whitespace-pre-wrap text-sm leading-relaxed">{a.comment}</p>}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}

function CodeLines({ lines }: { lines: string[] }) {
  return (
    <pre className="overflow-x-auto rounded-2xl border border-lavender-100 bg-white py-3 font-mono text-[13px] leading-relaxed">
      {lines.map((line, i) => (
        <div key={i} className="flex">
          <span className="w-10 shrink-0 select-none pr-3 text-right text-ink-faint">{i + 1}</span>
          <span className="whitespace-pre pr-4">{line || " "}</span>
        </div>
      ))}
    </pre>
  );
}
