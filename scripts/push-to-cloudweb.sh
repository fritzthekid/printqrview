#!/usr/bin/env bash
# Ausgabe-Hook (siehe internal/output.Hook / PRINTTOQRVIEW_DISPLAY_HOOK):
# wird von backend/sendfile/webshare nach jedem Erfolg als
# "push-to-cloudweb.sh <qr-png-pfad> <link>" aufgerufen und schickt das
# Ergebnis per HTTP an einen lokal laufenden cloudweb-Server (siehe
# cmd/cloudweb) statt an den Raspberry-Pi-Framebuffer
# (raspi/scripts/toraspi.sh).
#
# CLOUDWEB_URL: Basis-URL des Servers, Default http://127.0.0.1:40080
#   (muss auf demselben Host laufen - der Push-Endpoint akzeptiert nur
#   Anfragen von localhost).
# CLOUDWEB_PASSWORD: optional, wird mitgeschickt und auf der Anzeigeseite
#   dargestellt (z. B. falls der Freigabelink zusätzlich passwortgeschützt
#   ist - aktuell erzeugt dieses Projekt keine solchen Passwörter selbst,
#   das Feld ist nur vorbereitet).
set -euo pipefail

PNG="$1"
LINK="$2"

CLOUDWEB_URL="${CLOUDWEB_URL:-http://127.0.0.1:40080}"

curl -sf -X POST "$CLOUDWEB_URL/push" \
  -F "file=@${PNG}" \
  -F "link=${LINK}" \
  ${CLOUDWEB_PASSWORD:+-F "password=${CLOUDWEB_PASSWORD}"}
