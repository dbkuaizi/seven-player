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

mkdir -p release

(
  cd frontend
  npm install
  npm run build -q
)

GOOS=linux GOARCH=arm64 CGO_ENABLED=1 \
  go build -tags production -trimpath -buildvcs=false -ldflags="-w -s" \
  -o release/seven-player-linux-arm64 .

chmod +x release/seven-player-linux-arm64

file release/seven-player-linux-arm64
upx -9 release/seven-player-linux-arm64
file release/seven-player-linux-arm64
du -h release/seven-player-linux-arm64
