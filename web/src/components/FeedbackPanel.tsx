import { FeedbackMeta, FeedbackMode, RatingKind } from "@/lib/api";
import ComprehensionCheck from "@/components/ComprehensionCheck";
import { Spinner } from "@/components/ui";
import { formatFeedback, informalHeadings } from "@/lib/format";

interface FeedbackPanelProps {
  mode: FeedbackMode;
  feedbackText: string;
  meta: FeedbackMeta | null;
  isLoading: boolean;
  error: string | null;
  onRequestFeedback: () => void;
  canRequestFeedback: boolean;
  disabledReason?: string;
  rating?: { kind: RatingKind; assignmentId?: string };
}

const MODE_INFO: Record<FeedbackMode, { intro: React.ReactNode; note: string; button: string; box: string; rule: string; accent: string }> = {
  formal: {
    intro: (
      <>
        Retroalimentación formativa basada en una teoría educativa ampliamente estudiada y respaldada por la investigación, que se guía por
        tres preguntas: <strong className="text-ink">¿Hacia dónde voy?</strong>, <strong className="text-ink">¿Cómo voy?</strong> y{" "}
        <strong className="text-ink">¿Qué sigue?</strong>
      </>
    ),
    note: "La idea es acompañarte en el proceso: darte pistas y orientarte para que encuentres la solución por tu cuenta, en lugar de dártela directamente. No es una calificación, sino una ayuda para aprender. Tu docente siempre tiene la última palabra.",
    button: "Pedir retroalimentación formal",
    box: "bg-mint-50",
    rule: "border-mint-200",
    accent: "text-mint-700",
  },
  informal: {
    intro: (
      <>
        Orientación práctica, como la que recibirías de un monitor, organizada en tres preguntas:{" "}
        <strong className="text-ink">¿Qué pide el ejercicio?</strong>, <strong className="text-ink">¿Qué está funcionando y qué puedo mejorar?</strong>{" "}
        (nombres de variables, ciclos, funciones, organización del código, entre otros) y <strong className="text-ink">¿Qué puedo hacer ahora?</strong>
      </>
    ),
    note: "La idea es acompañarte mientras aprendes, darte pistas y ayudarte a encontrar tus propios errores, sin resolverte el ejercicio. No es una calificación, sino una herramienta de apoyo. Tu docente siempre tiene la última palabra.",
    button: "Pedir retroalimentación informal",
    box: "bg-peach-50",
    rule: "border-peach-200",
    accent: "text-peach-700",
  },
};

export default function FeedbackPanel({
  mode,
  feedbackText,
  meta,
  isLoading,
  error,
  onRequestFeedback,
  canRequestFeedback,
  disabledReason,
  rating,
}: FeedbackPanelProps) {
  const hasContent = feedbackText.length > 0;
  const info = MODE_INFO[mode];
  const text = mode === "informal" ? informalHeadings(feedbackText) : feedbackText;

  return (
    <div className="flex h-full min-h-[300px] flex-col">
      <div className="flex-1 overflow-auto p-5">
        {error && !isLoading && (
          <div className="mb-3 rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</div>
        )}

        {!hasContent && !isLoading && !error && (
          <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
            <span className={`flex h-14 w-14 items-center justify-center rounded-2xl ${info.box} ${info.accent}`}>
              <svg className="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.8} aria-hidden>
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 0 1-2.555-.337A5.972 5.972 0 0 1 5.41 20.97a5.969 5.969 0 0 1-.474-.065 4.48 4.48 0 0 0 .978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25Z" />
              </svg>
            </span>
            <p className="max-w-sm text-sm text-ink-soft">{info.intro}</p>
            <p className="max-w-sm text-xs text-ink-faint">{info.note}</p>
            {disabledReason ? (
              <p className="max-w-sm rounded-2xl bg-lavender-50 px-4 py-3 text-sm font-semibold text-lavender-700">{disabledReason}</p>
            ) : (
              <button
                onClick={onRequestFeedback}
                disabled={!canRequestFeedback}
                className="mt-1 rounded-full bg-lavender-600 px-5 py-2 text-sm font-bold text-white shadow-soft
                  transition hover:bg-lavender-700 disabled:opacity-50"
              >
                {info.button}
              </button>
            )}
          </div>
        )}

        {(hasContent || isLoading) && (
          <div className={`rounded-2xl p-4 ${info.box}`}>
            {isLoading && !hasContent && (
              <p className={`flex items-center gap-2 text-sm font-semibold ${info.accent}`}>
                <Spinner className="h-4 w-4 border-current" /> Analizando tu código...
              </p>
            )}
            <div
              className="text-sm leading-relaxed text-ink"
              dangerouslySetInnerHTML={{
                __html:
                  formatFeedback(text) +
                  (isLoading && hasContent ? '<span class="animate-pulse text-lavender-500">▊</span>' : ""),
              }}
            />
            {!isLoading && (
              <p className={`mt-4 border-t pt-3 text-xs text-ink-faint ${info.rule}`}>
                Generado por IA{meta?.time_ms ? ` en ${(meta.time_ms / 1000).toFixed(1)} s` : ""}. Es una
                orientación y puede equivocarse, tu docente tiene la última palabra.
              </p>
            )}
            {!isLoading && !error && rating && (
              <div className="mt-3">
                <ComprehensionCheck key={`${mode}-${meta?.time_ms ?? 0}`} kind={rating.kind} assignmentId={rating.assignmentId} />
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
