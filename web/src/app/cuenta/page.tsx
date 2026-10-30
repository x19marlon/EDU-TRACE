"use client";

import { useState } from "react";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import RecoveryCodeDialog from "@/components/RecoveryCodeDialog";
import { Avatar, Card } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import { changePassword, newRecoveryCode } from "@/lib/api";
import { ROLE_LABELS } from "@/lib/role";

const inputClass =
  "w-full rounded-2xl border border-lavender-100 bg-lavender-50/60 px-4 py-2.5 text-sm placeholder-ink-faint " +
  "focus:border-lavender-300 focus:bg-white focus:outline-none focus:ring-4 focus:ring-lavender-100";

function PasswordForm() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (next !== confirm) {
      setMessage({ ok: false, text: "Las dos contraseñas nuevas no coinciden" });
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      await changePassword(current, next);
      setCurrent("");
      setNext("");
      setConfirm("");
      setMessage({ ok: true, text: "Contraseña actualizada. Se cerró la sesión en tus otros dispositivos." });
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : "No se pudo cambiar la contraseña" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-3">
      <input
        type="password"
        required
        autoComplete="current-password"
        placeholder="Contraseña actual"
        value={current}
        onChange={(e) => setCurrent(e.target.value)}
        className={inputClass}
      />
      <input
        type="password"
        required
        minLength={8}
        maxLength={128}
        autoComplete="new-password"
        placeholder="Nueva contraseña (mín. 8 caracteres)"
        value={next}
        onChange={(e) => setNext(e.target.value)}
        className={inputClass}
      />
      <input
        type="password"
        required
        minLength={8}
        maxLength={128}
        autoComplete="new-password"
        placeholder="Repite la nueva contraseña"
        value={confirm}
        onChange={(e) => setConfirm(e.target.value)}
        className={inputClass}
      />
      {message && (
        <p className={`rounded-2xl px-4 py-2 text-sm font-semibold ${message.ok ? "bg-mint-50 text-mint-700" : "bg-peach-50 text-peach-700"}`}>
          {message.text}
        </p>
      )}
      <button
        type="submit"
        disabled={busy}
        className="rounded-full bg-lavender-600 px-5 py-2.5 text-sm font-bold text-white shadow-soft transition hover:bg-lavender-700 disabled:opacity-40"
      >
        {busy ? "Guardando..." : "Cambiar contraseña"}
      </button>
    </form>
  );
}

function RecoverySection() {
  const { user, setUser } = useAuth();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [code, setCode] = useState<string | null>(null);

  if (!user) return null;

  const generate = async (e: React.FormEvent) => {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const { recovery_code } = await newRecoveryCode(password);
      setPassword("");
      setCode(recovery_code);
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo generar el código");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3">
      {user.has_recovery ? (
        <p className="rounded-2xl bg-mint-50 px-4 py-2 text-sm font-semibold text-mint-700">
          Tienes un código de recuperación activo. Si lo perdiste, genera uno nuevo: el anterior deja de servir.
        </p>
      ) : (
        <p className="rounded-2xl bg-amber-50 px-4 py-2 text-sm font-semibold text-amber-800">
          Aún no tienes código de recuperación. Sin él no podrás recuperar tu cuenta si olvidas la contraseña.
        </p>
      )}
      <form onSubmit={generate} className="flex flex-wrap gap-2">
        <input
          type="password"
          required
          autoComplete="current-password"
          placeholder="Confirma con tu contraseña"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          className={`${inputClass} min-w-0 flex-1`}
        />
        <button
          type="submit"
          disabled={busy || !password}
          className="shrink-0 rounded-full bg-lavender-100 px-5 py-2.5 text-sm font-bold text-lavender-700 transition hover:bg-lavender-200 disabled:opacity-40"
        >
          {busy ? "Generando..." : user.has_recovery ? "Generar uno nuevo" : "Generar código"}
        </button>
      </form>
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}
      {code && (
        <RecoveryCodeDialog
          code={code}
          email={user.email}
          onDone={() => {
            setCode(null);
            setUser({ ...user, has_recovery: true });
          }}
        />
      )}
    </div>
  );
}

function AccountView() {
  const { user } = useAuth();
  if (!user) return null;
  return (
    <div className="mx-auto max-w-3xl space-y-5">
      <section className="rise-in flex items-center gap-4 rounded-3xl bg-white p-5 shadow-soft sm:p-6">
        <Avatar name={user.name} className="h-14 w-14 text-lg ring-4 ring-lavender-50" />
        <div className="min-w-0">
          <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Mi cuenta</h1>
          <p className="truncate text-sm text-ink-soft">
            {user.name} · {user.email} · {ROLE_LABELS[user.role]}
          </p>
        </div>
      </section>
      <Card title="Cambiar contraseña" className="rise-in">
        <PasswordForm />
      </Card>
      <Card title="Código de recuperación" className="rise-in">
        <p className="mb-3 text-sm text-ink-soft">
          Si olvidas tu contraseña, en la pantalla de inicio eliges «¿Olvidaste tu contraseña?» y usas tu email y este código para
          crear una nueva. Cada código sirve una sola vez.
        </p>
        <RecoverySection />
      </Card>
    </div>
  );
}

export default function AccountPage() {
  return (
    <RequireAuth>
      <AppShell>
        <AccountView />
      </AppShell>
    </RequireAuth>
  );
}
