"use client";

import { useState } from "react";
import CodeEditor from "@/components/CodeEditor";
import { SyntaxExample } from "@/lib/api";

const noop = () => {};

/** Agrupa los ejemplos por tema, conservando el orden del catálogo. */
export function byTopic<T extends SyntaxExample>(examples: T[]): { topic: string; examples: T[] }[] {
  const groups: { topic: string; examples: T[] }[] = [];
  for (const e of examples) {
    const last = groups[groups.length - 1];
    if (last?.topic === e.topic) last.examples.push(e);
    else groups.push({ topic: e.topic, examples: [e] });
  }
  return groups;
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      onClick={async () => {
        try {
          await navigator.clipboard.writeText(text);
          setCopied(true);
          setTimeout(() => setCopied(false), 1500);
        } catch {
          /* portapapeles bloqueado: el código sigue visible */
        }
      }}
      className="rounded-full bg-white px-3 py-1 text-xs font-bold text-lavender-700 ring-1 ring-lavender-200 transition hover:bg-lavender-50"
    >
      {copied ? "¡Copiado!" : "Copiar código"}
    </button>
  );
}

/** Explicación, código y salida de un ejemplo. */
export default function SyntaxExampleView({ example, action }: { example: SyntaxExample; action?: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <p className="whitespace-pre-line text-sm leading-relaxed text-ink">{example.explanation}</p>
      <div className="overflow-hidden rounded-2xl ring-1 ring-lavender-100">
        <CodeEditor value={example.code} onChange={noop} readOnly compact label={`Código del ejemplo: ${example.title}`} />
      </div>
      <div className={`grid gap-3 ${example.stdin ? "sm:grid-cols-2" : ""}`}>
        {example.stdin && (
          <div>
            <p className="mb-1 text-xs font-bold text-ink-soft">Entrada de ejemplo</p>
            <pre className="overflow-x-auto rounded-2xl bg-lavender-50/70 p-3 font-mono text-xs">{example.stdin}</pre>
          </div>
        )}
        <div>
          <p className="mb-1 text-xs font-bold text-ink-soft">Salida del programa</p>
          <pre className="overflow-x-auto rounded-2xl bg-lavender-50/70 p-3 font-mono text-xs">{example.output}</pre>
        </div>
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <CopyButton text={example.code} />
        {action}
      </div>
    </div>
  );
}
