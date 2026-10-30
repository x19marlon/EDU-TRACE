"use client";

import { useEffect, useState } from "react";
import { Assignment, SubmissionMeta } from "@/lib/api";
import { deadlineState, DeadlineTone, formatDeadline, formatShortDuration } from "@/lib/format";

const TONES: Record<DeadlineTone, string> = {
  ok: "bg-mint-100 text-mint-700",
  soon: "bg-amber-100 text-amber-800",
  urgent: "bg-peach-100 text-peach-700",
  late: "bg-peach-100 text-peach-700",
  closed: "bg-ink/10 text-ink",
};

export function useNow(intervalMs = 30000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(id);
  }, [intervalMs]);
  return now;
}

type DeadlineFields = Pick<Assignment, "due_at" | "allow_late" | "late_until" | "late_penalty">;

export function DeadlineBadge({ assignment, withDetail = false }: { assignment: DeadlineFields; withDetail?: boolean }) {
  const now = useNow();
  const state = deadlineState(assignment, now);
  if (!state) return null;
  return (
    <span className={withDetail ? "block space-y-1" : "inline-flex"}>
      <span
        className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-[11px] font-bold ${TONES[state.tone]}`}
        title={withDetail ? undefined : state.detail}
      >
        <svg className="h-3.5 w-3.5" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24" aria-hidden>
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
        </svg>
        {state.title}
      </span>
      {withDetail && <span className="block text-xs leading-relaxed text-ink-soft">{state.detail}</span>}
    </span>
  );
}

export function LatePill({ meta }: { meta: Pick<SubmissionMeta, "kind" | "late_seconds" | "due_at"> }) {
  if (meta.kind === "attempt" || !meta.late_seconds) return null;
  return (
    <span
      className="rounded-full bg-peach-100 px-2.5 py-0.5 text-[11px] font-bold text-peach-700"
      title={meta.due_at ? `Fecha límite: ${formatDeadline(meta.due_at)}` : undefined}
    >
      Tardía · {formatShortDuration(meta.late_seconds)}
    </span>
  );
}
