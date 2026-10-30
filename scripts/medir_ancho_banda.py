import gzip
import json
import os
import re
import sys
import urllib.error
import urllib.request
from http.cookiejar import CookieJar

WEB = os.environ.get("WEB", "http://localhost:3000")
API = os.environ.get("API", "http://127.0.0.1:8080")
ANCHOS = [int(x) for x in os.environ.get("ANCHOS_KBPS", "256,1000,3000").split(",")]
PAGINAS = ["/login", "/", "/historial", "/cuenta", "/docente", "/privacidad"]
ESTUDIANTE = ("ana.rodriguez@est.edu.co", "Alumno2026!")

opener = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(CookieJar()))


def bajar(url, comprimido=True, data=None):
    headers = {"Accept-Encoding": "gzip"} if comprimido else {}
    if data is not None:
        headers["Content-Type"] = "application/json"
    req = urllib.request.Request(url, data=data, headers=headers)
    with opener.open(req, timeout=60) as resp:
        raw = resp.read()
        texto = gzip.decompress(raw) if resp.headers.get("Content-Encoding") == "gzip" else raw
        return len(raw), texto.decode("utf-8", "replace"), resp.headers.get("Content-Encoding") == "gzip"


def segundos(bytes_, kbps):
    return bytes_ * 8 / (kbps * 1000)


def kb(n):
    return f"{n / 1024:7.1f} KB"


try:
    _, html, _ = bajar(WEB + "/login")
except urllib.error.URLError as e:
    sys.exit(f"  ✗ No se pudo abrir {WEB}: {e.reason}")
if "webpack-hmr" in html or "/_next/static/development/" in html:
    print("  ⚠ El frontend está en modo desarrollo: mide con «npm run build && npm run start» para obtener el peso real.\n")

vistos, total_unico, sin_gzip = set(), 0, []
print(f"1. Primera carga de cada página ({WEB}), comprimida")
print(f"   {'página':12} {'HTML':>10} {'JS':>10} {'CSS':>10} {'fuentes':>10} {'total':>10}   " + "   ".join(f"a {k} kbps" for k in ANCHOS))
for pagina in PAGINAS:
    peso_html, html, comp = bajar(WEB + pagina)
    if not comp:
        sin_gzip.append(pagina)
    partes = {"js": 0, "css": 0, "fuentes": 0}
    recursos = set(re.findall(r'(?:src|href)="(/_next/static/[^"]+\.(?:js|css))"', html))
    for recurso in sorted(recursos):
        peso, texto, comp = bajar(WEB + recurso)
        if not comp and peso > 1024:
            sin_gzip.append(recurso)
        if recurso.endswith(".css"):
            partes["css"] += peso
            for cara in re.findall(r"@font-face\{[^}]*\}", texto):
                rango = re.search(r"unicode-range:([^;}]+)", cara)
                fuente = re.search(r"url\((/_next/static/media/[^)]+\.woff2)\)", cara)
                if not fuente or (rango and "u+00??" not in rango.group(1).lower()):
                    continue
                fuente = fuente.group(1)
                peso_f, _, _ = bajar(WEB + fuente)
                partes["fuentes"] += peso_f
                if fuente not in vistos:
                    vistos.add(fuente)
                    total_unico += peso_f
        else:
            partes["js"] += peso
        if recurso not in vistos:
            vistos.add(recurso)
            total_unico += peso
    total = peso_html + sum(partes.values())
    total_unico += peso_html
    tiempos = "   ".join(f"{segundos(total, k):9.1f} s" for k in ANCHOS)
    print(f"   {pagina:12} {kb(peso_html):>10} {kb(partes['js']):>10} {kb(partes['css']):>10} {kb(partes['fuentes']):>10} {kb(total):>10}   {tiempos}")
print(f"   Todas las páginas juntas (lo compartido se descarga una vez): {kb(total_unico)}")
print("   El editor de código se descarga aparte al abrir el compilador.")

print(f"\n2. Respuestas de la API ({API}) para un estudiante")
try:
    bajar(API + "/api/auth/login", data=json.dumps({"email": ESTUDIANTE[0], "password": ESTUDIANTE[1]}).encode())
    for ruta in ["/api/auth/me", "/api/student/groups", "/api/student/assignments", "/api/student/submissions", "/api/student/history"]:
        con, _, comp = bajar(API + ruta)
        sin, _, _ = bajar(API + ruta, comprimido=False)
        print(f"   {ruta:28} {kb(sin)} → {kb(con)} con gzip{'' if comp or sin < 200 else '  ⚠ sin comprimir'}")
except urllib.error.HTTPError as e:
    print(f"   ⚠ No se pudo entrar como {ESTUDIANTE[0]} ({e.code}); carga los datos de prueba para medir la API.")
except urllib.error.URLError as e:
    print(f"   ⚠ No se pudo conectar con {API}: {e.reason}")

print("\n3. Durante el uso: compilar envía y recibe unos pocos KB; la retroalimentación de la IA llega en streaming (unos 2 KB de texto).")
if sin_gzip:
    print("   ⚠ Sin compresión: " + ", ".join(sin_gzip[:5]))
