#!/usr/bin/env sh
set -eu

VERSION="${CNB_BRANCH:-v1.0.0}"
mkdir -p release

{
  printf 'Seven Player %s\n\n' "$VERSION"
  printf '面向 115 用户的 Windows 外部播放器体验增强工具。\n\n'
  printf '## 下载\n\n'
  printf '%s\n' '- Windows x64: `seven-player-windows-amd64.exe`'
  printf '%s\n' '- Linux x64: `seven-player-linux-amd64`'
  printf '%s\n\n' '- Linux ARM64: `seven-player-linux-arm64`'
  printf '以上附件均由 CNB 在线构建生成，并使用 `upx -9` 压缩。\n\n'
  printf 'Windows 用户下载 exe 后即可运行。Linux 版本为理论支持产物，需要系统具备 GTK/WebKitGTK 运行环境；如遇兼容问题，可以自行编译或提交 Issue。\n'
} > release/RELEASE_NOTES.md
