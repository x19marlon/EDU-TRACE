"use client";

import { useEffect, useState } from "react";
import { getReviewPlan, ReviewEstimate, ReviewPlan } from "@/lib/api";
import { Card, Spinner } from "@/components/ui";
import { formatDeadline } from "@/lib/format";

export function formatMinutes(minutes: number): string {
  const m = Math.round(minutes);
  if (m < 60) return `${Math.max(1, m)} min`;
  const h = Math.floor(m / 60);
  const rest = m % 60;
  return rest ? `${h} h ${rest} min` : `${h} h`;
}

function Chip({ value, label, tone }: { value: number; label: string; tone: string }) {
  if (value === 0) return null;
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${tone}`}>
      {value} {label}
    </span>
  );
}

function Workshop({ e }: { e: ReviewEstimate }) {
  const done = e.students ? Math.round((e.returned / e.students) * 100) : 0;
  if (e.pending === 0) {
    return (
      <li className="flex items-center gap-2 rounded-2xl bg-mint-50 px-4 py-2.5 text-sm">
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-mint-200 text-[11px] font-bold text-mint-700">✓</span>
        <span className="min-w-0 flex-1 truncate font-bold">{e.title}</span>
        <span className="text-xs text-mint-700">
          {e.returned} {e.returned === 1 ? "revisión devuelta" : "revisiones devueltas"}
        </span>
      </li>
    );
  }
  return (
    <li className="space-y-2.5 rounded-2xl bg-lavender-50/70 p-4">
      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <span className="min-w-0 truncate font-bold">{e.title}</span>
        {e.due_at && <span className="text-[11px] text-ink-faint">Fecha límite: {formatDeadline(e.due_at)}</span>}
      </div>
      <div className="flex flex-wrap items-end gap-x-6 gap-y-1">
        <p>
          <span className="block text-[10px] font-bold uppercase tracking-wider text-ink-faint">Te tomará unos</span>
          <span className="text-3xl font-extrabold tabular-nums text-lavender-700">{formatMinutes(e.total_minutes)}</span>
        </p>
        <p className="pb-1">
          <span className="block text-[10px] font-bold uppercase tracking-wider text-ink-faint">Por entrega</span>
          <span className="text-lg font-extrabold tabular-nums">{formatMinutes(e.per_submission)}</span>
        </p>
        <p className="pb-1">
          <span className="block text-[10px] font-bold uppercase tracking-wider text-ink-faint">Por revisar</span>
          <span className="text-lg font-extrabold tabular-nums">
            {e.pending}
            <span className="text-sm font-bold text-ink-faint"> de {e.students}</span>
          </span>
        </p>
      </div>
      <div className="h-2 overflow-hidden rounded-full bg-white" role="progressbar" aria-valuenow={done} aria-valuemin={0} aria-valuemax={100} aria-label={`${done} % revisado`}>
        <div className="h-full rounded-full bg-mint-400 transition-all" style={{ width: `${done}%` }} />
      </div>
      <div className="flex flex-wrap gap-1.5">
        <Chip value={e.not_compiling} label={e.not_compiling === 1 ? "no compila" : "no compilan"} tone="bg-peach-100 text-peach-700" />
        <Chip value={e.failing_tests} label={e.failing_tests === 1 ? "no pasa todos los casos" : "no pasan todos los casos"} tone="bg-amber-100 text-amber-800" />
        <Chip value={e.all_passing} label={e.all_passing === 1 ? "pasa todos los casos" : "pasan todos los casos"} tone="bg-mint-100 text-mint-700" />
        <span className="rounded-full bg-white px-2.5 py-0.5 text-[11px] font-bold text-lavender-700">
          {e.with_ai === e.pending ? "Estimado por la IA en todas" : `Estimado por la IA en ${e.with_ai} de ${e.pending}`}
        </span>
      </div>
    </li>
  );
}

export default function ReviewTimeCard({ groupId, refreshKey }: { groupId: string; refreshKey: number }) {
  const [plan, setPlan] = useState<ReviewPlan | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getReviewPlan(groupId)
      .then(setPlan)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo estimar el tiempo de revisión"));
  }, [groupId, refreshKey]);

  const pending = plan?.assignments.filter((a) => a.pending > 0) ?? [];
  const total = pending.reduce((n, a) => n + a.total_minutes, 0);

  return (
    <Card
      title="Tiempo de revisión"
      className="rise-in"
      action={plan && pending.length > 0 && <span className="text-xs font-bold text-lavender-700">{formatMinutes(total)} en total</span>}
    >
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm text-peach-700">{error}</p>}
      {!plan ? (
        !error && (
          <p className="flex items-center gap-2 text-sm text-ink-soft">
            <Spinner className="h-4 w-4 border-lavender-500" /> Estimando...
          </p>
        )
      ) : plan.assignments.length === 0 ? (
        <p className="text-sm text-ink-soft">Cuando lleguen entregas a los talleres, aquí verás cuánto tiempo te tomará revisarlas.</p>
      ) : (
        <div className="space-y-3">
          <p className="text-xs text-ink-soft">
            Al preparar cada propuesta, la IA estima los minutos que necesitas para leer la entrega, ajustar la retroalimentación y
            devolverla. Tú decides cuánto tiempo le dedicas.
            {plan.measured_minutes !== undefined &&
              ` Tu ritmo real: ${formatMinutes(plan.measured_minutes)} por entrega (${plan.measured} ${plan.measured === 1 ? "revisión medida" : "revisiones medidas"})${
                plan.estimated_minutes !== undefined ? `; la IA había estimado ${formatMinutes(plan.estimated_minutes)}` : ""
              }.`}
          </p>
          <ul className="space-y-2">
            {plan.assignments.map((e) => (
              <Workshop key={e.assignment_id} e={e} />
            ))}
          </ul>
        </div>
      )}
    </Card>
  );
}
