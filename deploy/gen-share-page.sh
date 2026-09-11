#!/usr/bin/env bash
# Erzeugt aus web/share.html eine personalisierte Kopie mit fest
# eingebackenen Nextcloud-Zugangsdaten, damit auf dem Handy keine Eingabe
# mehr nötig ist: eine Datei aufs Handy kopieren, öffnen, fertig.
#
# Empfehlung: dafür ein eigenes App-Passwort vergeben (Nextcloud-Weboberfläche
# -> Einstellungen -> Sicherheit -> "Neues App-Passwort erstellen", z. B.
# benannt "handy-share"), getrennt vom App-Passwort des CUPS-Backends -
# unabhängig widerrufbar, falls das Handy verloren geht. Noch besser: ein
# eigener, auf einen Ordner beschränkter Nextcloud-Benutzer statt des
# Haupt-Accounts (wie es der bestehende "printer"-Nutzer für den
# CUPS-Backend-Teil schon vormacht).
#
# Aufruf:
#   ./deploy/gen-share-page.sh /etc/printtoqrview/backend.env > mein-handy.html
# oder mit eigenen Werten in der Umgebung:
#   NC_BASE_URL=... NC_USERNAME=... NC_PASSWORD=... ./deploy/gen-share-page.sh > mein-handy.html
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE="$REPO_ROOT/web/share.html"

if [ "${1:-}" ]; then
  if [ ! -r "$1" ]; then
    echo "Env-Datei nicht lesbar: $1" >&2
    exit 1
  fi
  set -a
  # shellcheck disable=SC1090
  . "$1"
  set +a
fi

: "${NC_BASE_URL:?NC_BASE_URL fehlt (Env-Datei als Argument übergeben oder Variablen exportieren)}"
: "${NC_USERNAME:?NC_USERNAME fehlt}"
: "${NC_PASSWORD:?NC_PASSWORD fehlt}"
TARGET_DIR="${NC_TARGET_DIR:-/PrinterUploads}"
EXPIRE_DAYS="${NC_LINK_EXPIRE_DAYS:-1}"

if ! [[ "$EXPIRE_DAYS" =~ ^[0-9]+$ ]]; then
  echo "NC_LINK_EXPIRE_DAYS muss eine Zahl sein, ist aber: $EXPIRE_DAYS" >&2
  exit 1
fi

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

CONFIG_JSON=$(printf '{"baseUrl":"%s","username":"%s","password":"%s","targetDir":"%s","linkExpireDays":%s}' \
  "$(json_escape "${NC_BASE_URL%/}")" \
  "$(json_escape "$NC_USERNAME")" \
  "$(json_escape "$NC_PASSWORD")" \
  "$(json_escape "$TARGET_DIR")" \
  "$EXPIRE_DAYS")

# CONFIG_JSON über ENVIRON statt "awk -v" einschleusen: "-v"-Zuweisungen
# interpretieren Backslash-Escapes (POSIX-awk-Eigenheit) und würden unser
# gerade erzeugtes \" bzw. \\ wieder entwerten - ENVIRON-Werte bleiben
# dagegen unverändert.
PRINTTOQRVIEW_CFG_JSON="$CONFIG_JSON" awk '
  /<!--PRINTTOQRVIEW_CONFIG-->/ { print "<script>window.PRINTTOQRVIEW_CONFIG = " ENVIRON["PRINTTOQRVIEW_CFG_JSON"] ";</script>"; next }
  { print }
' "$TEMPLATE"
