#!/usr/bin/env python3
"""Agrega estudiantes con perfiles variados a Intro a la Programación - Grupo A, para
ver todos los cuadrantes y categorías del gráfico Nota vs. esfuerzo.

Requiere el backend corriendo y haber ejecutado antes scripts/seed_test_data.sh. Todo
pasa por la API. Se puede ejecutar varias veces: quien ya existe se omite.

Uso:
    python3 scripts/seed_activity.py
    API=http://127.0.0.1:8092 python3 scripts/seed_activity.py
"""

import json
import os
import sys
import urllib.error
import urllib.request
from http.cookiejar import CookieJar

API = os.environ.get("API", "http://127.0.0.1:8080")
TEACHER = ("maria.garcia@universidad.edu.co", "Profesor2026!")
PASSWORD = "Alumno2026!"
COURSE = "Introducción a la Programación 2026-1"
PROGRAM = "#include <iostream>\nint main() {\n    std::cout << \"Intento %d\" << std::endl;\n    return 0;\n}\n"

# (nombre, correo, nota del docente o None, intentos, ¿entrega?)
PROFILES = [
    ("Valeria Rojas", "valeria.rojas", 1.5, 1, True),    # poca práctica y nota baja
    ("Mateo Silva", "mateo.silva", 2.4, 4, True),        # practica, falta comprensión
    ("Camila Vega", "camila.vega", 1.8, 8, True),        # se esfuerza sin lograrlo
    ("Tomás Pardo", "tomas.pardo", 3.2, 1, True),        # aprueba con poca práctica
    ("Nora Quintero", "nora.quintero", 3.5, 4, True),    # en progreso
    ("Iván Mora", "ivan.mora", 3.1, 7, True),            # persistente
    ("Julián Soto", "julian.soto", 4.2, 4, True),        # buen ritmo
    ("Elena Cruz", "elena.cruz", 4.7, 7, True),          # lo logra con esfuerzo
    ("Sara Nieto", "sara.nieto", None, 2, True),         # falta tu calificación
    ("Pablo Ruiz", "pablo.ruiz", None, 3, False),        # trabaja sin entregar
    ("Rita Luna", "rita.luna", None, 0, False),          # sin actividad
]


class Client:
    def __init__(self):
        self.opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(CookieJar()))

    def call(self, method, path, data=None):
        body = json.dumps(data).encode() if data is not None else None
        req = urllib.request.Request(API + path, data=body, method=method, headers={"Content-Type": "application/json"})
        with self.opener.open(req, timeout=60) as resp:
            raw = resp.read().decode()
        return json.loads(raw) if raw.strip() else None


def main():
    teacher = Client()
    teacher.call("POST", "/api/auth/login", {"email": TEACHER[0], "password": TEACHER[1]})
    course = next((c for c in teacher.call("GET", "/api/teacher/courses") if c["name"] == COURSE), None)
    if not course or not course["groups"]:
        sys.exit(f"No encontré «{COURSE}». Ejecuta primero scripts/seed_test_data.sh.")
    group_id = course["groups"][0]["id"]
    join_code = teacher.call("GET", f"/api/teacher/groups/{group_id}")["join_code"]
    assignments = teacher.call("GET", f"/api/teacher/groups/{group_id}/assignments")
    if not assignments:
        sys.exit("Esa clase no tiene talleres. Ejecuta primero scripts/seed_test_data.sh.")
    assignment_id = assignments[0]["id"]

    for name, user, grade, attempts, submits in PROFILES:
        student = Client()
        try:
            student.call("POST", "/api/auth/register", {
                "name": name, "email": f"{user}@est.edu.co", "password": PASSWORD, "role": "student", "consent": True,
            })
        except urllib.error.HTTPError as e:
            if e.code != 409:
                raise
            print(f"  = {name} (ya existe)")
            continue
        student.call("POST", "/api/student/groups/join", {"code": join_code})
        context = {"group_id": group_id, "assignment_id": assignment_id, "stdin": ""}
        for i in range(attempts):
            student.call("POST", "/api/compile", {**context, "code": PROGRAM % i})
        if submits:
            sub = student.call("POST", "/api/student/submissions", {**context, "code": PROGRAM % attempts})
            if grade is not None:
                teacher.call("PUT", f"/api/teacher/submissions/{sub['id']}/review", {
                    "feedback": "Retroalimentación de prueba.", "grade": grade, "return": True,
                })
        print(f"  ✓ {name}")
    print(f"\nListo. Contraseña de estas cuentas: {PASSWORD}. Abre la clase como docente y mira «Nota vs. esfuerzo».")


if __name__ == "__main__":
    try:
        main()
    except urllib.error.HTTPError as e:
        sys.exit(f"Error {e.code} en {e.url}: {e.read().decode().strip()}")
    except urllib.error.URLError as e:
        sys.exit(f"Error: {e}\n¿Está el backend corriendo en {API}?")
