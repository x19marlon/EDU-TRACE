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

| Curso | Clase | Alumnos |
|---|---|---|
| Introducción a la Programación 2026-1 | Grupo A - Mañana | 3 |
| Programación Avanzada 2026-1 | Grupo B - Tarde | 2 |

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

| Alumno | Taller | Resultado en el sandbox | Estado |
|---|---|---|---|
| Ana Rodríguez | Taller 1: promedio de notas | `Promedio: 4.17` | **4.5**, devuelta |
| Ana Rodríguez | Taller 2: suma con ciclos | `Suma: 10` (esperado 15) | **3.2**, devuelta |
| Ana Rodríguez | Taller 3: el mayor de tres | `Mayor: 12` | En revisión (borrador con 4.0, no devuelta); **tardía** ~27 h |
| Luis Martínez | Taller 2: suma con ciclos | `Suma: 15` | **5.0**, devuelta |
| Carlos Pérez | Taller 1: promedio de notas | `Promedio: 4` (división entera) | **2.8**, devuelta |

Fechas límite (relativas al momento de cargar los datos): el Taller 1 vence en 5 días; el Taller 2 vence en 20 horas y no acepta entregas tardías; el Taller 3 venció 27 horas antes y acepta tardías con una penalización sugerida de 0.5 por día iniciado (la entrega de Ana queda tardía, 2 días iniciados → −1.0 sugerido).

Ana también tiene dos intentos de compilación en el Taller 2 (uno sin compilar por un `;` faltante), visibles en su proceso desde la vista del profesor.

- **Como Ana** (`ana.rodriguez@est.edu.co`): en «Mis entregas» ve sus dos notas con la retroalimentación del profesor y la tercera entrega «En revisión».
- **Como profesora** (`maria.garcia@universidad.edu.co`): en *Grupo A - Mañana* ve las cinco entregas con su estado (devuelta o borrador), el diagrama «Rendimiento por RAE» y el proceso de cada estudiante.

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

