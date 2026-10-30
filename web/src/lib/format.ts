/** Fecha relativa en español ("Hace 5 min"). */
export function relativeTime(iso?: string | null): string {
  if (!iso || iso.startsWith("0001")) return "Sin actividad";
  const date = new Date(iso);
  const mins = Math.round((Date.now() - date.getTime()) / 60000);
  if (mins < 1) return "Ahora mismo";
  if (mins < 60) return `Hace ${mins} min`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `Hace ${hours} h`;
  const days = Math.round(hours / 24);
  if (days < 30) return `Hace ${days} ${days === 1 ? "día" : "días"}`;
  return date.toLocaleDateString("es", { dateStyle: "medium" });
}

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString("es", { dateStyle: "medium", timeStyle: "short" });
}

function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/**
 * Formato tipo Markdown básico para la respuesta de la IA. El texto se escapa
 * primero: la salida del modelo puede contener HTML (p. ej. inducido por el código enviado).
 */
export function formatFeedback(text: string): string {
  return escapeHtml(text.trim().replace(/\n{3,}/g, "\n\n"))
    .replace(
      /```(\w*)\n([\s\S]*?)```/g,
      '<pre class="bg-white border border-mint-200 rounded-xl p-3 my-2 overflow-x-auto font-mono text-xs"><code>$2</code></pre>'
    )
    .replace(/\*\*(.+?)\*\*/g, '<strong class="text-ink">$1</strong>')
    .replace(/^### (.+)$/gm, '<h4 class="font-bold text-lavender-700 mt-4 mb-1">$1</h4>')
    .replace(/^## (.+)$/gm, '<h3 class="font-extrabold text-lavender-700 text-base mt-4 mb-2">$1</h3>')
    .replace(/`([^`\n]+)`/g, '<code class="bg-lavender-100 text-lavender-700 px-1.5 py-0.5 rounded font-mono text-xs">$1</code>')
    .replace(/^(\s*)[-*+]\s+/gm, '$1<span class="text-mint-600 font-bold">•</span> ')
    .replace(/\n/g, "<br />")
    // Los títulos ya tienen margen: sin saltos extra antes ni después.
    .replace(/(<\/h[34]>)(<br \/>)+/g, "$1")
    .replace(/(<br \/>)+(<h[34])/g, "$2");
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  const days = `${d} ${d === 1 ? "día" : "días"}`;
  if (d > 0) return h > 0 ? `${days} ${h} h` : days;
  if (h > 0) return m > 0 ? `${h} h ${m} min` : `${h} h`;
  if (m > 0) return `${m} min`;
  return "menos de 1 min";
}

export function formatShortDuration(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return h > 0 ? `${d} d ${h} h` : `${d} d`;
  if (h > 0) return m > 0 ? `${h} h ${m} min` : `${h} h`;
  return m > 0 ? `${m} min` : "< 1 min";
}

export const startedDays = (lateSeconds: number) => Math.max(1, Math.ceil(lateSeconds / 86400));

export function formatDeadline(iso: string): string {
  return new Date(iso).toLocaleString("es", {
    weekday: "long",
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export type DeadlineTone = "ok" | "soon" | "urgent" | "late" | "closed";

export interface DeadlineState {
  tone: DeadlineTone;
  title: string;
  detail: string;
  canSubmit: boolean;
}

export function deadlineState(
  a: { due_at?: string; allow_late: boolean; late_until?: string; late_penalty?: number },
  now: number = Date.now()
): DeadlineState | null {
  if (!a.due_at) return null;
  const due = new Date(a.due_at).getTime();
  const penalty = a.late_penalty ? ` Penalización sugerida: −${a.late_penalty.toFixed(1)} por cada día de retraso iniciado.` : "";
  if (now < due) {
    const left = (due - now) / 1000;
    const tone: DeadlineTone = left <= 24 * 3600 ? "urgent" : left <= 72 * 3600 ? "soon" : "ok";
    const after = !a.allow_late
      ? " Después no se reciben entregas."
      : a.late_until
        ? ` Después se reciben como tardías hasta el ${formatDeadline(a.late_until)}.${penalty}`
        : ` Después se reciben como tardías.${penalty}`;
    return { tone, title: `Vence en ${formatDuration(left)}`, detail: `Fecha límite: ${formatDeadline(a.due_at)}.${after}`, canSubmit: true };
  }
  const over = (now - due) / 1000;
  if (!a.allow_late) {
    return {
      tone: "closed",
      title: "Plazo cerrado",
      detail: `La fecha límite fue el ${formatDeadline(a.due_at)} (hace ${formatDuration(over)}) y este taller no recibe entregas tardías.`,
      canSubmit: false,
    };
  }
  if (a.late_until && now >= new Date(a.late_until).getTime()) {
    return {
      tone: "closed",
      title: "Plazo cerrado",
      detail: `Las entregas tardías se recibieron hasta el ${formatDeadline(a.late_until)}.`,
      canSubmit: false,
    };
  }
  const until = a.late_until
    ? ` Se reciben tardías hasta el ${formatDeadline(a.late_until)} (quedan ${formatDuration((new Date(a.late_until).getTime() - now) / 1000)}).`
    : "";
  return {
    tone: "late",
    title: `Venció hace ${formatDuration(over)}`,
    detail: `La fecha límite fue el ${formatDeadline(a.due_at)}. Si entregas ahora quedará como tardía.${until}${penalty}`,
    canSubmit: true,
  };
}

export const LEVEL_STYLE: Record<"bajo" | "medio" | "alto", string> = {
  bajo: "bg-peach-100 text-peach-700",
  medio: "bg-amber-100 text-amber-800",
  alto: "bg-mint-100 text-mint-700",
};

export const PERFORMANCE_BANDS = [
  { min: 4.25, label: "Ejemplar de alto nivel", range: "[4.25–5]", tone: "bg-mint-100 text-mint-700" },
  { min: 3.75, label: "Ejemplar", range: "[3.75–4.25)", tone: "bg-mint-100 text-mint-700" },
  { min: 3, label: "Competente", range: "[3–3.75)", tone: "bg-amber-100 text-amber-800" },
  { min: 2, label: "Bajo el estándar", range: "[2–3)", tone: "bg-peach-100 text-peach-700" },
  { min: 0, label: "Insatisfactorio", range: "[0–2)", tone: "bg-peach-100 text-peach-700" },
];

export function performanceBand(grade: number) {
  return PERFORMANCE_BANDS.find((b) => grade >= b.min) ?? PERFORMANCE_BANDS[PERFORMANCE_BANDS.length - 1];
}
