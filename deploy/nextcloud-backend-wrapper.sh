#!/bin/sh
# Wird von cupsd als /usr/lib/cups/backend/nextcloud aufgerufen. CUPS startet
# Backends mit einer minimalen Umgebung, deshalb müssen die NC_*-Variablen
# aus einer root-only lesbaren Datei nachgeladen werden, bevor das
# eigentliche (kompilierte) Backend läuft.
set -a
[ -r /etc/printtoqrview/backend.env ] && . /etc/printtoqrview/backend.env
set +a
exec /usr/local/lib/printtoqrview/backend "$@"
