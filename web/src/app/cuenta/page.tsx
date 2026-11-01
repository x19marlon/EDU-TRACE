"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import RequireAuth from "@/components/RequireAuth";
import AppShell from "@/components/AppShell";
import RecoveryCodeDialog from "@/components/RecoveryCodeDialog";
import { Avatar, Card } from "@/components/ui";
import { useAuth } from "@/components/AuthProvider";
import {
  changePassword,
  deleteAccount,
  exportMyData,
  getNotifications,
  newRecoveryCode,
  setEmailNotifications,
  updateProfile,
} from "@/lib/api";
import { formatDateTime } from "@/lib/format";
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

function EmailNotices() {
  const [state, setState] = useState<{ email: boolean; available: boolean } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getNotifications()
      .then((list) => setState({ email: list.email, available: list.email_available }))
      .catch((err) => setError(err instanceof Error ? err.message : "No se pudo cargar tu preferencia"));
  }, []);

  const toggle = async (email: boolean) => {
    if (!state) return;
    setBusy(true);
    setError(null);
    try {
      await setEmailNotifications(email);
      setState({ ...state, email });
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo guardar tu preferencia");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3 text-sm">
      <p className="text-ink-soft">
        La campana de arriba te avisa de las novedades: revisiones devueltas o actualizadas, preguntas nuevas o respondidas y
        comentarios de monitores. Si no abres un aviso en 2 minutos, te llega también por correo.
      </p>
      {state && (
        <label className="flex items-start gap-3 rounded-2xl bg-lavender-50/60 px-4 py-3">
          <input
            type="checkbox"
            checked={state.email}
            disabled={busy}
            onChange={(e) => toggle(e.target.checked)}
            className="mt-0.5 h-4 w-4 accent-lavender-600"
          />
          <span>
            <span className="font-semibold text-ink">Recibir los avisos también por correo</span>
            {!state.available && (
              <span className="block text-ink-soft">El servidor todavía no tiene configurado el envío de correos: por ahora solo verás la campana.</span>
            )}
          </span>
        </label>
      )}
      {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 font-semibold text-peach-700">{error}</p>}
    </div>
  );
}

function MyData() {
  const { user, setUser } = useAuth();
  const router = useRouter();
  const [name, setName] = useState(user?.name ?? "");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState<"name" | "export" | "delete" | null>(null);
  const [message, setMessage] = useState<{ ok: boolean; text: string } | null>(null);
  const [confirming, setConfirming] = useState(false);

  if (!user) return null;

  const run = async (kind: "name" | "export" | "delete", action: () => Promise<string | null>) => {
    setBusy(kind);
    setMessage(null);
    try {
      const text = await action();
      if (text) setMessage({ ok: true, text });
    } catch (err) {
      setMessage({ ok: false, text: err instanceof Error ? err.message : "No se pudo completar la acción" });
    } finally {
      setBusy(null);
    }
  };

  const saveName = (e: React.FormEvent) => {
    e.preventDefault();
    run("name", async () => {
      setUser(await updateProfile(name.trim()));
      return "Nombre actualizado.";
    });
  };

  const download = () =>
    run("export", async () => {
      const blob = await exportMyData();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "edutrace-mis-datos.json";
      a.click();
      URL.revokeObjectURL(url);
      return "Descargamos un archivo JSON con todos tus datos.";
    });

  const remove = (e: React.FormEvent) => {
    e.preventDefault();
    run("delete", async () => {
      await deleteAccount(password);
      setUser(null);
      router.replace("/login");
      return null;
    });
  };

  return (
    <div className="space-y-4">
      <p className="text-sm text-ink-soft">
        {user.consent_at
          ? `Autorizaste el tratamiento de tus datos el ${formatDateTime(user.consent_at)}. `
          : "Aún no registras tu autorización. "}
        Lee la{" "}
        <Link href="/privacidad" target="_blank" className="font-bold text-lavender-600 underline">
          política de tratamiento de datos
        </Link>
        .
      </p>
      <form onSubmit={saveName} className="flex flex-wrap gap-2">
        <input
          value={name}
          onChange={(e) => setName(e.target.value)}
          minLength={2}
          maxLength={80}
          required
          aria-label="Tu nombre"
          className={`${inputClass} min-w-0 flex-1`}
        />
        <button
          type="submit"
          disabled={busy !== null || name.trim() === user.name || name.trim().length < 2}
          className="shrink-0 rounded-full bg-lavender-100 px-5 py-2.5 text-sm font-bold text-lavender-700 transition hover:bg-lavender-200 disabled:opacity-40"
        >
          {busy === "name" ? "Guardando..." : "Corregir nombre"}
        </button>
      </form>
      <div className="flex flex-wrap items-center gap-2">
        <button
          onClick={download}
          disabled={busy !== null}
          className="rounded-full bg-mint-100 px-5 py-2.5 text-sm font-bold text-mint-700 transition hover:bg-mint-200 disabled:opacity-40"
        >
          {busy === "export" ? "Preparando..." : "Descargar mis datos"}
        </button>
        <span className="text-xs text-ink-faint">Tu perfil, tus clases, tus entregas e intentos con su código y las revisiones que recibiste.</span>
      </div>
      <div className="rounded-2xl border border-peach-200 p-4">
        {!confirming ? (
          <div className="flex flex-wrap items-center gap-3">
            <p className="min-w-0 flex-1 text-sm text-ink-soft">
              Eliminar tu cuenta borra tus entregas, intentos, revisiones y registros. No se puede deshacer.
              {user.role === "teacher" && " Si eres profesor, primero debes eliminar tus materias."}
            </p>
            <button
              onClick={() => setConfirming(true)}
              className="rounded-full bg-white px-4 py-2 text-sm font-bold text-peach-700 ring-1 ring-peach-200 transition hover:bg-peach-50"
            >
              Eliminar mi cuenta
            </button>
          </div>
        ) : (
          <form onSubmit={remove} className="space-y-2">
            <p className="text-sm font-bold text-peach-700">Confirma con tu contraseña para eliminar tu cuenta y todos tus datos.</p>
            <div className="flex flex-wrap gap-2">
              <input
                type="password"
                required
                autoComplete="current-password"
                placeholder="Tu contraseña"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={`${inputClass} min-w-0 flex-1`}
              />
              <button
                type="submit"
                disabled={busy !== null || !password}
                className="shrink-0 rounded-full bg-peach-600 px-5 py-2.5 text-sm font-bold text-white transition hover:bg-peach-700 disabled:opacity-40"
              >
                {busy === "delete" ? "Eliminando..." : "Eliminar definitivamente"}
              </button>
              <button type="button" onClick={() => setConfirming(false)} className="rounded-full px-4 py-2 text-sm font-bold text-ink-soft hover:bg-lavender-50">
                Cancelar
              </button>
            </div>
          </form>
        )}
      </div>
      {message && (
        <p className={`rounded-2xl px-4 py-2 text-sm font-semibold ${message.ok ? "bg-mint-50 text-mint-700" : "bg-peach-50 text-peach-700"}`}>
          {message.text}
        </p>
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
      <Card title="Avisos" className="rise-in scroll-mt-24" id="avisos">
        <EmailNotices />
      </Card>
      <Card title="Mis datos" className="rise-in">
        <MyData />
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
