#!/usr/bin/env bash
# Teilt eine Datei über eine der bereits eingerichteten CUPS-Warteschlangen -
# gleicher Aufruf wie sendfile, aber ohne dass der aufrufende Nutzer selbst
# an die Nextcloud-Zugangsdaten kommen muss (die liegen ausschließlich beim
# CUPS-Backend, siehe /etc/printtoqrview/backend.env). Setzt voraus, dass
# deploy/install.sh auf diesem Rechner bereits gelaufen ist.
#
# Aufruf:
#   scripts/share.sh pfad/zu/test.zip
#   scripts/share.sh pfad/zu/test.zip "Anderer Titel.zip"
#   cat test.zip | scripts/share.sh - test.zip
#   PRINTER=CloudWeb scripts/share.sh pfad/zu/test.zip   # andere Warteschlange
set -euo pipefail

PRINTER="${PRINTER:-CloudToRaspi}"

if [ $# -lt 1 ]; then
  echo "Aufruf: $0 <pfad|-> [titel]" >&2
  exit 1
fi

PATH_ARG="$1"

if [ "$PATH_ARG" = "-" ]; then
  TITLE="${2:-file}"
  exec lp -d "$PRINTER" -t "$TITLE"
fi

TITLE="${2:-$(basename "$PATH_ARG")}"
exec lp -d "$PRINTER" -t "$TITLE" "$PATH_ARG"
