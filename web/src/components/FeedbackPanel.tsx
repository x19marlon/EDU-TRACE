import { FeedbackMeta } from "@/lib/api";
import { Spinner } from "@/components/ui";
import { formatFeedback } from "@/lib/format";

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
    <div className="flex h-full min-h-[300px] flex-col">
      <div className="flex-1 overflow-auto p-5">
        {error && !isLoading && (
          <div className="mb-3 rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</div>
        )}

        {!hasContent && !isLoading && !error && (
          <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
            <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-mint-100 text-mint-700">
              <svg className="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.8} aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z" />
              </svg>
            </span>
            <p className="max-w-sm text-sm text-ink-soft">
              La IA te acompaña con tres preguntas: <strong className="text-ink">¿Hacia dónde voy?</strong>,{" "}
              <strong className="text-ink">¿Cómo voy?</strong> y <strong className="text-ink">¿Qué sigue?</strong>
            </p>
            <p className="max-w-sm text-xs text-ink-faint">
              Te da pistas para que llegues tú a la solución, no te la resuelve. No es una calificación:
              tu docente tiene la última palabra.
            </p>
            <button
              onClick={onRequestFeedback}
              disabled={!canRequestFeedback}
              className="mt-1 rounded-full bg-lavender-600 px-5 py-2 text-sm font-bold text-white shadow-soft
                transition hover:bg-lavender-700 disabled:opacity-50"
            >
              Pedir orientación
            </button>
          </div>
        )}

        {(hasContent || isLoading) && (
          <div className="rounded-2xl bg-mint-50 p-4">
            {isLoading && !hasContent && (
              <p className="flex items-center gap-2 text-sm font-semibold text-mint-700">
                <Spinner className="h-4 w-4 border-mint-600" /> Analizando tu código...
              </p>
            )}
            <div
              className="text-sm leading-relaxed text-ink"
              dangerouslySetInnerHTML={{
                __html:
                  formatFeedback(feedbackText) +
                  (isLoading && hasContent ? '<span class="animate-pulse text-lavender-500">▊</span>' : ""),
              }}
            />
            {!isLoading && (
              <p className="mt-4 border-t border-mint-200 pt-3 text-xs text-ink-faint">
                Generado por IA{meta?.model ? ` (${meta.model}, ${(meta.time_ms / 1000).toFixed(1)} s)` : ""}. Es una
                orientación y puede equivocarse: no es una calificación y tu docente tiene la última palabra.
              </p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
