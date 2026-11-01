"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import { getStudentReport, StudentReport } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

const ORIGIN: Record<string, string> = { ia: "IA, aprobada sin cambios", ia_editada: "IA, editada", docente: "Escrita por el profesor" };
const LEVEL: Record<string, string> = { bajo: "Bajo", medio: "Medio", alto: "Alto" };

function lateText(seconds?: number): string {
  if (!seconds) return "";
  const hours = seconds / 3600;
  return hours < 24 ? ` (${Math.ceil(hours)} h tarde)` : ` (${Math.ceil(hours / 24)} d tarde)`;
}

function Report() {
  const { id, estudiante } = useParams<{ id: string; estudiante: string }>();
  const [report, setReport] = useState<StudentReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getStudentReport(id, estudiante)
      .then(setReport)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo preparar el reporte"));
  }, [id, estudiante]);

  if (error) return <p className="m-6 rounded-2xl bg-peach-50 p-4 text-sm text-peach-700">{error}</p>;
  if (!report) return <p className="m-6 text-sm text-ink-soft">Preparando el reporte...</p>;

  const cell = "border-b border-lavender-100 px-2 py-1.5 align-top";
  return (
    <main className="mx-auto max-w-4xl space-y-6 bg-white p-6 text-ink print:max-w-none print:p-0">
      <div className="flex flex-wrap items-center justify-between gap-2 print:hidden">
        <Link href={`/docente/clases/${id}`} className="text-sm font-bold text-lavender-700 hover:underline">
          ← Volver a la clase
        </Link>
        <button
          type="button"
          onClick={() => window.print()}
          className="rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white hover:bg-lavender-700"
        >
          Descargar PDF
        </button>
      </div>

      <header className="border-b-2 border-lavender-200 pb-3">
        <p className="text-xs font-bold uppercase tracking-wider text-lavender-700">EduTrace · Reporte de seguimiento</p>
        <h1 className="mt-1 text-2xl font-extrabold">{report.name}</h1>
        <p className="text-sm text-ink-soft">
          {report.email} · {report.course_name} · {report.group_name} · generado el {formatDateTime(report.generated_at)}
        </p>
      </header>

      <section aria-labelledby="resumen">
        <h2 id="resumen" className="mb-2 text-lg font-bold">Resumen</h2>
        <dl className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-5">
          {[
            ["Promedio", report.average !== undefined ? report.average.toFixed(1) : "—"],
            ["Entregas oficiales", String(report.officials)],
            ["Intentos de compilación", String(report.attempts)],
            ["Preguntas hechas", String(report.questions)],
            ["Última actividad", report.last_activity ? formatDateTime(report.last_activity) : "—"],
          ].map(([k, v]) => (
            <div key={k} className="rounded-xl bg-lavender-50 px-3 py-2 print:bg-white print:ring-1 print:ring-lavender-100">
              <dt className="text-xs text-ink-soft">{k}</dt>
              <dd className="font-bold">{v}</dd>
            </div>
          ))}
        </dl>
      </section>

      {report.alerts.length > 0 && (
        <section aria-labelledby="alertas">
          <h2 id="alertas" className="mb-2 text-lg font-bold">Señales de alerta</h2>
          <ul className="list-disc space-y-0.5 pl-5 text-sm">
            {report.alerts.map((a) => (
              <li key={a}>{a}</li>
            ))}
          </ul>
        </section>
      )}

      {report.raes.length > 0 && (
        <section aria-labelledby="raes">
          <h2 id="raes" className="mb-2 text-lg font-bold">Nivel por resultado de aprendizaje</h2>
          <table className="w-full text-left text-sm">
            <thead className="text-xs text-ink-soft">
              <tr>
                <th scope="col" className={cell}>RAE</th>
                <th scope="col" className={cell}>Descripción</th>
                <th scope="col" className={cell}>Último nivel</th>
              </tr>
            </thead>
            <tbody>
              {report.raes.map((r) => (
                <tr key={r.code}>
                  <td className={`${cell} font-bold`}>{r.code}</td>
                  <td className={cell}>{r.description}</td>
                  <td className={cell}>{r.level ? LEVEL[r.level] : "Sin evaluar"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      )}

      <section aria-labelledby="talleres">
        <h2 id="talleres" className="mb-2 text-lg font-bold">Talleres</h2>
        {report.assignments.length === 0 ? (
          <p className="text-sm text-ink-soft">La clase no tiene talleres.</p>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="text-xs text-ink-soft">
              <tr>
                <th scope="col" className={cell}>Taller</th>
                <th scope="col" className={cell}>Última entrega</th>
                <th scope="col" className={cell}>Casos</th>
                <th scope="col" className={cell}>Nota</th>
                <th scope="col" className={cell}>Retroalimentación</th>
              </tr>
            </thead>
            <tbody>
              {report.assignments.map((a) => (
                <tr key={a.title}>
                  <td className={cell}>
                    <span className="font-semibold">{a.title}</span>
                    {a.due_at && <span className="block text-xs text-ink-soft">Fecha límite: {formatDateTime(a.due_at)}</span>}
                  </td>
                  <td className={cell}>
                    {a.submitted_at ? `${formatDateTime(a.submitted_at)}${lateText(a.late_seconds)}` : "No entregó"}
                    {a.submissions > 1 && <span className="block text-xs text-ink-soft">{a.submissions} envíos</span>}
                  </td>
                  <td className={cell}>{a.tests_total > 0 ? `${a.tests_passed}/${a.tests_total}` : "—"}</td>
                  <td className={cell}>
                    {a.grade !== undefined ? <span className="font-bold">{a.grade.toFixed(1)}</span> : "—"}
                    {(a.criteria ?? []).map((c) => (
                      <span key={c.id} className="block text-xs text-ink-soft">
                        {c.name}: {c.score !== undefined ? c.score.toFixed(1) : "—"}/{c.points.toFixed(1)}
                      </span>
                    ))}
                  </td>
                  <td className={cell}>{a.returned_at ? ORIGIN[a.origin ?? "docente"] : a.submitted_at ? "Pendiente" : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      <section aria-labelledby="notas" className="break-inside-avoid">
        <h2 id="notas" className="mb-2 text-lg font-bold">Notas de la tutoría</h2>
        <div className="space-y-6 pt-2" aria-hidden>
          {[0, 1, 2, 3, 4].map((i) => (
            <div key={i} className="border-b border-lavender-200" />
          ))}
        </div>
      </section>
    </main>
  );
}

export default function ReportPage() {
  return (
    <RequireAuth role="teacher">
      <Report />
    </RequireAuth>
  );
}
