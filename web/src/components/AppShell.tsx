"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import Logo, { Wordmark } from "@/components/Logo";
import UserMenu from "@/components/UserMenu";
import NotificationBell from "@/components/NotificationBell";
import { useAuth } from "@/components/AuthProvider";
import { Role } from "@/lib/api";

interface NavItem {
  href: string;
  label: string;
  tabLabel?: string;
  icon: React.ReactNode;
  roles?: Role[];
}

const iconProps = {
  className: "h-5 w-5",
  fill: "none",
  stroke: "currentColor",
  strokeWidth: 1.8,
  viewBox: "0 0 24 24",
  "aria-hidden": true,
} as const;

const NAV: NavItem[] = [
  {
    href: "/",
    label: "Compilador",
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M17.25 6.75 22.5 12l-5.25 5.25m-10.5 0L1.5 12l5.25-5.25m7.5-3-4.5 16.5" />
      </svg>
    ),
  },
  {
    href: "/historial",
    label: "Historial",
    tabLabel: "Mi historial",
    roles: ["student"],
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6h4.5m4.5 0a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
      </svg>
    ),
  },
  {
    href: "/practica",
    label: "Práctica",
    roles: ["student"],
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M4.26 10.147a60.438 60.438 0 0 0-.491 6.347A48.62 48.62 0 0 1 12 20.904a48.62 48.62 0 0 1 8.232-4.41 60.46 60.46 0 0 0-.491-6.347m-15.482 0a50.636 50.636 0 0 0-2.658-.813A59.906 59.906 0 0 1 12 3.493a59.903 59.903 0 0 1 10.399 5.84c-.896.248-1.783.52-2.658.814m-15.482 0A50.717 50.717 0 0 1 12 13.489a50.702 50.702 0 0 1 7.74-3.342M6.75 15a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Zm0 0v-3.675A55.378 55.378 0 0 1 12 8.443m-7.007 11.55A5.981 5.981 0 0 0 6.75 15.75v-1.5" />
      </svg>
    ),
  },
  {
    href: "/progreso",
    label: "Progreso",
    tabLabel: "Mi progreso",
    roles: ["student"],
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M2.25 18 9 11.25l4.306 4.306a11.95 11.95 0 0 1 5.814-5.518l2.74-1.22m0 0-5.94-2.281m5.94 2.28-2.28 5.941" />
      </svg>
    ),
  },
  {
    href: "/docente",
    label: "Materias",
    tabLabel: "Mis materias",
    roles: ["teacher"],
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />
      </svg>
    ),
  },
  {
    href: "/monitor",
    label: "Clases",
    tabLabel: "Mis clases",
    roles: ["monitor"],
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M7.5 8.25h9m-9 3H12m-9.75 1.51c0 1.6 1.123 2.994 2.707 3.227 1.129.166 2.27.293 3.423.379.35.026.67.21.865.501L12 21l2.755-4.133a1.14 1.14 0 0 1 .865-.501 48.172 48.172 0 0 0 3.423-.379c1.584-.233 2.707-1.626 2.707-3.228V6.741c0-1.602-1.123-2.995-2.707-3.228A48.394 48.394 0 0 0 12 3c-2.392 0-4.744.175-7.043.513C3.373 3.746 2.25 5.14 2.25 6.741v6.018Z" />
      </svg>
    ),
  },
];

export default function AppShell({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const pathname = usePathname();
  const items = NAV.filter((n) => !n.roles || (user && n.roles.includes(user.role)));
  const isActive = (href: string) => (href === "/" ? pathname === "/" : pathname.startsWith(href));

  return (
    <div className="min-h-screen flex">
      {/* Barra lateral */}
      <aside className="hidden md:flex sticky top-0 h-screen w-24 shrink-0 flex-col items-center gap-1 bg-white border-r border-lavender-100 py-5 z-10">
        <Link href="/" className="flex flex-col items-center gap-1 mb-6">
          <Logo className="h-10 w-10" />
          <Wordmark />
        </Link>
        <nav className="flex flex-col gap-2 w-full px-3">
          {items.map((item) => {
            const active = isActive(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={`flex flex-col items-center gap-1 rounded-2xl py-2.5 text-[11px] font-semibold transition-colors ${
                  active
                    ? "bg-lavender-100 text-lavender-700"
                    : "text-ink-soft hover:bg-lavender-50 hover:text-lavender-700"
                }`}
              >
                {item.icon}
                {item.label}
              </Link>
            );
          })}
        </nav>
      </aside>

      <div className="relative flex-1 min-w-0 flex flex-col overflow-hidden">
        {/* Fondo pastel: solo detrás de los márgenes, nunca a través del contenido */}
        <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
          <div className="absolute inset-x-0 top-0 h-56 bg-lavender-100/80" />
        </div>

        {/* Barra superior */}
        <header className="relative z-30 flex items-center justify-between gap-3 px-4 sm:px-6 py-4">
          <Link href="/" aria-label="EduTrace, inicio" className="hidden min-[440px]:flex md:hidden items-center gap-2">
            <Logo className="h-8 w-8" />
          </Link>
          <nav aria-label="Secciones" className="flex min-w-0 items-center gap-1 rounded-full bg-white p-1 shadow-soft">
            {items.map((item) => {
              const active = isActive(item.href);
              const compact = items.length > 2;
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  aria-current={active ? "page" : undefined}
                  className={`flex items-center gap-1.5 whitespace-nowrap rounded-full px-3 py-1.5 text-sm font-semibold transition-colors lg:px-4 ${
                    active ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {compact && <span className="md:hidden">{item.icon}</span>}
                  <span className={compact ? "sr-only md:not-sr-only" : ""}>{item.tabLabel ?? item.label}</span>
                </Link>
              );
            })}
          </nav>
          <div className="flex items-center gap-2">
            <NotificationBell />
            <UserMenu />
          </div>
        </header>

        <main className="relative z-10 flex-1 px-4 sm:px-6 pb-6">{children}</main>
      </div>
    </div>
  );
}
