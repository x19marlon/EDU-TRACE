"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/components/AuthProvider";
import { Role } from "@/lib/api";

/**
 * Muestra el contenido solo con sesión iniciada (y el rol indicado, si se pasa).
 * La protección real la hace el backend; esto solo evita mostrar pantallas vacías.
 */
export default function RequireAuth({
  role,
  children,
}: {
  role?: Role;
  children: React.ReactNode;
}) {
  const { user, isLoading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;
    if (!user) router.replace("/login");
    else if (role && user.role !== role) router.replace("/");
  }, [user, isLoading, role, router]);

  if (isLoading || !user || (role && user.role !== role)) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-[3px] border-lavender-200 border-t-lavender-500" />
      </div>
    );
  }

  return <>{children}</>;
}
