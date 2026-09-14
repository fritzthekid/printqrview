#!/usr/bin/env bash
# Ausgabe-Hook (siehe internal/output.Hook / PRINTTOQRVIEW_DISPLAY_HOOK):
# wird von backend/sendfile/webshare nach jedem Erfolg als
# "push-to-cloudweb.sh <qr-png-pfad> <link> [passwort]" aufgerufen und
# schickt das Ergebnis per HTTP an einen lokal laufenden cloudweb-Server
# (siehe cmd/cloudweb) statt an den Raspberry-Pi-Framebuffer
# (raspi/scripts/toraspi.sh). Das dritte Argument (falls vorhanden) ist die
# von cmd/backend aus "nc-password-seed=..." abgeleitete Anzeige-Kurzform
# (crypt, siehe internal/sharepassword, scripts/share-to-web.sh) - nicht
# das volle Freigabe-Passwort.
#
# CLOUDWEB_URL: Basis-URL des Servers, Default http://127.0.0.1:40080
#   (muss auf demselben Host laufen - der Push-Endpoint akzeptiert nur
#   Anfragen von localhost).
# CLOUDWEB_PASSWORD: Fallback, falls kein drittes Argument übergeben wurde
#   (z. B. bei manuellen Aufrufen ohne cmd/backend).
set -euo pipefail

if [ $# -lt 2 ]; then
  cat >&2 <<USAGE
Aufruf: $0 <qr-png-datei> <link> [passwort]

Das ist der interne Ausgabe-Hook (siehe PRINTTOQRVIEW_DISPLAY_HOOK) - kein
Upload-Tool. Er erwartet eine bereits fertige QR-Code-PNG (z. B. eine Datei
aus /var/lib/printtoqrview/tmp/*.png) und reicht sie nur an cloudweb weiter,
lädt selbst nichts nach Nextcloud hoch.

Zum Teilen + Anzeigen einer beliebigen Datei stattdessen:
  PRINTER=CloudWeb scripts/share.sh <datei>
  scripts/share-to-web.sh <datei> <name>   # zusätzlich passwortgeschützt
USAGE
  exit 1
fi

PNG="$1"
LINK="$2"
PASSWORD="${3:-${CLOUDWEB_PASSWORD:-}}"

CLOUDWEB_URL="${CLOUDWEB_URL:-http://127.0.0.1:40080}"

curl -sf -X POST "$CLOUDWEB_URL/push" \
  -F "file=@${PNG}" \
  -F "link=${LINK}" \
  ${PASSWORD:+-F "password=${PASSWORD}"}
