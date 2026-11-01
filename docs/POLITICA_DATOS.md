# Política de datos y de uso de la IA (RNF-05)

Qué datos trata EduTrace, adónde van y quién los ve. La versión para estudiantes está en `/privacidad`; su versión vigente es `2026-11` (`auth.PrivacyVersion`). Cuando cambia la versión, cada cuenta debe volver a autorizar al entrar y ve un resumen de los cambios.

## Datos que se guardan

| Dato | Dónde | Quién lo ve |
|---|---|---|
| Nombre, email, rol, fecha de la autorización | `users` | La persona; los profesores de la materia (el dueño y quienes la comparten) y los monitores de sus clases ven el nombre y el email |
| Contraseña, código de recuperación y código que genera el profesor (vence en 24 h) | `users`, solo el hash | Nadie |
| Código, entrada, salida, mensajes del compilador y resultados de los casos | `submissions`, `submission_tests` | El estudiante, el profesor de la clase y los monitores asignados |
| Retroalimentación, nota y niveles por RAE | `reviews`, `review_raes` | El profesor; el estudiante cuando se aprueba |
| Marcas y comentarios de monitores | `code_annotations`, `monitor_feedback` | El profesor; el estudiante cuando el monitor las envía |
| Bitácora de intervenciones | `audit_log` | El profesor de la clase |
| Preguntas sobre la retroalimentación y sus respuestas | `review_questions` | El estudiante, el profesor de la clase y sus monitores |
| «¿Entendiste esta retroalimentación?» y su comentario | `feedback_ratings` | El estudiante; el profesor solo ve el resumen y los comentarios, sin nombres |
| Tiempos de la IA, con el modelo y la versión del prompt | `ai_requests`, sin usuario | El profesor, en forma agregada por clase |
| Modelo y versión del prompt de cada propuesta | `reviews.ai_model`, `reviews.ai_prompt` | El profesor |
| Avisos («tu profesor devolvió la revisión de…») y si se leyeron o se enviaron por correo | `notifications` | La persona |
| Preferencia de avisos por correo | `users.email_notifications` | La persona |
| Resultados de la práctica (intentos y casos que pasa por ejercicio) | `practice_results` | La persona |
| Puntajes por criterio de la rúbrica | `review_criteria` | El profesor; el estudiante cuando se devuelve la revisión |
| Casos de prueba propios del estudiante | Solo en su navegador (`localStorage`) | La persona |

## Qué sale del sistema

**Nada sale del servidor.** El modelo corre en Ollama, en la misma máquina o en la red institucional (`OLLAMA_BASE_URL`). EduTrace no llama a servicios de IA externos.

Al modelo solo se envía:
- el código, numerado por líneas;
- la entrada y la salida del programa, y los mensajes del compilador;
- el enunciado y los criterios del taller, y los RAE de la materia;
- el resultado de los casos de prueba. De los casos ocultos solo va «no pasa», sin su contenido.

Nunca se envía:
- el nombre, el email ni el identificador del estudiante;
- notas anteriores ni datos de otros estudiantes. El análisis del proceso tampoco incluye el nombre (`internal/handler/analysis.go`).

Si se cambia `OLLAMA_BASE_URL` a un servicio fuera de la institución, esta política deja de cumplirse. Hay que actualizarla y pedir una autorización nueva.

Las alertas tempranas, la detección de código similar y el reporte por estudiante se calculan al consultarlos con los datos anteriores; no se guardan aparte. Son apoyo para el profesor, no decisiones automáticas.

## Avisos por correo

Si el servidor tiene SMTP (`SMTP_HOST`, `SMTP_FROM`), un aviso que nadie abre en 2 minutos se envía también por correo, salvo que la persona lo haya desactivado en «Mi cuenta» → «Avisos». El correo solo dice qué pasó, en qué taller, y trae el enlace: nunca incluye código, notas ni el texto de la retroalimentación. Con `SMTP_PORT=587` el envío usa STARTTLS, y con `465`, TLS directo. La contraseña de SMTP nunca viaja sin cifrar.

## Supervisión humana

- Cada clase empieza en modo **supervisado**: la IA prepara un borrador y el estudiante solo ve lo que el profesor aprueba.
- El modo **inmediato** es una decisión explícita del profesor y queda registrada en la bitácora.
- En modo inmediato cada estudiante tiene un máximo de `AI_REQUESTS_PER_HOUR` pedidos a la IA por hora (20 por defecto).
- Un taller en **modo evaluación** apaga la IA y el chat para los estudiantes de la clase desde que se crea hasta su fecha límite (o el cierre de entregas tardías).
- Toda salida de la IA indica que la generó una IA.
- Cada revisión devuelta indica si el texto lo propuso la IA, lo editó el profesor o lo escribió el profesor, y quién la aprobó.

## Derechos del titular

| Derecho | Dónde se ejerce |
|---|---|
| Conocer y obtener copia | «Mi cuenta» → «Descargar mis datos» (`GET /api/auth/export`) |
| Rectificar | «Mi cuenta» → «Mis datos» (`PATCH /api/auth/profile`) |
| Revocar y suprimir | «Mi cuenta» → «Eliminar mi cuenta» (`POST /api/auth/delete`). Borra en cascada entregas, revisiones, marcas, preguntas, valoraciones, trabajos de la cola y entradas de la bitácora sobre sus entregas. Si la cuenta era de un monitor o un profesor, su nombre en la bitácora y en las respuestas se reemplaza por «Cuenta eliminada». Un profesor debe eliminar antes sus materias. |
| Prueba de la autorización | La fecha y la versión aceptada aparecen en «Mi cuenta» y en la descarga |

## Conservación

| Qué | Cuándo se borra |
|---|---|
| Materia, con sus clases, talleres, archivos, entregas, revisiones, preguntas, valoraciones y bitácora | `RETENTION_DAYS` días (180 por defecto) después de que el profesor cierra el periodo. Al cerrarlo, los estudiantes reciben un aviso con la fecha. El profesor puede reabrirlo antes y entonces no se borra. |
| Cuenta de estudiante o monitor sin clases | Cuando pasan `RETENTION_DAYS` días sin actividad. Las cuentas de profesor nunca se borran solas. |
| Avisos | A los 90 días. |
| Tiempos de la IA (sin usuario) | Al año. |
| Sesiones | Al vencer. |

- El borrado lo hace el backend cada 6 horas (`Store.Purge`). Con `RETENTION_DAYS=0` no se borra nada automáticamente.
- Los respaldos diarios (`scripts/respaldar.sh`) guardan los últimos `BACKUP_KEEP` (14 por defecto). Por eso, un dato borrado sigue en los respaldos hasta 14 días más.
- `data/` y los respaldos no se suben al repositorio (`.gitignore`: `/data/`, `/data-backup-*/` y `/respaldos/`).
