import { FeedbackMeta } from "@/lib/api";

interface FeedbackPanelProps {
  feedbackText: string;
  meta: FeedbackMeta | null;
  isLoading: boolean;
  error: string | null;
  onRequestFeedback: () => void;
  canRequestFeedback: boolean;
}

export default function FeedbackPanel({
  feedbackText,
  meta,
  isLoading,
  error,
  onRequestFeedback,
  canRequestFeedback,
}: FeedbackPanelProps) {
  const hasContent = feedbackText.length > 0;

  return (
    <div className="h-full flex flex-col bg-gray-900 rounded-lg border border-gray-700 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700 bg-gray-800/50">
        <div className="flex items-center gap-2">
          <span className="text-lg">🤖</span>
          <h3 className="text-sm font-semibold text-gray-300">
            Retroalimentación IA
          </h3>
          {isLoading && (
            <span className="flex items-center gap-1 text-xs text-purple-400">
              <span className="animate-pulse">●</span> generando...
            </span>
          )}
        </div>
        <button
          onClick={onRequestFeedback}
          disabled={isLoading || !canRequestFeedback}
          className="px-3 py-1.5 text-xs font-medium rounded-md transition-colors
            bg-purple-600 hover:bg-purple-700 text-white
            disabled:bg-gray-700 disabled:text-gray-500 disabled:cursor-not-allowed"
        >
          {isLoading ? "Analizando..." : "Pedir retroalimentación"}
        </button>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        {error && !isLoading && (
          <div className="bg-red-900/30 border border-red-800 rounded-md p-3 mb-3">
            <p className="text-red-400 text-sm">{error}</p>
          </div>
        )}

        {!hasContent && !isLoading && !error && (
          <div className="flex flex-col items-center justify-center h-full text-center gap-3">
            <span className="text-4xl">📝</span>
            <div>
              <p className="text-gray-400 text-sm">
                Escribe y compila tu código para recibir retroalimentación de la IA.
              </p>
              <p className="text-gray-600 text-xs mt-2">
                La IA te dará sugerencias constructivas basadas en el syllabus del curso.
                <br />
                No es una calificación.
              </p>
            </div>
          </div>
        )}

        {(hasContent || isLoading) && (
          <div>
            <div className="prose prose-invert prose-sm max-w-none">
              <div
                className="text-gray-300 text-sm leading-relaxed whitespace-pre-wrap"
                dangerouslySetInnerHTML={{
                  __html: formatFeedback(feedbackText) + (isLoading ? '<span class="animate-pulse text-purple-400">▊</span>' : ''),
                }}
              />
            </div>

            {/* Footer info — only show when done */}
            {meta && !isLoading && (
              <div className="flex items-center gap-3 mt-4 pt-3 border-t border-gray-800 text-xs text-gray-500">
                <span>🧠 Modelo: {meta.model}</span>
                <span>⏱ {(meta.time_ms / 1000).toFixed(1)}s</span>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

/** Basic markdown-like formatting for the AI response. */
function formatFeedback(text: string): string {
  return text
    .replace(/\*\*(.+?)\*\*/g, "<strong>$1</strong>")
    .replace(/^### (.+)$/gm, '<h4 class="text-white font-semibold mt-4 mb-1">$1</h4>')
    .replace(/^## (.+)$/gm, '<h3 class="text-white font-semibold text-base mt-4 mb-2">$1</h3>')
    .replace(/`([^`]+)`/g, '<code class="bg-gray-800 px-1 rounded text-purple-300">$1</code>')
    .replace(
      /```(\w*)\n([\s\S]*?)```/g,
      '<pre class="bg-gray-950 border border-gray-800 rounded p-3 my-2 overflow-x-auto"><code>$2</code></pre>'
    )
    .replace(/\n/g, "<br />");
}
