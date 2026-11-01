# Integración, despliegue y operación

## CI: `.github/workflows/ci.yaml`

Corre en cada *push* a `main` o `develop`, en cada pull request, a mano desde la pestaña Actions y cada lunes a las 06:00 (hora de Colombia), para enterarse de vulnerabilidades nuevas aunque nadie suba cambios.

| Trabajo | Qué verifica |
|---|---|
| Backend (Go) | `go build`, `go vet`, `go test` (incluye las pruebas que compilan C++, las del sandbox y la de respaldo y restauración) y `govulncheck` |
| Frontend (Next.js) | `npm ci`, `npm audit --audit-level=high`, `tsc --noEmit`, `npm run lint` y `npm run build` |
| Accesibilidad (WCAG 2.1 AA) | Levanta el backend y el frontend, crea un profesor, una clase, un taller, un estudiante, una entrega y una revisión, y revisa con axe las páginas públicas, las del estudiante y las del profesor (`web/e2e/accesibilidad.spec.ts`). Falla con cualquier problema de nivel A o AA. |

Si un *push* nuevo llega a la misma rama, se cancela la corrida anterior.

`.github/dependabot.yml` abre cada semana pull requests con las actualizaciones de Go, npm y las acciones de GitHub. Los saltos de versión mayor de Next.js, ESLint y Tailwind se hacen a mano, porque cambian la configuración.

Para probar la accesibilidad en local, con el backend y el frontend en marcha:

```bash
cd web
npx playwright install --only-shell chromium
E2E_API=http://localhost:8080 E2E_WEB=http://localhost:3000 E2E_TEACHER_CODE=<el TEACHER_CODE del backend> npm run test:a11y
```

## CD: `.github/workflows/cd.yaml`

Se dispara cuando el CI termina bien después de un *push* a `main` (no con la corrida semanal). También se puede lanzar a mano en «Run workflow», con un commit concreto para volver a una versión anterior.

Si la variable `STAGING_PATH` existe, cada versión se despliega **primero en el entorno de pruebas** (environment `pruebas`) y solo si arranca bien pasa a producción (environment `produccion`). Si a `produccion` se le agregan revisores obligatorios, alguien revisa la versión en el dominio de pruebas y aprueba el paso a producción.

El servidor necesita GPU, Ollama y `bubblewrap`, así que el despliegue corre en el propio servidor con `scripts/desplegar.sh`:
1. lleva el repositorio al commit indicado;
2. compila `bin/edutrace-server`, `bin/edutrace-respaldo` y el frontend;
3. reinicia `edutrace-backend` y `edutrace-web`;
4. espera hasta 60 s a que respondan `/api/health` y la página de ingreso del frontend.

Si la compilación o el arranque fallan, vuelve al commit anterior, lo recompila y lo reinicia. El workflow queda en rojo.

Hay dos formas de llegar al servidor. Se elige con la variable `DEPLOY_MODE`; sin ella, el CD no hace nada.

| `DEPLOY_MODE` | Cuándo usarlo | Cómo llega |
|---|---|---|
| `ssh` | El servidor es accesible desde internet | GitHub entra por SSH y ejecuta el script |
| `self-hosted` | El servidor está en la red de la universidad | Un runner de GitHub instalado en el servidor ejecuta el script |

## Entorno de pruebas (opcional, recomendado)

Es una segunda copia de EduTrace en el mismo servidor, con sus propios datos, puertos y dominio. Comparte Ollama con producción.

```bash
sudo mkdir -p /opt/edutrace-pruebas && sudo chown edutrace: /opt/edutrace-pruebas
sudo -u edutrace git clone https://github.com/x19marlon/EDU-TRACE.git /opt/edutrace-pruebas
sudo -u edutrace cp /opt/edutrace/.env /opt/edutrace-pruebas/.env
sudo -u edutrace nano /opt/edutrace-pruebas/.env
```

En el `.env` de pruebas cambia `PORT=8081`, `WEB_PORT=3001`, el dominio de pruebas en `PUBLIC_URL`, `NEXT_PUBLIC_API_URL` y `ALLOWED_ORIGINS`, y usa otro `TEACHER_CODE`. Deja `SMTP_HOST` vacío para que las pruebas no envíen correos a estudiantes reales.

```bash
sudo cp /opt/edutrace/deploy/edutrace-pruebas-backend.service /opt/edutrace/deploy/edutrace-pruebas-web.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo -u edutrace env SERVICIOS="edutrace-pruebas-backend edutrace-pruebas-web" /opt/edutrace-pruebas/scripts/desplegar.sh origin/main
sudo systemctl enable --now edutrace-pruebas-backend edutrace-pruebas-web
```

El dominio de pruebas va en `EDUTRACE_DOMINIO_PRUEBAS` (junto a `EDUTRACE_DOMINIO` en `/etc/caddy/edutrace.env`); el `Caddyfile` le agrega `X-Robots-Tag: noindex` para que no aparezca en buscadores. En GitHub, agrega la variable `STAGING_PATH=/opt/edutrace-pruebas`.

## Preparar el servidor (una sola vez)

Requisitos: Go 1.26 o superior, Node.js 22, `git`, `g++`, `bubblewrap` 0.8 o superior, `util-linux` (`prlimit` y `choom`), `curl`, `python3`, Caddy y Ollama con el modelo de `.env`. `rsync` solo si se copian los respaldos a otra máquina.

```bash
sudo useradd --system --create-home --home-dir /opt/edutrace --shell /bin/bash edutrace
sudo -u edutrace git clone https://github.com/x19marlon/EDU-TRACE.git /opt/edutrace
sudo -u edutrace cp /opt/edutrace/.env.example /opt/edutrace/.env
sudo -u edutrace chmod 600 /opt/edutrace/.env
sudo -u edutrace nano /opt/edutrace/.env
```

En `.env` de producción, con el dominio real en lugar de `edutrace.ejemplo.edu.co`:

```bash
TEACHER_CODE=<un código largo y aleatorio>
PUBLIC_URL=https://edutrace.ejemplo.edu.co
NEXT_PUBLIC_API_URL=https://edutrace.ejemplo.edu.co
ALLOWED_ORIGINS=https://edutrace.ejemplo.edu.co
COOKIE_SECURE=true
TRUST_PROXY=true
```

Después:

```bash
cd /opt/edutrace/deploy
sudo cp edutrace-backend.service edutrace-web.service edutrace-respaldo.service edutrace-respaldo.timer \
  edutrace-vigilar.service edutrace-vigilar.timer /etc/systemd/system/
sudo install -m 440 sudoers-edutrace /etc/sudoers.d/edutrace
sudo systemctl daemon-reload
sudo -u edutrace /opt/edutrace/scripts/desplegar.sh origin/main
sudo systemctl enable --now edutrace-backend edutrace-web edutrace-respaldo.timer edutrace-vigilar.timer
```

La regla de `sudoers` solo permite al usuario `edutrace` reiniciar los servicios de producción y de pruebas, y detener o iniciar el backend (para restaurar respaldos). Los registros se ven con `journalctl -u edutrace-backend -u edutrace-web`.

## HTTPS

`deploy/Caddyfile` publica EduTrace en un solo dominio: `/api/*` va al backend (puerto 8080) y todo lo demás al frontend (puerto 3000). Caddy obtiene y renueva el certificado solo, agrega HSTS y no almacena las respuestas en búfer, así que la retroalimentación sigue llegando por partes. El backend y el frontend escuchan solo en `127.0.0.1`.

```bash
sudo cp /opt/edutrace/deploy/Caddyfile /etc/caddy/Caddyfile
echo 'EDUTRACE_DOMINIO=edutrace.ejemplo.edu.co' | sudo tee /etc/caddy/edutrace.env
sudo systemctl edit caddy   # agrega: [Service]  EnvironmentFile=/etc/caddy/edutrace.env
sudo systemctl restart caddy
```

El dominio debe apuntar al servidor y los puertos 80 y 443 deben estar abiertos. Si la universidad entrega su propio certificado, se cambia la línea del dominio por `{$EDUTRACE_DOMINIO} { tls /ruta/cert.pem /ruta/llave.pem` y lo demás queda igual.

Con `TRUST_PROXY=true`, el backend toma la IP real del estudiante del encabezado `X-Forwarded-For` que pone Caddy. Sin esa opción, todos los pedidos parecerían venir de `127.0.0.1` y el bloqueo por intentos fallidos bloquearía a todo el mundo a la vez.

## Sandbox y límites del servicio

Cada programa de un estudiante corre en `bwrap`, sin red y con estos límites:

| Límite | Al ejecutar | Al compilar |
|---|---|---|
| Procesos | 8 | 16 |
| Memoria (espacio de direcciones) | 256 MB | 1 GB |
| `/tmp` | 8 MB | 64 MB |
| Tamaño de cada archivo | 1 MB | 64 MB |
| CPU | `RUN_TIMEOUT_SECS` + 1 s | `COMPILE_TIMEOUT_SECS` + 1 s |

A lo sumo `MAX_SANDBOXES` programas compilan o corren a la vez (por defecto, uno por núcleo). Los demás esperan hasta 60 s. Si el sistema se queda sin memoria, el kernel mata primero los programas de los estudiantes (`choom -n 1000`) y no el servidor.

`edutrace-backend.service` agrega `NoNewPrivileges`, `PrivateTmp`, `ProtectSystem=full`, `ProtectHome`, `TasksMax=1024` y `MemoryMax=4G`: aunque algo escape a los límites anteriores, no puede tumbar el resto de la máquina. Ajusta `MemoryHigh` y `MemoryMax` a la memoria del servidor; Ollama corre en su propio servicio y no cuenta para ese límite.

Al arrancar, el backend comprueba que el sandbox funcione con sus límites y no arranca si no lo logra.

## Respaldos

`edutrace-respaldo.timer` ejecuta `scripts/respaldar.sh` cada día a las 02:30. El script:
1. copia la base con `VACUUM INTO`, una copia consistente aunque EduTrace siga funcionando;
2. empaca la base y los archivos de los talleres en `respaldos/edutrace-AAAAMMDD-HHMMSS.tar.gz`;
3. **verifica el respaldo recién creado**: lo extrae en una carpeta temporal, revisa la integridad de la base (`PRAGMA integrity_check`) y que estén todos los archivos de los talleres. Si algo falla, lo borra y termina con error;
4. borra los respaldos más antiguos y deja los últimos `BACKUP_KEEP` (14);
5. si `RESPALDO_REMOTO` tiene un destino de `rsync` (por ejemplo `respaldos@otra-maquina:/srv/edutrace/`), copia allí la carpeta. Sin una copia fuera del servidor, una falla del disco se lleva también los respaldos.

Los respaldos contienen el código y los datos de todos los estudiantes: la carpeta queda con permisos `700` y los archivos con `600`.

### Respaldos cifrados

Con `BACKUP_RECIPIENT` en `.env`, cada respaldo se cifra con [age](https://age-encryption.org) y queda como `edutrace-AAAAMMDD-HHMMSS.tar.gz.age`. Se usa cifrado de clave pública: el servidor solo guarda la clave pública, así que ni quien entre al servidor ni quien vea la copia de `RESPALDO_REMOTO` puede leer los respaldos. El respaldo se verifica completo **antes** de cifrarlo.

```bash
sudo -u edutrace /opt/edutrace/bin/edutrace-respaldo -generar-clave
```

Imprime la clave privada y la pública. Guarda la privada **fuera del servidor** (gestor de contraseñas o USB): sin ella no se puede restaurar ningún respaldo cifrado. Pon la pública en `.env` como `BACKUP_RECIPIENT`.

Para verificar o restaurar un respaldo cifrado, copia la clave privada al servidor solo mientras la necesitas y apúntala con `BACKUP_IDENTITY`:

```bash
sudo -u edutrace env BACKUP_IDENTITY=/ruta/clave.txt /opt/edutrace/scripts/restaurar.sh /opt/edutrace/respaldos/edutrace-20261101-023000.tar.gz.age
```

Para revisar un respaldo sin tocar nada:

```bash
sudo -u edutrace /opt/edutrace/bin/edutrace-respaldo -verificar /opt/edutrace/respaldos/edutrace-20261101-023000.tar.gz
```

Para restaurar:

```bash
sudo -u edutrace /opt/edutrace/scripts/restaurar.sh /opt/edutrace/respaldos/edutrace-20261101-023000.tar.gz
```

El script verifica el respaldo, lo extrae en una carpeta nueva, pide confirmación, detiene el backend, deja los datos actuales en `data.antes-<fecha>`, pone los del respaldo y vuelve a iniciar. Si EduTrace no arranca, devuelve los datos anteriores.

La restauración se prueba en cada CI (`TestBackupAndRestore`): se crea una base con datos, se respalda, se restaura en otra carpeta y EduTrace la abre y deja entrar al profesor. Conviene repetir la restauración en el servidor una vez por periodo, en una copia de prueba.

## Monitoreo y alertas

`GET /api/health` responde:

```json
{"status": "ok", "checks": {"base": true, "ia": true, "disco": true, "cola": true}}
```

- `base`: la base de datos responde. Si falla, el estado es `caido` y el código HTTP es 503.
- `ia`: Ollama responde.
- `disco`: queda al menos `DISK_MIN_FREE_PERCENT` % libre en `DATA_DIR`.
- `cola`: la cola de la IA no lleva más de 30 minutos sin terminar ningún trabajo teniendo trabajos esperando.

Si algo falla que no sea la base, el estado es `degradado`, pero el código sigue siendo 200: el despliegue no se revierte porque Ollama esté apagado.

`edutrace-vigilar.timer` ejecuta `scripts/vigilar.sh` cada 5 minutos. Además de lo anterior, revisa que el frontend responda y que haya un respaldo de menos de 26 horas. Avisa una vez cuando aparece un problema, lo repite cada 6 horas si sigue y avisa cuando todo vuelve a la normalidad.

| Variable | Para qué |
|---|---|
| `ALERTA_URL` | Adónde se envía el aviso. Con [ntfy](https://ntfy.sh) basta un tema (`https://ntfy.sh/edutrace-<algo-aleatorio>`) y la aplicación en el teléfono. |
| `ALERTA_JSON=true` | Envía `{"text": …, "content": …}`, el formato de los webhooks de Slack, Discord y Google Chat. |
| `LATIDO_URL` | Una URL de Healthchecks.io o el «push» de Uptime Kuma. Se visita solo cuando todo está bien: si el servidor entero se cae, ese servicio externo deja de recibir visitas y avisa. |

## Avisos por correo

Si se definen `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD` y `SMTP_FROM` (por ejemplo `EduTrace <no-responder@ejemplo.edu.co>`), EduTrace envía por correo los avisos que nadie abrió en 2 minutos, agrupados por persona. Con el puerto 587 usa STARTTLS, y con 465, TLS directo. Si el servidor rechaza una dirección, ese aviso no se reintenta; si el servidor de correo no responde, se reintenta cada minuto durante 24 horas. Sin SMTP, los avisos se ven solo en la campana.

## Configurar GitHub

En el repositorio, «Settings»:

- **Variables** (Secrets and variables → Actions → Variables):
  - `DEPLOY_MODE`: `ssh` o `self-hosted`;
  - `DEPLOY_PATH`: opcional, por defecto `/opt/edutrace`;
  - `STAGING_PATH`: opcional; con ella cada versión pasa primero por el entorno de pruebas.
- **Secrets**, solo para `ssh`:
  - `DEPLOY_HOST` y `DEPLOY_USER` (`edutrace`);
  - `DEPLOY_SSH_KEY`: una llave privada cuya pública esté en `/opt/edutrace/.ssh/authorized_keys`;
  - `DEPLOY_KNOWN_HOSTS`: la salida de `ssh-keyscan <servidor>`.
- **Environments `pruebas` y `produccion`**: se crean solos con el primer despliegue. Si a `produccion` se le agregan revisores obligatorios, cada versión espera una aprobación después de pasar por pruebas.
- **Runner del servidor**, solo para `self-hosted`:
  1. Actions → Runners → New self-hosted runner, y seguir las instrucciones como usuario `edutrace`;
  2. agregarle la etiqueta `edutrace`;
  3. instalarlo como servicio con `sudo ./svc.sh install edutrace`.
- **Dependabot** (Code security): activar «Dependabot alerts» y «Dependabot security updates».

Para subir cambios en `.github/workflows/` por HTTPS, el token de GitHub necesita el permiso `workflow`.
