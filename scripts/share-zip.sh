#!/usr/bin/env bash
# Kurzform von "PRINTER=CloudWeb scripts/share.sh ..." - Warteschlange fest
# auf CloudWeb (cloudweb-Anzeige) statt CloudToRaspi (Pi-Display)
# voreingestellt, für den Fall, dass man das Ergebnis im Browser statt auf
# dem Pi sehen will.
#
# Trotz des Namens nicht auf ZIP-Dateien beschränkt - share.sh ist generisch
# (die Endung wird am tatsächlichen Inhalt festgemacht, siehe
# internal/filenames.DetectExtension), der Name spiegelt nur den
# ursprünglichen Anwendungsfall wider.
#
# Aufruf: wie scripts/share.sh
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec env PRINTER=CloudWeb "$SCRIPT_DIR/share.sh" "$@"
