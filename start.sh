#!/usr/bin/env bash
set -e

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

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
export SANDBOX="${SANDBOX:-bwrap}"
export TEACHER_CODE="${TEACHER_CODE:-admin123}"
export OLLAMA_BASE_URL="${OLLAMA_BASE_URL:-http://127.0.0.1:11434}"
export OLLAMA_MODEL="${OLLAMA_MODEL:-llama3.1:8b}"
WEB_PORT="${WEB_PORT:-3000}"

echo -e "${BLUE}Liberando puertos $PORT y $WEB_PORT${NC}"
fuser -k "$PORT/tcp" 2>/dev/null || true
fuser -k "$WEB_PORT/tcp" 2>/dev/null || true

# 1. Dependencias del sistema
echo -e "\n${BLUE}Verificando dependencias del sistema${NC}"
MISSING_PKGS=()
if ! command -v g++ &> /dev/null; then MISSING_PKGS+=("g++"); fi
if ! command -v bwrap &> /dev/null; then MISSING_PKGS+=("bubblewrap"); fi
if ! command -v prlimit &> /dev/null || ! command -v choom &> /dev/null; then MISSING_PKGS+=("util-linux"); fi
if ! command -v curl &> /dev/null; then MISSING_PKGS+=("curl"); fi
if ! command -v fuser &> /dev/null; then MISSING_PKGS+=("psmisc"); fi

if [ ${#MISSING_PKGS[@]} -gt 0 ]; then
    echo -e "${YELLOW}Instalando paquetes faltantes: ${MISSING_PKGS[*]}${NC}"
    if command -v pacman &> /dev/null; then
        sudo pacman -S --needed --noconfirm "${MISSING_PKGS[@]/g++/gcc}"
    else
        sudo apt update && sudo apt install -y "${MISSING_PKGS[@]}"
    fi
else
    echo -e "${GREEN} Paquetes del sistema listos${NC}"
fi

# 2. Go 1.24
echo -e "\n${BLUE}Verificando Go (>=1.24)${NC}"
LOCAL_GO_DIR="$HOME/.local/go1.24"
if [ -x "$LOCAL_GO_DIR/bin/go" ]; then
    export PATH="$LOCAL_GO_DIR/bin:$PATH"
fi

GO_OK=false
if command -v go &> /dev/null; then
    GO_VER_STR=$(go version | awk '{print $3}' | sed 's/go//')
    IFS='.' read -r G_MAJ G_MIN G_PATCH <<< "$GO_VER_STR"
    if [ "$G_MAJ" -gt 1 ] || { [ "$G_MAJ" -eq 1 ] && [ "$G_MIN" -ge 24 ]; }; then
        GO_OK=true
        echo -e "${GREEN}✓ Go $GO_VER_STR${NC}"
    fi
fi

if [ "$GO_OK" = false ]; then
    echo -e "${YELLOW}Instalando Go 1.24 en $LOCAL_GO_DIR${NC}"
    mkdir -p "$HOME/.local"
    curl -fsSL https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -o /tmp/go1.24.tar.gz
    rm -rf "$LOCAL_GO_DIR"
    mkdir -p "$LOCAL_GO_DIR"
    tar -C "$LOCAL_GO_DIR" --strip-components=1 -xzf /tmp/go1.24.tar.gz
    rm -f /tmp/go1.24.tar.gz
    export PATH="$LOCAL_GO_DIR/bin:$PATH"
    echo -e "${GREEN}✓ Go 1.24 instalado${NC}"
fi

# 3. Node.js y dependencias frontend
echo -e "\n${BLUE}Verificando Node.js y dependencias web${NC}"
if ! command -v node &> /dev/null || ! command -v npm &> /dev/null; then
    echo -e "${YELLOW}Instalando Node.js LTS${NC}"
    if command -v pacman &> /dev/null; then
        sudo pacman -S --needed --noconfirm nodejs npm
    else
        curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
        sudo apt install -y nodejs
    fi
fi
if ! NODE_ERR=$(node -v 2>&1); then
    echo -e "${RED}Node.js está instalado pero no arranca:${NC}"
    echo -e "${RED}  $NODE_ERR${NC}"
    echo -e "${RED}  Suele pasar tras una actualización parcial del sistema; actualiza o reinstala nodejs y vuelve a intentar.${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Node $NODE_ERR y npm $(npm -v)${NC}"

LOCK_SUM="$(sha256sum web/package-lock.json | cut -c1-16)"
if [ ! -d "web/node_modules" ] || [ "$(cat web/node_modules/.edutrace-lock 2>/dev/null)" != "$LOCK_SUM" ]; then
    echo -e "${YELLOW}Instalando las dependencias de web/ (cambió package-lock.json)${NC}"
    (cd web && npm install --no-audit --no-fund)
    echo "$LOCK_SUM" > web/node_modules/.edutrace-lock
else
    echo -e "${GREEN}✓ Dependencias de web/ al día${NC}"
fi

# 4. Ollama y modelo
echo -e "\n${BLUE}Verificando Ollama y el modelo $OLLAMA_MODEL${NC}"
if ! command -v ollama &> /dev/null; then
    echo -e "${YELLOW}Instalando Ollama${NC}"
    curl -fsSL https://ollama.com/install.sh | sh
fi

if ! curl -s "$OLLAMA_BASE_URL/api/tags" &> /dev/null; then
    echo -e "${YELLOW}Iniciando Ollama${NC}"
    ollama serve &> /tmp/edutrace_ollama.log &
    for i in {1..15}; do
        curl -s "$OLLAMA_BASE_URL/api/tags" &> /dev/null && break
        sleep 1
    done
fi

MODEL_TAG="$OLLAMA_MODEL"
[[ "$MODEL_TAG" == *:* ]] || MODEL_TAG="$MODEL_TAG:latest"
if ollama list | awk 'NR > 1 {print $1}' | grep -qx "$MODEL_TAG"; then
    echo -e "${GREEN}✓ Modelo $MODEL_TAG listo${NC}"
else
    echo -e "${YELLOW}Descargando $MODEL_TAG${NC}"
    ollama pull "$MODEL_TAG"
fi

# 5. Compilar e iniciar servicios
echo -e "\n${BLUE}Compilando e iniciando servicios${NC}"

if [ "$SANDBOX" = "bwrap" ]; then
    PROBE=(--ro-bind /usr /usr --proc /proc --dev /dev --tmpfs /tmp --unshare-all)
    for p in /bin /sbin /lib /lib64 /lib32; do
        if [ -L "$p" ]; then
            PROBE+=(--symlink "$(readlink "$p")" "$p")
        elif [ -d "$p" ]; then
            PROBE+=(--ro-bind "$p" "$p")
        fi
    done
    if bwrap "${PROBE[@]}" -- /usr/bin/true &> /dev/null; then
        echo -e "${GREEN}✓ Sandbox bwrap disponible${NC}"
    else
        export SANDBOX="none"
        echo -e "${RED} bwrap no puede crear el sandbox requerido${NC}"
        echo -e "${RED} Se usa SANDBOX=none: el código de los estudiantes se ejecuta sin aislamiento. Solo para desarrollo confirmado.${NC}"
    fi
fi

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
go build -o bin/server ./cmd/server
echo -e "${GREEN} Backend compilado en bin/server${NC}"

cleanup() {
    echo -e "\n${YELLOW}Deteniendo servicios${NC}"
    kill 0 2>/dev/null || true
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

./bin/server &
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
