import type { Metadata } from "next";
import Link from "next/link";
import Logo from "@/components/Logo";

export const metadata: Metadata = {
  title: "Política de datos · EduTrace",
};

const VERSION = "2026-11";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="space-y-2">
      <h2 className="text-lg font-extrabold text-lavender-700">{title}</h2>
      <div className="space-y-2 text-sm leading-relaxed text-ink-soft">{children}</div>
    </section>
  );
}

export default function PrivacyPage() {
  return (
    <main className="mx-auto max-w-3xl space-y-6 p-4 sm:p-8">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <Link href="/login" className="flex items-center gap-2">
          <Logo className="h-10 w-10" />
          <span className="text-xl font-extrabold tracking-tight">
            Edu<span className="text-lavender-600">Trace</span>
          </span>
        </Link>
        <span className="rounded-full bg-lavender-50 px-3 py-1 text-xs font-bold text-lavender-700">Versión {VERSION}</span>
      </header>

      <article className="space-y-6 rounded-3xl bg-white p-6 shadow-soft sm:p-8">
        <div className="space-y-2">
          <h1 className="text-2xl font-extrabold tracking-tight sm:text-3xl">Política de tratamiento de datos personales</h1>
          <p className="text-sm text-ink-soft">
            EduTrace apoya la retroalimentación formativa en Introducción a la Programación. Esta política explica qué datos trata, para
            qué, quién los ve y cómo ejerces tus derechos según la Ley 1581 de 2012, el Decreto 1377 de 2013 y la Circular Externa 002
            de 2024 de la Superintendencia de Industria y Comercio sobre inteligencia artificial.
          </p>
        </div>

        <Section title="Responsable">
          <p>
            La institución educativa que ofrece la asignatura y opera este servidor de EduTrace. Para cualquier solicitud escribe al
            profesor de tu clase o al administrador de EduTrace de tu institución.
          </p>
        </Section>

        <Section title="Qué datos tratamos">
          <ul className="list-disc space-y-1 pl-5">
            <li>De tu cuenta: nombre, email, rol y la fecha en que autorizaste el tratamiento. La contraseña y el código de recuperación se guardan cifrados con un hash; nadie puede leerlos.</li>
            <li>De tu trabajo: el código que compilas y envías, la entrada y la salida de tus programas, los mensajes del compilador y el resultado de los casos de prueba.</li>
            <li>De tu proceso: la retroalimentación que recibes, la revisión y la nota de tu profesor, los comentarios y marcas de los monitores, tus preguntas sobre la retroalimentación con sus respuestas, si dijiste haberla entendido y el registro de quién revisó y aprobó cada retroalimentación. Tu profesor ve las respuestas a «¿Entendiste esta retroalimentación?» resumidas y sin tu nombre.</li>
            <li>De uso: cuántas veces compilas o pides retroalimentación, tu última actividad y cuánto tarda la IA en responder. Estos tiempos se guardan sin tu nombre.</li>
            <li>De la práctica: qué ejercicios intentaste, cuántas veces y cuántos casos pasaste. Tus casos de prueba propios se guardan solo en tu navegador.</li>
            <li>Avisos: las novedades que te mostramos en la campana (por ejemplo, «tu profesor devolvió la revisión») y si los leíste o te los enviamos por correo.</li>
          </ul>
        </Section>

        <Section title="Para qué los usamos">
          <ul className="list-disc space-y-1 pl-5">
            <li>Darte retroalimentación formativa sobre tu código y permitir que tu profesor la revise, la edite y la apruebe.</li>
            <li>Que tu profesor evalúe tus entregas y acompañe tu proceso de aprendizaje en la asignatura.</li>
            <li>Medir, de forma agregada, si la retroalimentación llega a tiempo y si la plataforma funciona bien.</li>
            <li>Avisarte en la plataforma y, si lo permites, por correo, cuando hay novedades en tus entregas o preguntas. El correo solo dice qué pasó y en qué taller; nunca incluye tu código, tu nota ni el texto de la retroalimentación. Puedes desactivarlo en «Mi cuenta» → «Avisos».</li>
          </ul>
          <p>No usamos tus datos para publicidad, no los vendemos y no los compartimos con terceros.</p>
        </Section>

        <Section title="Inteligencia artificial">
          <ul className="list-disc space-y-1 pl-5">
            <li>El modelo de IA funciona en el mismo servidor de EduTrace. Ningún dato se envía a servicios de IA externos.</li>
            <li>Al modelo solo le llegan el código, la entrada y la salida del programa, los mensajes del compilador, el enunciado del taller y los resultados de los casos de prueba. Nunca tu nombre, tu email ni tus notas anteriores.</li>
            <li>La IA no califica ni decide. En el modo supervisado, que es el predeterminado, ninguna retroalimentación generada por IA te llega sin que tu profesor la apruebe. Siempre se indica cuándo participó la IA.</li>
            <li>Cada propuesta de la IA guarda qué modelo y qué versión de las instrucciones la generó, para que tu profesor pueda comparar su calidad.</li>
            <li>Durante una evaluación (parcial o quiz) que tu profesor marque así, la IA y el chat se desactivan para tu clase hasta la fecha límite.</li>
          </ul>
        </Section>

        <Section title="Quién puede ver tus datos">
          <p>
            Tú; el profesor de cada clase en la que estás inscrito y los profesores con quienes comparte la materia; y los monitores
            que asignen a tu clase, que ven tus entregas para retroalimentarte pero no califican. Tus profesores pueden ver alertas sobre
            tu avance (por ejemplo, talleres sin entregar) y generar un reporte de seguimiento para una tutoría contigo. Nadie más tiene
            acceso desde la plataforma.
          </p>
        </Section>

        <Section title="Tus derechos y cómo ejercerlos">
          <p>Según el artículo 8 de la Ley 1581 de 2012 puedes:</p>
          <ul className="list-disc space-y-1 pl-5">
            <li>Conocer y descargar tus datos: «Mi cuenta» → «Descargar mis datos».</li>
            <li>Actualizar y rectificar tu nombre: «Mi cuenta» → «Mis datos».</li>
            <li>Revocar la autorización y pedir la supresión: «Mi cuenta» → «Eliminar mi cuenta». Se borran tu cuenta, tus entregas, revisiones y registros asociados.</li>
            <li>Pedir prueba de tu autorización, preguntar cómo se usan tus datos y presentar quejas ante la Superintendencia de Industria y Comercio.</li>
          </ul>
        </Section>

        <Section title="Conservación y seguridad">
          <ul className="list-disc space-y-1 pl-5">
            <li>
              Cuando tu profesor cierra el periodo de una materia, recibes un aviso con la fecha en que se borrarán sus datos. En ese
              momento se borran automáticamente las entregas, revisiones, preguntas y registros de esa materia (por defecto, 180 días
              después del cierre). Antes puedes descargarlos desde «Mi cuenta».
            </li>
            <li>Si tu cuenta queda sin clases y sin actividad durante ese mismo plazo, también se borra.</li>
            <li>Los avisos se borran a los 90 días y los tiempos de la IA, que no tienen tu nombre, al año.</li>
            <li>
              Si olvidas tu contraseña y no tienes tu código de recuperación, tu profesor puede generarte un código de un solo uso que
              vence en 24 horas. Recibirás un aviso cada vez que lo haga.
            </li>
            <li>
              Las contraseñas se guardan con hash, la conexión va cifrada (HTTPS), la sesión usa una cookie que el navegador no expone a
              scripts, el código de los estudiantes se ejecuta en un entorno aislado sin red y con límites de procesos, memoria y disco,
              y cada consulta comprueba en la base de datos que tengas permiso para ver lo que pides.
            </li>
          </ul>
        </Section>

        <Section title="Menores de edad">
          <p>Si eres menor de 18 años, necesitas la autorización de tu representante legal para crear la cuenta.</p>
        </Section>
      </article>

      <p className="text-center text-xs text-ink-faint">
        <Link href="/login" className="font-bold text-lavender-600 hover:underline">
          Volver a EduTrace
        </Link>
      </p>
    </main>
  );
}
