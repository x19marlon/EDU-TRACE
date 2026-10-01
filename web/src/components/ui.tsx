"use client";

import { useState } from "react";

const AVATAR_TONES = [
  "bg-lavender-200 text-lavender-700",
  "bg-mint-200 text-mint-700",
  "bg-peach-200 text-peach-700",
];

export function initials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((w) => w[0]!.toUpperCase())
    .join("");
}

/** Círculo con iniciales; el color se deriva del nombre para que sea estable. */
export function Avatar({ name, className = "h-9 w-9 text-xs" }: { name: string; className?: string }) {
  let hash = 0;
  for (const ch of name) hash = (hash * 31 + ch.charCodeAt(0)) | 0;
  const tone = AVATAR_TONES[Math.abs(hash) % AVATAR_TONES.length];
  return (
    <span
      aria-hidden
      className={`inline-flex shrink-0 items-center justify-center rounded-full font-bold ${tone} ${className}`}
    >
      {initials(name)}
    </span>
  );
}

/** Tarjeta blanca con esquinas grandes, como en el diseño. */
export function Card({
  title,
  action,
  className = "",
  surface = "bg-white",
  bodyClassName = "p-5",
  children,
}: {
  title?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
  surface?: string;
  bodyClassName?: string;
  children: React.ReactNode;
}) {
  return (
    <section className={`${surface} rounded-3xl shadow-soft border border-lavender-100/70 flex flex-col ${className}`}>
      {title && (
        <header className="flex items-center justify-between gap-3 px-5 pt-4">
          <h2 className="font-bold text-ink">{title}</h2>
          {action}
        </header>
      )}
      <div className={`flex-1 min-h-0 ${bodyClassName}`}>{children}</div>
    </section>
  );
}

export function Spinner({ className = "h-4 w-4 border-white" }: { className?: string }) {
  return <span aria-hidden className={`inline-block animate-spin rounded-full border-2 border-t-transparent ${className}`} />;
}

/** Anillo de progreso (0-100). */
export function ProgressRing({ value, className = "h-28 w-28" }: { value: number; className?: string }) {
  const r = 42;
  const c = 2 * Math.PI * r;
  return (
    <div className={`relative shrink-0 ${className}`}>
      <svg viewBox="0 0 100 100" className="h-full w-full -rotate-90" aria-hidden>
        <circle cx="50" cy="50" r={r} fill="none" stroke="#EDE7FC" strokeWidth="11" />
        <circle
          cx="50"
          cy="50"
          r={r}
          fill="none"
          stroke="#6FC2A2"
          strokeWidth="11"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c * (1 - value / 100)}
          style={{ transition: "stroke-dashoffset 1s cubic-bezier(0.16, 1, 0.3, 1)" }}
        />
      </svg>
      <span className="absolute inset-0 flex items-center justify-center text-2xl font-extrabold tabular-nums">
        {value}%
      </span>
    </div>
  );
}

/** Formulario de una línea (nombre + botón), para crear materias y clases. */
export function InlineCreate({
  placeholder,
  button,
  onCreate,
}: {
  placeholder: string;
  button: string;
  onCreate: (name: string) => Promise<void>;
}) {
  const [name, setName] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setError(null);
    try {
      await onCreate(name.trim());
      setName("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo crear");
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-1.5">
      <div className="flex gap-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={placeholder}
          maxLength={80}
          className="min-w-0 flex-1 rounded-full border border-lavender-100 bg-lavender-50/60 px-4 py-2 text-sm
            placeholder-ink-faint focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100"
        />
        <button
          type="submit"
          disabled={busy || name.trim().length < 2}
          className="shrink-0 rounded-full bg-lavender-600 px-4 py-2 text-sm font-bold text-white transition
            hover:bg-lavender-700 disabled:opacity-40"
        >
          {busy ? "Creando..." : button}
        </button>
      </div>
      {error && <p className="px-2 text-xs font-semibold text-peach-700">{error}</p>}
    </form>
  );
}
