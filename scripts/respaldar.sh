#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

BIN="${BIN:-bin/edutrace-respaldo}"
if [ ! -x "$BIN" ]; then
  go build -o "$BIN" ./cmd/respaldo
fi
"$BIN"

if [ -n "${RESPALDO_REMOTO:-}" ]; then
  echo "Copiando los respaldos a $RESPALDO_REMOTO"
  rsync -a --delete "${BACKUP_DIR:-respaldos}/" "$RESPALDO_REMOTO"
fi
