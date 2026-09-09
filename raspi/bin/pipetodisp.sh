#!/bin/bash
set -euo pipefail

cat > /home/eduard/fb-drop/pic.png

TIMEOUT=1

if [[ $# > 0 ]]; then 
   TIMEOUT=$1
fi

setsid bash -c "
if [[  $# > 0 ]]; then
   sleep $TIMEOUT
   cp /home/eduard/work/icons/black.png /home/eduard/fb-drop/
fi
" < /dev/null > /dev/null 2>&1 &
disown

