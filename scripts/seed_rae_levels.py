import json
import os
import sys
import urllib.error
import urllib.request
from http.cookiejar import CookieJar

API = os.environ.get("API", "http://127.0.0.1:8080")
TEACHER = ("maria.garcia@universidad.edu.co", "Profesor2026!")
COURSE = "Introducción a la Programación 2026-1"
GROUP = "Grupo A - Mañana"

RAES = [
    ("RAE 1", "Diseña algoritmos que resuelven problemas sencillos con secuencia, condicionales y ciclos."),
    ("RAE 2", "Implementa programas en C++ usando tipos de datos, entrada y salida adecuados."),
    ("RAE 3", "Verifica sus programas con casos de prueba y depura los errores que encuentra."),
]

CARLOS_FEEDBACK = (
    "Tu programa compila y lee bien las tres notas. Con 4, 5 y 5 imprime 4 y el promedio real es 4.67.\n\n"
    "Revisa la línea 7: ¿de qué tipo son `a`, `b` y `c`, y qué tipo de división hace C++ con ellos? "
    "Prueba a mano antes de volver a enviarlo."
)

PLAN = [
    ("Ana Rodríguez", "Taller 1: promedio de notas", True, None, {"RAE 1": "medio", "RAE 2": "alto", "RAE 3": "alto"}),
    ("Ana Rodríguez", "Taller 2: suma con ciclos", True, None, {"RAE 1": "medio", "RAE 2": "alto", "RAE 3": "bajo"}),
    ("Ana Rodríguez", "Taller 3: el mayor de tres", False, None, {"RAE 1": "alto", "RAE 2": "alto", "RAE 3": "medio"}),
    ("Luis Martínez", "Taller 2: suma con ciclos", True, None, {"RAE 1": "alto", "RAE 2": "alto", "RAE 3": "alto"}),
    ("Carlos Pérez", "Taller 1: promedio de notas", True, (2.8, CARLOS_FEEDBACK), {"RAE 1": "medio", "RAE 2": "bajo", "RAE 3": "bajo"}),
]

opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(CookieJar()))


def call(method, path, body=None):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(API + path, data=data, method=method, headers={"Content-Type": "application/json"})
    try:
        with opener.open(req) as resp:
            raw = resp.read()
            return json.loads(raw) if raw else None
    except urllib.error.HTTPError as e:
        try:
            msg = json.loads(e.read()).get("error", e.reason)
        except ValueError:
            msg = e.reason
        sys.exit(f"  ✗ {method} {path}: {msg}")
    except urllib.error.URLError as e:
        sys.exit(f"  ✗ No se pudo conectar con {API}: {e.reason}")


def find(items, key, value, what):
    for item in items:
        if item[key] == value:
            return item
    sys.exit(f"  ✗ No se encontró {what} «{value}». Carga primero ./scripts/seed_test_data.sh")


call("POST", "/api/auth/login", {"email": TEACHER[0], "password": TEACHER[1], "role": "teacher"})
course = find(call("GET", "/api/teacher/courses"), "name", COURSE, "la materia")

rae_ids = {r["code"]: r["id"] for r in course.get("raes", [])}
for code, description in RAES:
    if code not in rae_ids:
        rae_ids[code] = call("POST", f"/api/teacher/courses/{course['id']}/raes", {"code": code, "description": description})["id"]
print(f"  ✓ {len(RAES)} RAE en «{COURSE}»")

group = find(course["groups"], "name", GROUP, "la clase")
students = {s["name"]: s["id"] for s in call("GET", f"/api/teacher/groups/{group['id']}")["students"]}
assignments = {a["title"]: a["id"] for a in call("GET", f"/api/teacher/groups/{group['id']}/assignments")}
submissions = call("GET", f"/api/teacher/groups/{group['id']}/submissions")

for name, title, return_it, fallback, levels in PLAN:
    student_id = students.get(name)
    assignment_id = assignments.get(title)
    match = [s for s in submissions if s["student_id"] == student_id and s.get("assignment_id") == assignment_id]
    if not match:
        sys.exit(f"  ✗ No hay entrega de {name} en «{title}»")
    sid = match[0]["id"]
    review = call("GET", f"/api/teacher/submissions/{sid}/review")
    feedback, grade = review.get("feedback", ""), review.get("grade")
    if not feedback and fallback:
        grade, feedback = fallback
    call("PUT", f"/api/teacher/submissions/{sid}/review", {
        "feedback": feedback,
        "grade": grade,
        "return": return_it,
        "rae_levels": {rae_ids[code]: level for code, level in levels.items()},
    })
    shown = ", ".join(f"{code} {level}" for code, level in levels.items())
    state = "devuelta" if return_it else "en revisión"
    print(f"  ✓ {name} · {title}: {grade} ({state}) · {shown}")
