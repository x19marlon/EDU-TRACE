import { ICON_DARK, ICON_ORANGE, NAME_DARK, NAME_ORANGE } from "@/components/logo-paths";

const DARK = "#222A2F";
const ORANGE = "#F17222";

export default function Logo({ className = "h-10 w-auto", full = false }: { className?: string; full?: boolean }) {
  return (
    <svg viewBox={full ? "29 250 1875 1309" : "350 250 1252 987"} className={className} role="img" aria-label="EduTrace">
      <g transform="translate(0 1928) scale(0.1 -0.1)">
        <path fill={DARK} d={ICON_DARK} />
        <path fill={ORANGE} d={ICON_ORANGE} />
        {full && (
          <>
            <path fill={DARK} d={NAME_DARK} />
            <path fill={ORANGE} d={NAME_ORANGE} />
          </>
        )}
      </g>
    </svg>
  );
}
