#!/usr/bin/env bash
# Cross-compile cloudweb.exe + fileshare.exe für Windows (amd64), von Linux
# oder macOS aus - kein Windows-Toolchain nötig, reines Go, kein cgo.
# Siehe doc/anforderung_windows.md.
#
# Aufruf: deploy/windows/build.sh [ausgabeverzeichnis]
# Default-Ausgabeverzeichnis: dist/windows
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
out_dir="${1:-"$repo_root/dist/windows"}"

mkdir -p "$out_dir"
cd "$repo_root"

export GOOS=windows GOARCH=amd64 CGO_ENABLED=0

go build -o "$out_dir/cloudweb.exe" ./cmd/cloudweb
go build -o "$out_dir/fileshare.exe" ./cmd/fileshare
cp deploy/windows/install.ps1 "$out_dir/"
cp deploy/windows/userinstall.ps1 "$out_dir/"
cp deploy/windows/backend.env.example "$out_dir/"
cp deploy/windows/HowToInstall.txt "$out_dir/"

# In ein ZIP packen - Windows warnt sonst bei jeder einzeln heruntergeladenen
# .exe per SmartScreen; als ZIP nur einmal beim Entpacken.
rm -f "$out_dir/cloudweb.zip"
zip -j "$out_dir/cloudweb.zip" \
    "$out_dir"/cloudweb.exe \
    "$out_dir"/fileshare.exe \
    "$out_dir"/install.ps1 \
    "$out_dir"/userinstall.ps1 \
    "$out_dir"/backend.env.example \
    "$out_dir"/HowToInstall.txt

echo "Fertig: $out_dir"
ls -la "$out_dir"
