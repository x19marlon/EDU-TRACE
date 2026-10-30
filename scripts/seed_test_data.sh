#!/usr/bin/env bash
# Carga usuarios de prueba, cursos y clases en EDU-TRACE
set -e
API="${API:-http://127.0.0.1:8080}"
ENV_FILE="$(dirname "$0")/../.env"
if [ -z "$TEACHER_CODE" ] && [ -f "$ENV_FILE" ]; then
  TEACHER_CODE=$(grep -E '^TEACHER_CODE=.+' "$ENV_FILE" | tail -1 | cut -d= -f2-)
fi
TEACHER_CODE="${TEACHER_CODE:-admin123}"

echo "=== 1. Registrando profesor ==="
PROF_COOKIE=$(curl -s -c - "$API/api/auth/register" \
  -H "Content-Type: application/json" \
  -d "$(python3 -c "import json, sys; print(json.dumps({'name': 'Prof. María García', 'email': 'maria.garcia@universidad.edu.co', 'password': 'Profesor2026!', 'role': 'teacher', 'teacher_code': sys.argv[1], 'consent': True}))" "$TEACHER_CODE")" \
  | grep edutrace_session | awk '{print $NF}')
echo "  Cookie profesor: $PROF_COOKIE"

echo "=== 2. Creando 2 cursos ==="
COURSE1=$(curl -s -b "edutrace_session=$PROF_COOKIE" "$API/api/teacher/courses" \
  -H "Content-Type: application/json" \
  -d '{"name":"Introducción a la Programación 2026-1"}')
COURSE1_ID=$(echo "$COURSE1" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "  Curso 1: $COURSE1_ID"

COURSE2=$(curl -s -b "edutrace_session=$PROF_COOKIE" "$API/api/teacher/courses" \
  -H "Content-Type: application/json" \
  -d '{"name":"Programación Avanzada 2026-1"}')
COURSE2_ID=$(echo "$COURSE2" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
echo "  Curso 2: $COURSE2_ID"

echo "=== 3. Creando clases (grupos) ==="
GROUP1=$(curl -s -b "edutrace_session=$PROF_COOKIE" "$API/api/teacher/courses/$COURSE1_ID/groups" \
  -H "Content-Type: application/json" \
  -d '{"name":"Grupo A - Mañana"}')
GROUP1_ID=$(echo "$GROUP1" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
GROUP1_CODE=$(echo "$GROUP1" | python3 -c "import sys,json; print(json.load(sys.stdin)['join_code'])")
echo "  Grupo 1 (Intro Prog): $GROUP1_ID  Código: $GROUP1_CODE"

GROUP2=$(curl -s -b "edutrace_session=$PROF_COOKIE" "$API/api/teacher/courses/$COURSE2_ID/groups" \
  -H "Content-Type: application/json" \
  -d '{"name":"Grupo B - Tarde"}')
GROUP2_ID=$(echo "$GROUP2" | python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")
GROUP2_CODE=$(echo "$GROUP2" | python3 -c "import sys,json; print(json.load(sys.stdin)['join_code'])")
echo "  Grupo 2 (Prog Avanzada): $GROUP2_ID  Código: $GROUP2_CODE"
curl -s -b "edutrace_session=$PROF_COOKIE" -X PATCH "$API/api/teacher/groups/$GROUP2_ID/ai-mode" \
  -H "Content-Type: application/json" -d '{"ai_mode":"inmediata"}' > /dev/null
echo "  Grupo A en modo supervisado (predeterminado) · Grupo B en modo inmediato"

echo "=== 4. Registrando 5 estudiantes y asignándolos a clases ==="

register_and_join() {
  local name="$1" email="$2" join_code="$3" group_label="$4"
  local cookie
  cookie=$(curl -s -c - "$API/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"email\":\"$email\",\"password\":\"Alumno2026!\",\"role\":\"student\",\"consent\":true}" \
    | grep edutrace_session | awk '{print $NF}')
  curl -s -b "edutrace_session=$cookie" "$API/api/student/groups/join" \
    -H "Content-Type: application/json" \
    -d "{\"code\":\"$join_code\"}" > /dev/null
  echo "  ✓ $name ($email) → $group_label"
}

# Clase 1: Intro a la Programación (3 estudiantes)
register_and_join "Carlos Pérez"    "carlos.perez@est.edu.co"    "$GROUP1_CODE" "Intro Prog - Grupo A"
register_and_join "Ana Rodríguez"   "ana.rodriguez@est.edu.co"   "$GROUP1_CODE" "Intro Prog - Grupo A"
register_and_join "Luis Martínez"   "luis.martinez@est.edu.co"   "$GROUP1_CODE" "Intro Prog - Grupo A"

# Clase 2: Programación Avanzada (2 estudiantes)
register_and_join "Sofía López"     "sofia.lopez@est.edu.co"     "$GROUP2_CODE" "Prog Avanzada - Grupo B"
register_and_join "Diego Ramírez"   "diego.ramirez@est.edu.co"   "$GROUP2_CODE" "Prog Avanzada - Grupo B"

field() {
  python3 -c "import sys, json
d = json.load(sys.stdin)
if isinstance(d, dict) and 'error' in d:
    sys.exit('  ✗ ' + d['error'])
print(d[sys.argv[1]])" "$1"
}

login_cookie() {
  curl -s -c - "$API/api/auth/login" -H "Content-Type: application/json" \
    -d "{\"email\":\"$1\",\"password\":\"$2\"}" | grep edutrace_session | awk '{print $NF}'
}

create_assignment() {
  local extra=()
  for kv in "${@:5}"; do extra+=(--form-string "$kv"); done
  curl -s -b "edutrace_session=$PROF_COOKIE" "$API/api/teacher/groups/$1/assignments" \
    --form-string "title=$2" --form-string "statement=$3" --form-string "rubric=$4" "${extra[@]}" | field id
}

iso_utc() {
  date -u -d "$1" +%Y-%m-%dT%H:%M:%SZ
}

attempt() {
  python3 -c "import json, sys; print(json.dumps({'group_id': sys.argv[1], 'assignment_id': sys.argv[2], 'code': sys.argv[3], 'stdin': sys.argv[4]}))" "$2" "$3" "$4" "$5" \
    | curl -s -b "edutrace_session=$1" "$API/api/compile" -H "Content-Type: application/json" --data @- > /dev/null
}

submit() {
  python3 -c "import json, sys; print(json.dumps({'group_id': sys.argv[1], 'assignment_id': sys.argv[2], 'code': sys.argv[3], 'stdin': sys.argv[4], 'ai_feedback': '', 'ai_feedback_informal': ''}))" "$2" "$3" "$4" "$5" \
    | curl -s -b "edutrace_session=$1" "$API/api/student/submissions" -H "Content-Type: application/json" --data @- | field id
}

review() {
  python3 -c "import json, sys; print(json.dumps({'feedback': sys.argv[1], 'grade': float(sys.argv[2]), 'return': sys.argv[3] == 'true'}))" "$4" "$2" "$3" \
    | curl -s -b "edutrace_session=$PROF_COOKIE" -X PUT "$API/api/teacher/submissions/$1/review" -H "Content-Type: application/json" --data @- | field status
}

echo "=== 5. Publicando talleres en Intro a la Programación - Grupo A ==="
T1=$(create_assignment "$GROUP1_ID" "Taller 1: promedio de notas" \
  "Lee tres notas (de 0.0 a 5.0) y muestra su promedio con dos decimales.
Ejemplo: entrada 4.5 3.0 5.0 -> salida Promedio: 4.17" \
  "Compila (1.0) · Lee bien la entrada (1.0) · Promedio correcto (2.0) · Legibilidad (1.0)" \
  "due_at=$(iso_utc '+5 days')" "allow_late=true" \
  'tests=[{"name":"Ejemplo del enunciado","stdin":"4.5 3.0 5.0","expected":"Promedio: 4.17","match":"contiene","hidden":false},{"name":"Notas enteras","stdin":"4 5 5","expected":"Promedio: 4.67","match":"contiene","hidden":false},{"name":"Promedio bajo","stdin":"0 0 1","expected":"Promedio: 0.33","match":"contiene","hidden":true}]')
T2=$(create_assignment "$GROUP1_ID" "Taller 2: suma con ciclos" \
  "Lee un entero n y muestra la suma de 1 hasta n usando un ciclo.
Ejemplo: entrada 5 -> salida Suma: 15" \
  "Compila (1.0) · Usa un ciclo (1.0) · Resultado correcto (2.0) · Legibilidad (1.0)" \
  "due_at=$(iso_utc '+20 hours')" "allow_late=false" \
  'tests=[{"name":"Ejemplo del enunciado","stdin":"5","expected":"Suma: 15","match":"contiene","hidden":false},{"name":"n = 1","stdin":"1","expected":"Suma: 1","match":"contiene","hidden":false},{"name":"n = 10","stdin":"10","expected":"Suma: 55","match":"contiene","hidden":true}]')
T3=$(create_assignment "$GROUP1_ID" "Taller 3: el mayor de tres" \
  "Lee tres enteros y muestra el mayor usando condicionales.
Ejemplo: entrada 7 12 3 -> salida Mayor: 12" \
  "Compila (1.0) · Usa condicionales (1.0) · Resultado correcto (2.0) · Legibilidad (1.0)" \
  "due_at=$(iso_utc '-27 hours')" "allow_late=true" "late_penalty=0.5" \
  'tests=[{"name":"Ejemplo del enunciado","stdin":"7 12 3","expected":"Mayor: 12","match":"contiene","hidden":false},{"name":"Repetidos","stdin":"5 5 2","expected":"Mayor: 5","match":"contiene","hidden":false},{"name":"Negativos","stdin":"-3 -8 -1","expected":"Mayor: -1","match":"contiene","hidden":true}]')
echo "  ✓ 3 talleres con 3 casos de prueba cada uno (1 oculto): el 1 vence en 5 días, el 2 en 20 h (sin tardías) y el 3 venció hace 27 h (tardías con −0.5 por día)"

ANA_COOKIE=$(login_cookie "ana.rodriguez@est.edu.co" "Alumno2026!")
CARLOS_COOKIE=$(login_cookie "carlos.perez@est.edu.co" "Alumno2026!")
LUIS_COOKIE=$(login_cookie "luis.martinez@est.edu.co" "Alumno2026!")

CODE_PROMEDIO=$(cat <<'EOF'
#include <iostream>
#include <iomanip>
using namespace std;

int main() {
    double n1, n2, n3;
    cin >> n1 >> n2 >> n3;
    double promedio = (n1 + n2 + n3) / 3;
    cout << fixed << setprecision(2);
    cout << "Promedio: " << promedio << endl;
    return 0;
}
EOF
)

CODE_PROMEDIO_ENTERO=$(cat <<'EOF'
#include <iostream>
using namespace std;

int main() {
    int a, b, c;
    cin >> a >> b >> c;
    double promedio = (a + b + c) / 3;
    cout << "Promedio: " << promedio << endl;
    return 0;
}
EOF
)

CODE_SUMA_SIN_PUNTO=$(cat <<'EOF'
#include <iostream>
using namespace std;

int main() {
    int n;
    cin >> n;
    int suma = 0
    for (int i = 1; i < n; i++) {
        suma += i;
    }
    cout << "Suma: " << suma << endl;
    return 0;
}
EOF
)

CODE_SUMA=$(cat <<'EOF'
#include <iostream>
using namespace std;

int main() {
    int n;
    cin >> n;
    int suma = 0;
    for (int i = 1; i < n; i++) {
        suma += i;
    }
    cout << "Suma: " << suma << endl;
    return 0;
}
EOF
)

CODE_SUMA_OK=$(cat <<'EOF'
#include <iostream>
using namespace std;

int main() {
    int n;
    cin >> n;
    int suma = 0;
    for (int i = 1; i <= n; i++) {
        suma += i;
    }
    cout << "Suma: " << suma << endl;
    return 0;
}
EOF
)

CODE_MAYOR=$(cat <<'EOF'
#include <iostream>
using namespace std;

int main() {
    int a, b, c;
    cin >> a >> b >> c;
    int mayor = a;
    if (b > mayor) mayor = b;
    if (c > mayor) mayor = c;
    cout << "Mayor: " << mayor << endl;
    return 0;
}
EOF
)

echo "=== 6. Enviando entregas (se compilan y ejecutan en el sandbox) ==="
ANA_T1=$(submit "$ANA_COOKIE" "$GROUP1_ID" "$T1" "$CODE_PROMEDIO" "4.5 3.0 5.0")
attempt "$ANA_COOKIE" "$GROUP1_ID" "$T2" "$CODE_SUMA_SIN_PUNTO" "5"
attempt "$ANA_COOKIE" "$GROUP1_ID" "$T2" "$CODE_SUMA" "5"
ANA_T2=$(submit "$ANA_COOKIE" "$GROUP1_ID" "$T2" "$CODE_SUMA" "5")
ANA_T3=$(submit "$ANA_COOKIE" "$GROUP1_ID" "$T3" "$CODE_MAYOR" "7 12 3")
CARLOS_T1=$(submit "$CARLOS_COOKIE" "$GROUP1_ID" "$T1" "$CODE_PROMEDIO_ENTERO" "4 5 5")
LUIS_T2=$(submit "$LUIS_COOKIE" "$GROUP1_ID" "$T2" "$CODE_SUMA_OK" "5")
echo "  ✓ Ana: 3 entregas y 2 intentos · Carlos: 1 entrega · Luis: 1 entrega"
echo "  ✓ Casos de prueba: Ana pasa 3/3, 0/3 y 3/3 · Carlos 0/3 · Luis 3/3"
echo "  ✓ Las 5 entregas oficiales entraron a la cola: la IA prepara sus propuestas en segundo plano"

echo "=== 7. Calificando ==="
review "$ANA_T1" 4.5 true "Muy bien: lees las tres notas con \`double\` y usas paréntesis para sumar antes de dividir, por eso el promedio sale correcto (4.17 con 4.5, 3.0 y 5.0). La salida con dos decimales cumple el enunciado.

Para el 5.0: ¿qué debería pasar si alguien escribe una nota mayor que 5 o negativa? Piensa cómo validarlo antes de calcular." > /dev/null
review "$ANA_T2" 3.2 true "Compila y el acumulador \`suma\` empieza bien en 0. Con n = 5 tu programa imprime 10 y el enunciado espera 15.

Revisa la condición del ciclo de la línea 8: escribe a mano qué valores toma \`i\` en cada vuelta. ¿Se alcanza a sumar el 5?" > /dev/null
review "$ANA_T3" 4.0 false "Correcto con 7, 12 y 3. Pendiente revisar los casos con números repetidos antes de devolverla." > /dev/null
review "$LUIS_T2" 5.0 true "Excelente: el ciclo recorre de 1 hasta n incluido y el resultado es correcto. El código es claro y bien indentado." > /dev/null
echo "  ✓ Ana: 4.5 (Taller 1) y 3.2 (Taller 2) devueltas, Taller 3 en revisión · Luis: 5.0"

echo "=== 8. RAE de la materia y niveles de logro ==="
API="$API" python3 "$(dirname "$0")/seed_rae_levels.py"

echo ""
echo "======================================================"
echo "  ✅ Datos de prueba cargados exitosamente"
echo "======================================================"
echo ""
echo "  PROFESOR:"
echo "    Email:    maria.garcia@universidad.edu.co"
echo "    Password: Profesor2026!"
echo ""
echo "  ESTUDIANTES (Password: Alumno2026!):"
echo "    Clase 1 - Intro a la Programación / Grupo A:"
echo "      carlos.perez@est.edu.co"
echo "      ana.rodriguez@est.edu.co"
echo "      luis.martinez@est.edu.co"
echo "    Clase 2 - Programación Avanzada / Grupo B:"
echo "      sofia.lopez@est.edu.co"
echo "      diego.ramirez@est.edu.co"
echo ""
echo "  NOTAS (escala 0.0 a 5.0):"
echo "    ana.rodriguez@est.edu.co: Taller 1 = 4.5, Taller 2 = 3.2, Taller 3 en revisión (entregado tarde)"
echo "    luis.martinez@est.edu.co: Taller 2 = 5.0"
echo "    carlos.perez@est.edu.co:  Taller 1 = 2.8"
echo "  RAE: los 4 oficiales del curso 33698 en Introducción a la Programación, con niveles en cada revisión"
echo "  IA: Grupo A supervisado (el estudiante solo ve lo que aprueba el profesor) · Grupo B inmediato"
echo ""
