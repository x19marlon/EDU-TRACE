export interface CompileResult {
  success: boolean;
  compiler_output: string;
  program_output: string;
  exit_code: number;
  compile_time_ms: number;
  run_time_ms: number;
  error?: string;
}

export interface FeedbackMeta {
  model: string;
  time_ms: number;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type Role = "student" | "teacher";

export interface Activity {
  compiles: number;
  feedbacks: number;
  last_active_at?: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  role: Role;
  activity: Activity;
}

// ---------- Materias, clases y envíos ----------

export interface GroupSummary {
  id: string;
  name: string;
  join_code: string;
  student_count: number;
  submission_count: number;
}

export interface CourseSummary {
  id: string;
  name: string;
  groups: GroupSummary[];
}

export interface GroupStudent {
  id: string;
  name: string;
  email: string;
  activity: Activity;
  submission_count: number; // entregas oficiales
  last_submission_at?: string;
  attempt_count: number; // intentos de compilación
  last_attempt_at?: string;
}

export interface GroupDetail {
  id: string;
  name: string;
  join_code: string;
  course_id: string;
  course_name: string;
  students: GroupStudent[];
}

export type SubmissionKind = "official" | "attempt";

export interface SubmissionMeta {
  id: string;
  kind: SubmissionKind;
  error_summary?: string;
  group_id: string;
  assignment_id?: string;
  student_id: string;
  created_at: string;
  success: boolean;
  run_error?: string;
  exit_code: number;
  lines: number;
  has_ai_feedback: boolean;
}

export interface Submission extends SubmissionMeta {
  code: string;
  stdin: string;
  compiler_output: string;
  program_output: string;
  ai_feedback: string;
  ai_feedback_informal: string;
}

export interface AssignmentFile {
  id: string;
  name: string;
  size: number;
}

export interface Assignment {
  id: string;
  group_id: string;
  title: string;
  statement: string;
  due_at?: string;
  created_at: string;
  files: AssignmentFile[];
}

export interface StudentAssignment extends Assignment {
  group_name: string;
  course_name: string;
}

export interface StudentGroup {
  id: string;
  name: string;
  course_name: string;
  teacher_name: string;
}

/** Error 401: la sesión no existe o expiró. */
export class UnauthorizedError extends Error {}

async function throwForStatus(response: Response): Promise<never> {
  const errorData = await response.json().catch(() => null);
  const message = errorData?.error || `Server error: ${response.status}`;
  if (response.status === 401) throw new UnauthorizedError(message);
  throw new Error(message);
}

/** fetch con la cookie de sesión incluida (el backend está en otro puerto). */
function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  return fetch(`${API_BASE}${path}`, { ...init, credentials: "include" });
}

async function postJSON<T>(path: string, body: unknown): Promise<T> {
  const response = await apiFetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

export async function getMe(): Promise<User | null> {
  const response = await apiFetch("/api/auth/me");
  if (response.status === 401) return null;
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

export function login(email: string, password: string, role: Role): Promise<User> {
  return postJSON("/api/auth/login", { email, password, role });
}

export function register(data: {
  name: string;
  email: string;
  password: string;
  role: Role;
  teacher_code?: string;
}): Promise<User> {
  return postJSON("/api/auth/register", data);
}

export async function logout(): Promise<void> {
  await apiFetch("/api/auth/logout", { method: "POST" });
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await apiFetch(path);
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

// Docente
export const listCourses = () => getJSON<CourseSummary[]>("/api/teacher/courses");
export const createCourse = (name: string) => postJSON<CourseSummary>("/api/teacher/courses", { name });
export const createGroup = (courseId: string, name: string) =>
  postJSON<GroupSummary>(`/api/teacher/courses/${encodeURIComponent(courseId)}/groups`, { name });
export const getGroup = (groupId: string) =>
  getJSON<GroupDetail>(`/api/teacher/groups/${encodeURIComponent(groupId)}`);
export const listGroupSubmissions = (groupId: string, studentId?: string, kind: SubmissionKind | "all" = "official") => {
  const q = new URLSearchParams({ kind });
  if (studentId) q.set("student", studentId);
  return getJSON<SubmissionMeta[]>(`/api/teacher/groups/${encodeURIComponent(groupId)}/submissions?${q}`);
};

/** Análisis con IA del proceso de un estudiante, para el profesor (SSE). */
export async function streamStudentAnalysis(
  groupId: string,
  studentId: string,
  onToken: (token: string) => void
): Promise<FeedbackMeta> {
  const response = await apiFetch(
    `/api/teacher/groups/${encodeURIComponent(groupId)}/students/${encodeURIComponent(studentId)}/analysis`,
    { method: "POST" }
  );
  if (!response.ok) await throwForStatus(response);
  return readSSE(response, onToken);
}
export const getSubmission = (id: string) =>
  getJSON<Submission>(`/api/teacher/submissions/${encodeURIComponent(id)}`);

// Estudiante
export const listMyGroups = () => getJSON<StudentGroup[]>("/api/student/groups");
export const joinGroup = (code: string) => postJSON<StudentGroup>("/api/student/groups/join", { code });
export const submitCode = (data: {
  group_id: string;
  assignment_id?: string;
  code: string;
  stdin: string;
  ai_feedback: string;
  ai_feedback_informal: string;
}) => postJSON<SubmissionMeta>("/api/student/submissions", data);
export const listMyAssignments = () => getJSON<StudentAssignment[]>("/api/student/assignments");

// Talleres
export const listGroupAssignments = (groupId: string) =>
  getJSON<Assignment[]>(`/api/teacher/groups/${encodeURIComponent(groupId)}/assignments`);

/** Crea un taller (multipart: title, statement, due_at opcional y files). */
export async function createAssignment(groupId: string, form: FormData): Promise<Assignment> {
  const response = await apiFetch(`/api/teacher/groups/${encodeURIComponent(groupId)}/assignments`, {
    method: "POST",
    body: form, // el navegador pone el Content-Type multipart con su boundary
  });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

/** URL de descarga de un adjunto (la cookie de sesión viaja con el enlace). */
export const assignmentFileUrl = (assignmentId: string, fileId: string) =>
  `${API_BASE}/api/assignments/${encodeURIComponent(assignmentId)}/files/${encodeURIComponent(fileId)}`;

/**
 * Compila y ejecuta. Si el estudiante indica su clase (y taller), la compilación
 * se guarda como intento para que su profesor vea el proceso.
 */
export function compileCode(
  code: string,
  stdin: string = "",
  context?: { group_id: string; assignment_id?: string }
): Promise<CompileResult> {
  return postJSON("/api/compile", { code, stdin, ...context });
}

/**
 * Datos de la última compilación de exactamente el código que se envía.
 * Si el código cambió desde entonces, se envía `compiled: false`.
 */
export type FeedbackMode = "formal" | "informal";

export interface FeedbackContext {
  mode: FeedbackMode;
  assignment_id?: string;
  code: string;
  compiled: boolean;
  success?: boolean;
  compiler_output?: string;
  stdin?: string;
  program_output?: string;
  exit_code?: number;
  run_error?: string;
}

/**
 * Streams AI feedback via Server-Sent Events.
 * onToken is called for each chunk of text as it arrives.
 * Returns metadata (model, time) when the stream completes.
 */
export async function streamFeedback(
  context: FeedbackContext,
  onToken: (token: string) => void
): Promise<FeedbackMeta> {
  const response = await apiFetch("/api/feedback", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(context),
  });

  if (!response.ok) await throwForStatus(response);
  return readSSE(response, onToken);
}

/** Lee una respuesta SSE del backend: tokens, "error" y "done" con metadatos. */
async function readSSE(response: Response, onToken: (token: string) => void): Promise<FeedbackMeta> {
  const reader = response.body?.getReader();
  if (!reader) throw new Error("No se recibió respuesta del servidor");

  const decoder = new TextDecoder();
  let meta: FeedbackMeta = { model: "", time_ms: 0 };
  let streamError: string | null = null;
  let buffer = "";
  // Se conserva entre lecturas: "event:" y "data:" pueden llegar en trozos distintos.
  let eventType = "message";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });

    const lines = buffer.split("\n");
    buffer = lines.pop() || ""; // Línea incompleta: se espera al siguiente trozo.

    for (const line of lines) {
      if (line.startsWith("event: ")) {
        eventType = line.slice(7).trim();
      } else if (line.startsWith("data: ")) {
        let data: unknown;
        try {
          data = JSON.parse(line.slice(6));
        } catch {
          data = line.slice(6);
        }

        if (eventType === "done") {
          meta = data as FeedbackMeta;
        } else if (eventType === "error") {
          streamError =
            (data as { error?: string })?.error ?? String(data);
        } else if (data && typeof (data as { token?: unknown }).token === "string") {
          onToken((data as { token: string }).token);
        }
      } else if (line === "") {
        eventType = "message"; // Fin del evento SSE.
      }
    }
  }

  if (streamError) throw new Error(streamError);
  return meta;
}
