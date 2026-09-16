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
cp deploy/windows/backend.env.example "$out_dir/"

echo "Fertig: $out_dir"
ls -la "$out_dir"
