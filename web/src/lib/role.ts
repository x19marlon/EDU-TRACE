import { Role } from "@/lib/api";

// Último rol elegido para entrar (lo recuerda el login y lo fija "Cambiar de rol").
const ROLE_KEY = "edutrace_role";

export function loadRole(): Role {
  try {
    return localStorage.getItem(ROLE_KEY) === "teacher" ? "teacher" : "student";
  } catch {
    return "student";
  }
}

export function saveRole(role: Role) {
  try {
    localStorage.setItem(ROLE_KEY, role);
  } catch {
    /* almacenamiento bloqueado: no pasa nada */
  }
}
