import {
  getGroup,
  getMonitorGroup,
  listCourses,
  listExercises,
  listGroupAssignments,
  listMonitorAssignments,
  listMonitorGroups,
  listMyAssignments,
  listMyGroups,
  Role,
} from "@/lib/api";

export type SearchKind = "Estudiantes" | "Clases" | "Talleres" | "Ejercicios" | "Materias" | "Secciones";

const KIND_ORDER: SearchKind[] = ["Estudiantes", "Clases", "Talleres", "Ejercicios", "Materias", "Secciones"];
const PER_KIND = 5;
const CACHE_MS = 60_000;

export interface SearchItem {
  key: string;
  kind: SearchKind;
  title: string;
  detail: string;
  href: string;
  terms?: string;
}

const section = (title: string, detail: string, href: string, terms = ""): SearchItem => ({
  key: `seccion-${href}`,
  kind: "Secciones",
  title,
  detail,
  href,
  terms,
});

const COMMON_SECTIONS = [
  section("Compilador", "Escribe y compila tu código", "/", "editor programar c++ entrada"),
  section("Mi cuenta", "Contraseña, código de recuperación, avisos y tus datos", "/cuenta", "perfil correo notificaciones"),
  section("Política de tratamiento de datos", "Cómo cuidamos tu información", "/privacidad", "privacidad ley 1581"),
  section("Inicio", "Qué es EduTrace y cómo funciona", "/inicio", "home ayuda"),
];

const STUDY_SECTIONS = [
  section("Mi historial", "Tus entregas, intentos y retroalimentación", "/historial", "revisiones notas"),
  section("Práctica", "Ejercicios para practicar y prepararte para el parcial", "/practica", "banco"),
  section("Mi progreso", "Tus notas y tu avance en cada RAE", "/progreso", "resultados aprendizaje"),
];

const ROLE_SECTIONS: Record<Role, SearchItem[]> = {
  student: STUDY_SECTIONS,
  teacher: [
    section("Mis materias", "Tus materias, clases y resultados de aprendizaje", "/docente", "rae cursos"),
    section("Historial de cursos", "Materias en curso y de periodos anteriores", "/docente/historial", "cerrados"),
    section("Cómo va el curso", "Participación, notas y RAE de toda la materia", "/docente/progreso", "progreso general"),
  ],
  monitor: [section("Mi monitoría", "Las clases que acompañas", "/monitor", "grupos clases"), ...STUDY_SECTIONS],
};

export const sectionItems = (role: Role) => [...ROLE_SECTIONS[role], ...COMMON_SECTIONS];

async function studentItems(): Promise<SearchItem[]> {
  const [groups, assignments, exercises] = await Promise.all([listMyGroups(), listMyAssignments(), listExercises()]);
  return [
    ...assignments.map((a): SearchItem => ({
      key: `taller-${a.id}`,
      kind: "Talleres",
      title: a.title,
      detail: `${a.group_name}, ${a.course_name}`,
      href: `/?clase=${a.group_id}&taller=${a.id}`,
    })),
    ...groups.map((g): SearchItem => ({
      key: `clase-${g.id}`,
      kind: "Clases",
      title: g.name,
      detail: `${g.course_name}, con ${g.teacher_name}`,
      href: `/?clase=${g.id}`,
    })),
    ...exercises.map((e): SearchItem => ({
      key: `ejercicio-${e.id}`,
      kind: "Ejercicios",
      title: e.title,
      detail: e.topic,
      href: `/?ejercicio=${e.id}`,
    })),
  ];
}

async function teacherItems(): Promise<SearchItem[]> {
  const courses = await listCourses();
  const perClass = await Promise.all(
    courses.flatMap((c) =>
      c.groups.map(async (g): Promise<SearchItem[]> => {
        const [detail, assignments] = await Promise.all([getGroup(g.id), listGroupAssignments(g.id)]);
        const place = `${g.name}, ${c.name}`;
        return [
          { key: `clase-${g.id}`, kind: "Clases", title: g.name, detail: c.name, href: `/docente/clases/${g.id}` },
          ...assignments.map((a): SearchItem => ({
            key: `taller-${a.id}`,
            kind: "Talleres",
            title: a.title,
            detail: place,
            href: `/docente/clases/${g.id}#talleres`,
          })),
          ...detail.students.map((s): SearchItem => ({
            key: `estudiante-${g.id}-${s.id}`,
            kind: "Estudiantes",
            title: s.name,
            detail: place,
            href: `/docente/clases/${g.id}?estudiante=${s.id}`,
            terms: s.email,
          })),
        ];
      })
    )
  );
  const subjects = courses.map((c): SearchItem => ({
    key: `materia-${c.id}`,
    kind: "Materias",
    title: c.name,
    detail: c.closed_at ? "Periodo cerrado" : "En curso",
    href: `/docente/progreso?materia=${c.id}`,
  }));
  return [...perClass.flat(), ...subjects];
}

async function monitorItems(): Promise<SearchItem[]> {
  const [own, groups] = await Promise.all([studentItems(), listMonitorGroups()]);
  const perClass = await Promise.all(
    groups.map(async (g): Promise<SearchItem[]> => {
      const [detail, assignments] = await Promise.all([getMonitorGroup(g.id), listMonitorAssignments(g.id)]);
      const place = `Monitoría: ${g.name}, ${g.course_name}`;
      return [
        { key: `clase-${g.id}`, kind: "Clases", title: g.name, detail: `Monitoría: ${g.course_name}, con ${g.teacher_name}`, href: `/monitor/clases/${g.id}` },
        ...assignments.map((a): SearchItem => ({
          key: `taller-${a.id}`,
          kind: "Talleres",
          title: a.title,
          detail: place,
          href: `/monitor/clases/${g.id}`,
        })),
        ...detail.students.map((s): SearchItem => ({
          key: `estudiante-${g.id}-${s.id}`,
          kind: "Estudiantes",
          title: s.name,
          detail: place,
          href: `/monitor/clases/${g.id}?estudiante=${s.id}`,
          terms: s.email,
        })),
      ];
    })
  );
  return [...own, ...perClass.flat()];
}

const LOADERS: Record<Role, () => Promise<SearchItem[]>> = {
  student: studentItems,
  teacher: teacherItems,
  monitor: monitorItems,
};

let cached: { userId: string; at: number; items: SearchItem[] } | null = null;

export async function loadSearchItems(userId: string, role: Role): Promise<SearchItem[]> {
  if (cached && cached.userId === userId && Date.now() - cached.at < CACHE_MS) return cached.items;
  const items = await LOADERS[role]();
  cached = { userId, at: Date.now(), items };
  return items;
}

const clean = (text: string) =>
  text
    .normalize("NFD")
    .replace(/\p{M}/gu, "")
    .toLowerCase();

export function searchItems(items: SearchItem[], query: string): { kind: SearchKind; items: SearchItem[] }[] {
  const words = clean(query).split(/\s+/).filter(Boolean);
  if (words.length === 0) return [];
  const found = items.flatMap((item) => {
    const title = clean(item.title);
    const all = `${title} ${clean(item.detail)} ${clean(item.terms ?? "")}`;
    if (!words.every((w) => all.includes(w))) return [];
    const rank = title.startsWith(words[0]) ? 0 : title.includes(` ${words[0]}`) ? 1 : title.includes(words[0]) ? 2 : 3;
    return [{ item, rank }];
  });
  found.sort((a, b) => a.rank - b.rank || a.item.title.localeCompare(b.item.title, "es"));
  return KIND_ORDER.flatMap((kind) => {
    const inKind = found.filter((f) => f.item.kind === kind).slice(0, PER_KIND);
    return inKind.length ? [{ kind, best: inKind[0].rank, items: inKind.map((f) => f.item) }] : [];
  }).sort((a, b) => a.best - b.best);
}
