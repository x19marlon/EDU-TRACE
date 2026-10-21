"use client";

import { useEffect, useState } from "react";
import { listMySubmissions, StudentSubmission } from "@/lib/api";
import { Card } from "@/components/ui";
import { MonitorNotes } from "@/components/MonitorFeedback";
import { formatDateTime, formatFeedback } from "@/lib/format";


export default function MySubmissions({ refreshKey }: { refreshKey: number }) {
  const [items, setItems] = useState<StudentSubmission[] | null>(null);
  const [openId, setOpenId] = useState<string | null>(null);

  useEffect(() => {
    listMySubmissions()
      .then(setItems)
      .catch(() => setItems([]));
  }, [refreshKey]);

  if (!items || items.length === 0) return null;

  return (
    <Card title="Mis entregas" className="rise-in">
      <ul className="space-y-2">
        {items.map((s) => {
          const open = openId === s.id;
          return (
            <li key={s.id} className="rounded-2xl bg-lavender-50/60">
              <button
                onClick={() => setOpenId(open ? null : s.id)}
                aria-expanded={open}
                className="flex w-full flex-wrap items-center gap-x-3 gap-y-1 px-4 py-3 text-left"
              >
                <span className="font-bold">{s.course_name}</span>
                <span className="text-xs text-ink-soft">
                  {s.group_name} · {formatDateTime(s.created_at)}
                </span>
                <span className="flex-1" />
                {(s.monitor_feedback?.length ?? 0) > 0 && (
                  <span className="rounded-full bg-mint-50 px-3 py-1 text-xs font-bold text-mint-700">
                    {s.monitor_feedback!.length === 1 ? "1 comentario del monitor" : `${s.monitor_feedback!.length} comentarios del monitor`}
                  </span>
                )}
                {s.review ? (
                  <span className="rounded-full bg-mint-100 px-3 py-1 text-xs font-bold text-mint-700">
                    Revisada{s.review.grade !== undefined ? ` · ${s.review.grade.toFixed(1)} / 5.0` : ""}
                  </span>
                ) : (
                  <span className="rounded-full bg-white px-3 py-1 text-xs font-bold text-ink-soft">En revisión</span>
                )}
              </button>
              {open && s.review && (
                <div className="px-4 pb-4">
                  <div
                    className="rounded-2xl bg-white p-4 text-sm leading-relaxed"
                    dangerouslySetInnerHTML={{ __html: formatFeedback(s.review.feedback) }}
                  />
                  <p className="mt-2 text-xs text-ink-faint">
                    Revisada por tu profesor el {formatDateTime(s.review.returned_at)}.
                    {s.review.ai_assisted && " Tu profesor usó apoyo de IA; la calificación la definió tu profesor."}
                  </p>
                </div>
              )}
              {open && !s.review && (
                <p className="px-4 pb-4 text-sm text-ink-soft">Tu profesor aún no ha revisado esta entrega.</p>
              )}
              {open && (s.monitor_feedback?.length ?? 0) > 0 && (
                <div className="space-y-2 px-4 pb-4">
                  <p className="text-xs font-bold uppercase tracking-wider text-ink-soft">Retroalimentación de tu monitor</p>
                  <MonitorNotes notes={s.monitor_feedback!} />
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
