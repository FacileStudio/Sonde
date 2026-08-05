#!/usr/bin/env sh
# Vérifie une app Facile déployée en mono-conteneur, côté navigateur autant que côté API.
#
#   sh verify-app.sh capsule [chemin-config-au-boot]
#
# Existe parce qu'un /api/health vert ne dit rien de ce que le navigateur appelle :
# c'est exactement l'angle mort qui a laissé partir un front cassé en production.

set -eu

app="${1:?usage: sh scripts/verify-deploy.sh <app> [boot-path] [domaine]}"
boot="${2:-/api/auth/config}"
base="https://${app}.${3:-facile.studio}"
fails=0

check() { # libellé, attendu, obtenu
  if [ "$2" = "$3" ]; then
    printf '  ok    %-46s %s\n' "$1" "$3"
  else
    printf '  FAIL  %-46s %s (attendu %s)\n' "$1" "$3" "$2"
    fails=$((fails + 1))
  fi
}

code() { curl -s --max-time 15 -o /dev/null -w '%{http_code}' "$1"; }
ctype() { curl -s --max-time 15 -o /dev/null -w '%{content_type}' "$1" | cut -d';' -f1; }

printf '== %s ==\n' "$base"

# 1. Le front se charge, et c'est bien du HTML.
check "GET /"                        200 "$(code "$base/")"
check "GET / content-type"           "text/html" "$(ctype "$base/")"

# 2. Une route cliente inexistante côté serveur retombe sur l'index (routage SPA).
check "GET /une-route-cliente"       200 "$(code "$base/une-route-cliente-inventee")"

# 3. Un asset hashé réellement référencé par la page est servi.
asset=$(curl -s --max-time 15 "$base/" | grep -o '/_app/immutable/[A-Za-z0-9/_.-]*\.js' | head -1 || true)
if [ -n "$asset" ]; then
  check "GET $asset"                 200 "$(code "$base$asset")"
else
  printf '  WARN  aucun asset _app/immutable trouvé dans la page\n'
fi

# 4. Un bundle absent doit 404, surtout pas recevoir l'index : sinon le navigateur
#    signale une erreur de MIME et la vraie cause reste invisible.
check "GET bundle absent"            404 "$(code "$base/_app/immutable/absent.OLDHASH.js")"

# 5. L'appel que le front fait au démarrage. C'est celui-ci qui manquait.
check "GET $boot"                    200 "$(code "$base$boot")"
check "GET $boot content-type"       "application/json" "$(ctype "$base$boot")"

# 6. Le préfixe ne doit pas être doublé.
check "GET /api$boot (doit 404)"     404 "$(code "$base/api$boot")"

# 7. Santé.
check "GET /api/health"              200 "$(code "$base/api/health")"
check "GET /api/ready"               200 "$(code "$base/api/ready")"

# 8. Une écriture sur une route inconnue doit être refusée, jamais absorbée en 2xx :
#    un client machine prendrait un 200 pour une livraison réussie.
check "POST route inconnue"          405 "$(curl -s --max-time 15 -o /dev/null -w '%{http_code}' -X POST "$base/route-inexistante")"

# 9. Aucun /api/api dans ce que la page charge réellement.
doubled=0
for f in $(curl -s --max-time 15 "$base/" | grep -o '/_app/immutable/[A-Za-z0-9/_.-]*\.js' | sort -u); do
  n=$(curl -s --max-time 10 "$base$f" | grep -c '/api/api' || true)
  doubled=$((doubled + n))
done
check "occurrences /api/api"         0 "$doubled"

if [ "$fails" -eq 0 ]; then
  printf '  => %s OK\n' "$app"
else
  printf '  => %s : %d ECHEC(S)\n' "$app" "$fails"
fi
exit "$fails"
