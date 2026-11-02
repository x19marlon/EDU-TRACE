import type { Metadata } from "next";
import Link from "next/link";
import Logo from "@/components/Logo";
import HomeActions from "@/components/HomeActions";

export const metadata: Metadata = {
  title: "EduTrace: programa, comprende, avanza",
  description: "Practica, aprende de tus errores y recibe orientación paso a paso. Tu docente acompaña tu progreso.",
};

const ROLES = [
  {
    title: "Si estás aprendiendo",
    surface: "bg-lavender-50",
    items: [
      "Compila tu código desde el navegador y entiende cada error, en español y con su línea.",
      "Comprueba tu programa con casos de prueba antes de enviarlo.",
      "Pide pistas que te orientan, sin darte la respuesta.",
      "Revisa tu historial y mira cómo vas avanzando en cada resultado de aprendizaje.",
    ],
  },
  {
    title: "Si estás enseñando",
    surface: "bg-mint-50",
    items: [
      "Organiza tus materias y clases, y publica talleres con sus archivos y casos de prueba.",
      "Revisa cada propuesta de la IA: la editas, la apruebas o la descartas.",
      "Identifica quién podría necesitar apoyo y cuándo acercarte a conversar.",
      "Consulta cómo va tu curso, clase por clase.",
    ],
  },
  {
    title: "Si estás acompañando",
    surface: "bg-peach-50",
    items: [
      "Sigue el trabajo de los estudiantes de las clases en las que eres monitor.",
      "Deja comentarios sobre su código y responde sus preguntas.",
      "Da retroalimentación sin asignar calificaciones: las notas siguen siendo del profesor.",
      "Para tu propio aprendizaje, tienes las mismas herramientas que cualquier estudiante.",
    ],
  },
];

const STEPS = [
  ["Únete a tu clase.", "Tu profesor te comparte un código de acceso y con él entras en un momento."],
  ["Escribe y compila.", "Programa en C++ sin instalar nada. Si tu programa pide datos, los escribes y ves cómo responde."],
  ["Aprende de tus errores.", "Cuando algo falla, te explicamos qué significa. Si aún no sabes cómo seguir, pide una pista: van de la más general a la más específica."],
  ["Recibe retroalimentación.", "Envía tu entrega y tu docente te devuelve sus comentarios y tu nota, para que sepas qué mejorar."],
];

const TOPICS = ["Entrada y salida", "Variables", "Aritmética", "Condicionales", "Ciclos", "Funciones", "Arreglos", "Matrices", "Estructuras", "Archivos de texto"];

const CODE_LINE = "whitespace-pre";

function CodePreview() {
  return (
    <div className="rounded-3xl border border-lavender-100/70 bg-white p-4 shadow-lift sm:p-5">
      <div className="mb-3 flex items-center justify-between gap-2">
        <span className="font-bold">Así se ve por dentro</span>
        <span className="rounded-full bg-lavender-100 px-2.5 py-0.5 text-xs font-bold text-lavender-700">Ejemplo</span>
      </div>
      <pre tabIndex={0} aria-label="Ejemplo de un programa en C++" className="overflow-x-auto rounded-2xl bg-lavender-50/70 p-3 font-mono text-xs leading-relaxed text-ink">
        <code>
          <span className={`block ${CODE_LINE} text-mint-700`}>#include &lt;iostream&gt;</span>
          <span className={`block ${CODE_LINE}`}><span className="text-lavender-700">using namespace</span> std;</span>
          <span className={`block ${CODE_LINE}`}><span className="text-mint-700">int</span> main() {"{"}</span>
          <span className={`block ${CODE_LINE}`}>{"    "}<span className="text-mint-700">int</span> numero;</span>
          <span className={`block ${CODE_LINE}`}>{"    "}cout &lt;&lt; <span className="text-peach-700">&quot;Ingrese un número: &quot;</span>;</span>
          <span className={`block ${CODE_LINE}`}>{"    "}cin &gt;&gt; numero;</span>
          <span className={`block ${CODE_LINE}`}>{"    "}cout &lt;&lt; <span className="text-peach-700">&quot;El doble es &quot;</span> &lt;&lt; numero * 2;</span>
          <span className={`block ${CODE_LINE}`}>{"}"}</span>
        </code>
      </pre>
      <div className="mt-3 rounded-2xl bg-peach-100 p-3">
        <p className="text-xs font-bold text-peach-700">Resultado</p>
        <p className="mt-1 font-mono text-xs leading-relaxed">
          Ingrese un número:
          <br />
          El doble es 42
        </p>
      </div>
      <div className="mt-3 rounded-2xl bg-mint-50 p-3 text-sm">
        <p className="font-bold text-mint-700">¿Qué sigue?</p>
        <p className="mt-0.5 leading-relaxed">Tu programa ya funciona con números enteros. Prueba con 2.5: ¿qué tipo de dato necesitarías para no perder los decimales?</p>
      </div>
    </div>
  );
}

export default function HomePage() {
  return (
    <div className="relative min-h-screen overflow-hidden">
      <div aria-hidden className="pointer-events-none absolute inset-0">
        <div className="blob bg-lavender-200/70 h-[30rem] w-[30rem] -top-40 -left-32" />
        <div className="blob bg-mint-200/70 h-80 w-[28rem] top-[28rem] -right-40" style={{ animationDelay: "-7s" }} />
        <div className="blob bg-peach-200/70 h-96 w-96 -bottom-48 left-1/4" style={{ animationDelay: "-14s" }} />
      </div>

      <header className="relative z-10 mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-5 sm:px-6">
        <Link href="/inicio" aria-label="EduTrace, inicio" className="flex items-center gap-2">
          <Logo full className="h-14 w-auto" />
        </Link>
        <nav aria-label="Tu cuenta" className="flex flex-wrap items-center gap-2">
          <HomeActions />
        </nav>
      </header>

      <main className="relative z-10 mx-auto max-w-6xl space-y-6 px-4 pb-8 sm:px-6">
        <section className="rise-in relative grid items-center gap-8 overflow-hidden rounded-[2rem] bg-lavender-100 p-6 shadow-lift sm:p-10 md:grid-cols-2 lg:p-12">
          <div aria-hidden className="pointer-events-none absolute inset-0 overflow-hidden">
            <div className="blob bg-mint-200 h-64 w-80 -bottom-16 -left-20" />
            <div className="blob bg-peach-200/80 h-40 w-40 top-10 right-1/3" style={{ animationDelay: "-5s" }} />
          </div>
          <div className="relative">
            <p className="text-sm font-bold text-lavender-700">De javerianos para javerianos</p>
            <h1 className="mt-3 text-4xl font-extrabold leading-tight tracking-tight sm:text-5xl">
              Programa. Comprende. <span className="text-lavender-700">Avanza.</span>
            </h1>
            <p className="mt-4 text-lg font-semibold leading-snug">
              Practica, aprende de tus errores y recibe orientación paso a paso. Tu docente acompaña tu progreso.
            </p>
            <p className="mt-3 text-ink-soft">
              EduTrace es el lugar para aprender a programar en C++ con calma: escribes y compilas desde el navegador, y cuando algo no sale,
              entiendes por qué.
            </p>
            <div className="mt-6 flex flex-wrap gap-2">
              <HomeActions large />
            </div>
          </div>
          <div className="relative min-w-0">
            <CodePreview />
          </div>
        </section>

        <section aria-labelledby="para-quien" className="rise-in">
          <h2 id="para-quien" className="text-2xl font-extrabold tracking-tight">Un mismo lugar para aprender y enseñar</h2>
          <div className="mt-4 grid gap-4 md:grid-cols-3">
            {ROLES.map((role) => (
              <div key={role.title} className={`rounded-3xl border border-lavender-100/70 p-5 shadow-soft ${role.surface}`}>
                <h3 className="font-bold">{role.title}</h3>
                <ul className="mt-3 space-y-2 text-sm leading-relaxed">
                  {role.items.map((item) => (
                    <li key={item} className="flex gap-2">
                      <span aria-hidden className="font-bold text-lavender-700">✓</span>
                      <span>{item}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </section>

        <section aria-labelledby="como-funciona" className="rise-in rounded-3xl border border-lavender-100/70 bg-white p-5 shadow-soft sm:p-7">
          <h2 id="como-funciona" className="text-2xl font-extrabold tracking-tight">¿Cómo funciona EduTrace?</h2>
          <ol className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {STEPS.map(([lead, text], i) => (
              <li key={lead} className="rounded-2xl bg-lavender-50/70 p-4">
                <span aria-hidden className="flex h-8 w-8 items-center justify-center rounded-full bg-lavender-600 text-sm font-extrabold text-white">{i + 1}</span>
                <p className="mt-3 font-bold">{lead}</p>
                <p className="mt-1 text-sm leading-relaxed text-ink-soft">{text}</p>
              </li>
            ))}
          </ol>
        </section>

        <div className="grid gap-4 md:grid-cols-2">
          <section aria-labelledby="ia" className="rise-in rounded-3xl border border-lavender-100/70 bg-mint-50 p-5 shadow-soft sm:p-7">
            <h2 id="ia" className="text-2xl font-extrabold tracking-tight">La IA te orienta. Tu docente decide.</h2>
            <p className="mt-3 text-sm leading-relaxed">
              La idea es acompañarte en el proceso: darte pistas y orientarte para que encuentres la solución por tu cuenta, en lugar de dártela
              directamente. No es una calificación, sino una ayuda para aprender.
            </p>
            <ul className="mt-3 flex flex-wrap gap-2 text-sm font-bold">
              {["¿Hacia dónde voy?", "¿Cómo voy?", "¿Qué sigue?"].map((q) => (
                <li key={q} className="rounded-full bg-white px-3 py-1 text-mint-700 ring-1 ring-mint-200">{q}</li>
              ))}
            </ul>
            <p className="mt-3 text-sm leading-relaxed">
              Tu docente siempre tiene la última palabra, y es quien define en cada clase cómo llega la retroalimentación de la IA.
            </p>
          </section>
          <section aria-labelledby="practica" className="rise-in rounded-3xl border border-lavender-100/70 bg-white p-5 shadow-soft sm:p-7">
            <h2 id="practica" className="text-2xl font-extrabold tracking-tight">Practica antes del parcial</h2>
            <p className="mt-3 text-sm leading-relaxed text-ink-soft">
              Tienes más de 40 ejercicios, ordenados por tema y por dificultad. Cada uno se revisa automáticamente con sus propios casos de prueba.
              No tienes que enviárselos a tu profesor: son solo para que practiques y veas cómo vas avanzando.
            </p>
            <ul className="mt-3 flex flex-wrap gap-1.5 text-xs font-semibold">
              {TOPICS.map((topic) => (
                <li key={topic} className="rounded-full bg-lavender-50 px-2.5 py-1 text-lavender-700">{topic}</li>
              ))}
            </ul>
          </section>
        </div>

        <section aria-labelledby="datos" className="rise-in rounded-3xl border border-lavender-100/70 bg-peach-50 p-5 shadow-soft sm:p-7">
          <h2 id="datos" className="text-2xl font-extrabold tracking-tight">Queremos que tengas tranquilidad sobre tu información</h2>
          <div className="mt-3 max-w-3xl space-y-2 text-sm leading-relaxed">
            <p>
              Tus datos y el código que compartes permanecen en nuestro servidor. Además, la inteligencia artificial funciona allí mismo y no tiene
              acceso a tu nombre ni a tu correo electrónico.
            </p>
            <p>
              En cualquier momento puedes consultar, descargar, corregir o solicitar la eliminación de tus datos desde «Mi cuenta». Si quieres conocer
              todos los detalles, te invitamos a leer nuestra{" "}
              <Link href="/privacidad" className="font-bold text-lavender-700 underline">
                política de tratamiento de datos
              </Link>
              .
            </p>
          </div>
        </section>

        <section aria-labelledby="empieza" className="rise-in flex flex-wrap items-center justify-between gap-4 rounded-3xl bg-lavender-100 p-5 shadow-soft sm:p-7">
          <div className="min-w-0 flex-1 basis-72">
            <h2 id="empieza" className="text-2xl font-extrabold tracking-tight">Tu próximo programa empieza aquí</h2>
            <p className="mt-1 text-sm">Crea tu cuenta en un minuto y pídele a tu profesor el código para unirte a su clase.</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <HomeActions large />
          </div>
        </section>
      </main>

      <footer className="relative z-10 mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-2 px-4 pb-8 text-sm text-ink-soft sm:px-6">
        <p className="font-bold text-lavender-700">De javerianos para javerianos.</p>
        <Link href="/privacidad" className="font-semibold hover:text-lavender-700">
          Política de tratamiento de datos
        </Link>
      </footer>
    </div>
  );
}
