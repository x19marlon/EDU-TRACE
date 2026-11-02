"use client";

import { useEffect, useId, useMemo, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";
import { Spinner } from "@/components/ui";
import { Role } from "@/lib/api";
import { loadSearchItems, searchItems, sectionItems, SearchItem } from "@/lib/search";

const PLACEHOLDER: Record<Role, string> = {
  student: "Buscar talleres, ejercicios, clases...",
  teacher: "Buscar estudiantes, clases, talleres...",
  monitor: "Buscar estudiantes, talleres, ejercicios...",
};

export default function SearchBar() {
  const { user } = useAuth();
  const router = useRouter();
  const listId = useId();
  const box = useRef<HTMLDivElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);
  const [items, setItems] = useState<SearchItem[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        input.current?.focus();
        input.current?.select();
      }
    };
    const onClick = (e: MouseEvent) => {
      if (!box.current?.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("keydown", onKey);
    document.addEventListener("mousedown", onClick);
    return () => {
      document.removeEventListener("keydown", onKey);
      document.removeEventListener("mousedown", onClick);
    };
  }, []);

  const groups = useMemo(() => (user ? searchItems([...(items ?? []), ...sectionItems(user.role)], query) : []), [user, items, query]);
  const results = useMemo(() => groups.flatMap((g) => g.items), [groups]);
  const typed = query.trim().length > 0;
  const showList = open && results.length > 0;
  const optionId = (i: number) => `${listId}-opcion-${i}`;

  if (!user) return null;

  const load = async () => {
    setFailed(false);
    try {
      setItems(await loadSearchItems(user.id, user.role));
    } catch {
      setItems([]);
      setFailed(true);
    }
  };

  const leave = () => {
    setQuery("");
    setOpen(false);
    setActive(0);
    input.current?.blur();
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Escape") {
      if (open && typed) setOpen(false);
      else input.current?.blur();
    } else if (showList && e.key === "ArrowDown") {
      e.preventDefault();
      setActive((i) => (i + 1) % results.length);
    } else if (showList && e.key === "ArrowUp") {
      e.preventDefault();
      setActive((i) => (i - 1 + results.length) % results.length);
    } else if (showList && e.key === "Enter") {
      e.preventDefault();
      router.push(results[Math.min(active, results.length - 1)].href);
      leave();
    }
  };

  return (
    <div ref={box} role="search" className="relative">
      <svg className="pointer-events-none absolute left-3.5 top-1/2 h-4 w-4 -translate-y-1/2 text-ink-faint" fill="none" stroke="currentColor" strokeWidth={2} viewBox="0 0 24 24" aria-hidden>
        <path strokeLinecap="round" strokeLinejoin="round" d="m21 21-4.35-4.35M17 10.5a6.5 6.5 0 1 1-13 0 6.5 6.5 0 0 1 13 0Z" />
      </svg>
      <input
        ref={input}
        type="search"
        role="combobox"
        aria-label="Buscar en EduTrace"
        aria-autocomplete="list"
        aria-expanded={showList}
        aria-controls={listId}
        aria-activedescendant={showList ? optionId(Math.min(active, results.length - 1)) : undefined}
        autoComplete="off"
        value={query}
        placeholder={PLACEHOLDER[user.role]}
        onChange={(e) => {
          setQuery(e.target.value);
          setOpen(true);
          setActive(0);
        }}
        onFocus={() => {
          setOpen(true);
          load();
        }}
        onKeyDown={onKeyDown}
        className="w-full rounded-full border-0 bg-white py-2 pl-10 pr-4 text-sm text-ink shadow-soft placeholder-ink-faint focus:outline-none focus:ring-4 focus:ring-lavender-200"
      />
      <span role="status" className="sr-only">
        {typed && open ? `${results.length} ${results.length === 1 ? "resultado" : "resultados"}` : ""}
      </span>
      {showList && (
        <div id={listId} role="listbox" aria-label="Resultados de la búsqueda" className="absolute inset-x-0 top-full z-40 mt-2 max-h-[70vh] overflow-auto rounded-2xl bg-white p-1.5 shadow-lift ring-1 ring-lavender-100">
          {groups.map((group) => (
            <div key={group.kind} role="group" aria-labelledby={`${listId}-${group.kind}`}>
              <div id={`${listId}-${group.kind}`} className="px-3 pb-1 pt-2 text-[11px] font-bold uppercase tracking-wider text-ink-faint">
                {group.kind}
              </div>
              {group.items.map((item) => {
                const index = results.indexOf(item);
                return (
                  <Link
                    key={item.key}
                    id={optionId(index)}
                    role="option"
                    aria-selected={index === Math.min(active, results.length - 1)}
                    href={item.href}
                    onMouseEnter={() => setActive(index)}
                    onClick={leave}
                    className={`block rounded-xl px-3 py-2 ${index === Math.min(active, results.length - 1) ? "bg-lavender-100" : ""}`}
                  >
                    <span className="block truncate text-sm font-bold text-ink">{item.title}</span>
                    <span className="block truncate text-xs text-ink-soft">{item.detail}</span>
                  </Link>
                );
              })}
            </div>
          ))}
          {failed && <p className="px-3 py-2 text-xs text-peach-700">Por ahora solo pudimos buscar en las secciones. Inténtalo de nuevo en un momento.</p>}
        </div>
      )}
      {open && typed && results.length === 0 && (
        <div className="absolute inset-x-0 top-full z-40 mt-2 rounded-2xl bg-white p-4 text-sm text-ink-soft shadow-lift ring-1 ring-lavender-100">
          {items === null ? (
            <span className="flex items-center gap-2">
              <Spinner className="h-4 w-4 border-lavender-500" /> Buscando...
            </span>
          ) : (
            <>No encontramos nada con «{query.trim()}». Prueba con otra palabra.</>
          )}
        </div>
      )}
    </div>
  );
}
