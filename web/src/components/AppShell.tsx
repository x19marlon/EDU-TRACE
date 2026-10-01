"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import Logo, { Wordmark } from "@/components/Logo";
import UserMenu from "@/components/UserMenu";
import { useAuth } from "@/components/AuthProvider";

interface NavItem {
  href: string;
  label: string;
  tabLabel?: string;
  icon: React.ReactNode;
  teacherOnly?: boolean;
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
    href: "/docente",
    label: "Materias",
    tabLabel: "Mis materias",
    teacherOnly: true,
    icon: (
      <svg {...iconProps}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 0 1 3 19.875v-6.75ZM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V8.625ZM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 0 1-1.125-1.125V4.125Z" />
      </svg>
    ),
  },
];

export default function AppShell({ children }: { children: React.ReactNode }) {
  const { user } = useAuth();
  const pathname = usePathname();
  const items = NAV.filter((n) => !n.teacherOnly || user?.role === "teacher");
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
          <Link href="/" className="md:hidden flex items-center gap-2">
            <Logo className="h-8 w-8" />
          </Link>
          <nav className="flex items-center gap-1 rounded-full bg-white p-1 shadow-soft">
            {items.map((item) => {
              const active = isActive(item.href);
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  className={`rounded-full px-4 py-1.5 text-sm font-semibold transition-colors ${
                    active ? "bg-lavender-100 text-lavender-700" : "text-ink-soft hover:text-ink"
                  }`}
                >
                  {item.tabLabel ?? item.label}
                </Link>
              );
            })}
          </nav>
          <UserMenu />
        </header>

        <main className="relative z-10 flex-1 px-4 sm:px-6 pb-6">{children}</main>
      </div>
    </div>
  );
}
