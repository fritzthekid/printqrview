#!/bin/sh
# Wird von cupsd als /usr/lib/cups/backend/nextcloud aufgerufen. CUPS startet
# Backends mit einer minimalen Umgebung, deshalb müssen die NC_*-Variablen
# aus einer root-only lesbaren Datei nachgeladen werden, bevor das
# eigentliche (kompilierte) Backend läuft.
set -a
[ -r /etc/printtoqrview/backend.env ] && . /etc/printtoqrview/backend.env
set +a

# CUPS setzt $PRINTER auf den Namen der Warteschlange, über die gedruckt
# wurde - so lassen sich mehrere Warteschlangen mit demselben Backend-Code
# fest auf unterschiedliche Anzeige-Ziele routen, statt einen einzigen
# globalen Hook in backend.env zu teilen. Ohne Treffer greift der
# PRINTTOQRVIEW_DISPLAY_HOOK-Default aus backend.env (falls gesetzt).
case "$PRINTER" in
  CloudToRaspi)
    export PRINTTOQRVIEW_DISPLAY_HOOK=/usr/local/lib/printtoqrview/toraspi.sh
    ;;
  CloudWeb)
    export PRINTTOQRVIEW_DISPLAY_HOOK=/usr/local/lib/printtoqrview/push-to-cloudweb.sh
    ;;
esac

exec /usr/local/lib/printtoqrview/backend "$@"
