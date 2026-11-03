"use client";

import { useEffect, useMemo, useState } from "react";
import { Card, SmallButton } from "@/components/ui";
import SyntaxExampleView, { byTopic } from "@/components/SyntaxExampleView";
import { GroupSyntaxExample, listGroupExamples, setGroupExamples } from "@/lib/api";

function ExampleRow({
  example,
  disabled,
  onToggle,
}: {
  example: GroupSyntaxExample;
  disabled: boolean;
  onToggle: (enabled: boolean) => void;
}) {
  const [open, setOpen] = useState(false);
  return (
    <li className="rounded-2xl bg-lavender-50/50 px-3 py-2">
      <div className="flex items-center gap-3">
        <label className="flex min-w-0 flex-1 cursor-pointer items-center gap-3 text-sm font-semibold">
          <input
            type="checkbox"
            className="h-4 w-4 shrink-0 accent-lavender-600"
            checked={example.enabled}
            disabled={disabled}
            onChange={(e) => onToggle(e.target.checked)}
          />
          <span className="min-w-0">{example.title}</span>
        </label>
        <SmallButton onClick={() => setOpen((o) => !o)}>{open ? "Ocultar" : "Ver"}</SmallButton>
      </div>
      {open && (
        <div className="mt-3 border-t border-lavender-100 pt-3">
          <SyntaxExampleView example={example} />
        </div>
      )}
    </li>
  );
}

/** El docente elige qué ejemplos de sintaxis ven los estudiantes de la clase. */
export default function ClassExamplesCard({ groupId }: { groupId: string }) {
  const [examples, setExamples] = useState<GroupSyntaxExample[] | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    listGroupExamples(groupId)
      .then((list) => !cancelled && setExamples(list))
      .catch((err) => !cancelled && setError(err instanceof Error ? err.message : "No se pudieron cargar los ejemplos"));
    return () => {
      cancelled = true;
    };
  }, [groupId]);

  const topics = useMemo(() => byTopic(examples ?? []), [examples]);
  const active = (examples ?? []).filter((e) => e.enabled).length;

  const change = async (ids: string[], enabled: boolean) => {
    const before = examples;
    setBusy(true);
    setError(null);
    setExamples((list) => list?.map((e) => (ids.includes(e.id) ? { ...e, enabled } : e)) ?? null);
    try {
      setExamples(await setGroupExamples(groupId, ids, enabled));
    } catch (err) {
      setExamples(before);
      setError(err instanceof Error ? err.message : "No se pudieron cambiar los ejemplos");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Card
      title="Ejemplos de sintaxis"
      className="rise-in"
      action={<SmallButton onClick={() => setExpanded((v) => !v)}>{expanded ? "Ocultar" : "Elegir ejemplos"}</SmallButton>}
    >
      {examples && (
        <p className="text-sm text-ink-soft">
          {active === 0
            ? "Tus estudiantes todavía no ven ningún ejemplo. Agrégalos a medida que avances en los temas."
            : `Tus estudiantes ven ${active} de ${examples.length} ejemplos, en la sección Ejemplos.`}
        </p>
      )}
      {error && <p className="mt-2 rounded-2xl bg-peach-50 p-3 text-sm text-peach-700">{error}</p>}
      {!examples && !error && <p className="text-sm text-ink-soft">Cargando...</p>}
      {expanded && (
        <div className="mt-4 space-y-3">
          {topics.map(({ topic, examples: items }) => {
            const on = items.filter((e) => e.enabled).length;
            return (
              <details key={topic} className="rounded-2xl ring-1 ring-lavender-100">
                <summary className="cursor-pointer rounded-2xl px-4 py-3 font-bold hover:bg-lavender-50">
                  {topic} <span className="ml-1 text-xs font-semibold text-ink-soft">{on} de {items.length} agregados</span>
                </summary>
                <div className="space-y-2 px-3 pb-3">
                  <div className="flex flex-wrap gap-2 px-1">
                    <SmallButton disabled={busy || on === items.length} onClick={() => change(items.map((e) => e.id), true)}>
                      Agregar todo el tema
                    </SmallButton>
                    <SmallButton disabled={busy || on === 0} onClick={() => change(items.map((e) => e.id), false)}>
                      Quitar todo el tema
                    </SmallButton>
                  </div>
                  <ul className="space-y-1.5">
                    {items.map((e) => (
                      <ExampleRow key={e.id} example={e} disabled={busy} onToggle={(enabled) => change([e.id], enabled)} />
                    ))}
                  </ul>
                </div>
              </details>
            );
          })}
        </div>
      )}
    </Card>
  );
}
