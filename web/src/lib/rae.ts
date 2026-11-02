import { RAELevel, StudentPoint } from "@/lib/api";
import { LevelSegment } from "@/components/LevelBar";

export const OFFICIAL_RAES = [
  {
    code: "RAE 1",
    description:
      "Identificar información que permite caracterizar un problema informático de complejidad baja, mediante el uso de los principios básicos de Lógica e Ingeniería de Sistemas.",
  },
  {
    code: "RAE 2",
    description:
      "Conocer las construcciones básicas de programación para diseñar soluciones a problemas informáticos de complejidad baja, mediante algoritmos, utilizando el pensamiento computacional.",
  },
  {
    code: "RAE 3",
    description:
      "Conocer los procedimientos para diseñar la organización de los datos requeridos para solucionar el problema utilizando colecciones de datos.",
  },
  {
    code: "RAE 4",
    description:
      "Determinar las herramientas necesarias para implementar, en un lenguaje de programación, los algoritmos diseñados, permitiendo su modificación.",
  },
];

export function gradeLevel(grade: number): RAELevel {
  return grade < 3 ? "bajo" : grade < 3.75 ? "medio" : "alto";
}

export const LEVEL_INFO: Record<RAELevel, { label: string; range: string; meaning: string }> = {
  bajo: { label: "Bajo", range: "Normalmente, nota menor a 3.0", meaning: "Requiere acompañamiento" },
  medio: { label: "Medio", range: "Normalmente, nota de 3.0 a 3.74", meaning: "Va en camino" },
  alto: { label: "Alto", range: "Normalmente, nota de 3.75 a 5.0", meaning: "Logro consolidado" },
};

const LEVEL_MARGIN = 0.5;
const LEVEL_FLOOR: Record<RAELevel, number> = { bajo: 0, medio: 3, alto: 3.75 };
const LEVEL_CEILING: Record<RAELevel, number> = { bajo: 3, medio: 3.75, alto: 5 };

export type GradeNudge = "up" | "down";

export function gradeNudge(grade: number, level: RAELevel): GradeNudge | null {
  if (gradeLevel(grade) === level) return null;
  return grade >= LEVEL_CEILING[level] ? "up" : "down";
}

export function gradeFarFromLevel(grade: number, level: RAELevel): boolean {
  const nudge = gradeNudge(grade, level);
  if (!nudge) return false;
  return (nudge === "up" ? grade - LEVEL_CEILING[level] : LEVEL_FLOOR[level] - grade) >= LEVEL_MARGIN;
}

export const NUDGE_NOTE: Record<RAELevel, Partial<Record<GradeNudge, string>>> = {
  bajo: { up: "Nota algo más alta de lo habitual en este nivel: podría estar saliendo adelante." },
  medio: {
    up: "Nota algo más alta de lo habitual en este nivel: podría estar cerca del nivel alto.",
    down: "Nota algo más baja de lo habitual en este nivel: podría necesitar refuerzo.",
  },
  alto: { down: "Nota algo más baja de lo habitual en este nivel: conviene ver si se mantiene." },
};

export function raeReading(level: RAELevel, grade: number): string {
  if (gradeFarFromLevel(grade, level)) return "La nota y el nivel se alejan: conviene revisarlos.";
  const nudge = gradeNudge(grade, level);
  return nudge ? NUDGE_NOTE[level][nudge] ?? LEVEL_INFO[level].meaning : LEVEL_INFO[level].meaning;
}

export type RaeBucket = RAELevel | "revisar" | "sin_evaluar";

export function raeBucket(student: StudentPoint, raeId: string): RaeBucket {
  const standing = student.standings[raeId];
  if (!standing) return "sin_evaluar";
  return gradeFarFromLevel(standing.grade, standing.level) ? "revisar" : standing.level;
}

export const LEVEL_COLOR: Record<RaeBucket, string> = {
  bajo: "bg-peach-600",
  medio: "bg-amber-500",
  alto: "bg-mint-500",
  revisar: "bg-lavender-600",
  sin_evaluar: "bg-lavender-100",
};

const BUCKET_LABEL: Record<RaeBucket, string> = {
  bajo: "Bajo",
  medio: "Medio",
  alto: "Alto",
  revisar: "Para revisar",
  sin_evaluar: "Sin evaluar",
};

export function raeSegments(buckets: RaeBucket[]): LevelSegment[] {
  return (["bajo", "medio", "alto", "revisar", "sin_evaluar"] as const).map((key) => ({
    key,
    label: BUCKET_LABEL[key],
    color: LEVEL_COLOR[key],
    count: buckets.filter((b) => b === key).length,
  }));
}
