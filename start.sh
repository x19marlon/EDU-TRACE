#!/usr/bin/env bash
set -e

# ==============================================================================
# Script de arranque y configuración automática para EDU-TRACE
# ==============================================================================

GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Liberar puertos en caso de que hayan quedado procesos colgados
echo -e "${BLUE}=== [0/5] Liberando puertos anteriores (8080 y 3000) ===${NC}"
fuser -k 8080/tcp 2>/dev/null || true
fuser -k 3000/tcp 2>/dev/null || true

echo -e "${BLUE}=== [1/5] Verificando dependencias del sistema ===${NC}"
MISSING_PKGS=()
if ! command -v g++ &> /dev/null; then MISSING_PKGS+=("g++"); fi
if ! command -v bwrap &> /dev/null; then MISSING_PKGS+=("bubblewrap"); fi
if ! command -v prlimit &> /dev/null; then MISSING_PKGS+=("util-linux"); fi
if ! command -v curl &> /dev/null; then MISSING_PKGS+=("curl"); fi
if ! command -v fuser &> /dev/null; then MISSING_PKGS+=("psmisc"); fi

if [ ${#MISSING_PKGS[@]} -gt 0 ]; then
    echo -e "${YELLOW}Instalando paquetes del sistema faltantes: ${MISSING_PKGS[*]}.${NC}"
    sudo apt update && sudo apt install -y "${MISSING_PKGS[@]}"
else
    echo -e "${GREEN}✓ Paquetes base del sistema OK.${NC}"
fi

# ------------------------------------------------------------------------------
# 2. Configurar Go 1.24
# ------------------------------------------------------------------------------
echo -e "\n${BLUE}=== [2/5] Verificando Go (>= 1.24) ===${NC}"
LOCAL_GO_DIR="$HOME/.local/go1.24"
if [ -x "$LOCAL_GO_DIR/bin/go" ]; then
    export PATH="$LOCAL_GO_DIR/bin:$PATH"
fi

GO_OK=false
if command -v go &> /dev/null; then
    GO_VER_STR=$(go version | awk '{print $3}' | sed 's/go//')
    IFS='.' read -r G_MAJ G_MIN G_PATCH <<< "$GO_VER_STR"
    if [ "$G_MAJ" -gt 1 ] || ([ "$G_MAJ" -eq 1 ] && [ "$G_MIN" -ge 24 ]); then
        GO_OK=true
        echo -e "${GREEN}✓ Go $GO_VER_STR disponible.${NC}"
    fi
fi

if [ "$GO_OK" = false ]; then
    echo -e "${YELLOW}Instalando Go 1.24 en $LOCAL_GO_DIR...${NC}"
    mkdir -p "$HOME/.local"
    curl -fsSL https://go.dev/dl/go1.24.1.linux-amd64.tar.gz -o /tmp/go1.24.tar.gz
    rm -rf "$LOCAL_GO_DIR"
    mkdir -p "$LOCAL_GO_DIR"
    tar -C "$LOCAL_GO_DIR" --strip-components=1 -xzf /tmp/go1.24.tar.gz
    rm -f /tmp/go1.24.tar.gz
    export PATH="$LOCAL_GO_DIR/bin:$PATH"
    echo -e "${GREEN}✓ Go 1.24 configurado.${NC}"
fi

# ------------------------------------------------------------------------------
# 3. Node.js y dependencias frontend
# ------------------------------------------------------------------------------
echo -e "\n${BLUE}=== [3/5] Verificando Node.js y dependencias Web ===${NC}"
if ! command -v node &> /dev/null || ! command -v npm &> /dev/null; then
    echo -e "${YELLOW}Instalando Node.js LTS...${NC}"
    curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
    sudo apt install -y nodejs
fi
echo -e "${GREEN}✓ Node $(node -v) y npm $(npm -v) listos.${NC}"

if [ ! -d "web/node_modules" ]; then
    echo -e "${YELLOW}Instalando dependencias en web/...${NC}"
    (cd web && npm install)
else
    echo -e "${GREEN}✓ Dependencias de web/ ya instaladas.${NC}"
fi

# ------------------------------------------------------------------------------
# 4. Ollama e IA
# ------------------------------------------------------------------------------
echo -e "\n${BLUE}=== [4/5] Verificando Ollama y Modelo LLM ===${NC}"
if ! command -v ollama &> /dev/null; then
    echo -e "${YELLOW}Instalando Ollama...${NC}"
    curl -fsSL https://ollama.com/install.sh | sh
fi

if ! curl -s http://127.0.0.1:11434/api/tags &> /dev/null; then
    echo -e "${YELLOW}Iniciando servicio de Ollama...${NC}"
    ollama serve &> /tmp/edutrace_ollama.log &
    sleep 3
fi

MODEL="llama3.2:latest"
if ! ollama list | grep -q "llama3.2"; then
    echo -e "${YELLOW}Descargando modelo $MODEL...${NC}"
    ollama pull "$MODEL"
else
    echo -e "${GREEN}✓ Modelo $MODEL listo.${NC}"
fi

# ------------------------------------------------------------------------------
# 5. Compilar backend y lanzar todo
# ------------------------------------------------------------------------------
echo -e "\n${BLUE}=== [5/5] Compilando e iniciando servicios ===${NC}"

export TEACHER_CODE="${TEACHER_CODE:-admin123}"
export OLLAMA_HOST="${OLLAMA_HOST:-http://127.0.0.1:11434}"
export HOST="127.0.0.1"
export PORT="8080"

# Detección de sandbox: Ubuntu 24.04 restringe namespaces sin privilegios por defecto.
# Si bwrap falla al probarlo, activamos SANDBOX=none para que el backend funcione en desarrollo local.
if bwrap --ro-bind /usr /usr --dev /dev --proc /proc --tmpfs /tmp --unshare-net -- /usr/bin/true &> /dev/null; then
    export SANDBOX="bwrap"
    echo -e "${GREEN}✓ Sandbox 'bwrap' disponible.${NC}"
else
    export SANDBOX="none"
    echo -e "${YELLOW}⚠️  Ubuntu 24.04 restringe userns no privilegiados para bwrap.${NC}"
    echo -e "${YELLOW}    Usando SANDBOX=none para entorno de desarrollo local.${NC}"
fi

echo -e "Compilando Backend Go..."
mkdir -p bin
go build -o bin/server ./cmd/server
echo -e "${GREEN}✓ Backend compilado en bin/server.${NC}"

cleanup() {
    echo -e "\n${YELLOW}Deteniendo servicios...${NC}"
    kill 0 2>/dev/null || true
    exit 0
}
trap cleanup SIGINT SIGTERM EXIT

# 1. Iniciar Backend
echo -e "${GREEN}Iniciando Backend en http://127.0.0.1:8080...${NC}"
./bin/server &
BACKEND_PID=$!

# Esperar a que el backend esté listo
for i in {1..10}; do
    if curl -s http://127.0.0.1:8080/ &> /dev/null || curl -s http://127.0.0.1:8080/api/auth/me &> /dev/null; then
        echo -e "${GREEN}✓ Backend respondiendo correctamente.${NC}"
        break
    fi
    sleep 1
done

# 2. Iniciar Frontend Web en puerto 3000
echo -e "${GREEN}Iniciando Frontend Web en http://localhost:3000...${NC}"
(cd web && PORT=3000 npm run dev -- -p 3000) &
FRONTEND_PID=$!

echo -e "\n${GREEN}====================================================${NC}"
echo -e "${GREEN}   EDU-TRACE iniciado correctamente!${NC}"
echo -e "${GREEN}   👉 Abre en tu navegador: http://localhost:3000${NC}"
echo -e "${GREEN}   - Backend API: http://localhost:8080${NC}"
echo -e "${GREEN}   - Ollama IA:   http://127.0.0.1:11434 ($MODEL)${NC}"
echo -e "${YELLOW}   Código de registro docente: ${TEACHER_CODE}${NC}"
echo -e "${GREEN}====================================================${NC}"
echo -e "Presiona [Ctrl+C] para detener todo."

wait
