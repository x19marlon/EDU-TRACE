# Integración con la plataforma institucional (RNF-13)

**Estado: pendiente.** La integración de EduTrace con la plataforma institucional se hará más adelante.

## Plataforma institucional

La institución usa **Campus Virtual**, que funciona sobre **Brightspace (D2L)**. En su configuración actual no se pueden importar notas desde archivos CSV, así que exportar a CSV no sirve como integración.

Mientras tanto, EduTrace funciona por separado: el estudiante trabaja y recibe la retroalimentación en EduTrace, y las notas se pasan a Campus Virtual a mano.

## Lo que ya existe en EduTrace

En la clase, los botones «Exportar notas (CSV)» y «Exportar revisiones (CSV)» descargan las notas y las revisiones devueltas. Sirven como registro del profesor y para abrir en Excel o Google Sheets, pero **no** para cargar en Campus Virtual.

| Archivo | Una fila por | Columnas |
|---|---|---|
| `notas-<materia>-<clase>.csv` | Estudiante | Nombre, Dirección de correo, una columna por taller con la nota de la última entrega devuelta, Promedio |
| `revisiones-<materia>-<clase>.csv` | Revisión devuelta | Nombre, Dirección de correo, Taller, Entregada, Retraso (horas), Casos que pasa, Nota, Nota propuesta por la IA, Devuelta, Origen del texto, Modelo de IA, Minutos de revisión, Retroalimentación |

## Integración que se va a buscar

Opciones que habrá que evaluar con quienes administran Campus Virtual:

- **LTI 1.3 (LTI Advantage):** EduTrace se registra como herramienta externa en Brightspace. El estudiante y el profesor entran desde el curso sin otra contraseña, la lista de inscritos llega sola (*Names and Role Provisioning Services*) y las notas aprobadas se devuelven al libro de calificaciones (*Assignment and Grade Services*).
- **API de Brightspace (Valence):** EduTrace consulta los cursos y publica las notas con credenciales de una aplicación registrada en la institución.

En ambos casos se necesita que la institución registre EduTrace en Brightspace y entregue las credenciales correspondientes (ID de cliente, ID de despliegue y claves).
