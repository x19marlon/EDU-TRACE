#!/usr/bin/env bash
# Inicia EduTrace en Windows para desarrollo. Se ejecuta en Git Bash: ./startW.sh
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*) ;;
    *)
        echo -e "${RED}Este script es para Windows (Git Bash). En Linux usa ./start.sh${NC}"
        exit 1
        ;;
esac

# 0. Configuración y puertos
if [ -f .env ]; then
    while IFS='=' read -r key value || [ -n "$key" ]; do
        [[ "$key" =~ ^[A-Z_][A-Z0-9_]*$ ]] || continue
        value="${value%$'\r'}"
        value="${value#\"}"; value="${value%\"}"
        value="${value#\'}"; value="${value%\'}"
        if [ -z "${!key}" ] && [ -n "$value" ]; then
            export "$key=$value"
        fi
    done < .env
fi
export HOST="${HOST:-127.0.0.1}"
export PORT="${PORT:-8080}"
export TEACHER_CODE="${TEACHER_CODE:-admin123}"
export OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://127.0.0.1:11434}"
export OLLAMA_MODEL="${OLLAMA_MODEL:-llama3.1:8b}"
WEB_PORT="${WEB_PORT:-3000}"

# bubblewrap solo existe en Linux.
export SANDBOX=none

# Termina los procesos que escuchan en un puerto. Se reconocen por la dirección remota
# :0 porque netstat traduce LISTENING según el idioma de Windows. //F y //T llevan doble
# barra para que Git Bash no las convierta en rutas.
free_port() {
    local pids
    pids="$(netstat -ano 2>/dev/null | tr -d '\r' | awk -v port=":$1" '$1 == "TCP" && $3 ~ /:0$/ && substr($2, length($2) - length(port) + 1) == port {print $5}' | sort -u)"
    for pid in $pids; do
        taskkill //F //T //PID "$pid" > /dev/null 2>&1 || true
    done
}

echo -e "${BLUE}Liberando puertos $PORT y $WEB_PORT${NC}"
free_port "$PORT"
free_port "$WEB_PORT"

# 1. Dependencias
echo -e "\n${BLUE}Verificando dependencias${NC}"
INSTALLED=false
require() {
    local command_name="$1" package="$2" label="$3"
    if command -v "$command_name" &> /dev/null; then
        echo -e "${GREEN}✓ $label${NC}"
        return
    fi
    if ! command -v winget &> /dev/null; then
        echo -e "${RED}Falta $label y no está winget para instalarlo. Instálalo a mano (paquete $package) y vuelve a intentar.${NC}"
        exit 1
    fi
    echo -e "${YELLOW}Instalando $label${NC}"
    winget install --id "$package" --exact --silent --accept-package-agreements --accept-source-agreements
    INSTALLED=true
}
require go GoLang.Go "Go"
require node OpenJS.NodeJS.LTS "Node.js"
require g++ BrechtSanders.WinLibs.POSIX.UCRT "g++ (WinLibs)"
require ollama Ollama.Ollama "Ollama"
if [ "$INSTALLED" = true ]; then
    echo -e "\n${YELLOW}Se instalaron programas nuevos: cierra Git Bash, ábrelo otra vez y vuelve a ejecutar ./startW.sh${NC}"
    exit 0
fi

# 2. Dependencias del frontend
LOCK_SUM="$(sha256sum web/package-lock.json | cut -c1-16)"
if [ ! -d "web/node_modules" ] || [ "$(cat web/node_modules/.edutrace-lock 2>/dev/null)" != "$LOCK_SUM" ]; then
    echo -e "${YELLOW}Instalando las dependencias de web/${NC}"
    (cd web && npm install --no-audit --no-fund)
    echo "$LOCK_SUM" > web/node_modules/.edutrace-lock
else
    echo -e "${GREEN}✓ Dependencias de web/ al día${NC}"
fi

# 3. Ollama y modelo
echo -e "\n${BLUE}Verificando Ollama y el modelo $OLLAMA_MODEL${NC}"
if ! curl -s "$OLLAMA_BASE_URL/api/tags" &> /dev/null; then
    echo -e "${YELLOW}Iniciando Ollama${NC}"
    ollama serve &> /dev/null &
    for i in {1..15}; do
        curl -s "$OLLAMA_BASE_URL/api/tags" &> /dev/null && break
        sleep 1
    done
fi

MODEL_TAG="$OLLAMA_MODEL"
[[ "$MODEL_TAG" == *:* ]] || MODEL_TAG="$MODEL_TAG:latest"
if ollama list | tr -d '\r' | awk 'NR > 1 {print $1}' | grep -qx "$MODEL_TAG"; then
    echo -e "${GREEN}✓ Modelo $MODEL_TAG listo${NC}"
else
    echo -e "${YELLOW}Descargando $MODEL_TAG${NC}"
    ollama pull "$MODEL_TAG"
fi

# 4. Compilar e iniciar servicios
echo -e "\n${BLUE}Compilando e iniciando servicios${NC}"
echo -e "${RED}Sin aislamiento: en Windows el código de los estudiantes se ejecuta directamente en este computador. Solo para desarrollo.${NC}"

# Los programas de los estudiantes escriben sus archivos en /tmp, que en Windows es
# la carpeta tmp de la unidad donde corren.
SYSTEM_DRIVE="${SYSTEMDRIVE:-C:}"
mkdir -p "/${SYSTEM_DRIVE%:}/tmp" 2>/dev/null || echo -e "${YELLOW}No se pudo crear la carpeta tmp en $SYSTEM_DRIVE: los programas que crean archivos no funcionarán.${NC}"

DEV_FIRMA="$( { cat web/package-lock.json web/next.config.mjs web/tailwind.config.ts web/tsconfig.json web/postcss.config.mjs 2>/dev/null; node -v; echo "$PORT"; } | sha256sum | cut -c1-16)"
if [ "$1" = "--limpio" ]; then
    echo -e "${YELLOW}Borrando la caché de Next (web/.next y web/.next-dev)${NC}"
    rm -rf web/.next web/.next-dev
elif [ "$(cat web/.next-dev.firma 2>/dev/null)" != "$DEV_FIRMA" ]; then
    echo -e "${YELLOW}Cambiaron las dependencias o la configuración: se reconstruye la caché de desarrollo (web/.next-dev)${NC}"
    rm -rf web/.next-dev
fi
echo "$DEV_FIRMA" > web/.next-dev.firma

mkdir -p bin
go build -o bin/server.exe ./cmd/server
echo -e "${GREEN}✓ Backend compilado en bin/server.exe${NC}"

# kill no alcanza a los programas de Windows: se liberan por sus puertos.
cleanup() {
    trap - SIGINT SIGTERM EXIT
    echo -e "\n${YELLOW}Deteniendo servicios${NC}"
    free_port "$PORT"
    free_port "$WEB_PORT"
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

./bin/server.exe &
BACKEND_PID=$!
for i in {1..20}; do
    if curl -s "http://$HOST:$PORT/api/health" &> /dev/null; then
        echo -e "${GREEN}✓ Backend en http://$HOST:$PORT${NC}"
        break
    fi
    if ! kill -0 "$BACKEND_PID" 2> /dev/null; then
        echo -e "${RED}El backend se detuvo al arrancar; revisa los mensajes de arriba.${NC}"
        exit 1
    fi
    sleep 1
done

(cd web && NEXT_PUBLIC_API_URL="http://localhost:$PORT" npm run dev -- -p "$WEB_PORT") &

echo -e "\n${GREEN}Edutrace inicializado!!${NC}"
echo -e "${GREEN}Abre en tu navegador: http://localhost:$WEB_PORT${NC}"
echo -e "Backend: http://$HOST:$PORT · Ollama: $OLLAMA_BASE_URL ($MODEL_TAG) · Sandbox: $SANDBOX"
echo -e "Presiona Ctrl+C para detener todo."

wait
