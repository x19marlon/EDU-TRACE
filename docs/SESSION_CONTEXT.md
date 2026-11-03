# Contexto del proyecto para nuevas sesiones

Resumen para retomar el trabajo. El detalle está en el [README](../README.md); credenciales y datos de prueba en [TEST_DATA.md](TEST_DATA.md).

## Cómo correrlo

- Linux: `./start.sh`. Windows, solo para desarrollo y sin sandbox: `./startW.sh` en Git Bash.
- Datos de prueba: `./scripts/seed_test_data.sh`; para llenar el gráfico, además `python3 scripts/seed_activity.py`.
- Pruebas: `go test ./...` y `cd web && npx tsc --noEmit && npm run lint && npm run build`.

## Funcionalidades recientes

- **Ejemplos de sintaxis** (`internal/examples`, `ClassExamplesCard`, `/ejemplos`): catálogo de 48 ejemplos. El docente los activa por clase y los estudiantes solo ven los activos. Probarlos en el compilador no crea intentos ni toca el borrador.
- **Gráfico Nota vs. esfuerzo** (`internal/auth/performance.go`, `PerformanceChart`): 9 cuadrantes más 3 situaciones sin nota, con cómo va y qué necesita cada estudiante. Solo cuentan las notas del docente, no la propuesta de la IA sin revisar.
- **Documentos por materia** (`syllabus/`, `internal/ai/syllabus.go`): la IA recibe los documentos de la materia del curso de la clase. Solo existe `intro-programacion/`; un curso que no coincide con una materia recibe un aviso de que no hay documentos.
- **Fallas de ejecución** (`crashMessage` en `internal/compiler/compiler.go`): abort, segfault y división entre cero tienen código de salida y explicación.

## Pendientes

- Confirmar en un servidor con `bwrap` los ejemplos de archivos y los códigos de salida de las fallas.
- Probar `startW.sh` en un Windows real.
- Agregar los documentos de Programación Avanzada en `syllabus/`.
- Kubernetes: el backend usa SQLite, así que no escala a varias réplicas sin cambiar la base; primer paso propuesto, Dockerfiles y manifiestos en `deploy/k8s/`.
