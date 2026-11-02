"use client";

import { useState } from "react";

export interface LevelSegment {
  key: string;
  label: string;
  count: number;
  color: string;
}

export default function LevelBar({ segments, label, unit }: { segments: LevelSegment[]; label: string; unit: [string, string] }) {
  const [hover, setHover] = useState<string | null>(null);
  const total = segments.reduce((n, s) => n + s.count, 0);
  const named = (n: number) => `${n} ${n === 1 ? unit[0] : unit[1]}`;
  const percent = (n: number) => (total ? Math.round((n / total) * 100) : 0);
  const shown = segments.filter((s) => s.count > 0);
  const active = shown.find((s) => s.key === hover);
  const before = active ? shown.slice(0, shown.indexOf(active)).reduce((n, s) => n + s.count, 0) : 0;
  const center = active ? ((before + active.count / 2) / total) * 100 : 0;

  return (
    <div className="space-y-1.5">
      <div className="relative">
        <div
          role="img"
          aria-label={`${label}: ${segments.map((s) => `${s.label}, ${named(s.count)}`).join("; ")}`}
          className="flex h-6 items-center gap-0.5"
        >
          {total === 0 && <div className="h-3 w-full rounded bg-lavender-100" />}
          {shown.map((s) => (
            <div
              key={s.key}
              onMouseEnter={() => setHover(s.key)}
              onMouseLeave={() => setHover(null)}
              className="flex h-full min-w-[6px] items-center"
              style={{ flexBasis: 0, flexGrow: s.count }}
            >
              <div className={`h-3 w-full ${s.color} ${s === shown[0] ? "rounded-l" : ""} ${s === shown[shown.length - 1] ? "rounded-r" : ""}`} />
            </div>
          ))}
        </div>
        {active && (
          <div
            className="pointer-events-none absolute bottom-full z-10 mb-1 w-max max-w-[14rem] -translate-x-1/2 rounded-xl bg-white px-3 py-1.5 text-xs shadow-lift ring-1 ring-lavender-100"
            style={{ left: `${Math.min(85, Math.max(15, center))}%` }}
          >
            <span className="font-bold text-ink">{active.label}:</span> {named(active.count)} ({percent(active.count)} %)
          </div>
        )}
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-ink-soft">
        {segments.map((s) => (
          <li key={s.key} className="flex items-center gap-1.5">
            <span aria-hidden className={`h-2.5 w-2.5 rounded-sm ${s.color}`} />
            {s.label}: <span className="font-bold tabular-nums text-ink">{s.count}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
