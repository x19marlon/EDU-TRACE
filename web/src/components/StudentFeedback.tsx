"use client";

import { FeedbackOrigin, StudentSubmission } from "@/lib/api";
import { MonitorNotes } from "@/components/MonitorFeedback";
import ComprehensionCheck from "@/components/ComprehensionCheck";
import { StudentQuestions } from "@/components/FeedbackQuestions";
import { formatDateTime, formatFeedback, LEVEL_STYLE, performanceBand } from "@/lib/format";

export function ReviewStatusPill({ item }: { item: StudentSubmission }) {
  return item.review ? (
    <span className="rounded-full bg-mint-100 px-3 py-1 text-xs font-bold text-mint-700">
      Revisada
      {item.review.grade !== undefined ? ` · ${item.review.grade.toFixed(1)} / 5.0 · ${performanceBand(item.review.grade).label}` : ""}
    </span>
  ) : (
    <span className="rounded-full bg-white px-3 py-1 text-xs font-bold text-ink-soft">En revisión</span>
  );
}

export function MonitorCountPill({ item }: { item: StudentSubmission }) {
  const n = item.monitor_feedback?.length ?? 0;
  if (n === 0) return null;
  return (
    <span className="rounded-full bg-mint-50 px-3 py-1 text-xs font-bold text-mint-700">
      {n === 1 ? "Comentario del monitor" : `${n} comentarios del monitor`}
    </span>
  );
}

export function AnnotationsPill({ item }: { item: StudentSubmission }) {
  const n = item.monitor_annotations ?? 0;
  if (n === 0) return null;
  return (
    <span className="rounded-full bg-lavender-100 px-3 py-1 text-xs font-bold text-lavender-700">
      {n === 1 ? "1 marca en tu código" : `${n} marcas en tu código`}
    </span>
  );
}

const ORIGIN_NOTE: Record<FeedbackOrigin, string> = {
  ia: "Texto propuesto por la IA y aprobado sin cambios por",
  ia_editada: "Texto propuesto por la IA, editado y aprobado por",
  docente: "Retroalimentación escrita por",
};

export function SubmissionFeedback({ item }: { item: StudentSubmission }) {
  const review = item.review;
  const origin = review?.origin || (review?.ai_assisted ? "ia_editada" : "docente");
  const teacher = review?.teacher_name || "tu profesor";
  return (
    <div className="space-y-3">
      {review ? (
        <div>
          <div className="mb-2 flex flex-wrap items-center gap-2">
            {origin !== "docente" ? (
              <span className="rounded-full bg-lavender-100 px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
                Generado con IA · aprobado por {teacher}
              </span>
            ) : (
              <span className="rounded-full bg-mint-100 px-2.5 py-0.5 text-[11px] font-bold text-mint-700">Escrito por {teacher}</span>
            )}
          </div>
          <div
            className="rounded-2xl bg-white p-4 text-sm leading-relaxed"
            dangerouslySetInnerHTML={{ __html: formatFeedback(review.feedback) }}
          />
          {review.raes && review.raes.length > 0 && (
            <ul className="mt-2 space-y-1.5 rounded-2xl bg-white p-3">
              <li className="text-xs font-bold uppercase tracking-wider text-ink-soft">Nivel de logro por RAE</li>
              {review.raes.map((r) => (
                <li key={r.code} className="flex items-start gap-2 text-sm">
                  <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-[11px] font-bold capitalize ${LEVEL_STYLE[r.level]}`}>
                    {r.level}
                  </span>
                  <span>
                    <span className="font-bold">{r.code}</span> <span className="text-ink-soft">{r.description}</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="mt-2 text-xs text-ink-faint">
            {ORIGIN_NOTE[origin]} {teacher} el {formatDateTime(review.returned_at)}.
            {review.ai_assisted && " La calificación la definió tu profesor, no la IA."} Si tienes dudas sobre esta retroalimentación,
            pregúntale a {teacher} aquí abajo o a tu monitor.
          </p>
          <div className="mt-2">
            <ComprehensionCheck kind="revision" submissionId={item.id} initial={review.understood} />
          </div>
        </div>
      ) : (
        <p className="text-sm text-ink-soft">Tu profesor aún no ha revisado esta entrega.</p>
      )}
      {(review || item.has_ai_feedback || (item.questions?.length ?? 0) > 0) && (
        <StudentQuestions submissionId={item.id} initial={item.questions ?? []} />
      )}
      {(item.monitor_feedback?.length ?? 0) > 0 && (
        <div className="space-y-2">
          <p className="text-xs font-bold uppercase tracking-wider text-ink-soft">Retroalimentación de tu monitor</p>
          <MonitorNotes notes={item.monitor_feedback!} />
        </div>
      )}
    </div>
  );
}
