#!/bin/bash
# Ausgabe-Hook fuer backend/sendfile (siehe internal/output.Hook, aktiviert
# per PRINTTOQRVIEW_DISPLAY_HOOK=/pfad/zu/toraspi.sh in backend.env).
#
# Aufruf durch den Go-Code: toraspi.sh <qr-png-pfad> <link>
#
# Komponiert den QR-Code mit dem Link als Text daneben und schickt das
# Ergebnis per ssh an das Display eines Raspberry Pi
# (raspi/bin/pipetodisp.sh + raspi/services/fb-image-watcher.service dort).
set -euo pipefail

PNG="$1"
LINK="$2"

RASPI_HOST="${RASPI_HOST:-raspi.local}"
RASPI_DISPLAY_TIMEOUT="${RASPI_DISPLAY_TIMEOUT:-10}"
OUT="$(dirname "$PNG")/ausgabe.png"

TEXT="${LINK#https://}"
TEXT="${TEXT#http://}"
TEXT="$(printf '%s' "$TEXT" | fold -w 24 -s)"

convert "$PNG" \( -size 180x320 -gravity center -background black -fill white -pointsize 20 label:"$TEXT" \) +append "$OUT"
ssh "$RASPI_HOST" pipetodisp.sh "$RASPI_DISPLAY_TIMEOUT" < "$OUT"
