"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { listMySubmissions, StudentSubmission } from "@/lib/api";
import { Card } from "@/components/ui";
import { AnnotationsPill, MonitorCountPill, ReviewStatusPill, SubmissionFeedback } from "@/components/StudentFeedback";
import { formatDateTime } from "@/lib/format";
import { LatePill } from "@/components/Deadline";

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
    <Card
      title="Mis entregas"
      className="rise-in"
      action={
        <Link href="/historial" className="text-xs font-bold text-lavender-600 hover:text-lavender-700">
          Ver historial completo →
        </Link>
      }
    >
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
                <span className="font-bold">{s.assignment_title || "Práctica libre"}</span>
                <span className="text-xs text-ink-soft">
                  {s.course_name} · {s.group_name} · {formatDateTime(s.created_at)}
                </span>
                <span className="flex-1" />
                <LatePill meta={s} />
                <AnnotationsPill item={s} />
                <MonitorCountPill item={s} />
                <ReviewStatusPill item={s} />
              </button>
              {open && (
                <div className="space-y-3 px-4 pb-4">
                  {(s.monitor_annotations ?? 0) > 0 && (
                    <p className="rounded-2xl bg-lavender-100/70 px-4 py-2 text-sm text-lavender-700">
                      Tu monitor marcó tu código.{" "}
                      <Link href="/historial" className="font-bold underline">
                        Míralo en Mi historial →
                      </Link>
                    </p>
                  )}
                  <SubmissionFeedback item={s} />
                </div>
              )}
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
