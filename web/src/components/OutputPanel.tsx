import { CompileResult } from "@/lib/api";

interface OutputPanelProps {
  result: CompileResult | null;
  isLoading: boolean;
  error: string | null;
}

export default function OutputPanel({ result, isLoading, error }: OutputPanelProps) {
  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full min-h-[400px] bg-gray-900 rounded-lg border border-gray-700 p-6">
        <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-blue-500 mb-4" />
        <p className="text-gray-400 text-sm">Compilando...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="h-full min-h-[400px] bg-gray-900 rounded-lg border border-red-800 p-4">
        <div className="flex items-center gap-2 mb-3">
          <span className="px-2 py-1 rounded text-xs font-bold bg-red-900 text-red-300">
            ERROR
          </span>
        </div>
        <pre className="text-red-400 text-sm font-mono whitespace-pre-wrap">{error}</pre>
      </div>
    );
  }

  if (!result) {
    return (
      <div className="flex flex-col items-center justify-center h-full min-h-[400px] bg-gray-900 rounded-lg border border-gray-700 p-6">
        <svg className="w-12 h-12 text-gray-600 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5}
            d="M14.25 9.75L16.5 12l-2.25 2.25m-4.5 0L7.5 12l2.25-2.25M6 20.25h12A2.25 2.25 0 0020.25 18V6A2.25 2.25 0 0018 3.75H6A2.25 2.25 0 003.75 6v12A2.25 2.25 0 006 20.25z" />
        </svg>
        <p className="text-gray-500 text-sm text-center">
          Escribe tu código C++ y presiona <strong>Compilar</strong> para ver el resultado.
        </p>
      </div>
    );
  }

  return (
    <div className="h-full min-h-[400px] bg-gray-900 rounded-lg border border-gray-700 p-4 overflow-auto">
      {/* Status Badge */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          {result.success ? (
            <span className="px-2 py-1 rounded text-xs font-bold bg-green-900 text-green-300">
              ✓ COMPILACIÓN EXITOSA
            </span>
          ) : (
            <span className="px-2 py-1 rounded text-xs font-bold bg-red-900 text-red-300">
              ✗ ERROR DE COMPILACIÓN
            </span>
          )}
          {result.error && (
            <span className="px-2 py-1 rounded text-xs bg-yellow-900 text-yellow-300">
              {result.error}
            </span>
          )}
        </div>
        <span className="text-xs text-gray-500">
          código: {result.exit_code}
        </span>
      </div>

      {/* Compiler Output */}
      {result.compiler_output && (
        <div className="mb-4">
          <h3 className="text-xs font-semibold text-gray-400 uppercase mb-1">
            Salida del compilador
          </h3>
          <pre className="text-sm font-mono whitespace-pre-wrap bg-gray-950 rounded p-3 text-red-400 border border-gray-800">
            {result.compiler_output}
          </pre>
        </div>
      )}

      {/* Program Output */}
      {result.success && (
        <div className="mb-4">
          <h3 className="text-xs font-semibold text-gray-400 uppercase mb-1">
            Salida del programa
          </h3>
          <pre className="text-sm font-mono whitespace-pre-wrap bg-gray-950 rounded p-3 text-green-400 border border-gray-800">
            {result.program_output || "(sin salida)"}
          </pre>
        </div>
      )}

      {/* Timing */}
      <div className="flex gap-4 text-xs text-gray-500 mt-3 pt-3 border-t border-gray-800">
        <span>⏱ Compilación: {result.compile_time_ms}ms</span>
        {result.success && <span>⏱ Ejecución: {result.run_time_ms}ms</span>}
      </div>
    </div>
  );
}
