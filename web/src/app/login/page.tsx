"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { useAuth } from "@/components/AuthProvider";
import Logo from "@/components/Logo";
import { login, recoverAccount, register, Role, SessionUser } from "@/lib/api";
import RecoveryCodeDialog from "@/components/RecoveryCodeDialog";
import { homeFor, isRole, loadRole, ROLE_LABELS, saveRole } from "@/lib/role";

const ROLE_OPTIONS: { value: Role; label: string }[] = (["student", "teacher", "monitor"] as const).map((value) => ({
  value,
  label: ROLE_LABELS[value],
}));

/** Pestañas para elegir con qué tipo de cuenta entrar, visibles de una vez. */
function RoleTabs({ role, onChange }: { role: Role; onChange: (r: Role) => void }) {
  return (
    <div role="radiogroup" aria-label="Entrar como" className="grid grid-cols-3 gap-1 rounded-full bg-lavender-50 p-1">
      {ROLE_OPTIONS.map((o) => (
        <button
          key={o.value}
          type="button"
          role="radio"
          aria-checked={role === o.value}
          onClick={() => onChange(o.value)}
          className={`rounded-full py-2 text-sm font-bold transition ${
            role === o.value ? "bg-white text-lavender-700 shadow-soft" : "text-ink-soft hover:text-ink"
          }`}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}

const SUBTITLES: Record<"login" | "register", Record<Role, string>> = {
  login: {
    student: "Entra para seguir practicando.",
    teacher: "Entra para acompañar a tu grupo.",
    monitor: "Entra para retroalimentar a los estudiantes de tus clases.",
  },
  register: {
    student: "Empieza a practicar en un minuto.",
    teacher: "Necesitarás el código de profesor de tu institución.",
    monitor: "Después, pídele a tu profesor que te agregue a su clase con este email.",
  },
};

const inputClass =
  "w-full rounded-2xl border border-lavender-100 bg-lavender-50/60 px-4 py-3 text-sm text-ink " +
  "placeholder-ink-faint transition focus:border-lavender-300 focus:bg-white focus:outline-none " +
  "focus:ring-4 focus:ring-lavender-100";

export default function LoginPage() {
  const { user, isLoading, setUser } = useAuth();
  const router = useRouter();

  const [mode, setMode] = useState<"login" | "register" | "recover">("login");
  const [recoveryCode, setRecoveryCode] = useState("");
  const [confirm, setConfirm] = useState("");
  const [pending, setPending] = useState<{ user: SessionUser; code: string; title: string } | null>(null);
  const [role, setRole] = useState<Role>("student");

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [teacherCode, setTeacherCode] = useState("");
  const [consent, setConsent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // ?rol=teacher|student (desde "Cambiar de rol") tiene prioridad sobre la última elección.
  useEffect(() => {
    const fromUrl = new URLSearchParams(window.location.search).get("rol");
    if (isRole(fromUrl)) {
      setRole(fromUrl);
      saveRole(fromUrl);
    } else {
      setRole(loadRole());
    }
  }, []);

  useEffect(() => {
    if (!isLoading && user) router.replace(homeFor(user.role));
  }, [user, isLoading, router]);

  const chooseRole = (r: Role) => {
    setRole(r);
    saveRole(r);
    setError(null);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    try {
      if (mode === "recover") {
        if (password !== confirm) {
          setError("Las dos contraseñas no coinciden");
          return;
        }
        const u = await recoverAccount(email, recoveryCode, password);
        setPending({ user: u, code: u.recovery_code ?? "", title: "Listo: este es tu nuevo código de recuperación" });
        return;
      }
      const u =
        mode === "login"
          ? await login(email, password, role)
          : await register({
              name,
              email,
              password,
              role,
              consent,
              ...(role === "teacher" && { teacher_code: teacherCode }),
            });
      if (u.recovery_code) setPending({ user: u, code: u.recovery_code, title: "Guarda tu código de recuperación" });
      else setUser(u);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error de conexión con el servidor");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="relative min-h-screen overflow-hidden flex items-center justify-center p-4 sm:p-8">
      {/* Formas pastel de fondo */}
      <div aria-hidden className="pointer-events-none absolute inset-0">
        <div className="blob bg-lavender-200/70 h-[30rem] w-[30rem] -top-40 -left-32" />
        <div className="blob bg-mint-200/70 h-80 w-[28rem] top-1/3 -right-40" style={{ animationDelay: "-7s" }} />
        <div className="blob bg-peach-200/70 h-96 w-96 -bottom-48 left-1/4" style={{ animationDelay: "-14s" }} />
      </div>

      <div className="relative grid w-full max-w-5xl rounded-[2rem] bg-white/60 shadow-lift backdrop-blur-xl md:grid-cols-2">
        {/* Presentación */}
        <div className="relative hidden md:flex flex-col justify-center gap-10 rounded-l-[2rem] bg-lavender-100 p-10 lg:p-12">
          <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden rounded-l-[2rem]">
            <div className="blob bg-mint-200 h-64 w-80 -bottom-16 -right-20" />
            <div className="blob bg-peach-200/80 h-40 w-40 top-24 -right-10" style={{ animationDelay: "-5s" }} />
          </div>

          <div className="relative rise-in">
            <div className="mb-8 flex items-center gap-3">
              <Logo animated className="h-20 w-20 drop-shadow-sm" />
              <span className="text-3xl font-extrabold tracking-tight">
                Edu<span className="text-lavender-600">Trace</span>
              </span>
            </div>
            <h1 className="text-4xl font-extrabold leading-tight tracking-tight">
              Aprende a programar
              <br />
              <span className="text-lavender-600">con guía, paso a paso.</span>
            </h1>
            <p className="mt-3 max-w-sm text-ink-soft">
              Compila tu código, entiende tus errores y descubre qué sigue. La IA te orienta con pistas,
              sin darte la solución, y tu docente sigue siendo quien decide.
            </p>
          </div>
        </div>

        {/* Formulario */}
        <div className="relative rounded-[2rem] bg-white p-7 sm:p-10 md:rounded-l-none">
          <div className="md:hidden mb-6 flex items-center gap-2">
            <Logo animated className="h-10 w-10" />
            <span className="text-xl font-extrabold tracking-tight">
              Edu<span className="text-lavender-600">Trace</span>
            </span>
          </div>

          <div className="rise-in">
            {mode !== "recover" && (
              <>
                <p className="mb-2 text-xs font-bold uppercase tracking-wider text-ink-faint">Entrar como</p>
                <RoleTabs role={role} onChange={chooseRole} />
                <div className="mb-7" />
              </>
            )}
            <h2 className="text-2xl font-extrabold tracking-tight">
              {mode === "login" ? "¡Hola de nuevo!" : mode === "register" ? "Crea tu cuenta" : "Recupera tu cuenta"}
            </h2>
            <p className="mt-1 mb-6 text-sm text-ink-soft">
              {mode === "recover"
                ? "Escribe tu email, el código de recuperación que guardaste (o el que te entregó tu profesor) y una contraseña nueva."
                : SUBTITLES[mode][role]}
            </p>

            <form onSubmit={handleSubmit} className="space-y-3.5">
              {mode === "register" && (
                <input
                  type="text"
                  required
                  minLength={2}
                  maxLength={80}
                  autoComplete="name"
                  placeholder="Nombre completo"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className={inputClass}
                />
              )}
              <input
                type="email"
                required
                autoComplete="email"
                placeholder="Email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className={inputClass}
              />
              {mode === "recover" && (
                <input
                  type="text"
                  required
                  autoComplete="off"
                  spellCheck={false}
                  placeholder="Código (XXXX-XXXX-XXXX)"
                  aria-label="Código de recuperación o código de tu profesor"
                  value={recoveryCode}
                  onChange={(e) => setRecoveryCode(e.target.value.toUpperCase())}
                  className={`${inputClass} font-mono tracking-wider`}
                />
              )}
              <input
                type="password"
                required
                minLength={mode === "login" ? undefined : 8}
                maxLength={128}
                autoComplete={mode === "login" ? "current-password" : "new-password"}
                placeholder={
                  mode === "register" ? "Contraseña (mín. 8 caracteres)" : mode === "recover" ? "Nueva contraseña (mín. 8 caracteres)" : "Contraseña"
                }
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className={inputClass}
              />
              {mode === "recover" && (
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
              )}
              {mode === "login" && (
                <div className="text-right">
                  <button
                    type="button"
                    onClick={() => {
                      setMode("recover");
                      setPassword("");
                      setError(null);
                    }}
                    className="text-xs font-bold text-lavender-600 hover:text-lavender-700"
                  >
                    ¿Olvidaste tu contraseña?
                  </button>
                </div>
              )}
              {mode === "recover" && (
                <p className="text-xs text-ink-faint">
                  El código se te mostró al crear la cuenta (también puedes generar uno nuevo desde «Mi cuenta» cuando entres). Si nunca
                  guardaste uno, pide ayuda al administrador de EduTrace.
                </p>
              )}
              {mode === "register" && role === "teacher" && (
                <input
                  type="password"
                  required
                  autoComplete="off"
                  placeholder="Código de profesor"
                  value={teacherCode}
                  onChange={(e) => setTeacherCode(e.target.value)}
                  className={inputClass}
                />
              )}

              {mode === "register" && (
                <label className="flex items-start gap-2.5 rounded-xl border border-lavender-100 bg-lavender-50/50 p-3 text-[11px] leading-relaxed text-ink-soft">
                  <input
                    type="checkbox"
                    required
                    checked={consent}
                    onChange={(e) => setConsent(e.target.checked)}
                    className="mt-0.5 h-4 w-4 shrink-0 accent-lavender-600"
                  />
                  <span>
                    <span className="font-semibold text-ink">Autorizo el tratamiento de mis datos personales</span> (Ley 1581 de 2012) con
                    fines académicos de la asignatura, según la{" "}
                    <Link href="/privacidad" target="_blank" className="font-bold text-lavender-600 underline">
                      política de tratamiento de datos
                    </Link>
                    . Mis datos y mi código no salen del servidor de EduTrace: la IA funciona en el propio servidor. Puedo consultar,
                    descargar, corregir o eliminar mis datos desde «Mi cuenta».
                  </span>
                </label>
              )}

              {error && (
                <p role="alert" className="rounded-2xl bg-peach-50 px-4 py-2.5 text-sm font-semibold text-peach-700">
                  {error}
                </p>
              )}

              <button
                type="submit"
                disabled={submitting || (mode === "register" && !consent)}
                className="w-full rounded-full bg-lavender-600 py-3 font-bold text-white shadow-soft transition
                  hover:bg-lavender-700 hover:shadow-lift active:scale-[0.99] disabled:opacity-50"
              >
                {submitting ? "Un momento..." : mode === "login" ? "Entrar" : mode === "register" ? "Registrarme" : "Cambiar contraseña y entrar"}
              </button>
            </form>

            <p className="text-center text-sm text-ink-soft mt-6">
              {mode === "login" ? "¿No tienes cuenta? " : mode === "register" ? "¿Ya tienes cuenta? " : "¿La recordaste? "}
              <button
                type="button"
                onClick={() => {
                  setMode(mode === "login" ? "register" : "login");
                  setError(null);
                }}
                className="font-bold text-lavender-600 hover:text-lavender-700"
              >
                {mode === "login" ? "Regístrate" : "Inicia sesión"}
              </button>
            </p>
          </div>
        </div>
      </div>
      {pending && (
        <RecoveryCodeDialog
          code={pending.code}
          email={pending.user.email}
          title={pending.title}
          onDone={() => {
            const u = pending.user;
            setPending(null);
            setUser(u);
          }}
        />
      )}
    </div>
  );
}
