#!/usr/bin/env bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive
VERSION="${CNB_BRANCH:-v1.0.0}"

echo "deb http://deb.debian.org/debian bookworm-backports main" > /etc/apt/sources.list.d/bookworm-backports.list
dpkg --add-architecture arm64

apt-get update
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  file \
  git \
  build-essential \
  pkg-config \
  libgtk-3-dev \
  libwebkit2gtk-4.1-dev \
  gcc-aarch64-linux-gnu \
  g++-aarch64-linux-gnu \
  libc6-dev-arm64-cross \
  libgtk-3-dev:arm64 \
  libwebkit2gtk-4.1-dev:arm64
apt-get install -y -t bookworm-backports --no-install-recommends upx-ucl
ln -sf /usr/bin/upx-ucl /usr/local/bin/upx

curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
apt-get install -y --no-install-recommends nodejs

go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.80
export PATH="$(go env GOPATH)/bin:${PATH}"

mkdir -p release

wails3 task build
cp bin/seven-player.exe release/seven-player-windows-amd64.exe

GO_BUILD_FLAGS=(-tags production -trimpath -buildvcs=false '-ldflags=-w -s')

GOOS=linux GOARCH=amd64 CGO_ENABLED=1 \
  go build "${GO_BUILD_FLAGS[@]}" -o release/seven-player-linux-amd64 .

PKG_CONFIG_ALLOW_CROSS=1 \
PKG_CONFIG_LIBDIR=/usr/lib/aarch64-linux-gnu/pkgconfig:/usr/share/pkgconfig \
CC=aarch64-linux-gnu-gcc \
CXX=aarch64-linux-gnu-g++ \
GOOS=linux GOARCH=arm64 CGO_ENABLED=1 \
  go build "${GO_BUILD_FLAGS[@]}" -o release/seven-player-linux-arm64 .

chmod +x release/seven-player-linux-amd64 release/seven-player-linux-arm64

for artifact in \
  release/seven-player-windows-amd64.exe \
  release/seven-player-linux-amd64 \
  release/seven-player-linux-arm64
do
  file "$artifact"
  upx -9 "$artifact"
done

file release/seven-player-windows-amd64.exe release/seven-player-linux-amd64 release/seven-player-linux-arm64
du -h release/seven-player-windows-amd64.exe release/seven-player-linux-amd64 release/seven-player-linux-arm64

{
  printf 'Seven Player %s\n\n' "$VERSION"
  printf '面向 115 用户的 Windows 外部播放器体验增强工具。\n\n'
  printf '## 下载\n\n'
  printf -- '- Windows x64: `seven-player-windows-amd64.exe`\n'
  printf -- '- Linux x64: `seven-player-linux-amd64`\n'
  printf -- '- Linux ARM64: `seven-player-linux-arm64`\n\n'
  printf '以上附件均由 CNB 在线构建生成，并已使用 `upx -9` 压缩。\n\n'
  printf 'Windows 用户下载 exe 后即可运行。Linux 版本为理论支持产物，需要系统具备 GTK/WebKitGTK 运行环境；如遇兼容问题，可以自行编译或提交 Issue。\n'
} > release/RELEASE_NOTES.md
