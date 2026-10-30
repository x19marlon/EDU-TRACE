# Integración con la plataforma institucional (RNF-13)

EduTrace no reemplaza el LMS de la institución: el estudiante trabaja y recibe retroalimentación en EduTrace, y las notas se llevan al LMS con un archivo CSV.

## Qué se exporta

En la clase, botones «Exportar notas (CSV)» y «Exportar revisiones (CSV)». Solo se exportan revisiones aprobadas y devueltas; los borradores no.

| Archivo | Una fila por | Columnas |
|---|---|---|
| `notas-<materia>-<clase>.csv` | Estudiante | Nombre, Dirección de correo, una columna por taller con la nota de la última entrega devuelta, Promedio |
| `revisiones-<materia>-<clase>.csv` | Revisión devuelta | Nombre, Dirección de correo, Taller, Entregada, Retraso (horas), Casos que pasa, Nota, Nota propuesta por la IA, Devuelta, Origen del texto, Minutos de revisión, Retroalimentación |

Formato del archivo:
- UTF-8 con BOM, para que Excel muestre bien las tildes.
- Separado por comas, con saltos de línea CRLF.
- Notas de 0 a 5 con punto decimal.
- Las celdas de texto que empiezan con `=`, `+`, `-` o `@` llevan un apóstrofo delante, para que una hoja de cálculo no las ejecute como fórmula.

## Moodle

1. Crea en el libro de calificaciones un ítem por taller con calificación máxima 5.
2. Ve a «Calificaciones» → «Importar» → «Archivo CSV» y sube `notas-….csv` (separador: coma; codificación: UTF-8).
3. En «Mapear desde» elige «Dirección de correo» y en «Mapear a» elige «Dirección de correo».
4. Asigna cada columna de taller a su ítem e ignora la columna «Promedio».

## Canvas

Canvas solo importa su propia plantilla. Por eso:
1. Exporta el libro de calificaciones de Canvas.
2. Copia las notas desde `notas-….csv`, buscándolas por correo (por ejemplo, con `BUSCARV`).
3. Vuelve a importar el archivo en Canvas.

## Google Sheets o Excel

Abre el archivo directamente. En Google Sheets: «Archivo» → «Importar» → «Subir», con separador «coma».

## Pendiente

LTI 1.3 permitiría entrar a EduTrace desde el LMS sin otra contraseña y devolver las notas automáticamente. Requiere registrar EduTrace como herramienta en el LMS de la institución.
