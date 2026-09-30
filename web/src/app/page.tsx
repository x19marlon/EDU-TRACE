"use client";

import { useState, useCallback, useRef } from "react";
import dynamic from "next/dynamic";
import OutputPanel from "@/components/OutputPanel";
import FeedbackPanel from "@/components/FeedbackPanel";
import { compileCode, streamFeedback, CompileResult, FeedbackMeta } from "@/lib/api";

// CodeMirror no es SSR-safe, se carga solo en el cliente.
const CodeEditor = dynamic(() => import("@/components/CodeEditor"), {
  ssr: false,
  loading: () => (
    <div className="w-full min-h-[400px] bg-gray-900 rounded-lg border border-gray-700 flex items-center justify-center">
      <p className="text-gray-500">Cargando editor...</p>
    </div>
  ),
});

const DEFAULT_CODE = `#include <iostream>
using namespace std;

int main() {
    cout << "¡Hola, mundo!" << endl;
    return 0;
}
`;

export default function Home() {
  const [code, setCode] = useState(DEFAULT_CODE);
  const [stdin, setStdin] = useState("");
  const [compileResult, setCompileResult] = useState<CompileResult | null>(null);
  const [isCompiling, setIsCompiling] = useState(false);
  const [compileError, setCompileError] = useState<string | null>(null);

  const [feedbackText, setFeedbackText] = useState("");
  const [feedbackMeta, setFeedbackMeta] = useState<FeedbackMeta | null>(null);
  const [isFeedbackLoading, setIsFeedbackLoading] = useState(false);
  const [feedbackError, setFeedbackError] = useState<string | null>(null);
  const feedbackRef = useRef("");

  const [activeTab, setActiveTab] = useState<"output" | "feedback">("output");

  const handleCompile = useCallback(async () => {
    setIsCompiling(true);
    setCompileError(null);
    setCompileResult(null);
    setFeedbackText("");
    setFeedbackMeta(null);
    setFeedbackError(null);
    feedbackRef.current = "";
    setActiveTab("output");

    try {
      const result = await compileCode(code, stdin);
      setCompileResult(result);
    } catch (err) {
      setCompileError(
        err instanceof Error ? err.message : "Error de conexión con el servidor"
      );
    } finally {
      setIsCompiling(false);
    }
  }, [code, stdin]);

  const handleFeedback = useCallback(async () => {
    setIsFeedbackLoading(true);
    setFeedbackError(null);
    setFeedbackText("");
    setFeedbackMeta(null);
    feedbackRef.current = "";
    setActiveTab("feedback");

    try {
      const meta = await streamFeedback(
        code,
        compileResult?.compiler_output || "",
        compileResult?.success || false,
        (token) => {
          feedbackRef.current += token;
          setFeedbackText(feedbackRef.current);
        }
      );
      setFeedbackMeta(meta);
    } catch (err) {
      setFeedbackError(
        err instanceof Error ? err.message : "Error de conexión con Ollama"
      );
    } finally {
      setIsFeedbackLoading(false);
    }
  }, [code, compileResult]);

  return (
    <div className="min-h-screen bg-gray-950 text-white">
      {/* Header */}
      <header className="border-b border-gray-800 px-6 py-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              EDU-TRACE
              <span className="text-blue-500 ml-2 text-lg font-normal">
                Compilador C++
              </span>
            </h1>
            <p className="text-gray-500 text-sm mt-1">
              Plataforma de apoyo para Introducción a la Programación
            </p>
          </div>
          <div className="flex items-center gap-3">
            <button
              onClick={handleCompile}
              disabled={isCompiling || !code.trim()}
              className="px-5 py-2.5 bg-blue-600 hover:bg-blue-700 disabled:bg-gray-700
                disabled:text-gray-500 text-white font-semibold rounded-lg
                transition-colors flex items-center gap-2"
            >
              {isCompiling ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-white" />
                  Compilando...
                </>
              ) : (
                <>
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                      d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                      d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  Compilar
                </>
              )}
            </button>
            <button
              onClick={handleFeedback}
              disabled={isFeedbackLoading || !code.trim()}
              className="px-5 py-2.5 bg-purple-600 hover:bg-purple-700 disabled:bg-gray-700
                disabled:text-gray-500 text-white font-semibold rounded-lg
                transition-colors flex items-center gap-2"
            >
              {isFeedbackLoading ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-t-2 border-b-2 border-white" />
                  Analizando...
                </>
              ) : (
                <>
                  <span>🤖</span>
                  Retroalimentación IA
                </>
              )}
            </button>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto p-6">
        <div className="grid grid-cols-1 lg:grid-cols-5 gap-6" style={{ height: "calc(100vh - 140px)" }}>
          {/* Left Panel — Code Editor (full height) */}
          <div className="lg:col-span-3 flex flex-col">
            <div className="flex items-center justify-between mb-2">
              <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider">
                Código fuente
              </h2>
              <span className="text-xs text-gray-600">C++ (g++ -std=c++17)</span>
            </div>
            <div className="flex-1">
              <CodeEditor value={code} onChange={setCode} />
            </div>
          </div>

          {/* Right Panel — Stdin + Output/Feedback */}
          <div className="lg:col-span-2 flex flex-col">
            {/* Stdin Input */}
            <div className="mb-3">
              <div className="flex items-center justify-between mb-1">
                <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider">
                  Entrada (stdin)
                </h2>
                <span className="text-xs text-gray-600">
                  Datos para cin / scanf
                </span>
              </div>
              <textarea
                value={stdin}
                onChange={(e) => setStdin(e.target.value)}
                placeholder="Escribe aquí la entrada para tu programa..."
                className="w-full h-24 bg-gray-900 border border-gray-700 rounded-lg p-3
                  text-sm font-mono text-green-400 placeholder-gray-600
                  focus:outline-none focus:border-blue-500 resize-y"
                spellCheck={false}
              />
            </div>

            {/* Tabs */}
            <div className="flex gap-1 mb-2">
              <button
                onClick={() => setActiveTab("output")}
                className={`px-3 py-1.5 text-xs font-semibold uppercase tracking-wider rounded-t-md transition-colors ${
                  activeTab === "output"
                    ? "bg-gray-800 text-white border border-gray-700 border-b-0"
                    : "text-gray-500 hover:text-gray-300"
                }`}
              >
                Resultado
              </button>
              <button
                onClick={() => setActiveTab("feedback")}
                className={`px-3 py-1.5 text-xs font-semibold uppercase tracking-wider rounded-t-md transition-colors flex items-center gap-1.5 ${
                  activeTab === "feedback"
                    ? "bg-gray-800 text-white border border-gray-700 border-b-0"
                    : "text-gray-500 hover:text-gray-300"
                }`}
              >
                🤖 Retroalimentación
                {isFeedbackLoading && (
                  <span className="animate-spin rounded-full h-3 w-3 border-t border-b border-purple-400" />
                )}
              </button>
            </div>

            {/* Tab content */}
            <div className="flex-1 min-h-0">
              {activeTab === "output" ? (
                <OutputPanel
                  result={compileResult}
                  isLoading={isCompiling}
                  error={compileError}
                />
              ) : (
                <FeedbackPanel
                  feedbackText={feedbackText}
                  meta={feedbackMeta}
                  isLoading={isFeedbackLoading}
                  error={feedbackError}
                  onRequestFeedback={handleFeedback}
                  canRequestFeedback={!!code.trim()}
                />
              )}
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}
