#!/usr/bin/env bash
set -euo pipefail

REPO="${REPO:-https://github.com/x19marlon/EDU-TRACE.git}"
RUTA="${RUTA:-data-backup-2026-10-21}"
TMP="$(mktemp -d)"
DIR="$TMP/repo.git"

echo "1. Copia espejo de $REPO en $DIR"
git clone --quiet --bare "$REPO" "$DIR"
cd "$DIR"
git for-each-ref --format='%(refname) %(objectname)' refs/heads refs/tags > "$TMP/antes.txt"

if ! git log --all --oneline -- "$RUTA" | grep -q .; then
  echo "   $RUTA no aparece en el historial: no hay nada que limpiar."
  exit 0
fi
echo "   Commits que contienen $RUTA:"
git log --all --format='     %h %ad %s' --date=short -- "$RUTA"

echo "2. Quitando $RUTA de todo el historial"
if command -v git-filter-repo > /dev/null; then
  git filter-repo --quiet --force --invert-paths --path "$RUTA"
else
  FILTER_BRANCH_SQUELCH_WARNING=1 git filter-branch --force \
    --index-filter "git rm -r -q --cached --ignore-unmatch '$RUTA'" \
    --prune-empty --tag-name-filter cat -- --all > /dev/null
  git for-each-ref --format='%(refname)' refs/original | xargs -r -n1 git update-ref -d
fi
git reflog expire --expire=now --all
git gc --quiet --prune=now

echo "3. Verificando"
if git log --all --oneline -- "$RUTA" | grep -q .; then
  echo "   ✗ $RUTA sigue en el historial. No se publica nada."
  exit 1
fi
if git rev-list --all --objects | grep -q "$RUTA"; then
  echo "   ✗ Quedan objetos de $RUTA. No se publica nada."
  exit 1
fi
git for-each-ref --format='%(refname) %(objectname)' refs/heads refs/tags > "$TMP/despues.txt"
CAMBIADAS=()
while read -r ref nuevo; do
  viejo="$(grep -F "$ref " "$TMP/antes.txt" | cut -d' ' -f2 || true)"
  if [ "$viejo" != "$nuevo" ]; then
    CAMBIADAS+=("$ref:$viejo:$nuevo")
    echo "   $ref: ${viejo:0:7} → ${nuevo:0:7}"
  fi
done < "$TMP/despues.txt"
echo "   ✓ $RUTA ya no está en ninguna rama ni etiqueta."

echo ""
echo "4. Publicar reescribe esas ramas en $REPO para todos los que tengan un clon."
read -r -p "   Escribe SI para hacer el force push (cualquier otra cosa cancela): " RESPUESTA
if [ "$RESPUESTA" != "SI" ]; then
  echo "   Cancelado. La copia limpia quedó en $DIR"
  exit 0
fi
for item in "${CAMBIADAS[@]}"; do
  IFS=: read -r ref viejo nuevo <<< "$item"
  git push --quiet --force-with-lease="$ref:$viejo" "$REPO" "$nuevo:$ref"
  echo "   ✓ $ref publicado"
done

echo ""
echo "5. Después de publicar:"
echo "   - En cada clon (también el tuyo): git fetch origin && git reset --soft origin/main"
echo "     (--soft conserva tus cambios sin commit; el contenido de main no cambia, solo su historial)."
echo "   - Pide a GitHub Support que purgue las vistas en caché de los commits viejos:"
echo "     https://support.github.com/contact (Remove cached views / sensitive data)."
echo "   - Revisa que no haya forks ni pull requests que conserven esos commits."
echo "   - Pide a las personas de ese respaldo que cambien su contraseña: sus hashes estuvieron publicados."
