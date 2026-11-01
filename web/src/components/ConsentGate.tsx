"use client";

import { useState } from "react";
import Link from "next/link";
import Logo from "@/components/Logo";
import { useAuth } from "@/components/AuthProvider";
import { recordConsent } from "@/lib/api";

export default function ConsentGate() {
  const { user, setUser, logout } = useAuth();
  const updated = !!user?.consent_outdated;
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const accept = async () => {
    setBusy(true);
    setError(null);
    try {
      setUser(await recordConsent());
    } catch (err) {
      setError(err instanceof Error ? err.message : "No se pudo registrar tu autorización");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div role="dialog" aria-modal="true" aria-labelledby="consent-title" className="w-full max-w-lg space-y-4 rounded-3xl bg-white p-6 shadow-lift sm:p-8">
        <div className="flex items-center gap-3">
          <Logo className="h-10 w-10" />
          <h1 id="consent-title" className="text-xl font-extrabold tracking-tight">
            {updated ? "Actualizamos la política de datos" : "Autorización de tratamiento de datos"}
          </h1>
        </div>
        {updated && (
          <p className="rounded-2xl bg-lavender-50 px-4 py-3 text-sm leading-relaxed text-ink">
            Qué cambió: EduTrace ahora te avisa en la plataforma y, si lo permites, por correo cuando hay novedades en tus entregas; y
            al cerrar el periodo de una materia sus datos se borran automáticamente después del plazo de conservación. Puedes desactivar
            los correos en «Mi cuenta».
          </p>
        )}
        <p className="text-sm leading-relaxed text-ink-soft">
          Para seguir usando EduTrace necesitamos que autorices el tratamiento de tus datos personales (Ley 1581 de 2012) con fines
          académicos de la asignatura. Tus datos y tu código no salen del servidor: la IA funciona en el propio servidor y nunca recibe tu
          nombre ni tu email.
        </p>
        <p className="text-sm leading-relaxed text-ink-soft">
          Puedes descargar, corregir o eliminar tus datos cuando quieras desde «Mi cuenta». Lee la{" "}
          <Link href="/privacidad" target="_blank" className="font-bold text-lavender-600 underline">
            política de tratamiento de datos
          </Link>{" "}
          completa.
        </p>
        {error && <p className="rounded-2xl bg-peach-50 px-4 py-2 text-sm font-semibold text-peach-700">{error}</p>}
        <div className="flex flex-wrap gap-2">
          <button
            onClick={accept}
            disabled={busy}
            className="rounded-full bg-lavender-600 px-5 py-2.5 text-sm font-bold text-white shadow-soft transition hover:bg-lavender-700 disabled:opacity-50"
          >
            {busy ? "Guardando..." : "Autorizo y continúo"}
          </button>
          <button
            onClick={() => logout()}
            className="rounded-full px-4 py-2.5 text-sm font-bold text-ink-soft transition hover:bg-lavender-50"
          >
            No autorizo, cerrar sesión
          </button>
        </div>
      </div>
    </div>
  );
}
