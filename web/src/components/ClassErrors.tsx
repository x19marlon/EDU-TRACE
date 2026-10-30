"use client";

import { useEffect, useState } from "react";
import { downloadGroupCSV, ErrorReport, FrequentError, getFrequentErrors } from "@/lib/api";
import { Card, SmallButton, Spinner } from "@/components/ui";
import { relativeTime } from "@/lib/format";

function ErrorList({ title, items, empty }: { title: string; items: FrequentError[]; empty: string }) {
  return (
    <div className="space-y-2">
      <h3 className="text-xs font-bold uppercase tracking-wider text-ink-soft">{title}</h3>
      {items.length === 0 ? (
        <p className="text-sm text-ink-soft">{empty}</p>
      ) : (
        <ol className="space-y-1.5">
          {items.map((e) => (
            <li key={e.message} className="rounded-2xl bg-lavender-50/60 px-3 py-2">
              <div className="flex flex-wrap items-baseline gap-x-2 gap-y-1">
                <code className="min-w-0 flex-1 break-words font-mono text-xs text-peach-700">{e.message}</code>
                <span className="rounded-full bg-white px-2 py-0.5 text-[11px] font-bold text-lavender-700">
                  {e.students} {e.students === 1 ? "estudiante" : "estudiantes"} · {e.count} {e.count === 1 ? "vez" : "veces"}
                </span>
              </div>
              {e.explanation && <p className="mt-1 text-sm">{e.explanation}</p>}
              <p className="mt-0.5 text-[11px] text-ink-faint">
                {e.assignments.length > 0 ? `${e.assignments.join(" · ")} · ` : ""}última vez {relativeTime(e.last_at).toLowerCase()}
              </p>
            </li>
          ))}
        </ol>
      )}
    </div>
  );
}

export function ExportButtons({ groupId }: { groupId: string }) {
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const download = async (kind: "notas" | "revisiones") => {
    setBusy(kind);
    setError(null);
    try {
      const { blob, filename } = await downloadGroupCSV(groupId, kind);
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo exportar");
    } finally {
      setBusy(null);
    }
  };
  return (
    <span className="flex flex-wrap items-center gap-1">
      <SmallButton onClick={() => download("notas")}>{busy === "notas" ? "Exportando..." : "Exportar notas (CSV)"}</SmallButton>
      <SmallButton onClick={() => download("revisiones")}>{busy === "revisiones" ? "Exportando..." : "Exportar revisiones (CSV)"}</SmallButton>
      {error && <span className="text-xs font-semibold text-peach-700">{error}</span>}
    </span>
  );
}

export default function ClassErrors({ groupId, refreshKey }: { groupId: string; refreshKey: number }) {
  const [wholeCourse, setWholeCourse] = useState(false);
  const [report, setReport] = useState<ErrorReport | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setReport(null);
    getFrequentErrors(groupId, wholeCourse)
      .then(setReport)
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudieron cargar los errores"));
  }, [groupId, wholeCourse, refreshKey]);

  return (
    <Card
      title="Errores frecuentes (RF-21)"
      className="rise-in"
      action={
        <div role="radiogroup" aria-label="Alcance" className="flex gap-1 rounded-full bg-lavender-50 p-1">
          {([
            [false, "Esta clase"],
            [true, "Toda la materia"],
          ] as const).map(([value, label]) => (
            <button
              key={label}
              type="button"
              role="radio"
              aria-checked={wholeCourse === value}
              onClick={() => setWholeCourse(value)}
              className={`rounded-full px-3 py-1 text-xs font-bold transition ${
                wholeCourse === value ? "bg-white text-lavender-700 shadow-soft" : "text-ink-soft hover:text-ink"
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      }
    >
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm text-peach-700">{error}</p>}
      {!report ? (
        !error && (
          <p className="flex items-center gap-2 text-sm text-ink-soft">
            <Spinner className="h-4 w-4 border-lavender-500" /> Analizando intentos y entregas...
          </p>
        )
      ) : (
        <div className="space-y-4">
          <p className="text-xs text-ink-soft">
            {report.analyzed} intentos y entregas analizados
            {report.scope === "materia" ? ` en ${report.groups} ${report.groups === 1 ? "clase" : "clases"}` : ""} ·{" "}
            {report.students_with_errors} {report.students_with_errors === 1 ? "estudiante tuvo" : "estudiantes tuvieron"} al menos un error.
            Ordenados por cuántos estudiantes los tuvieron.
          </p>
          <div className="grid gap-4 lg:grid-cols-2">
            <ErrorList title="Errores de compilación" items={report.compile} empty="Nadie ha tenido errores de compilación." />
            <div className="space-y-4">
              <ErrorList title="Problemas al ejecutar" items={report.runtime} empty="Ningún programa falló al ejecutarse." />
              <div className="space-y-2">
                <h3 className="text-xs font-bold uppercase tracking-wider text-ink-soft">Casos de prueba que más cuestan</h3>
                {report.tests.length === 0 ? (
                  <p className="text-sm text-ink-soft">Aún no hay casos de prueba fallidos.</p>
                ) : (
                  <ul className="space-y-1">
                    {report.tests.map((t) => (
                      <li key={`${t.assignment}-${t.test}`} className="flex flex-wrap items-baseline gap-x-2 rounded-xl bg-lavender-50/60 px-3 py-1.5 text-xs">
                        <span className="font-bold">{t.test}</span>
                        {t.hidden && <span className="text-ink-faint">(oculto)</span>}
                        <span className="text-ink-soft">{t.assignment}</span>
                        <span className="flex-1" />
                        <span className="font-semibold text-peach-700">
                          {t.not_passing} de {t.students} {t.students === 1 ? "estudiante" : "estudiantes"} aún no lo{" "}
                          {t.not_passing === 1 ? "pasa" : "pasan"}
                        </span>
                        <span className="text-ink-faint">
                          · {t.failures} {t.failures === 1 ? "fallo" : "fallos"} en {t.runs} {t.runs === 1 ? "ejecución" : "ejecuciones"}
                        </span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </div>
          </div>
        </div>
      )}
    </Card>
  );
}
