#!/bin/bash
set -euo pipefail

# Konfigurierbar wie fb-image-watcher.sh (dieselben Env-Vars, damit beide
# Skripte auf dasselbe Watch-Verzeichnis zeigen).
WATCH_DIR="${WATCH_DIR:-$HOME/fb-drop}"
BLACK_PNG="${BLACK_PNG:-$HOME/.fb-image-watcher/black.png}"

mkdir -p "$WATCH_DIR"
cat > "$WATCH_DIR/pic.png"

TIMEOUT=1

if [[ $# > 0 ]]; then
   TIMEOUT=$1
fi

setsid bash -c "
if [[  $# > 0 ]]; then
   sleep $TIMEOUT
   cp '$BLACK_PNG' '$WATCH_DIR/'
fi
" < /dev/null > /dev/null 2>&1 &
disown
