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

# Saubere Ausgangslage: verwaiste fbi-Prozesse aus einem vorherigen Lauf
# (z.B. nach Absturz/Neustart) beenden, bevor wir neue starten.
pkill -x fbi 2>/dev/null || true

if [ ! -f "$BLACK_PNG" ]; then
    RES=$(fbset -fb "$FB_DEVICE" -s | awk '/geometry/ {print $2"x"$3}')
    RES="${RES:-480x320}"
    convert -size "$RES" xc:black "$BLACK_PNG"
fi

CURRENT_FILE=/tmp/fb-image-watcher/current-file
GEN_FILE=/tmp/fb-image-watcher.pid

# "setsid cmd &" liefert in $! haeufig nur die PID des kurzlebigen
# setsid-Wrappers (der intern forkt, weil der Hintergrundjob schon
# Prozessgruppenleiter ist) statt der PID des eigentlichen fbi-Prozesses -
# darauf gestuetztes "kill $!" trifft also nie den echten fbi. Deshalb hier
# stattdessen ein pauschales "pkill -x fbi" (auf diesem dedizierten
# Display-Pi laeuft fbi ausschliesslich fuer diesen Zweck).
clear_screen() {
    pkill -x fbi 2>/dev/null || true
    fbi -d "$FB_DEVICE" -T 1 -noverbose -once "$BLACK_PNG"
}

show_image() {
    local file="$1"
    local token
    token=$(date +%s%N)

    pkill -x fbi 2>/dev/null || true
    mkdir -p "$(dirname "$CURRENT_FILE")"
    cp "$file" "$CURRENT_FILE"

    setsid fbi -d "$FB_DEVICE" -T 1 -noverbose -a "$CURRENT_FILE" &
    echo "$token" > "$GEN_FILE"
    (
        sleep "$TIMEOUT"
        # $GEN_FILE wird bei jedem neuen Bild überschrieben; nur löschen,
        # wenn in der Zwischenzeit kein neueres Bild angekommen ist.
        if [ "$(cat "$GEN_FILE" 2>/dev/null)" = "$token" ]; then
            clear_screen
        fi
    ) &
}

echo "Beobachte $WATCH_DIR (Anzeige auf $FB_DEVICE, Timeout ${TIMEOUT}s)..."

inotifywait -m -e close_write --format '%f' "$WATCH_DIR" | while read -r NAME; do
    FILE="$WATCH_DIR/$NAME"
    [ -f "$FILE" ] || continue
    show_image "$FILE"
    # sleep 1
    rm -f "$FILE"
done
