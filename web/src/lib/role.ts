import { Role } from "@/lib/api";

// Último rol elegido para entrar (lo recuerda el login y lo fija "Cambiar de rol").
const ROLE_KEY = "edutrace_role";

export function isRole(value: string | null): value is Role {
  return value === "student" || value === "teacher" || value === "monitor";
}

export function loadRole(): Role {
  try {
    const saved = localStorage.getItem(ROLE_KEY);
    return isRole(saved) ? saved : "student";
  } catch {
    return "student";
  }
}

export function homeFor(role: Role): string {
  if (role === "teacher") return "/docente";
  if (role === "monitor") return "/monitor";
  return "/";
}

export const ROLE_LABELS: Record<Role, string> = {
  student: "Estudiante",
  teacher: "Profesor",
  monitor: "Monitor",
};

export function saveRole(role: Role) {
  try {
    localStorage.setItem(ROLE_KEY, role);
  } catch {
    /* almacenamiento bloqueado: no pasa nada */
  }
}
