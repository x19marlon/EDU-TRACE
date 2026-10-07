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

/** "Entrega: 20 oct 2026, 18:59" (o "Entrega vencida: ..."). */
export function dueLabel(iso?: string): string | null {
  if (!iso) return null;
  const late = new Date(iso).getTime() < Date.now();
  return `${late ? "Entrega vencida" : "Entrega"}: ${formatDateTime(iso)}`;
}
