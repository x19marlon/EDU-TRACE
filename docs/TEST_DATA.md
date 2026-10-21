# Datos de prueba

Cuentas, cursos y clases de prueba para desarrollo local. **Solo para entornos locales**: no uses estas credenciales en un despliegue real.

## Cómo cargarlos

`data/` está en `.gitignore`, así que cada persona debe cargar los datos en su propia instancia:

1. Levanta la aplicación con `./start.sh` (exporta `TEACHER_CODE=admin123` si no está definido).
2. Con el backend corriendo en `http://127.0.0.1:8080`, ejecuta:

   ```bash
   ./scripts/seed_test_data.sh
   ```

El script registra al docente, crea los cursos y clases, y registra e inscribe a los alumnos. Al final imprime los códigos de ingreso de cada clase (se generan al azar en cada carga).

> El script usa `teacher_code: "admin123"`. Si arrancas el backend con otro `TEACHER_CODE`, el registro del docente fallará. Ejecútalo sobre una base limpia: si los correos ya existen, el registro devuelve error.

## Docente

| Nombre | Email | Contraseña |
|---|---|---|
| Prof. María García | `maria.garcia@universidad.edu.co` | `Profesor2026!` |

Para iniciar sesión como docente, abre el botón `⋮` de la tarjeta de login y elige **Docente**.

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
