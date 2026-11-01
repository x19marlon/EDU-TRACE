# Datos de prueba

Cuentas, cursos y clases de prueba para desarrollo local. **Solo para entornos locales**: no uses estas credenciales en un despliegue real.

## Cómo cargarlos

`data/` está en `.gitignore`, así que cada persona debe cargar los datos en su propia instancia:

1. Levanta la aplicación con `./start.sh` (exporta `TEACHER_CODE=admin123` si no está definido).
2. Con el backend corriendo en `http://127.0.0.1:8080`, ejecuta:

   ```bash
   ./scripts/seed_test_data.sh
   ```

El script registra al docente, crea los cursos y clases, registra e inscribe a los alumnos, publica tres talleres, envía entregas (se compilan de verdad en el sandbox) y las califica. Al final imprime los códigos de ingreso de cada clase (se generan al azar en cada carga).

Para cargar en otro backend: `API=http://127.0.0.1:8092 ./scripts/seed_test_data.sh`.

> El script registra al docente con el `TEACHER_CODE` de tu `.env` (o `admin123` si no hay), el mismo que usa `./start.sh`. Ejecútalo sobre una base limpia: si los correos ya existen, el registro devuelve error. Las credenciales quedan resumidas en `CREDENCIALES.txt` (en la raíz, fuera de git).

## Docente

| Nombre | Email | Contraseña |
|---|---|---|
| Prof. María García | `maria.garcia@universidad.edu.co` | `Profesor2026!` |

Para iniciar sesión como docente, elige la pestaña **Profesor** en la pantalla de login.

## Cursos y clases

| Curso | Clase | Alumnos | IA para el estudiante |
|---|---|---|---|
| Introducción a la Programación 2026-1 | Grupo A - Mañana | 3 | Supervisada (predeterminada) |
| Programación Avanzada 2026-1 | Grupo B - Tarde | 2 | Inmediata |

- **Supervisada**: el estudiante no puede pedir retroalimentación ni chat a la IA. La IA prepara una propuesta de cada entrega y el estudiante solo ve lo que la profesora aprueba (RNF-06).
- **Inmediata**: el estudiante pide retroalimentación y chat a la IA mientras practica.

El modo se cambia en la clase, en la tarjeta «Retroalimentación con IA».

## Alumnos

Todos usan la contraseña `Alumno2026!`.

| Nombre | Email | Clase |
|---|---|---|
| Carlos Pérez | `carlos.perez@est.edu.co` | Intro a la Programación - Grupo A |
| Ana Rodríguez | `ana.rodriguez@est.edu.co` | Intro a la Programación - Grupo A |
| Luis Martínez | `luis.martinez@est.edu.co` | Intro a la Programación - Grupo A |
| Sofía López | `sofia.lopez@est.edu.co` | Programación Avanzada - Grupo B |
| Diego Ramírez | `diego.ramirez@est.edu.co` | Programación Avanzada - Grupo B |

## Talleres y notas

En *Intro a la Programación - Grupo A* hay tres talleres. Las notas van de 0.0 a 5.0.

| Alumno | Taller | Resultado en el sandbox | Casos de prueba | Estado |
|---|---|---|---|---|
| Ana Rodríguez | Taller 1: promedio de notas | `Promedio: 4.17` | 3/3 | **4.5**, devuelta |
| Ana Rodríguez | Taller 2: suma con ciclos | `Suma: 10` (esperado 15) | 0/3 | **3.2**, devuelta |
| Ana Rodríguez | Taller 3: el mayor de tres | `Mayor: 12` | 3/3 | En revisión (borrador con 4.0, no devuelta); **tardía** ~27 h |
| Luis Martínez | Taller 2: suma con ciclos | `Suma: 15` | 3/3 | **5.0**, devuelta |
| Carlos Pérez | Taller 1: promedio de notas | `Promedio: 4` (división entera) | 0/3 | **2.8**, devuelta |

Cada taller tiene tres casos de prueba: el ejemplo del enunciado, otro visible y uno oculto. Los casos se ejecutan al compilar con el taller elegido y al entregar; el estudiante ve cuáles pasa y, de los ocultos, solo si pasan.

Las cinco entregas entran a la cola de la IA al enviarse. Con Ollama encendido, la propuesta del Taller 3 de Ana aparece en su revisión en menos de un minuto. Las que la profesora ya devolvió quedan como «omitidas».

Fechas límite (relativas al momento de cargar los datos): el Taller 1 vence en 5 días; el Taller 2 vence en 20 horas y no acepta entregas tardías; el Taller 3 venció 27 horas antes y acepta tardías con una penalización sugerida de 0.5 por día iniciado (la entrega de Ana queda tardía, 2 días iniciados → −1.0 sugerido).

Ana también tiene dos intentos de compilación en el Taller 2 (uno sin compilar por un `;` faltante), visibles en su proceso desde la vista del profesor.

- **Como Ana** (`ana.rodriguez@est.edu.co`):
  - En «Mis entregas» ve sus dos notas con la retroalimentación de la profesora, con su nombre y si hubo IA, y la tercera entrega «En revisión».
  - Los botones de IA aparecen deshabilitados porque su clase es supervisada.
- **Como Sofía** (`sofia.lopez@est.edu.co`): su clase es inmediata, así que puede pedir retroalimentación formal o informal y usar el chat.
- **Como profesora** (`maria.garcia@universidad.edu.co`), en *Grupo A - Mañana* ve:
  - las cinco entregas con sus casos y su estado;
  - el modo de IA y la cola;
  - los indicadores RNF-08, RNF-10 y RNF-11, con umbrales editables;
  - la bitácora;
  - el diagrama «Rendimiento por RAE» y el proceso de cada estudiante.

  En la revisión puede editar, aprobar o descartar la propuesta de la IA, y ver su historial.

## RAE y niveles de logro

*Introducción a la Programación 2026-1* usa los cuatro RAE oficiales del programa del curso 33698 (RAE 1: caracterizar el problema; RAE 2: diseñar con construcciones básicas; RAE 3: organizar los datos con colecciones; RAE 4: implementar en C++). Cada revisión marca el nivel de logro (bajo, medio o alto) de los RAE que el taller trabaja; el RAE 3 queda sin marcar porque estos talleres no usan colecciones. El diagrama de dispersión de la clase cruza la nota promedio (0 a 5) con ese nivel:

| Alumno | Nota promedio | RAE 1 | RAE 2 | RAE 3 | RAE 4 |
|---|---|---|---|---|---|
| Ana Rodríguez | 3.85 | medio | alto | — | medio |
| Luis Martínez | 5.0 | alto | alto | — | alto |
| Carlos Pérez | 2.8 | medio | medio | — | bajo |

Solo cuentan las revisiones devueltas (el Taller 3 de Ana está en borrador). Si la base ya tenía los datos de prueba de antes, este script carga los RAE oficiales (o actualiza el texto de los que ya existen) y los niveles:

```bash
python3 scripts/seed_rae_levels.py
```

## Preguntas, comprensión y exportación

- **Como Ana**, en «Mi historial» abre una entrega devuelta. Debajo de la retroalimentación responde «¿Entendiste esta retroalimentación?» y hace una pregunta.
- **Como profesora**, la pregunta aparece en «Preguntas de tus estudiantes» del tablero y como píldora en la entrega; la responde debajo del código. La comprensión aparece resumida, sin nombres, en «Indicadores de la clase» (RNF-09).
- En la clase, la profesora tiene «Errores frecuentes» (por clase o por materia) y los botones «Exportar notas (CSV)» y «Exportar revisiones (CSV)». La integración con Campus Virtual está pendiente; ver `docs/INTEGRACION.md`.

## Ancho de banda (RNF-12)

Con el frontend compilado (`npm run build && npm run start`):

```bash
WEB=http://localhost:3000 API=http://127.0.0.1:8080 python3 scripts/medir_ancho_banda.py
```

Muestra el peso de la primera carga de cada página y de las respuestas de la API, y el tiempo a 1 Mbps y a los 60–70 Mbps acordados, también con los 200 estudiantes abriendo la página a la vez. Para otros valores: `ANCHOS_KBPS=1000,30000 ESTUDIANTES=150`.

## Prueba de carga (RNF-10)

Con los datos cargados y Ollama encendido:

```bash
API=http://127.0.0.1:8080 python3 scripts/prueba_carga.py 20
```

El script envía 20 entregas al Taller 1, espera a que la IA prepare todas las propuestas y compara la peor demora con el umbral «Procesar cada lote» de la materia. Con `llama3.1:8b` cada propuesta tarda unos 25 s.

## Autorización de datos

Las cuentas de prueba se registran con la autorización de tratamiento de datos. Si una cuenta antigua no la tiene, al entrar ve primero el aviso «Autorización de tratamiento de datos».

