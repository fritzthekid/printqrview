#!/bin/bash
# Beobachtet ein Verzeichnis auf neue Bilddateien (z.B. per scp abgelegt),
# zeigt jede neue Datei fuer TIMEOUT Sekunden auf dem Framebuffer an und
# loescht die Anzeige danach wieder (schwarzes Bild).

set -euo pipefail

mkdir -p /home/eduard/fb-drop

WATCH_DIR="${WATCH_DIR:-$HOME/fb-drop}"
FB_DEVICE="${FB_DEVICE:-/dev/fb1}"
TIMEOUT="${TIMEOUT:-60}"
BLACK_PNG="${BLACK_PNG:-$HOME/.fb-image-watcher/black.png}"

mkdir -p "$WATCH_DIR" "$(dirname "$BLACK_PNG")"

if [ ! -f "$BLACK_PNG" ]; then
    RES=$(fbset -fb "$FB_DEVICE" -s | awk '/geometry/ {print $2"x"$3}')
    RES="${RES:-480x320}"
    convert -size "$RES" xc:black "$BLACK_PNG"
fi

CURRENT_PID=""

clear_screen() {
    [ -n "$CURRENT_PID" ] && kill "$CURRENT_PID" 2>/dev/null || true
    fbi -d "$FB_DEVICE" -T 1 -noverbose -once "$BLACK_PNG"
}

show_image() {
    local file="$1"
    currentfile=/tmp/fb-image-watcher/current-file
    [ -n "$CURRENT_PID" ] && kill "$CURRENT_PID" 2>/dev/null || true
    mkdir -p $(dirname /tmp/fb-image-watcher/current-file)
    cp $file $currentfile

    echo ls -l $file $currentfile
    setsid fbi -d "$FB_DEVICE" -T 1 -noverbose -a "$currentfile" &
    CURRENT_PID=$!
    (
        sleep "$TIMEOUT"
        if [ "$(cat /tmp/fb-image-watcher.pid 2>/dev/null)" = "$CURRENT_PID" ]; then
            clear_screen
        fi
    ) &
    echo "$CURRENT_PID" > /tmp/fb-image-watcher.pid
}

echo "Beobachte $WATCH_DIR (Anzeige auf $FB_DEVICE, Timeout ${TIMEOUT}s)..."

inotifywait -m -e close_write --format '%f' "$WATCH_DIR" | while read -r NAME; do
    FILE="$WATCH_DIR/$NAME"
    [ -f "$FILE" ] || continue
    show_image "$FILE"
    # sleep 1
    rm -f "$FILE"
done
