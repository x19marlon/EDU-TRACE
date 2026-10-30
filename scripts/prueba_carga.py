import json
import os
import sys
import time
import urllib.error
import urllib.request
from http.cookiejar import CookieJar

API = os.environ.get("API", "http://127.0.0.1:8080")
N = int(sys.argv[1]) if len(sys.argv) > 1 else 10
TEACHER = ("maria.garcia@universidad.edu.co", "Profesor2026!")
STUDENTS = ["ana.rodriguez@est.edu.co", "carlos.perez@est.edu.co", "luis.martinez@est.edu.co"]
GROUP = os.environ.get("CLASE", "Grupo A - Mañana")
TALLER = os.environ.get("TALLER", "Taller 1: promedio de notas")
LIMITE_MIN = int(os.environ.get("LIMITE_MIN", "120"))

CODE = """#include <iostream>
#include <iomanip>
using namespace std;

int main() {
    double a, b, c;
    cin >> a >> b >> c;
    double promedio = a + b + c / %d;
    cout << fixed << setprecision(2) << "Promedio: " << promedio << endl;
    return 0;
}
"""


def client(email, password, role):
    opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(CookieJar()))

    def call(method, path, body=None):
        data = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(API + path, data=data, method=method, headers={"Content-Type": "application/json"})
        try:
            with opener.open(req, timeout=120) as resp:
                raw = resp.read()
                return json.loads(raw) if raw else None
        except urllib.error.HTTPError as e:
            sys.exit(f"  ✗ {method} {path}: {json.loads(e.read() or b'{}').get('error', e.reason)}")
        except urllib.error.URLError as e:
            sys.exit(f"  ✗ No se pudo conectar con {API}: {e.reason}")

    call("POST", "/api/auth/login", {"email": email, "password": password, "role": role})
    return call


prof = client(*TEACHER, "teacher")
group = next((g for c in prof("GET", "/api/teacher/courses") for g in c["groups"] if g["name"] == GROUP), None)
if not group:
    sys.exit(f"  ✗ No existe la clase «{GROUP}». Carga primero ./scripts/seed_test_data.sh")
taller = next((a for a in prof("GET", f"/api/teacher/groups/{group['id']}/assignments") if a["title"] == TALLER), None)
if not taller:
    sys.exit(f"  ✗ No existe el taller «{TALLER}» en «{GROUP}»")
students = [client(email, "Alumno2026!", "student") for email in STUDENTS]

print(f"1. Enviando {N} entregas a «{TALLER}» ({GROUP})")
start = time.time()
ids = []
for i in range(N):
    sub = students[i % len(students)]("POST", "/api/student/submissions", {
        "group_id": group["id"], "assignment_id": taller["id"], "code": CODE % (3 + i),
        "stdin": "4.5 3.0 5.0", "ai_feedback": "", "ai_feedback_informal": "",
    })
    ids.append(sub["id"])
print(f"   ✓ {N} entregas guardadas en {time.time() - start:.1f} s (compiladas y con sus casos de prueba)")

print(f"2. Esperando a que la IA prepare las {N} propuestas (límite {LIMITE_MIN} min)")
pending = set(ids)
while pending and time.time() - start < LIMITE_MIN * 60:
    time.sleep(10)
    states = {s["id"]: s.get("ai_job") for s in prof("GET", f"/api/teacher/groups/{group['id']}/submissions") if s["id"] in pending}
    pending = {i for i, st in states.items() if st in ("pendiente", "procesando")}
    print(f"   {N - len(pending)}/{N} procesadas · {time.time() - start:.0f} s")

elapsed = time.time() - start
states = {s["id"]: s.get("ai_job") for s in prof("GET", f"/api/teacher/groups/{group['id']}/submissions") if s["id"] in ids}
done = sum(1 for st in states.values() if st == "lista")
failed = sum(1 for st in states.values() if st == "error")
print("")
print(f"3. Resultado: {done} listas, {failed} con error, {len(pending)} sin terminar en {elapsed / 60:.1f} min")
print(f"   Promedio por entrega: {elapsed / max(1, N - len(pending)):.0f} s")
metrics = prof("GET", f"/api/teacher/groups/{group['id']}/metrics")
limit = metrics["thresholds"].get("batch_hours")
batch = next((b for b in metrics["batches"] if b["assignment_id"] == taller["id"]), None)
if batch:
    print(f"   Peor demora del lote «{batch['title']}»: {batch['worst_ms'] / 60000:.1f} min", end="")
    print(f" · umbral {limit} h · {'NO CUMPLE' if batch['over'] else 'cumple'}" if limit else " · sin umbral definido para la materia")
sys.exit(1 if pending or failed else 0)
