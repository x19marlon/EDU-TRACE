import { AuditEntry } from "@/lib/api";
import { formatDateTime } from "@/lib/format";

const MODE_NAME: Record<string, string> = {
  supervisada: "Modo supervisado",
  inmediata: "Modo inmediato",
};

const lower = (text: string) => text.charAt(0).toLowerCase() + text.slice(1);
const upper = (text: string) => text.charAt(0).toUpperCase() + text.slice(1);
const finished = (text: string) => (/[.:]$/.test(text) ? text : `${text}.`);

const place = (e: AuditEntry) => `${e.student_name ? ` de ${e.student_name}` : ""}${e.assignment_title ? ` en «${e.assignment_title}»` : ""}`;
const told = (e: AuditEntry) => (e.detail ? `${e.actor_name} ${lower(e.detail)}` : e.actor_name);

const ACTIONS: Record<string, { title: string; sentence: (e: AuditEntry) => string; detailIsSentence?: boolean }> = {
  propuesta_ia: { title: "Propuesta de la IA", sentence: (e) => `${e.actor_name} preparó una propuesta de retroalimentación para la entrega${place(e)}` },
  propuesta_descartada: { title: "Propuesta descartada", sentence: (e) => `${told(e)}${place(e)}`, detailIsSentence: true },
  borrador_guardado: { title: "Borrador guardado", sentence: (e) => `${e.actor_name} guardó un borrador de la revisión${place(e)}` },
  aprobada: { title: "Revisión aprobada y devuelta", sentence: (e) => `${e.actor_name} aprobó y devolvió la revisión${place(e)}` },
  devuelta_actualizada: { title: "Devolución actualizada", sentence: (e) => `${e.actor_name} actualizó la revisión ya devuelta${place(e)}` },
  anotaciones_enviadas: {
    title: "Marcas enviadas",
    sentence: (e) => {
      const text = told(e).replace("al estudiante", e.student_name ? `a ${e.student_name}` : "al estudiante");
      return `${text}${e.assignment_title ? ` en «${e.assignment_title}»` : ""}`;
    },
    detailIsSentence: true,
  },
  modo_ia: { title: "Cambio en el modo de IA", sentence: (e) => `${e.actor_name} cambió la forma en que se entrega la retroalimentación de IA a los estudiantes:` },
  casos_reejecutados: { title: "Casos de prueba ejecutados de nuevo", sentence: (e) => `${e.actor_name} volvió a ejecutar los casos de prueba de la entrega${place(e)}` },
  pregunta_respondida: { title: "Pregunta respondida", sentence: (e) => `${told(e)}${place(e)}`, detailIsSentence: true },
  taller_creado: { title: "Taller creado", sentence: (e) => `${e.actor_name} creó el taller ${e.detail ?? ""}`, detailIsSentence: true },
  taller_editado: { title: "Taller editado", sentence: (e) => `${e.actor_name} editó el taller ${e.detail ?? ""}`, detailIsSentence: true },
  taller_eliminado: { title: "Taller eliminado", sentence: (e) => `${e.actor_name} eliminó el taller ${e.detail ?? ""}`, detailIsSentence: true },
  casos_actualizados: { title: "Casos de prueba actualizados", sentence: (e) => `${e.actor_name} actualizó los casos de prueba del taller ${e.detail ?? ""}`, detailIsSentence: true },
  codigo_restablecimiento: { title: "Código para restablecer contraseña", sentence: told, detailIsSentence: true },
  periodo_cerrado: { title: "Periodo cerrado", sentence: told, detailIsSentence: true },
  periodo_reabierto: { title: "Periodo reabierto", sentence: told, detailIsSentence: true },
  profesor_agregado: { title: "Profesor agregado", sentence: told, detailIsSentence: true },
  profesor_retirado: {
    title: "Profesor retirado",
    sentence: (e) => (e.detail?.startsWith(e.actor_name) ? e.detail : told(e)),
    detailIsSentence: true,
  },
  materia_copiada: {
    title: "Materia copiada",
    sentence: (e) => `${e.actor_name} creó esta materia como copia de ${(e.detail ?? "").replace("Copiada de ", "")}`,
    detailIsSentence: true,
  },
};

function detailLines(detail: string): string[] {
  return detail.split(" · ").filter((part) => !part.startsWith("modelo ")).flatMap((part) => {
    const change = part.match(/^(.*?)\s*(\S+) → (\S+)$/);
    if (!change) return [`${upper(part)}.`];
    const [, label, before, after] = change;
    if (MODE_NAME[before] || MODE_NAME[after]) return [`Antes: ${MODE_NAME[before] ?? before}.`, `Ahora: ${MODE_NAME[after] ?? after}.`];
    return [`Antes: ${label} ${before}.`, `Ahora: ${label} ${after}.`];
  });
}

export default function AuditItem({ entry, className = "" }: { entry: AuditEntry; className?: string }) {
  const action = ACTIONS[entry.action];
  const sentence = action ? action.sentence(entry) : `${entry.actor_name}${place(entry)}`;
  const lines = entry.detail && !(action?.detailIsSentence) ? detailLines(entry.detail) : [];
  return (
    <li className={`text-xs ${className}`}>
      <p className="text-ink-faint">{formatDateTime(entry.at)}</p>
      <p className="text-sm font-bold text-ink">{action?.title ?? entry.action}</p>
      <p className="text-ink-soft">{finished(sentence)}</p>
      {lines.map((line) => (
        <p key={line} className="text-ink-soft">
          {line}
        </p>
      ))}
    </li>
  );
}
