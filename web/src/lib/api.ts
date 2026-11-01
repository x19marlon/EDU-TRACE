export interface CompileResult {
  success: boolean;
  compiler_output: string;
  program_output: string;
  exit_code: number;
  compile_time_ms: number;
  run_time_ms: number;
  error?: string;
  tests?: TestResult[];
}

export type TestMatch = "contiene" | "exacta";

export interface TestCase {
  id?: string;
  name: string;
  stdin: string;
  expected: string;
  match: TestMatch;
  hidden: boolean;
}

export interface TestResult {
  test_id: string;
  name: string;
  hidden: boolean;
  passed: boolean;
  stdin?: string;
  expected?: string;
  output?: string;
  error?: string;
}

export type AIMode = "supervisada" | "inmediata";

export type JobStatus = "pendiente" | "procesando" | "lista" | "error" | "omitida";

export interface JobState {
  status: JobStatus;
  attempts: number;
  error?: string;
  enqueued_at: string;
  ahead: number;
}

export interface FeedbackMeta {
  model: string;
  time_ms: number;
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
export type Role = "student" | "teacher" | "monitor";

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
  has_recovery?: boolean;
  consent_at?: string;
}

export type SessionUser = User & { recovery_code?: string };

export interface GroupSummary {
  id: string;
  name: string;
  join_code: string;
  student_count: number;
  submission_count: number;
  ai_mode: AIMode;
}

export type RAELevel = "bajo" | "medio" | "alto";

export interface RAE {
  id: string;
  course_id: string;
  code: string;
  description: string;
}

export interface CourseSummary {
  id: string;
  name: string;
  groups: GroupSummary[];
  raes: RAE[];
}

export interface StudentPoint {
  id: string;
  name: string;
  average_grade?: number;
  graded: number;
  levels: Record<string, RAELevel>;
}

export interface RAEMatrix {
  raes: RAE[];
  students: StudentPoint[];
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

export interface GroupMonitor {
  id: string;
  name: string;
  email: string;
}

export interface GroupDetail {
  id: string;
  name: string;
  join_code: string;
  course_id: string;
  course_name: string;
  students: GroupStudent[];
  monitors: GroupMonitor[];
  ai_mode: AIMode;
}

export interface MonitorGroup {
  id: string;
  name: string;
  course_name: string;
  teacher_name: string;
  student_count: number;
  submission_count: number;
}

export interface MonitorFeedback {
  id: string;
  submission_id: string;
  monitor_id: string;
  monitor_name: string;
  body: string;
  created_at: string;
}

export type SubmissionKind = "official" | "attempt";

export interface ReviewState {
  status: "draft" | "returned";
  grade?: number;
}

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
  due_at?: string;
  late_seconds?: number;
  late_penalty?: number;
  tests_total?: number;
  tests_passed?: number;
  review_state?: ReviewState;
  monitor_feedback_count?: number;
  ai_job?: JobStatus;
  open_questions?: number;
}

export interface Question {
  id: string;
  submission_id: string;
  body: string;
  created_at: string;
  answer?: string;
  answered_by?: string;
  answered_role?: Role;
  answered_at?: string;
}

export interface OpenQuestion extends Question {
  group_id: string;
  group_name: string;
  student_name: string;
  assignment_title?: string;
}

export type Understood = "si" | "parcial" | "no";
export type RatingKind = "retro_formal" | "retro_informal" | "revision";

export interface FrequentError {
  message: string;
  explanation?: string;
  count: number;
  students: number;
  last_at: string;
  assignments: string[];
}

export interface HardTest {
  assignment: string;
  test: string;
  hidden: boolean;
  students: number;
  not_passing: number;
  failures: number;
  runs: number;
}

export interface ErrorReport {
  scope: "clase" | "materia";
  groups: number;
  analyzed: number;
  students_with_errors: number;
  compile: FrequentError[];
  runtime: FrequentError[];
  tests: HardTest[];
}

export interface Submission extends SubmissionMeta {
  code: string;
  stdin: string;
  compiler_output: string;
  program_output: string;
  ai_feedback: string;
  ai_feedback_informal: string;
  monitor_feedback?: MonitorFeedback[];
  annotations?: Annotation[];
  tests?: TestResult[];
  questions?: Question[];
}

export type AnnotationKind = "highlight" | "strike" | "good" | "note" | "drawing";

export interface Stroke {
  color: string;
  width: number;
  points: [number, number][];
}

export interface Annotation {
  id: string;
  submission_id: string;
  author_id: string;
  author_name: string;
  author_role: Role;
  kind: AnnotationKind;
  line_start?: number;
  line_end?: number;
  comment?: string;
  data?: { strokes: Stroke[] };
  sent_at?: string;
  created_at: string;
}

export interface AnnotationInput {
  kind: AnnotationKind;
  line_start?: number;
  line_end?: number;
  comment?: string;
  data?: { strokes: Stroke[] };
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
  rubric?: string;
  due_at?: string;
  allow_late: boolean;
  late_until?: string;
  late_penalty?: number;
  created_at: string;
  files: AssignmentFile[];
  tests: TestCase[];
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
  ai_mode: AIMode;
}

export interface Review {
  submission_id: string;
  ai_feedback?: string;
  ai_grade?: number;
  ai_reason?: string;
  ai_confidence?: "alta" | "media" | "baja";
  ai_at?: string;
  feedback: string;
  grade?: number;
  status: "draft" | "returned";
  returned_at?: string;
  grade_source: "ai_confirmed"|"profesor" |"";
  raes: ReviewRAE[];
  ai_discarded_at?: string;
  origin?: FeedbackOrigin;
  review_ms: number;
  ai_minutes?: number;
  ai_job?: JobState;
  ai_available: boolean;
}

export interface ReviewEstimate {
  assignment_id: string;
  title: string;
  due_at?: string;
  students: number;
  returned: number;
  pending: number;
  with_ai: number;
  not_compiling: number;
  failing_tests: number;
  all_passing: number;
  total_minutes: number;
  per_submission: number;
}

export interface ReviewPlan {
  assignments: ReviewEstimate[];
  measured: number;
  measured_minutes?: number;
  estimated_minutes?: number;
}

export type FeedbackOrigin = "ia" | "ia_editada" | "docente";

export interface ReviewRAE {
  rae_id: string;
  code: string;
  description: string;
  level?: RAELevel;
  ai_level?: RAELevel;
}

export interface StudentReview {
  feedback: string;
  grade?: number;
  returned_at: string;
  ai_assisted: boolean;
  raes?: { code: string; description: string; level: RAELevel }[];
  origin?: FeedbackOrigin;
  teacher_name?: string;
  understood?: Understood;
}
export interface StudentSubmission extends SubmissionMeta {
  group_name: string;
  course_name: string;
  assignment_title?: string;
  monitor_annotations?: number;
  review?: StudentReview;
  monitor_feedback?: MonitorFeedback[];
  questions?: Question[];
}

export class UnauthorizedError extends Error {}

async function throwForStatus(response: Response): Promise<never> {
  const errorData = await response.json().catch(() => null);
  const message = errorData?.error || `Server error: ${response.status}`;
  if (response.status === 401) throw new UnauthorizedError(message);
  throw new Error(message);
}


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

export function login(email: string, password: string, role: Role): Promise<SessionUser> {
  return postJSON("/api/auth/login", { email, password, role });
}

export function register(data: {
  name: string;
  email: string;
  password: string;
  role: Role;
  teacher_code?: string;
  consent: boolean;
}): Promise<SessionUser> {
  return postJSON("/api/auth/register", data);
}

export function recoverAccount(email: string, recoveryCode: string, newPassword: string): Promise<SessionUser> {
  return postJSON("/api/auth/recover", { email, recovery_code: recoveryCode, new_password: newPassword });
}

export async function changePassword(currentPassword: string, newPassword: string): Promise<void> {
  const response = await apiFetch("/api/auth/password", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
  });
  if (!response.ok) await throwForStatus(response);
}

export const newRecoveryCode = (password: string) =>
  postJSON<{ recovery_code: string }>("/api/auth/recovery-code", { password });

export const recordConsent = () => postJSON<User>("/api/auth/consent", {});
export const updateProfile = (name: string) => sendJSON<User>("PATCH", "/api/auth/profile", { name });

export async function exportMyData(): Promise<Blob> {
  const response = await apiFetch("/api/auth/export");
  if (!response.ok) await throwForStatus(response);
  return response.blob();
}

export async function deleteAccount(password: string): Promise<void> {
  const response = await apiFetch("/api/auth/delete", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ password }),
  });
  if (!response.ok) await throwForStatus(response);
}

export async function logout(): Promise<void> {
  await apiFetch("/api/auth/logout", { method: "POST" });
}

async function getJSON<T>(path: string): Promise<T> {
  const response = await apiFetch(path);
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

async function sendJSON<T>(method: "PATCH" | "PUT", path: string, body: unknown): Promise<T> {
  const response = await apiFetch(path, {
    method,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

async function deleteReq(path: string): Promise<void> {
  const response = await apiFetch(path, { method: "DELETE" });
  if (!response.ok) await throwForStatus(response);
}

const enc = encodeURIComponent;

export const renameCourse = (courseId: string, name: string) =>
  sendJSON<{ id: string; name: string }>("PATCH", `/api/teacher/courses/${enc(courseId)}`, { name });
export const deleteCourse = (courseId: string) => deleteReq(`/api/teacher/courses/${enc(courseId)}`);
export const renameGroup = (groupId: string, name: string) =>
  sendJSON<{ id: string; name: string }>("PATCH", `/api/teacher/groups/${enc(groupId)}`, { name });
export const deleteGroup = (groupId: string) => deleteReq(`/api/teacher/groups/${enc(groupId)}`);
export const regenerateJoinCode = (groupId: string) =>
  postJSON<{ join_code: string }>(`/api/teacher/groups/${enc(groupId)}/join-code`, {});
export const removeStudent = (groupId: string, studentId: string) =>
  deleteReq(`/api/teacher/groups/${enc(groupId)}/students/${enc(studentId)}`);
export const addMonitor = (groupId: string, email: string) =>
  postJSON<GroupMonitor>(`/api/teacher/groups/${enc(groupId)}/monitors`, { email });
export const removeMonitor = (groupId: string, monitorId: string) =>
  deleteReq(`/api/teacher/groups/${enc(groupId)}/monitors/${enc(monitorId)}`);
export const deleteAssignment = (assignmentId: string) => deleteReq(`/api/teacher/assignments/${enc(assignmentId)}`);
export const createRAE = (courseId: string, code: string, description: string) =>
  postJSON<RAE>(`/api/teacher/courses/${enc(courseId)}/raes`, { code, description });
export const updateRAE = (raeId: string, code: string, description: string) =>
  sendJSON<RAE>("PATCH", `/api/teacher/raes/${enc(raeId)}`, { code, description });
export const deleteRAE = (raeId: string) => deleteReq(`/api/teacher/raes/${enc(raeId)}`);
export const getRAEMatrix = (groupId: string) => getJSON<RAEMatrix>(`/api/teacher/groups/${enc(groupId)}/rae-matrix`);
export const deleteMonitorFeedbackAsTeacher = (feedbackId: string) =>
  deleteReq(`/api/teacher/monitor-feedback/${enc(feedbackId)}`);
export const setGroupAIMode = (groupId: string, aiMode: AIMode) =>
  sendJSON<{ ai_mode: AIMode }>("PATCH", `/api/teacher/groups/${enc(groupId)}/ai-mode`, { ai_mode: aiMode });
export const getGroupMetrics = (groupId: string) => getJSON<GroupMetrics>(`/api/teacher/groups/${enc(groupId)}/metrics`);
export const getGroupAudit = (groupId: string) => getJSON<AuditEntry[]>(`/api/teacher/groups/${enc(groupId)}/audit`);
export const retryDrafts = (groupId: string) =>
  postJSON<{ retried: number }>(`/api/teacher/groups/${enc(groupId)}/ai-queue/retry`, {});
export const setThresholds = (courseId: string, t: Thresholds) =>
  sendJSON<Thresholds>("PUT", `/api/teacher/courses/${enc(courseId)}/thresholds`, t);
export const getAIStatus = () => getJSON<AIStatus>("/api/teacher/ai-status");
export const discardReviewDraft = (submissionId: string, elapsedMs: number) =>
  postJSON<Review>(`/api/teacher/submissions/${enc(submissionId)}/review/discard`, { elapsed_ms: elapsedMs });
export function sendReviewTime(submissionId: string, elapsedMs: number): void {
  fetch(`${API_BASE}/api/teacher/submissions/${enc(submissionId)}/review/time`, {
    method: "POST",
    keepalive: true,
    credentials: "include",
    headers: { "Content-Type": "text/plain" },
    body: JSON.stringify({ elapsed_ms: elapsedMs }),
  }).catch(() => {});
}

export const getSubmissionAudit = (submissionId: string) =>
  getJSON<AuditEntry[]>(`/api/teacher/submissions/${enc(submissionId)}/audit`);
export const getReviewPlan = (groupId: string) => getJSON<ReviewPlan>(`/api/teacher/groups/${enc(groupId)}/review-plan`);
export const getFrequentErrors = (groupId: string, wholeCourse: boolean) =>
  getJSON<ErrorReport>(`/api/teacher/groups/${enc(groupId)}/errors${wholeCourse ? "?alcance=materia" : ""}`);
export const listOpenQuestions = () => getJSON<OpenQuestion[]>("/api/teacher/questions");
export const answerQuestion = (role: "teacher" | "monitor", questionId: string, answer: string) =>
  postJSON<Question>(`/api/${role}/questions/${enc(questionId)}/answer`, { answer });
export const askQuestion = (submissionId: string, body: string) =>
  postJSON<Question>(`/api/student/submissions/${enc(submissionId)}/questions`, { body });

export async function rateFeedback(data: {
  kind: RatingKind;
  understood: Understood;
  comment?: string;
  submission_id?: string;
  assignment_id?: string;
}): Promise<void> {
  const response = await apiFetch("/api/student/ratings", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!response.ok) await throwForStatus(response);
}

export async function downloadGroupCSV(groupId: string, kind: "notas" | "revisiones"): Promise<{ blob: Blob; filename: string }> {
  const response = await apiFetch(`/api/teacher/groups/${enc(groupId)}/export/${kind}.csv`);
  if (!response.ok) await throwForStatus(response);
  const disposition = response.headers.get("Content-Disposition") ?? "";
  const match = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(disposition);
  return { blob: await response.blob(), filename: match ? decodeURIComponent(match[1]) : `${kind}.csv` };
}

export const rerunTests = (submissionId: string) =>
  postJSON<TestResult[]>(`/api/teacher/submissions/${enc(submissionId)}/tests`, {});

export interface AuditEntry {
  id: number;
  at: string;
  actor_name: string;
  actor_role: Role | "sistema";
  action: string;
  submission_id?: string;
  student_name?: string;
  assignment_title?: string;
  detail?: string;
}

export interface Stat {
  count: number;
  avg_ms: number;
  p50_ms: number;
  p90_ms: number;
  max_ms: number;
  over: number;
}

export interface Thresholds {
  review_minutes?: number | null;
  batch_hours?: number | null;
  feedback_hours?: number | null;
}

export interface BatchMetric {
  assignment_id: string;
  title: string;
  submissions: number;
  done: number;
  failed: number;
  waiting: number;
  worst_ms: number;
  finished: boolean;
  over: boolean;
}

export interface GroupMetrics {
  course_id: string;
  course_name: string;
  ai_mode: AIMode;
  thresholds: Thresholds;
  review_time: Stat;
  review_estimate: Stat;
  over_estimate: number;
  turnaround: Stat;
  pending_reviews: number;
  pending_over: number;
  oldest_pending_ms: number;
  queue: { pending: number; running: number; done: number; failed: number; skipped: number; wait: Stat; run: Stat };
  batches: BatchMetric[];
  instant: {
    requests: number;
    failures: number;
    first_token: Stat;
    total: Stat;
    sections_checked: number;
    sections_ok: number;
    quality_checked: number;
    quality_ok: number;
    chats: number;
  };
  decisions: { ia: number; ia_editada: number; docente: number; descartadas: number };
  comprehension: {
    revisiones: RatingCount;
    inmediata: RatingCount;
    comentarios: { at: string; kind: RatingKind; understood: Understood; comment: string }[];
    meta: { min_si: number; max_no: number; min_respuestas: number };
    por_taller: (RatingCount & { assignment_id: string; title: string; total: number; suficientes: boolean; cumple: boolean })[];
    veredicto: "cumple" | "no_cumple" | "pocas_respuestas";
  };
}

export interface RatingCount {
  si: number;
  parcial: number;
  no: number;
}

export interface AIStatus {
  available: boolean;
  last_error?: string;
  last_error_at?: string;
  last_ok_at?: string;
  pending: number;
  running: number;
  failed: { group_id: string; group_name: string; course_name: string; failed: number; last_error: string; last_at?: string }[];
}

export async function updateAssignment(assignmentId: string, form: FormData): Promise<Assignment> {
  const response = await apiFetch(`/api/teacher/assignments/${enc(assignmentId)}`, { method: "PUT", body: form });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

export const listMonitorGroups = () => getJSON<MonitorGroup[]>("/api/monitor/groups");
export const getMonitorGroup = (groupId: string) => getJSON<GroupDetail>(`/api/monitor/groups/${enc(groupId)}`);
export const listMonitorSubmissions = (groupId: string, studentId?: string) => {
  const q = new URLSearchParams({ kind: "official" });
  if (studentId) q.set("student", studentId);
  return getJSON<SubmissionMeta[]>(`/api/monitor/groups/${enc(groupId)}/submissions?${q}`);
};
export const listMonitorAssignments = (groupId: string) =>
  getJSON<Assignment[]>(`/api/monitor/groups/${enc(groupId)}/assignments`);
export const getMonitorSubmission = (id: string) => getJSON<Submission>(`/api/monitor/submissions/${enc(id)}`);
export const addMonitorFeedback = (submissionId: string, body: string) =>
  postJSON<MonitorFeedback>(`/api/monitor/submissions/${enc(submissionId)}/feedback`, { body });
export const deleteMonitorFeedback = (feedbackId: string) => deleteReq(`/api/monitor/feedback/${enc(feedbackId)}`);

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

// revision profesor
export const getReview = (submissionId: string) =>
  getJSON<Review>(`/api/teacher/submissions/${encodeURIComponent(submissionId)}/review`);
export const generateReviewDraft = (submissionId: string) =>
  postJSON<Review>(`/api/teacher/submissions/${encodeURIComponent(submissionId)}/review/draft`, {});
export async function saveReview(
  submissionId: string,
  data: { feedback: string; grade: number | null; return: boolean; rae_levels?: Record<string, RAELevel | "">; elapsed_ms?: number }
): Promise<Review> {
  const response = await apiFetch(`/api/teacher/submissions/${encodeURIComponent(submissionId)}/review`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}

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
export const listMySubmissions = () => getJSON<StudentSubmission[]>("/api/student/submissions");

export interface StudentHistory {
  submissions: StudentSubmission[];
  attempts: StudentSubmission[];
}

export const createAnnotation = (role: "teacher" | "monitor", submissionId: string, input: AnnotationInput) =>
  postJSON<Annotation>(`/api/${role}/submissions/${encodeURIComponent(submissionId)}/annotations`, input);
export const deleteAnnotation = (role: "teacher" | "monitor", annotationId: string) =>
  deleteReq(`/api/${role}/annotations/${encodeURIComponent(annotationId)}`);
export const sendMonitorAnnotations = (submissionId: string) =>
  postJSON<{ sent: number; sent_at: string }>(`/api/monitor/submissions/${encodeURIComponent(submissionId)}/annotations/send`, {});

export const getMyHistory = () => getJSON<StudentHistory>("/api/student/history");
export const getMySubmission = (id: string) => getJSON<Submission>(`/api/student/submissions/${encodeURIComponent(id)}`);

// Talleres
export const listGroupAssignments = (groupId: string) =>
  getJSON<Assignment[]>(`/api/teacher/groups/${encodeURIComponent(groupId)}/assignments`);


export async function createAssignment(groupId: string, form: FormData): Promise<Assignment> {
  const response = await apiFetch(`/api/teacher/groups/${encodeURIComponent(groupId)}/assignments`, {
    method: "POST",
    body: form, 
  });
  if (!response.ok) await throwForStatus(response);
  return response.json();
}


export const assignmentFileUrl = (assignmentId: string, fileId: string) =>
  `${API_BASE}/api/assignments/${encodeURIComponent(assignmentId)}/files/${encodeURIComponent(fileId)}`;


export function compileCode(
  code: string,
  stdin: string = "",
  context?: { group_id: string; assignment_id?: string }
): Promise<CompileResult> {
  return postJSON("/api/compile", { code, stdin, ...context });
}


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
  tests?: TestResult[];
}


export async function streamFeedback(
  context: FeedbackContext,
  onToken: (token: string) => void,
  onReplace?: (text: string) => void
): Promise<FeedbackMeta> {
  const response = await apiFetch("/api/feedback", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(context),
  });

  if (!response.ok) await throwForStatus(response);
  return readSSE(response, onToken, onReplace);
}

/** Lee una respuesta SSE del backend: tokens, "error" y "done" con metadatos. */
async function readSSE(
  response: Response,
  onToken: (token: string) => void,
  onReplace?: (text: string) => void
): Promise<FeedbackMeta> {
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
        } else if (eventType === "replace") {
          const text = (data as { text?: unknown })?.text;
          if (typeof text === "string") onReplace?.(text);
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

export interface ChatMessage {
  role: "user" | "assistant";
  content: string;
}

export interface ChatContext {
  assignment_id?: string;
  code?: string;
  stdin?: string;
  messages: ChatMessage[];
}

export async function streamChat(
  context: ChatContext,
  onToken: (token: string) => void
): Promise<FeedbackMeta> {
  const response = await apiFetch("/api/chat", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(context),
  });

  if (!response.ok) await throwForStatus(response);
  return readSSE(response, onToken);
}
