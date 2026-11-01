#!/usr/bin/env bash
set -euo pipefail

COMMIT="${1:-origin/main}"
SERVICIOS="${SERVICIOS:-edutrace-backend edutrace-web}"
REINICIAR="${REINICIAR:-sudo systemctl restart}"
cd "$(dirname "$0")/.."

if [ ! -f .env ]; then
  echo "✗ Falta .env en $(pwd)"
  exit 1
fi
set -a
source .env
set +a
SALUD="${SALUD:-http://127.0.0.1:${PORT:-8080}/api/health}"
ANTERIOR="$(git rev-parse HEAD)"

construir() {
  echo "2. Compilando el backend"
  go build -o bin/edutrace-server ./cmd/server
  echo "3. Compilando el frontend"
  (cd web && npm ci --no-audit --no-fund && npm run build)
}

reiniciar() {
  echo "4. Reiniciando $SERVICIOS"
  $REINICIAR $SERVICIOS
  echo "5. Comprobando $SALUD"
  for _ in $(seq 1 30); do
    if curl -fsS "$SALUD" > /dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  return 1
}

echo "1. Actualizando el código a $COMMIT (ahora: ${ANTERIOR:0:7})"
git fetch --quiet origin
git checkout --quiet --force "$COMMIT"

if construir && reiniciar; then
  echo "✓ EduTrace $(git rev-parse --short HEAD) desplegado"
  exit 0
fi

echo "✗ El despliegue falló; vuelvo a ${ANTERIOR:0:7}"
git checkout --quiet --force "$ANTERIOR"
if construir && reiniciar; then
  echo "✓ Se restauró ${ANTERIOR:0:7}"
else
  echo "✗ Tampoco arrancó la versión anterior: revisa journalctl -u edutrace-backend -u edutrace-web"
fi
exit 1
