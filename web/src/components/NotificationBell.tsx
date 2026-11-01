"use client";

import { useCallback, useEffect, useId, useRef, useState } from "react";
import Link from "next/link";
import { useAuth } from "@/components/AuthProvider";
import { AppNotification, getNotifications, markNotificationsRead } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

const POLL_MS = 60_000;

export default function NotificationBell() {
  const { user } = useAuth();
  const [items, setItems] = useState<AppNotification[]>([]);
  const [unread, setUnread] = useState(0);
  const [open, setOpen] = useState(false);
  const [fresh, setFresh] = useState<Set<number>>(new Set());
  const ref = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);
  const panelId = useId();

  const load = useCallback(() => {
    getNotifications()
      .then((list) => {
        setItems(list.items);
        setUnread(list.unread);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (!user) return;
    load();
    const timer = setInterval(() => document.visibilityState === "visible" && load(), POLL_MS);
    window.addEventListener("focus", load);
    return () => {
      clearInterval(timer);
      window.removeEventListener("focus", load);
    };
  }, [user, load]);

  useEffect(() => {
    if (!open) return;
    const onClick = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      setOpen(false);
      buttonRef.current?.focus();
    };
    document.addEventListener("mousedown", onClick);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onClick);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  if (!user) return null;

  const toggle = () => {
    const next = !open;
    setOpen(next);
    if (!next || unread === 0) return;
    const ids = items.filter((n) => !n.read_at).map((n) => n.id);
    setFresh(new Set(ids));
    setUnread(0);
    markNotificationsRead(ids).catch(load);
  };

  const label = unread > 0 ? `Avisos: ${unread} sin leer` : "Avisos";

  return (
    <div ref={ref} className="relative">
      <button
        ref={buttonRef}
        onClick={toggle}
        aria-expanded={open}
        aria-controls={panelId}
        aria-label={label}
        className="relative flex h-10 w-10 items-center justify-center rounded-full bg-white text-ink-soft shadow-soft transition-colors hover:bg-lavender-50 hover:text-ink"
      >
        <svg className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth={1.8} viewBox="0 0 24 24" aria-hidden>
          <path strokeLinecap="round" strokeLinejoin="round" d="M14.857 17.082a23.848 23.848 0 0 0 5.454-1.31A8.967 8.967 0 0 1 18 9.75V9A6 6 0 0 0 6 9v.75a8.967 8.967 0 0 1-2.312 6.022c1.733.64 3.56 1.085 5.455 1.31m5.714 0a24.255 24.255 0 0 1-5.714 0m5.714 0a3 3 0 1 1-5.714 0" />
        </svg>
        {unread > 0 && (
          <span aria-hidden className="absolute -right-0.5 -top-0.5 flex h-5 min-w-5 items-center justify-center rounded-full bg-peach-700 px-1 text-[11px] font-bold text-white">
            {unread > 9 ? "9+" : unread}
          </span>
        )}
      </button>
      <span className="sr-only" aria-live="polite">{unread > 0 ? `Tienes ${unread} avisos sin leer` : ""}</span>

      {open && (
        <div
          id={panelId}
          className="fixed inset-x-4 top-20 z-30 rounded-2xl border border-lavender-100 bg-white p-2 shadow-lift sm:absolute sm:inset-x-auto sm:right-0 sm:top-auto sm:mt-2 sm:w-80"
        >
          <h2 className="px-2 pb-1 pt-1.5 text-sm font-extrabold text-ink">Avisos</h2>
          {items.length === 0 ? (
            <p className="px-2 py-3 text-sm text-ink-soft">No tienes avisos.</p>
          ) : (
            <ul className="max-h-96 space-y-1 overflow-y-auto">
              {items.map((n) => {
                const isNew = fresh.has(n.id) || !n.read_at;
                const body = (
                  <>
                    <span className="flex items-start gap-2">
                      {isNew && <span aria-hidden className="mt-1.5 h-2 w-2 shrink-0 rounded-full bg-lavender-600" />}
                      <span className={isNew ? "font-semibold text-ink" : "text-ink-soft"}>
                        {isNew && <span className="sr-only">Nuevo: </span>}
                        {n.title}
                      </span>
                    </span>
                    <span className="mt-0.5 block text-xs text-ink-soft">{formatDateTime(n.created_at)}</span>
                  </>
                );
                return (
                  <li key={n.id}>
                    {n.link ? (
                      <Link href={n.link} onClick={() => setOpen(false)} className="block rounded-xl px-2 py-2 text-sm hover:bg-lavender-50">
                        {body}
                      </Link>
                    ) : (
                      <div className="rounded-xl px-2 py-2 text-sm">{body}</div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          <Link href="/cuenta#avisos" onClick={() => setOpen(false)} className="mt-1 block rounded-xl px-2 py-2 text-xs font-bold text-lavender-700 hover:bg-lavender-50">
            Avisos por correo
          </Link>
        </div>
      )}
    </div>
  );
}
