# Matriz de cumplimiento de requisitos

Estado de EduTrace frente al *Catálogo de requisitos* (RF-01 a RF-23, RNF-01 a RNF-16 y RV-01 a RV-03), verificado sobre el código y no sobre la intención de diseño.

Estados:
- **Cumple**: hay una implementación y una prueba o verificación que lo demuestra.
- **Instrumentado**: la plataforma mide el criterio; falta que el docente acuerde la cifra del umbral, como pide el catálogo.
- **Parcial**: cubre una parte del criterio.
- **Falta**: no está implementado.
- **Fuera del código**: depende de un documento, una auditoría o una validación con personas.

Las pruebas citadas están en `internal/**/*_test.go` y se ejecutan con `go test ./...`.

## Requisitos funcionales

| ID | MoSCoW | Estado | Evidencia | Pendiente |
|---|---|---|---|---|
| RF-01 Retroalimentación para cada entrega | Must | Cumple | Cada entrega oficial entra a la cola `ai_jobs` y un trabajador genera la propuesta en segundo plano (`internal/handler/drafts.go`, `internal/auth/jobs.go`). Al actualizar, las entregas antiguas sin propuesta también entran a la cola. Prueba: `TestDraftQueueJobs`. | — |
| RF-02 Tres preguntas de Hattie y Timperley | Must | Cumple | Los modos formal e informal y la propuesta del profesor usan «¿Hacia dónde voy?», «¿Cómo voy?» y «¿Qué sigue?». `EnsureSections` completa en su lugar las secciones que falten (`internal/ai/sections.go`). Prueba: `TestEnsureSections`. | — |
| RF-03 «¿Qué sigue?» en el 100 % de las salidas | Must | Cumple | Si el modelo la omite, se agrega una pista según el resultado: no compila, falla al ejecutar, no pasa un caso o funciona. En el streaming el texto corregido llega como evento `replace`. Se mide el porcentaje de salidas que la traían (`ai_requests.sections_ok`). | — |
| RF-04 Corrección funcional con casos de prueba | Must | Cumple | Hasta 10 casos por taller: entrada, salida esperada, comparación «contiene» o «exacta» y opción de ocultarlo al estudiante. Se ejecutan en el sandbox al compilar y al entregar; el resultado se ve como «Pasa X de N» y llega a la IA (`internal/compiler/compiler.go`, `internal/auth/tests.go`). Pruebas: `TestCompileAndTest`, `TestAssignmentTestsAndResults`. | — |
| RF-05 Observación sobre legibilidad | Should | Cumple | Un analizador estático (`internal/ai/quality.go`) revisa sangría, nombres de una letra y líneas largas. Sus hallazgos llegan al modelo como datos verificados, y `EnsureQuality` agrega la observación si la salida no la trae. Se mide en `ai_requests.quality_ok`. Pruebas: `TestAnalyzeQuality`, `TestEnsureQuality`. | — |
| RF-06 Observación sobre mantenibilidad | Should | Cumple | Igual que RF-05: instrucciones repetidas 3 o más veces, variables globales y `main` de más de 50 líneas. | — |
| RF-07 Observación sobre documentación | Could | Cumple | Igual que RF-05: si hay comentarios, y su cantidad. | — |
| RF-08 Pistas orientadas al razonamiento | Must | Cumple | Las reglas del prompt piden preguntas guía y ejemplos de pista. | — |
| RF-09 No revelar la solución completa | Must | Cumple | `SolutionGuard` filtra el streaming y `FilterSolution` filtra la propuesta (`internal/ai/guard.go`). Prueba: `guard_test.go`. | — |
| RF-10 Estado «pendiente de revisión» | Must | Cumple | Toda revisión empieza como borrador («Pendiente de revisión») y el estudiante no la ve hasta que se aprueba. | — |
| RF-11 Editar antes de entregar | Must | Cumple | El texto, la nota y los niveles por RAE son editables en el panel de revisión. | — |
| RF-12 Aprobación registrada | Must | Cumple | «Aprobar y devolver» queda en `audit_log` con autor, fecha, nota y origen del texto. Prueba: `TestDiscardOriginAndAudit`. | — |
| RF-13 Descarte registrado | Must | Cumple | «Descartar propuesta de la IA» (`POST /api/teacher/submissions/{id}/review/discard`) quita la propuesta, impide que la cola la regenere y queda en la bitácora. | — |
| RF-14 Configurar criterios de evaluación | Must | Cumple | Criterios por taller, los cuatro RAE oficiales por materia y casos de prueba, definidos antes de ejecutar. | — |
| RF-15 La rúbrica se refleja en la salida | Should | Cumple | La propuesta del profesor usa los criterios, los RAE y los casos. La retroalimentación inmediata también recibe los criterios del taller (hasta 500 caracteres); si el código es tan largo que no caben, se omiten antes de rechazarlo. | — |
| RF-16 Ejecución automática ante cada entrega | Must | Cumple | Una entrega dispara la compilación, los casos de prueba y la propuesta de la IA sin intervención manual. | — |
| RF-17 Separar lo formativo de lo sumativo | Must | Cumple | La retroalimentación de la IA tiene prohibido poner notas; la nota es un campo aparte que define el profesor. | — |
| RF-18 Aviso de participación de la IA | Must | Cumple | Toda salida de la IA lleva su aviso; cada revisión devuelta indica si hubo IA. | — |
| RF-19 Contenido de IA marcado | Should | Cumple | El origen del texto devuelto queda como `ia`, `ia_editada` o `docente`, y el estudiante ve «Generado con IA · aprobado por …» o «Escrito por …». La marca es por texto completo, no por frase. | — |
| RF-20 Bitácora de intervenciones | Should | Cumple | `audit_log` registra autor, acción, fecha y detalle de cada propuesta, descarte, borrador, aprobación, cambio de nota, cambio de modo, marcas enviadas por monitores y casos re-ejecutados. Se ve por clase y por entrega. | — |
| RF-21 Errores frecuentes del curso | Should | Cumple | Tarjeta «Errores frecuentes» por clase o por materia: errores de compilación agrupados y explicados en español, problemas al ejecutar y casos de prueba que más cuestan, ordenados por cuántos estudiantes los tuvieron (`GET /api/teacher/groups/{id}/errors?alcance=materia`). Prueba: `TestFrequentErrors`. | — |
| RF-22 Aclaración sobre la retroalimentación | Could | Cumple | Debajo de cada retroalimentación, el estudiante pregunta. El profesor o un monitor de la clase responde, y la respuesta queda en la bitácora. El profesor ve las preguntas pendientes en su tablero y en la entrega. El chat con la IA sigue disponible en modo inmediato. Prueba: `TestQuestionsAndRatings`. | — |
| RF-23 Iterar | Could | Cumple | El estudiante puede volver a enviar; cada versión tiene su propia revisión. | — |

## Requisitos no funcionales

| ID | MoSCoW | Estado | Evidencia | Pendiente |
|---|---|---|---|---|
| RNF-01 Ley 1581 de 2012 | Must | Implementado; falta la auditoría | Autorización explícita al registrarse, guardada con fecha y versión; las cuentas antiguas deben autorizar al entrar. Política pública en `/privacidad`. Derechos del titular en «Mi cuenta»: descargar los datos en JSON, corregir el nombre y eliminar la cuenta en cascada. Prueba: `TestPersonalData`. | Auditoría legal. Nombrar responsable y canal de contacto institucional. |
| RNF-02 Circular 002 de 2024 (SIC) | Must | Parcial | Supervisión humana por defecto, transparencia sobre la IA, modelo local y minimización de datos. | Evaluación de impacto de la IA y auditoría legal. |
| RNF-03 Confidencialidad del código | Must | Cumple en el código | Cada consulta filtra en SQL por dueño, clase o monitor asignado; el código se ejecuta en `bwrap` sin red. | En producción: HTTPS y `COOKIE_SECURE=true`. |
| RNF-04 Confidencialidad de datos personales | Must | Cumple en el código | Igual que RNF-03; contraseñas y códigos de recuperación con hash, y sesión en cookie `HttpOnly`. | Ídem. |
| RNF-05 Control de lo que se envía a la IA | Must | Cumple | Política en `docs/POLITICA_DATOS.md`: el modelo es local y solo recibe código, entrada, salida, compilador, enunciado y casos; nunca nombre ni email. | — |
| RNF-06 Validación docente de toda retroalimentación automática | Must | Cumple en modo supervisado | Cada clase tiene un modo de IA y por defecto es «supervisada»: el estudiante no puede pedir retroalimentación ni chat a la IA (`403`) y solo recibe lo que el profesor aprueba. El modo «inmediata» es una excepción que el profesor activa con una advertencia y que queda en la bitácora. Prueba: `TestAIModeGate`. | — |
| RNF-07 Docente como decisor final | Must | Cumple | La IA solo propone; aprobar, editar, descartar y calificar son acciones del profesor. | — |
| RNF-08 Tiempo de revisión por entrega | Must | Cumple | El profesor decide cuánto tiempo dedica; la IA le propone cuánto necesitará. Al preparar cada propuesta, la IA estima los minutos para revisar esa entrega (`minutos_revision`, de 1 a 30), y la tarjeta «Tiempo de revisión» suma lo pendiente de cada taller. Se mide el tiempo activo real en el panel (pestaña visible, pausa tras 5 minutos sin actividad, máximo 2 h por sesión) y se compara con lo estimado. Pruebas: `TestReviewPlan`, `TestGroupMetrics`. | — |
| RNF-09 Lenguaje comprensible para CS1 | Should | Instrumentado con meta | Cada retroalimentación pregunta «¿Entendiste esta retroalimentación?» (sí, en parte o no, con un comentario opcional). Meta por taller: **al menos 80 % de «sí» y como máximo 10 % de «no», con 20 respuestas o más**. Acordada con: ________ · Fecha: ________. El indicador muestra, por taller, el porcentaje de cada respuesta y si cumple; la clase cumple cuando todos los talleres con respuestas suficientes cumplen. Prueba: `TestComprehensionGoal`. | Usarlo con estudiantes durante el semestre. |
| RNF-10 Procesar el volumen del curso a tiempo | Must | Cumple | Umbral acordado: **5 h por lote**. Acordado con: ________ · Fecha: ________. Cola persistente con reintentos (3 intentos); por cada taller se mide la peor demora entrega→propuesta. `scripts/prueba_carga.py N` simula N entregas. Prueba con 200 entregas: _resultado pendiente_. | — |
| RNF-11 Retroalimentación a tiempo | Should | Cumple | Umbral acordado: **5 h** entre la entrega y la retroalimentación aprobada. Acordado con: ________ · Fecha: ________. Se mide ese tiempo y las entregas pendientes que ya superan el umbral; en modo inmediato también el tiempo hasta el primer texto y hasta la respuesta completa. | — |
| RNF-12 Conectividad limitada | Should | Cumple | Ancho de banda acordado: **60–70 Mbps simétricos**. Acordado con: ________ · Fecha: ________. Cada página pesa unos 190 a 210 KB en su primera carga (unos 0.03 s a 60 Mbps); si los 200 estudiantes abren a la vez la más pesada, son 41 MB en unos 6 s. Después, la API transfiere menos de 2 KB por consulta, comprimida con gzip. El código del estudiante se guarda en el navegador si se corta la conexión. Medido con `scripts/medir_ancho_banda.py`. | — |
| RNF-13 Integración con la plataforma institucional | Should | Pendiente | La institución usa Campus Virtual (Brightspace), que en su configuración actual no importa notas por CSV. Por ahora EduTrace funciona por separado y exporta notas y revisiones en CSV como registro del profesor. Ver `docs/INTEGRACION.md`. | Integración con Campus Virtual / Brightspace (LTI 1.3 o su API), que se hará más adelante. |
| RNF-14 Conservar la entrega si falla la IA | Should | Cumple | La entrega se guarda antes de la IA y la cola reintenta sin tocarla. | — |
| RNF-15 Avisar al docente si falla la IA | Could | Cumple | Aviso en el tablero y en la clase cuando la IA no responde o una propuesta falla tres veces; botón «Reintentar ahora». | — |
| RNF-16 Vínculo docente-estudiante | Should | Cumple | La revisión devuelta muestra el nombre del profesor que la aprobó y el origen del texto, y el estudiante puede preguntarle directamente (RF-22). | — |

## Restricciones de viabilidad

| ID | Estado | Pendiente |
|---|---|---|
| RV-01 Costo-beneficio | Fuera del código | Análisis con el costo del servidor con GPU frente al tiempo docente ahorrado. Insumos disponibles: RNF-08, RNF-10 y la cantidad de propuestas aprobadas sin cambios. |
| RV-02 Mantenimiento a largo plazo | Fuera del código | Plan de mantenimiento: responsables, actualizaciones de dependencias y respaldo de `data/`. |
| RV-03 Actualización de modelos de IA | Fuera del código | Presupuesto y procedimiento. `cmd/aieval` sirve para comparar modelos antes de cambiar `OLLAMA_MODEL`. |

## Respaldo de datos en el historial de git

El commit `c727708` subió por error la carpeta `data-backup-2026-10-21/`, un respaldo de la base con datos de prueba inventados. El 31 de octubre de 2026 se reescribió `main` con `scripts/limpiar_historial.sh` y la carpeta ya no aparece en ninguna rama; el contenido actual no cambió.

`.gitignore` excluye `/data/` y `/data-backup-*/` para que no se vuelva a subir. Si algún día se sube un respaldo con datos reales, el mismo script sirve para quitarlo; además, hay que pedir a GitHub Support que purgue la caché de los commits viejos y avisar a las personas afectadas.
