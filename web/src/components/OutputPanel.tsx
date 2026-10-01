import { CompileResult } from "@/lib/api";
import { Spinner } from "@/components/ui";

interface OutputPanelProps {
  result: CompileResult | null;
  isLoading: boolean;
  error: string | null;
}

function Pill({ tone, children }: { tone: "mint" | "peach" | "amber"; children: React.ReactNode }) {
  const tones = {
    mint: "bg-mint-100 text-mint-700",
    peach: "bg-peach-100 text-peach-700",
    amber: "bg-amber-100 text-amber-800",
  };
  return <span className={`rounded-full px-3 py-1 text-xs font-bold ${tones[tone]}`}>{children}</span>;
}

export default function OutputPanel({ result, isLoading, error }: OutputPanelProps) {
  if (isLoading) {
    return (
      <div className="flex h-full min-h-[300px] flex-col items-center justify-center gap-3 p-6">
        <Spinner className="h-9 w-9 border-[3px] border-lavender-200 border-t-lavender-500" />
        <p className="text-sm font-semibold text-ink-soft">Compilando...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full min-h-[300px] p-5">
        <Pill tone="peach">Error</Pill>
        <pre className="mt-3 whitespace-pre-wrap rounded-2xl bg-peach-50 p-4 font-mono text-sm text-peach-700">{error}</pre>
      </div>
    );
  }

  if (!result) {
    return (
      <div className="flex h-full min-h-[300px] flex-col items-center justify-center gap-3 p-6 text-center">
        <span className="flex h-14 w-14 items-center justify-center rounded-2xl bg-lavender-100 text-lavender-600">
          <svg className="h-7 w-7" fill="none" stroke="currentColor" viewBox="0 0 24 24" strokeWidth={1.8} aria-hidden>
            <path strokeLinecap="round" strokeLinejoin="round" d="M5.25 5.653c0-.856.917-1.398 1.667-.986l11.54 6.347a1.125 1.125 0 0 1 0 1.972l-11.54 6.347a1.125 1.125 0 0 1-1.667-.986V5.653Z" />
          </svg>
        </span>
        <p className="max-w-xs text-sm text-ink-soft">
          Escribe tu código C++ y presiona <strong className="text-ink">Compilar</strong> para ver el resultado.
        </p>
      </div>
    );
  }

  return (
    <div className="h-full min-h-[300px] overflow-auto p-5">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          {result.success ? <Pill tone="mint">✓ Compilación exitosa</Pill> : <Pill tone="peach">✗ Error de compilación</Pill>}
          {result.error && <Pill tone="amber">{result.error}</Pill>}
        </div>
        {!result.error && result.success && (
          <span className="text-xs font-semibold text-ink-faint">código de salida: {result.exit_code}</span>
        )}
      </div>

      {result.compiler_output && (
        <div className="mb-4">
          <h3 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Salida del compilador</h3>
          <pre className="whitespace-pre-wrap rounded-2xl bg-peach-50 p-4 font-mono text-sm text-peach-700">
            {result.compiler_output}
          </pre>
        </div>
      )}

      {result.success && (
        <div className="mb-4">
          <h3 className="mb-1.5 text-xs font-bold uppercase tracking-wider text-ink-soft">Salida del programa</h3>
          <pre className="whitespace-pre-wrap rounded-2xl bg-mint-50 p-4 font-mono text-sm text-ink">
            {result.program_output || "(sin salida)"}
          </pre>
        </div>
      )}

      <div className="mt-3 flex gap-4 border-t border-lavender-100 pt-3 text-xs font-semibold text-ink-faint">
        <span>Compilación: {result.compile_time_ms} ms</span>
        {result.success && <span>Ejecución: {result.run_time_ms} ms</span>}
      </div>
    </div>
  );
}
