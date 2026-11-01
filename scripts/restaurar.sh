#!/usr/bin/env bash
set -euo pipefail

if [ $# -ne 1 ]; then
  echo "Uso: $0 respaldos/edutrace-AAAAMMDD-HHMMSS.tar.gz"
  exit 1
fi
ARCHIVO="$(realpath "$1")"
cd "$(dirname "$0")/.."
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi
DATOS="${DATA_DIR:-data}"
SERVICIO="${SERVICIO:-edutrace-backend}"
SYSTEMCTL="${SYSTEMCTL:-sudo systemctl}"
SALUD="${SALUD:-http://127.0.0.1:${PORT:-8080}/api/health}"
BIN="${BIN:-bin/edutrace-respaldo}"
if [ ! -x "$BIN" ]; then
  go build -o "$BIN" ./cmd/respaldo
fi
FECHA="$(date +%Y%m%d-%H%M%S)"
NUEVO="$DATOS.restaurando-$FECHA"
ANTES="$DATOS.antes-$FECHA"

echo "1. Verificando $ARCHIVO"
"$BIN" -verificar "$ARCHIVO"
echo "2. Extrayendo en $NUEVO"
"$BIN" -restaurar "$ARCHIVO" -en "$NUEVO"

read -r -p "Se detendrá $SERVICIO y los datos actuales pasarán a $ANTES. ¿Continuar? [s/N] " RESPUESTA
if [ "$RESPUESTA" != "s" ] && [ "$RESPUESTA" != "S" ]; then
  rm -rf "$NUEVO"
  echo "Cancelado: no se cambió nada."
  exit 1
fi

echo "3. Deteniendo $SERVICIO"
$SYSTEMCTL stop "$SERVICIO"
mv "$DATOS" "$ANTES"
mv "$NUEVO" "$DATOS"
echo "4. Iniciando $SERVICIO"
$SYSTEMCTL start "$SERVICIO"
for _ in $(seq 1 30); do
  if curl -fsS "$SALUD" > /dev/null 2>&1; then
    echo "✓ Restaurado. Los datos anteriores quedaron en $ANTES; bórralos cuando confirmes que todo está bien."
    exit 0
  fi
  sleep 2
done
echo "✗ EduTrace no arrancó con el respaldo; vuelvo a los datos anteriores"
$SYSTEMCTL stop "$SERVICIO"
mv "$DATOS" "$NUEVO"
mv "$ANTES" "$DATOS"
$SYSTEMCTL start "$SERVICIO"
exit 1
