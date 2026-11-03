# Documentos del curso para la IA

La IA no recibe todos los documentos en todas las peticiones: recibe los de la **materia** del curso con el que está trabajando. Este README no se envía al modelo.

## Estructura

```
syllabus/
  comun/                   (opcional) se envía en todas las materias
  intro-programacion/
    materia.json           nombre, alias y si es la materia por defecto
    criterios.md           criterios de retroalimentación
    programa.md            lo del programa oficial que no está en los criterios
  <otra-materia>/
    materia.json
    *.md
```

Solo se leen los `.md` dentro de las carpetas, en orden alfabético. Se cargan al arrancar el backend: después de cambiar algo hay que reiniciarlo.

## `materia.json`

```json
{
  "nombre": "Introducción a la Programación",
  "alias": ["Intro a la Programación", "Intro Programación"],
  "por_defecto": true
}
```

- **nombre** y **alias**: la materia se elige buscando alguno de estos textos dentro del nombre del curso que creó el docente, sin distinguir mayúsculas ni tildes. Así, el curso "Introducción a la Programación 2026-1" usa esta carpeta. Si coinciden varias materias, gana el texto más largo.
- **por_defecto**: la materia que se usa cuando la petición no tiene curso, por ejemplo un estudiante que practica sin elegir clase. Solo una materia puede tenerlo.

## Cómo se elige el curso de cada petición

| Petición | De dónde sale el curso |
|---|---|
| Retroalimentación (`/api/feedback`) y tutor (`/api/chat`) | La clase elegida por el estudiante (`group_id`) o, si no hay, la del taller (`assignment_id`) |
| Propuesta de revisión del docente | La clase de la entrega |
| Análisis del proceso de un estudiante | La clase que el docente está viendo |

Solo cuenta si el usuario pertenece a esa clase: es el docente dueño o un estudiante inscrito.

Si el curso no coincide con ninguna materia, la IA recibe solo `comun/` (o un aviso de que no hay documentos) y **no** los de otra materia. Cada petición deja en el log del backend qué materia usó (`syllabus selected`).

## Agregar una materia

1. Crea `syllabus/<materia>/` con su `materia.json` y sus `.md`.
2. Comprueba que el nombre o un alias aparezca en el nombre del curso de la app.
3. Reinicia el backend y revisa el log `syllabus loaded`: debe listar la materia.

Ten en cuenta el tamaño: los documentos van completos en cada petición y comparten el contexto del modelo (`OLLAMA_CONTEXT_SIZE`) con el código del estudiante y la respuesta. Por eso `programa.md` es un resumen: el programa completo está en [docs/PROGRAMA_INTRO_PROGRAMACION.md](../docs/PROGRAMA_INTRO_PROGRAMACION.md) y no se envía al modelo.
