#!/usr/bin/env bash
set -uo pipefail
cd "$(dirname "$0")/.."
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

API="${VIGILAR_API:-http://127.0.0.1:${PORT:-8080}}"
WEB="${VIGILAR_WEB:-http://127.0.0.1:3000}"
RESPALDOS="${BACKUP_DIR:-respaldos}"
HORAS_RESPALDO="${VIGILAR_HORAS_RESPALDO:-26}"
ESTADO="${STATE_DIRECTORY:-${TMPDIR:-/tmp}}/edutrace-vigilar.estado"
REPETIR="${VIGILAR_REPETIR_HORAS:-6}"

problemas=()
salud="$(curl -fsS -m 10 "$API/api/health" 2>/dev/null)" || salud=""
if [ -z "$salud" ]; then
  problemas+=("El backend no responde en $API.")
else
  for check in base ia disco cola; do
    if grep -q "\"$check\":false" <<< "$salud"; then
      case "$check" in
        base) problemas+=("La base de datos no responde.") ;;
        ia) problemas+=("Ollama no responde: no hay propuestas ni retroalimentación de la IA.") ;;
        disco) problemas+=("Queda menos del ${DISK_MIN_FREE_PERCENT:-10} % de disco libre en ${DATA_DIR:-data}.") ;;
        cola) problemas+=("La cola de la IA lleva más de 30 minutos sin avanzar.") ;;
      esac
    fi
  done
fi
if ! curl -fsS -m 10 -o /dev/null "$WEB/login" 2>/dev/null; then
  problemas+=("El frontend no responde en $WEB.")
fi
ultimo="$(ls -1t "$RESPALDOS"/edutrace-*.tar.gz "$RESPALDOS"/edutrace-*.tar.gz.age 2>/dev/null | head -n 1)"
if [ -z "$ultimo" ]; then
  problemas+=("No hay ningún respaldo en $RESPALDOS.")
elif [ $(( $(date +%s) - $(stat -c %Y "$ultimo") )) -gt $(( HORAS_RESPALDO * 3600 )) ]; then
  problemas+=("El último respaldo tiene más de $HORAS_RESPALDO horas: $(basename "$ultimo").")
fi

avisar() {
  echo "$1"
  [ -n "${ALERTA_URL:-}" ] || return 0
  if [ "${ALERTA_JSON:-false}" = "true" ]; then
    cuerpo="$(python3 -c 'import json,sys; t=sys.stdin.read(); print(json.dumps({"text": t, "content": t}))' <<< "$1")"
    curl -fsS -m 15 -H "Content-Type: application/json" -d "$cuerpo" "$ALERTA_URL" > /dev/null || echo "No se pudo enviar la alerta a ALERTA_URL"
  else
    curl -fsS -m 15 -H "Title: EduTrace" --data-binary "$1" "$ALERTA_URL" > /dev/null || echo "No se pudo enviar la alerta a ALERTA_URL"
  fi
}

anterior="$(cat "$ESTADO" 2>/dev/null || true)"
if [ ${#problemas[@]} -eq 0 ]; then
  [ -n "${LATIDO_URL:-}" ] && curl -fsS -m 10 -o /dev/null "$LATIDO_URL"
  if [ -n "$anterior" ]; then
    avisar "✓ EduTrace ($(hostname)) volvió a la normalidad."
  fi
  rm -f "$ESTADO"
  exit 0
fi

mensaje="✗ EduTrace ($(hostname)) tiene problemas:"
for p in "${problemas[@]}"; do
  mensaje+=$'\n'"- $p"
done
firma="$(printf '%s\n' "${problemas[@]}" | sha256sum | cut -c1-16)"
if [ "${anterior%% *}" != "$firma" ] || [ $(( $(date +%s) - ${anterior##* } )) -gt $(( REPETIR * 3600 )) ]; then
  avisar "$mensaje"
  echo "$firma $(date +%s)" > "$ESTADO"
else
  echo "$mensaje"
  echo "(ya se avisó; se repetirá en menos de $REPETIR horas si sigue igual)"
fi
exit 1
