"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Card, SmallButton } from "@/components/ui";
import { getStudentAlerts, StudentAlert } from "@/lib/api";

export default function StudentAlerts({
  groupId,
  refreshKey,
  onSelect,
}: {
  groupId: string;
  refreshKey: number;
  onSelect: (studentId: string) => void;
}) {
  const [alerts, setAlerts] = useState<StudentAlert[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getStudentAlerts(groupId)
      .then(setAlerts)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudieron calcular las alertas"));
  }, [groupId, refreshKey]);

  return (
    <Card
      title="Alertas tempranas"
      className="rise-in"
      action={alerts && <span className="text-xs font-semibold text-ink-soft">{alerts.length} {alerts.length === 1 ? "estudiante" : "estudiantes"}</span>}
    >
      <p className="mb-3 text-xs text-ink-soft">
        Estudiantes que no entregaron talleres vencidos, llevan 14 días sin actividad, acumulan 8 compilaciones seguidas con errores,
        tienen nivel bajo en 2 o más RAE o un promedio menor a 3.0. Son señales para conversar, no un diagnóstico.
      </p>
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm text-peach-700">{error}</p>}
      {alerts && alerts.length === 0 && <p className="text-sm text-ink-soft">Nadie necesita atención especial por ahora.</p>}
      {alerts && alerts.length > 0 && (
        <ul className="space-y-2">
          {alerts.map((a) => (
            <li key={a.student_id} className="rounded-2xl bg-amber-50 px-4 py-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="min-w-0 flex-1 font-bold">{a.name}</span>
                <SmallButton onClick={() => onSelect(a.student_id)}>Ver su proceso</SmallButton>
                <Link
                  href={`/docente/clases/${groupId}/reporte/${a.student_id}`}
                  className="rounded-full px-3 py-1 text-xs font-bold text-lavender-700 hover:bg-lavender-50"
                >
                  Reporte
                </Link>
              </div>
              <ul className="mt-1 list-disc space-y-0.5 pl-5 text-sm text-ink">
                {a.reasons.map((r) => (
                  <li key={r}>{r}</li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
