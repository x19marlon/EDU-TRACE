# Contexto de Sesión y Avance de Proyecto - EDU-TRACE

## 1. Script de Inicio Automatizado (`start.sh`)

- Valida e instala dependencias del sistema (g++, bwrap, util-linux, psmisc).
- Descarga y configura automáticamente Go 1.24 en `~/.local/go1.24` si el sistema tiene una versión inferior (requerido por `crypto/pbkdf2`).
- Detecta restricciones de user namespaces en Ubuntu 24.04 y conmuta a `SANDBOX=none` en local si es necesario.
- Compila el backend y ejecuta Backend (:8080), Frontend (:3000) y Ollama (:11434) de forma coordinada.

## 2. Nueva Funcionalidad: Tutor Chat Interactivo con IA

- Endpoint backend: `POST /api/chat` con streaming SSE (ver `internal/handler/routes.go`).
- System prompt pedagógico estricto: método socrático. Prohibido resolver o dar correcciones directas de la tarea del alumno; sólo ofrece pistas y ejemplos genéricos conceptuales.
- Frontend: pestaña "Tutor Chat IA" en el panel principal con soporte y renderizado tipográfico completo de Markdown (tablas, títulos, código resaltado).

## 3. Requisitos No Funcionales (RNF-01 al RNF-16)

- Matriz completa documentada en `docs/RNF_COMPLIANCE.md`.
- RNF-01 / RNF-02: aviso legal de tratamiento de datos en registro (`/login`).
- RNF-03 a RNF-07: inferencia 100% local (sin fugas a APIs externas), anonimización de datos en prompts y retroalimentación en modo "draft" (borrador) hasta aprobación docente.
- RNF-14 / RNF-15: conservación de entregas ante fallo de IA y alerta informativa en el panel de revisión.

## 4. Usuarios de Prueba Cargados (`data/edutrace.json`)

Script de recarga: `scripts/seed_test_data.sh` (detalle completo en `docs/TEST_DATA.md`)

- Docente: `maria.garcia@universidad.edu.co` / `Profesor2026!`
- Alumnos (password: `Alumno2026!`):
  - Clase 1 (Intro a la Programación - Grupo A):
    - `carlos.perez@est.edu.co`
    - `ana.rodriguez@est.edu.co`
    - `luis.martinez@est.edu.co`
  - Clase 2 (Programación Avanzada - Grupo B):
    - `sofia.lopez@est.edu.co`
    - `diego.ramirez@est.edu.co`
