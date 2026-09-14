#!/usr/bin/env bash
# Teilt eine Datei über die "CloudWeb"-Warteschlange, optional zusätzlich
# mit einem aus <name> (+ MAC-Adresse dieses Rechners, siehe
# internal/sharepassword, cmd/backend) abgeleiteten Passwort geschützt.
# Gedacht insbesondere für größere/sensiblere Dateien (z. B. ZIP-Archive),
# bei denen der Link/QR-Code allein nicht ausreichend geschützt sein soll.
#
# WICHTIG: Nur die angezeigte Kurzform (crypt) erscheint auf der
# cloudweb-Anzeige - <name> selbst muss der Empfänger unabhängig davon
# kennen (z. B. mündlich vereinbart) und beim Öffnen des Freigabelinks als
# "<name><crypt>" zusammengesetzt als Passwort eintippen.
#
# Aufruf:
#   scripts/share-to-web.sh pfad/zu/test.zip           # ohne Passwortschutz
#   scripts/share-to-web.sh pfad/zu/test.zip "name"    # mit Passwortschutz
#   cat test.zip | scripts/share-to-web.sh - "name"
set -euo pipefail

if [ $# -lt 1 ]; then
  echo "Aufruf: $0 <pfad|-> [name]" >&2
  exit 1
fi

PATH_ARG="$1"
NAME="${2:-}"

OPTS=()
if [ -n "$NAME" ]; then
  OPTS=(-o "nc-password-seed=${NAME}")
fi

if [ "$PATH_ARG" = "-" ]; then
  exec lp -d CloudWeb "${OPTS[@]}" -t "file"
fi

exec lp -d CloudWeb "${OPTS[@]}" -t "$(basename "$PATH_ARG")" "$PATH_ARG"
