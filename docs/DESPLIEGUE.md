# Integración y despliegue continuos

## CI: `.github/workflows/ci.yaml`

Corre en cada *push* a `main` o `develop`, en cada pull request y a mano desde la pestaña Actions. Tiene dos trabajos en paralelo:

| Trabajo | Qué verifica |
|---|---|
| Backend (Go) | `go build ./...`, `go vet ./...` y `go test ./...`, incluidas las pruebas que compilan C++ con `g++` |
| Frontend (Next.js) | `npm ci`, `tsc --noEmit`, `npm run lint` y `npm run build` |

Si un *push* nuevo llega a la misma rama, se cancela la corrida anterior.

## CD: `.github/workflows/cd.yaml`

Se dispara cuando el CI termina bien en `main`. También se puede lanzar a mano en «Run workflow», con un commit concreto para volver a una versión anterior.

El servidor necesita GPU, Ollama y `bubblewrap`, así que el despliegue corre en el propio servidor con `scripts/desplegar.sh`:
1. lleva el repositorio al commit indicado;
2. compila el backend en `bin/edutrace-server` y el frontend con `npm run build`;
3. reinicia `edutrace-backend` y `edutrace-web`;
4. espera hasta 60 s a que `/api/health` responda.

Si la compilación o el arranque fallan, vuelve al commit anterior, lo recompila y lo reinicia. El workflow queda en rojo.

Hay dos formas de llegar al servidor. Se elige con la variable `DEPLOY_MODE`; sin ella, el CD no hace nada.

| `DEPLOY_MODE` | Cuándo usarlo | Cómo llega |
|---|---|---|
| `ssh` | El servidor es accesible desde internet | GitHub entra por SSH y ejecuta el script |
| `self-hosted` | El servidor está en la red de la universidad | Un runner de GitHub instalado en el servidor ejecuta el script |

## Preparar el servidor (una sola vez)

Requisitos: Go 1.26 o superior, Node.js 22, `git`, `g++`, `bubblewrap`, `util-linux`, `curl` y Ollama con el modelo de `.env`.

```bash
sudo useradd --system --create-home --home-dir /opt/edutrace --shell /bin/bash edutrace
sudo -u edutrace git clone https://github.com/x19marlon/EDU-TRACE.git /opt/edutrace
sudo -u edutrace cp /opt/edutrace/.env.example /opt/edutrace/.env
sudo -u edutrace nano /opt/edutrace/.env
```

En `.env` de producción:
- `TEACHER_CODE`: el código para crear cuentas de profesor;
- `NEXT_PUBLIC_API_URL` y `ALLOWED_ORIGINS`: las direcciones públicas;
- `COOKIE_SECURE=true` cuando haya HTTPS.

```bash
sudo cp /opt/edutrace/deploy/edutrace-backend.service /opt/edutrace/deploy/edutrace-web.service /etc/systemd/system/
sudo install -m 440 /opt/edutrace/deploy/sudoers-edutrace /etc/sudoers.d/edutrace
sudo systemctl daemon-reload
sudo -u edutrace /opt/edutrace/scripts/desplegar.sh origin/main
sudo systemctl enable --now edutrace-backend edutrace-web
```

La regla de `sudoers` solo permite al usuario `edutrace` reiniciar esos dos servicios. Los registros se ven con `journalctl -u edutrace-backend -u edutrace-web`.

## Configurar GitHub

En el repositorio, «Settings»:

- **Variables** (Secrets and variables → Actions → Variables):
  - `DEPLOY_MODE`: `ssh` o `self-hosted`;
  - `DEPLOY_PATH`: opcional, por defecto `/opt/edutrace`.
- **Secrets**, solo para `ssh`:
  - `DEPLOY_HOST` y `DEPLOY_USER` (`edutrace`);
  - `DEPLOY_SSH_KEY`: una llave privada cuya pública esté en `/opt/edutrace/.ssh/authorized_keys`;
  - `DEPLOY_KNOWN_HOSTS`: la salida de `ssh-keyscan <servidor>`.
- **Environment `produccion`** (Environments): se crea solo con el primer despliegue. Si se le agregan revisores obligatorios, cada despliegue espera una aprobación.
- **Runner del servidor**, solo para `self-hosted`:
  1. Actions → Runners → New self-hosted runner, y seguir las instrucciones como usuario `edutrace`;
  2. agregarle la etiqueta `edutrace`;
  3. instalarlo como servicio con `sudo ./svc.sh install edutrace`.

Para subir cambios en `.github/workflows/` por HTTPS, el token de GitHub necesita el permiso `workflow`.
