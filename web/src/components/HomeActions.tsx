"use client";

import Link from "next/link";
import { useAuth } from "@/components/AuthProvider";
import { homeFor } from "@/lib/role";

const PRIMARY = "rounded-full bg-lavender-600 font-bold text-white shadow-soft transition hover:bg-lavender-700";
const SECONDARY = "rounded-full bg-white font-bold text-lavender-700 shadow-soft transition hover:bg-lavender-50";

export default function HomeActions({ large = false }: { large?: boolean }) {
  const { user } = useAuth();
  const size = large ? "px-6 py-3" : "px-4 py-2 text-sm";
  if (user) {
    return (
      <Link href={homeFor(user.role)} className={`${PRIMARY} ${size}`}>
        Ir a mi espacio
      </Link>
    );
  }
  return (
    <>
      <Link href="/login?modo=registro" className={`${PRIMARY} ${size}`}>
        Crear mi cuenta
      </Link>
      <Link href="/login" className={`${SECONDARY} ${size}`}>
        Ya tengo cuenta
      </Link>
    </>
  );
}
