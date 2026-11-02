"use client";

import { useEffect, useRef } from "react";
import { usePathname, useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";
import { Role } from "@/lib/api";
import { homeFor } from "@/lib/role";
import ConsentGate from "@/components/ConsentGate";

const roleAllowed = (wanted: Role | Role[] | undefined, role: Role) =>
  !wanted || (Array.isArray(wanted) ? wanted.includes(role) : wanted === role);

/**
 * Muestra el contenido solo con sesión iniciada (y el rol indicado, si se pasa).
 * La protección real la hace el backend; esto solo evita mostrar pantallas vacías.
 */
export default function RequireAuth({
  role,
  children,
}: {
  role?: Role | Role[];
  children: React.ReactNode;
}) {
  const { user, isLoading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const hadSession = useRef(false);
  if (user) hadSession.current = true;

  useEffect(() => {
    if (isLoading) return;
    if (!user) router.replace(pathname === "/" && !hadSession.current ? "/inicio" : "/login");
    else if (!roleAllowed(role, user.role)) router.replace(homeFor(user.role));
  }, [user, isLoading, role, router, pathname]);

  if (isLoading || !user || !roleAllowed(role, user.role)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-[3px] border-lavender-200 border-t-lavender-500" />
      </div>
    );
  }

  if (!user.consent_at) return <ConsentGate />;

  return <>{children}</>;
}
