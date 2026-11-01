"use client";

import { useState } from "react";
import { formatDateTime } from "@/lib/format";

export interface GradePoint {
  title: string;
  at: string;
  grade: number;
}

const W = 560;
const H = 220;
const PAD = { top: 16, right: 16, bottom: 28, left: 32 };
const MAX = 5;
const SERIES = "#7154CC";
const GRID = "#EDE7FC";
const MUTED = "#6B6680";

export default function GradeChart({ points, label }: { points: GradePoint[]; label: string }) {
  const [active, setActive] = useState<number | null>(null);
  const plotW = W - PAD.left - PAD.right;
  const plotH = H - PAD.top - PAD.bottom;
  const x = (i: number) => PAD.left + (points.length === 1 ? plotW / 2 : (i / (points.length - 1)) * plotW);
  const y = (g: number) => PAD.top + plotH - (g / MAX) * plotH;
  const path = points.map((p, i) => `${i ? "L" : "M"}${x(i).toFixed(1)},${y(p.grade).toFixed(1)}`).join(" ");

  const nearest = (clientX: number, rect: DOMRect) => {
    const px = ((clientX - rect.left) / rect.width) * W;
    let best = 0;
    points.forEach((_, i) => {
      if (Math.abs(x(i) - px) < Math.abs(x(best) - px)) best = i;
    });
    return best;
  };

  const shown = active !== null ? points[active] : null;
  return (
    <figure className="relative">
      <svg
        viewBox={`0 0 ${W} ${H}`}
        className="h-auto w-full touch-none"
        role="group"
        aria-label={`${label}: ${points.length} notas, la última ${points.at(-1)?.grade.toFixed(1) ?? "—"} de ${MAX}`}
        onPointerMove={(e) => setActive(nearest(e.clientX, e.currentTarget.getBoundingClientRect()))}
        onPointerLeave={() => setActive(null)}
      >
        {[0, 1, 2, 3, 4, 5].map((g) => (
          <g key={g}>
            <line x1={PAD.left} x2={W - PAD.right} y1={y(g)} y2={y(g)} stroke={GRID} strokeWidth={1} />
            <text x={PAD.left - 8} y={y(g)} dy="0.32em" textAnchor="end" fontSize={11} fill={MUTED}>
              {g}
            </text>
          </g>
        ))}
        <text x={W - PAD.right} y={y(3) - 5} textAnchor="end" fontSize={11} fill={MUTED}>
          3.0 aprueba
        </text>
        {active !== null && <line x1={x(active)} x2={x(active)} y1={PAD.top} y2={PAD.top + plotH} stroke={MUTED} strokeWidth={1} />}
        {points.length > 1 && <path d={path} fill="none" stroke={SERIES} strokeWidth={2} strokeLinejoin="round" strokeLinecap="round" />}
        {points.map((p, i) => (
          <circle
            key={i}
            cx={x(i)}
            cy={y(p.grade)}
            r={active === i ? 6 : 5}
            fill={SERIES}
            stroke="#FFFFFF"
            strokeWidth={2}
            tabIndex={0}
            role="img"
            aria-label={`${p.title}: ${p.grade.toFixed(1)}`}
            onFocus={() => setActive(i)}
            onBlur={() => setActive(null)}
            className="outline-none focus-visible:stroke-lavender-700"
          />
        ))}
        <text x={x(0)} y={H - 8} textAnchor={points.length === 1 ? "middle" : "start"} fontSize={11} fill={MUTED}>
          {points[0] && new Date(points[0].at).toLocaleDateString("es-CO", { day: "numeric", month: "short" })}
        </text>
        {points.length > 1 && (
          <text x={x(points.length - 1)} y={H - 8} textAnchor="end" fontSize={11} fill={MUTED}>
            {new Date(points[points.length - 1].at).toLocaleDateString("es-CO", { day: "numeric", month: "short" })}
          </text>
        )}
      </svg>
      {shown && active !== null && (
        <div
          role="status"
          className="pointer-events-none absolute top-0 rounded-xl bg-white px-3 py-2 text-xs shadow-lift ring-1 ring-lavender-100"
          style={{ left: `${Math.min(70, Math.max(0, (x(active) / W) * 100 - 15))}%` }}
        >
          <span className="block font-bold text-ink">{shown.title}</span>
          <span className="block text-ink-soft">
            Nota <span className="font-bold text-ink">{shown.grade.toFixed(1)}</span> · {formatDateTime(shown.at)}
          </span>
        </div>
      )}
    </figure>
  );
}
