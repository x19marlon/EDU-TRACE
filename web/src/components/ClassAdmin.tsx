"use client";

import { useState } from "react";
import { addMonitor, GroupMonitor, removeMonitor } from "@/lib/api";
import { Avatar, Card, SmallButton } from "@/components/ui";

export function MonitorsCard({
  groupId,
  monitors,
  onChange,
}: {
  groupId: string;
  monitors: GroupMonitor[];
  onChange: (monitors: GroupMonitor[]) => void;
}) {
  const [email, setEmail] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const add = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const m = await addMonitor(groupId, email.trim());
      onChange([...monitors.filter((x) => x.id !== m.id), m]);
      setEmail("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo agregar al monitor");
    } finally {
      setBusy(false);
    }
  };

  const remove = async (m: GroupMonitor) => {
    if (!window.confirm(`¿Quitar a ${m.name} como monitor de esta clase? Su retroalimentación ya enviada se conserva.`)) return;
    setError(null);
    try {
      await removeMonitor(groupId, m.id);
      onChange(monitors.filter((x) => x.id !== m.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo quitar al monitor");
    }
  };

  return (
    <Card
      title="Monitores"
      className="rise-in"
      action={<span className="text-xs font-semibold text-ink-faint">{monitors.length}</span>}
    >
      <p className="mb-3 text-sm text-ink-soft">
        Los monitores ven las entregas oficiales de esta clase y escriben retroalimentación a los estudiantes. No califican
        ni ven las notas: eso sigue siendo tuyo.
      </p>
      {monitors.length > 0 && (
        <ul className="mb-4 space-y-1">
          {monitors.map((m) => (
            <li key={m.id} className="flex items-center gap-3 rounded-2xl bg-mint-50/70 px-3 py-2">
              <Avatar name={m.name} />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-bold">{m.name}</span>
                <span className="block truncate text-xs text-ink-soft">{m.email}</span>
              </span>
              <SmallButton tone="danger" onClick={() => remove(m)}>
                Quitar
              </SmallButton>
            </li>
          ))}
        </ul>
      )}
      <form onSubmit={add} className="space-y-1.5">
        <div className="flex gap-2">
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder="Email del monitor"
            maxLength={254}
            className="min-w-0 flex-1 rounded-full border border-lavender-100 bg-lavender-50/60 px-4 py-2 text-sm
              placeholder-ink-faint focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100"
          />
          <button
            type="submit"
            disabled={busy || !email.trim()}
            className="shrink-0 rounded-full bg-mint-700 px-4 py-2 text-sm font-bold text-white transition hover:bg-ink disabled:opacity-40"
          >
            {busy ? "Agregando..." : "Agregar monitor"}
          </button>
        </div>
        <p className="px-2 text-xs text-ink-faint">La persona debe registrarse primero en EduTrace con el rol Monitor.</p>
        {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
      </form>
    </Card>
  );
}
