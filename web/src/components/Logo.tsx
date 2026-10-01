/**
 * Logo de EduTrace: un libro abierto del que brota una planta (aprender = crecer).
 * Con `animated`, el brote crece y las hojas se abren al montarse.
 */
export default function Logo({
  className = "h-10 w-10",
  animated = false,
}: {
  className?: string;
  animated?: boolean;
}) {
  const a = (cls: string) => (animated ? cls : undefined);

  return (
    <svg viewBox="0 0 64 64" className={className} role="img" aria-label="EduTrace">
      {/* Libro */}
      <g strokeLinejoin="round" strokeLinecap="round">
        <path
          d="M32 52 C24 46.5 15 46 7 48.5 V23 C15 20.5 24 21 32 26.5 Z"
          fill="#C3B4F1"
          stroke="#5C42AE"
          strokeWidth="2.6"
        />
        <path
          d="M32 52 C40 46.5 49 46 57 48.5 V23 C49 20.5 40 21 32 26.5 Z"
          fill="#A38DE6"
          stroke="#5C42AE"
          strokeWidth="2.6"
        />
        {/* Líneas de texto */}
        <g stroke="#FFFFFF" strokeOpacity="0.75" strokeWidth="2" fill="none">
          <path d="M13 31 C18 30 23 30.5 26.5 32.5" />
          <path d="M13 37.5 C18 36.5 23 37 26.5 39" />
          <path d="M37.5 32.5 C41 30.5 46 30 51 31" />
          <path d="M37.5 39 C41 37 46 36.5 51 37.5" />
        </g>
      </g>

      {/* Brote */}
      <g className={a("logo-sprout")}>
        <path d="M32 26 V11" stroke="#3A8F6F" strokeWidth="3" strokeLinecap="round" fill="none" />
        <path
          className={a("logo-leaf logo-leaf-left")}
          d="M31.5 17 C25 17.5 20.5 13.5 20 7 C27 6.5 31.5 10.5 31.5 17 Z"
          fill="#6FC2A2"
          stroke="#3A8F6F"
          strokeWidth="2"
          strokeLinejoin="round"
        />
        <path
          className={a("logo-leaf logo-leaf-right")}
          d="M32.5 13 C38.5 13.5 43 9.5 43.5 3.5 C37 3 32.5 7 32.5 13 Z"
          fill="#9AD6BE"
          stroke="#3A8F6F"
          strokeWidth="2"
          strokeLinejoin="round"
        />
      </g>
    </svg>
  );
}

/** Logo + nombre, para la barra lateral y la pantalla de entrada. */
export function Wordmark({ size = "md" }: { size?: "md" | "lg" }) {
  return (
    <span
      className={`font-extrabold tracking-tight text-ink ${size === "lg" ? "text-4xl" : "text-sm"}`}
    >
      Edu<span className="text-lavender-600">Trace</span>
    </span>
  );
}
