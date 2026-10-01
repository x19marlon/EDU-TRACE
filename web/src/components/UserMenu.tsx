"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";
import { Avatar } from "@/components/ui";
import { Role } from "@/lib/api";
import { saveRole } from "@/lib/role";

export default function UserMenu() {
  const { user, logout } = useAuth();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  if (!user) return null;

  const handleLogout = async () => {
    setOpen(false);
    await logout();
    router.replace("/login");
  };

  // Cada cuenta tiene un solo rol: cambiar de rol es entrar con la otra cuenta.
  const switchRole = async (role: Role) => {
    setOpen(false);
    if (role === user.role) return;
    saveRole(role); // el login lo lee aunque la redirección llegue sin ?rol=
    await logout();
    router.replace(`/login?rol=${role}`);
  };

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={`Cuenta de ${user.name}`}
        className="flex items-center gap-2 rounded-full bg-white py-1 pl-1 pr-3 shadow-soft
          hover:bg-lavender-50 transition-colors"
      >
        <Avatar name={user.name} className="h-8 w-8 text-xs" />
        <span className="hidden sm:block text-sm font-semibold text-ink max-w-[10rem] truncate">
          {user.name.split(" ")[0]}
        </span>
        <svg className="h-4 w-4 text-ink-faint" viewBox="0 0 20 20" fill="currentColor" aria-hidden>
          <path fillRule="evenodd" d="M5.23 7.21a.75.75 0 0 1 1.06.02L10 11.17l3.71-3.94a.75.75 0 1 1 1.08 1.04l-4.25 4.5a.75.75 0 0 1-1.08 0l-4.25-4.5a.75.75 0 0 1 .02-1.06Z" clipRule="evenodd" />
        </svg>
      </button>

      {open && (
        <div
          role="menu"
          className="absolute right-0 mt-2 w-64 bg-white border border-lavender-100 rounded-2xl shadow-lift z-30 p-1.5"
        >
          <div className="flex items-center gap-3 px-3 py-2.5">
            <Avatar name={user.name} className="h-10 w-10 text-sm" />
            <div className="min-w-0">
              <p className="text-sm font-bold text-ink truncate">{user.name}</p>
              <p className="text-xs text-ink-soft truncate">{user.email}</p>
            </div>
          </div>
          <div className="border-t border-lavender-100 my-1" />
          <p className="px-3 pt-1.5 pb-1 text-[11px] font-bold uppercase tracking-wider text-ink-faint">Rol</p>
          {(["student", "teacher"] as const).map((r) => (
            <button
              key={r}
              role="menuitemradio"
              aria-checked={user.role === r}
              onClick={() => switchRole(r)}
              className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-sm font-semibold transition-colors ${
                user.role === r ? "bg-lavender-50 text-lavender-700" : "text-ink-soft hover:bg-lavender-50 hover:text-ink"
              }`}
            >
              <span>
                {r === "teacher" ? "Profesor" : "Estudiante"}
                {user.role !== r && <span className="ml-1 text-xs font-normal text-ink-faint">(entrar con esa cuenta)</span>}
              </span>
              {user.role === r && <span aria-hidden>✓</span>}
            </button>
          ))}
          <div className="border-t border-lavender-100 my-1" />
          <button
            role="menuitem"
            onClick={handleLogout}
            className="w-full text-left rounded-xl px-3 py-2 text-sm font-semibold text-ink-soft hover:bg-peach-50 hover:text-peach-700"
          >
            Cerrar sesión
          </button>
        </div>
      )}
    </div>
  );
}
