#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/bookworm-backports.list

apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  file \
  git \
  build-essential \
  pkg-config \
  libgtk-3-dev \
  libwebkit2gtk-4.1-dev
apt-get install -y -t bookworm-backports --no-install-recommends upx-ucl
UPX_BIN="$(command -v upx || command -v upx-ucl)"
ln -sf "$UPX_BIN" /usr/local/bin/upx

curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt-get install -y --no-install-recommends nodejs

go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.80
export PATH="$(go env GOPATH)/bin:${PATH}"

mkdir -p release

wails3 task build
cp bin/seven-player.exe release/seven-player-windows-amd64.exe

GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" \
  -o release/seven-player-linux-amd64 .

chmod +x release/seven-player-linux-amd64

for artifact in \
  release/seven-player-windows-amd64.exe \
  release/seven-player-linux-amd64
do
  file "$artifact"
  upx -9 "$artifact"
done

file release/seven-player-windows-amd64.exe release/seven-player-linux-amd64
du -h release/seven-player-windows-amd64.exe release/seven-player-linux-amd64
