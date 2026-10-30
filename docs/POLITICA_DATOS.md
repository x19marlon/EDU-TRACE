# Política de datos y de uso de la IA (RNF-05)

Qué datos trata EduTrace, adónde van y quién los ve. La versión para estudiantes está en `/privacidad`; su versión vigente es `2026-10` (`auth.PrivacyVersion`).

## Datos que se guardan

| Dato | Dónde | Quién lo ve |
|---|---|---|
| Nombre, email, rol, fecha de la autorización | `users` | La persona; el profesor y los monitores de sus clases ven el nombre y el email |
| Contraseña y código de recuperación | `users`, solo el hash | Nadie |
| Código, entrada, salida, mensajes del compilador y resultados de los casos | `submissions`, `submission_tests` | El estudiante, el profesor de la clase y los monitores asignados |
| Retroalimentación, nota y niveles por RAE | `reviews`, `review_raes` | El profesor; el estudiante cuando se aprueba |
| Marcas y comentarios de monitores | `code_annotations`, `monitor_feedback` | El profesor; el estudiante cuando el monitor las envía |
| Bitácora de intervenciones | `audit_log` | El profesor de la clase |
| Tiempos de la IA | `ai_requests`, sin usuario | El profesor, en forma agregada por clase |

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

## Supervisión humana

- Cada clase empieza en modo **supervisado**: la IA prepara un borrador y el estudiante solo ve lo que el profesor aprueba.
- El modo **inmediato** es una decisión explícita del profesor y queda registrada en la bitácora.
- Toda salida de la IA indica que la generó una IA.
- Cada revisión devuelta indica si el texto lo propuso la IA, lo editó el profesor o lo escribió el profesor, y quién la aprobó.

## Derechos del titular

| Derecho | Dónde se ejerce |
|---|---|
| Conocer y obtener copia | «Mi cuenta» → «Descargar mis datos» (`GET /api/auth/export`) |
| Rectificar | «Mi cuenta» → «Mis datos» (`PATCH /api/auth/profile`) |
| Revocar y suprimir | «Mi cuenta» → «Eliminar mi cuenta» (`POST /api/auth/delete`). Borra en cascada entregas, revisiones, marcas, trabajos de la cola y entradas de la bitácora sobre sus entregas. Un profesor debe eliminar antes sus materias. |
| Prueba de la autorización | La fecha y la versión aceptada aparecen en «Mi cuenta» y en la descarga |

## Conservación

- Los datos viven mientras exista la cuenta y la clase. Al terminar el periodo, el profesor elimina la clase y con ella sus entregas.
- `data/` y los respaldos no se suben al repositorio (`.gitignore`: `/data/` y `/data-backup-*/`).
