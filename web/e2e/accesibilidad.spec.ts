import { APIRequestContext, Browser, expect, Page, request, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const API = process.env.E2E_API ?? "http://localhost:8080";
const TEACHER_CODE = process.env.E2E_TEACHER_CODE ?? "";
const PASSWORD = "clave-de-prueba-1";
const TAGS = ["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"];

type Session = { api: APIRequestContext; email: string; id: string };

let teacher: Session;
let student: Session;
let groupId = "";

async function call<T>(api: APIRequestContext, method: string, path: string, data?: unknown): Promise<T> {
  const response = await api.fetch(`${API}${path}`, { method, data });
  expect(response.ok(), `${method} ${path}: ${response.status()} ${await response.text()}`).toBeTruthy();
  return response.status() === 204 ? (undefined as T) : response.json();
}

async function account(role: "teacher" | "student", name: string): Promise<Session> {
  const api = await request.newContext();
  const email = `${role}-${Date.now()}@prueba.edu`;
  const user = await call<{ id: string }>(api, "POST", "/api/auth/register", { name, email, password: PASSWORD, role, teacher_code: TEACHER_CODE, consent: true });
  return { api, email, id: user.id };
}

async function pageAs(browser: Browser, session: Session): Promise<Page> {
  const context = await browser.newContext({ storageState: await session.api.storageState() });
  return context.newPage();
}

async function expectAccessible(page: Page, label: string) {
  await page.waitForLoadState("networkidle");
  const { violations } = await new AxeBuilder({ page }).withTags(TAGS).analyze();
  const report = violations.map(
    (v) => `${v.id} (${v.impact}): ${v.help} → ${v.nodes.slice(0, 4).map((n) => n.target.join(" ")).join(" | ")}`
  );
  expect(report, `${label}: problemas de accesibilidad`).toEqual([]);
}

test.beforeAll(async () => {
  expect(TEACHER_CODE, "falta E2E_TEACHER_CODE").not.toBe("");
  teacher = await account("teacher", "Elena Prueba");
  student = await account("student", "Ana Prueba");
  const course = await call<{ id: string }>(teacher.api, "POST", "/api/teacher/courses", { name: "Programación I" });
  const group = await call<{ id: string; join_code: string }>(teacher.api, "POST", `/api/teacher/courses/${course.id}/groups`, { name: "Grupo 1" });
  groupId = group.id;
  const due = new Date(Date.now() + 7 * 24 * 3600 * 1000).toISOString();
  const tests = JSON.stringify([
    { name: "Suma", stdin: "3", expected: "6", match: "contiene", hidden: false },
    { name: "Oculto", stdin: "4", expected: "10", match: "contiene", hidden: true },
  ]);
  const created = await teacher.api.fetch(`${API}/api/teacher/groups/${group.id}/assignments`, {
    method: "POST",
    multipart: {
      title: "Taller 1: suma",
      statement: "Lee n e imprime la suma de 1 a n.",
      due_at: due,
      tests,
      criteria: JSON.stringify([
        { name: "Compila", points: 1 },
        { name: "Resultado correcto", points: 4 },
      ]),
    },
  });
  expect(created.ok(), await created.text()).toBeTruthy();
  const assignment = await created.json();
  await call(student.api, "POST", "/api/student/groups/join", { code: group.join_code });
  const code = "#include <iostream>\nint main() {\n  int n, s = 0;\n  std::cin >> n;\n  for (int i = 1; i <= n; i++) s += i;\n  std::cout << s << std::endl;\n}\n";
  const sub = await call<{ id: string }>(student.api, "POST", "/api/student/submissions", {
    group_id: group.id,
    assignment_id: assignment.id,
    code,
    stdin: "3",
  });
  await call(teacher.api, "PUT", `/api/teacher/submissions/${sub.id}/review`, {
    feedback: "## ¿Hacia dónde voy?\nSumar de 1 a n.\n\n## ¿Cómo voy?\nFunciona.\n\n## ¿Qué sigue?\nPrueba con n = 0.",
    grade: 4.5,
    return: true,
  });
  await call(student.api, "POST", `/api/student/submissions/${sub.id}/questions`, { body: "¿Qué pasa si n es negativo?" });
});

test("páginas públicas", async ({ page }) => {
  for (const path of ["/login", "/login?rol=teacher", "/privacidad"]) {
    await page.goto(path);
    await expectAccessible(page, path);
  }
});

test("estudiante", async ({ browser }) => {
  const page = await pageAs(browser, student);
  for (const path of ["/", "/historial", "/cuenta", "/practica", "/progreso", "/?ejercicio=suma-1-n"]) {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
    await expectAccessible(page, path);
  }
  await page.goto("/");
  await page.getByRole("button", { name: /^Avisos/ }).click();
  await expect(page.getByRole("heading", { name: "Avisos" })).toBeVisible();
  await expectAccessible(page, "/ con los avisos abiertos");
});

test("profesor", async ({ browser }) => {
  const page = await pageAs(browser, teacher);
  for (const path of ["/docente", `/docente/clases/${groupId}`, `/docente/clases/${groupId}/reporte/${student.id}`, "/cuenta"]) {
    await page.goto(path);
    await expect(page.getByRole("heading", { level: 1 }).first()).toBeVisible();
    await expectAccessible(page, path);
  }
  await page.goto(`/docente/clases/${groupId}`);
  await page.getByRole("button", { name: /Ana Prueba.*Taller 1: suma/ }).click();
  await expect(page.getByText("Rúbrica").first()).toBeVisible();
  await expectAccessible(page, "revisión con rúbrica");
});

test.afterAll(async () => {
  await teacher?.api.dispose();
  await student?.api.dispose();
});
