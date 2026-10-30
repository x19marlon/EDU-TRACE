"use client";

import { useState } from "react";

export default function RecoveryCodeDialog({
  code,
  email,
  title = "Guarda tu código de recuperación",
  onDone,
}: {
  code: string;
  email: string;
  title?: string;
  onDone: () => void;
}) {
  const [saved, setSaved] = useState(false);
  const [copied, setCopied] = useState(false);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(code);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  };

  const download = () => {
    const text = `EduTrace · Código de recuperación\n\nCuenta: ${email}\nCódigo: ${code}\n\nÚsalo en «¿Olvidaste tu contraseña?» si no recuerdas tu contraseña. Sirve una sola vez: al usarlo recibirás uno nuevo.\n`;
    const url = URL.createObjectURL(new Blob([text], { type: "text/plain;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = "edutrace-codigo-de-recuperacion.txt";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="recovery-title" className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4">
      <div className="w-full max-w-md space-y-4 rounded-3xl bg-white p-6 shadow-lift">
        <h2 id="recovery-title" className="text-xl font-extrabold tracking-tight">
          {title}
        </h2>
        <p className="text-sm text-ink-soft">
          Si olvidas tu contraseña, con este código y tu email puedes crear una nueva desde «¿Olvidaste tu contraseña?». Guárdalo en
          un lugar seguro: <span className="font-semibold text-ink">no se vuelve a mostrar</span> y sirve una sola vez.
        </p>
        <p className="rounded-2xl bg-lavender-50 py-4 text-center font-mono text-2xl font-extrabold tracking-[0.2em] text-lavender-700">
          {code}
        </p>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            onClick={copy}
            className="rounded-full bg-lavender-100 px-4 py-2 text-sm font-bold text-lavender-700 transition hover:bg-lavender-200"
          >
            {copied ? "¡Copiado!" : "Copiar"}
          </button>
          <button
            type="button"
            onClick={download}
            className="rounded-full bg-lavender-100 px-4 py-2 text-sm font-bold text-lavender-700 transition hover:bg-lavender-200"
          >
            Descargar .txt
          </button>
        </div>
        <label className="flex items-center gap-2 text-sm font-semibold text-ink">
          <input type="checkbox" checked={saved} onChange={(e) => setSaved(e.target.checked)} className="h-4 w-4 accent-lavender-600" />
          Ya lo guardé en un lugar seguro
        </label>
        <button
          type="button"
          disabled={!saved}
          onClick={onDone}
          className="w-full rounded-full bg-lavender-600 py-3 font-bold text-white shadow-soft transition hover:bg-lavender-700 disabled:opacity-40"
        >
          Continuar
        </button>
      </div>
    </div>
  );
}
