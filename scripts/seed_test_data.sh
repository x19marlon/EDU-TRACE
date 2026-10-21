#!/usr/bin/env bash
# Carga usuarios de prueba, cursos y clases en EDU-TRACE
set -e
API="http://127.0.0.1:8080"

echo "=== 1. Registrando profesor ==="
PROF_COOKIE=$(curl -s -c - "$API/api/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"name":"Prof. María García","email":"maria.garcia@universidad.edu.co","password":"Profesor2026!","role":"teacher","teacher_code":"admin123"}' \
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

echo "=== 4. Registrando 5 estudiantes y asignándolos a clases ==="

register_and_join() {
  local name="$1" email="$2" join_code="$3" group_label="$4"
  local cookie
  cookie=$(curl -s -c - "$API/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"$name\",\"email\":\"$email\",\"password\":\"Alumno2026!\",\"role\":\"student\"}" \
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
