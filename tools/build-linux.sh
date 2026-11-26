#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
tag="$(tr -d '\r\n' < RELEASE_VERSION)"
arch="${1:-amd64}"
case "$arch" in amd64) label=x64;; arm64) label=arm64;; *) echo "Unsupported architecture: $arch" >&2; exit 2;; esac
name="AussieTV-Experimental-$tag-linux-$label"
stage="dist/$name"
mkdir -p "$stage" dist/linux
go run ./tools/iconbuilder
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags='-s -w' -o "$stage/aussietv" .
cp LICENSE "$stage/"
cp docs/LINUX.md "$stage/README.md"
cp config/teletext.json "$stage/teletext.json"
cp static/app-icon.svg "$stage/app-icon.svg"
tar -C dist -czf "dist/linux/$name.tar.gz" "$name"
(cd dist/linux && sha256sum "$name.tar.gz" > "$name.tar.gz.sha256")
