#!/usr/bin/env bash
# Baut alle Linux-Binaries an den Stellen, die deploy/install.sh erwartet -
# ein Befehl statt "go build -o X ./cmd/X" für jedes einzeln (Quelle
# wiederkehrender "install.sh installiert eine veraltete Binary"-Fehler).
#
# Aufruf: deploy/build.sh
# Danach: sudo ./deploy/install.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

go build -o backend ./cmd/backend
go build -o sendfile ./cmd/sendfile
go build -o webshare ./cmd/webshare
go build -o cloudweb ./cmd/cloudweb

echo "Fertig. Jetzt: sudo ./deploy/install.sh"
