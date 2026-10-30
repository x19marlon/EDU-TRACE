import { SubmissionMeta, TestResult } from "@/lib/api";

export function TestsPill({ meta }: { meta: Pick<SubmissionMeta, "tests_total" | "tests_passed"> }) {
  const total = meta.tests_total ?? 0;
  if (total === 0) return null;
  const passed = meta.tests_passed ?? 0;
  const tone = passed === total ? "bg-mint-100 text-mint-700" : passed === 0 ? "bg-peach-100 text-peach-700" : "bg-amber-100 text-amber-800";
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${tone}`} title="Casos de prueba del profesor que supera">
      Casos {passed}/{total}
    </span>
  );
}

function Field({ label, value, tone }: { label: string; value?: string; tone: string }) {
  return (
    <div className="min-w-0">
      <p className="mb-0.5 text-[10px] font-bold uppercase tracking-wider text-ink-faint">{label}</p>
      <pre className={`max-h-28 overflow-auto whitespace-pre-wrap rounded-xl p-2 font-mono text-[11px] ${tone}`}>{value || "(vacío)"}</pre>
    </div>
  );
}

export default function TestResults({
  results,
  title = "Casos de prueba del profesor",
  outputLabel = "Salida de tu programa",
}: {
  results: TestResult[];
  title?: string;
  outputLabel?: string;
}) {
  if (results.length === 0) return null;
  const passed = results.filter((r) => r.passed).length;
  const all = passed === results.length;
  return (
    <div className="space-y-2">
      <div className="flex flex-wrap items-center gap-2">
        <h4 className="text-xs font-bold uppercase tracking-wider text-ink-soft">{title}</h4>
        <span className={`rounded-full px-2.5 py-0.5 text-[11px] font-bold ${all ? "bg-mint-100 text-mint-700" : "bg-amber-100 text-amber-800"}`}>
          Pasa {passed} de {results.length}
        </span>
      </div>
      <ul className="space-y-1.5">
        {results.map((r, i) => (
          <li key={r.test_id || i}>
            <details className="group rounded-2xl bg-lavender-50/60 px-3 py-2" open={!r.passed && !r.hidden && results.length <= 3}>
              <summary className="flex cursor-pointer list-none items-center gap-2 text-sm">
                <span
                  className={`flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-[11px] font-bold ${
                    r.passed ? "bg-mint-200 text-mint-700" : "bg-peach-200 text-peach-700"
                  }`}
                  aria-label={r.passed ? "pasa" : "no pasa"}
                >
                  {r.passed ? "✓" : "✗"}
                </span>
                <span className="min-w-0 flex-1 truncate font-semibold">{r.name || `Caso ${i + 1}`}</span>
                {r.hidden && <span className="rounded-full bg-white px-2 py-0.5 text-[10px] font-bold text-ink-soft">oculto</span>}
                {r.error && <span className="hidden truncate text-xs text-peach-700 sm:inline">{r.error}</span>}
              </summary>
              {r.stdin === undefined && r.expected === undefined && r.hidden ? (
                <p className="mt-2 text-xs text-ink-soft">
                  Tu profesor mantiene oculto este caso: solo sabes si tu programa lo supera.
                </p>
              ) : (
                <div className="mt-2 grid gap-2 sm:grid-cols-3">
                  <Field label="Entrada" value={r.stdin} tone="bg-white" />
                  <Field label="Salida esperada" value={r.expected} tone="bg-mint-50" />
                  <Field label={outputLabel} value={r.output} tone={r.passed ? "bg-mint-50" : "bg-peach-50"} />
                </div>
              )}
              {r.error && <p className="mt-1.5 text-xs font-semibold text-peach-700">{r.error}</p>}
            </details>
          </li>
        ))}
      </ul>
    </div>
  );
}
