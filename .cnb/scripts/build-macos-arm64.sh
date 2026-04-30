#!/usr/bin/env bash
set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "This build must run on a macOS CNB runner. Current system: $(uname -a)"
  echo "Set CNB_MACOS_RUNNER_TAGS to the tag of your macOS runner, then trigger again."
  exit 1
fi

if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Warning: current macOS runner is $(uname -m); building darwin/arm64 anyway."
fi

export WAILS_VERSION="${WAILS_VERSION:-v2.12.0}"
export OUTPUT_ZIP="${OUTPUT_ZIP:-dist/PanPlayer115-darwin-arm64.zip}"
export PATH="$(go env GOPATH)/bin:${PATH}"

echo "System: $(sw_vers -productName) $(sw_vers -productVersion) ($(uname -m))"
go version
node --version
npm --version
xcode-select -p

go mod download
go install "github.com/wailsapp/wails/v2/cmd/wails@${WAILS_VERSION}"
wails version

npm ci --prefix frontend
wails build -platform darwin/arm64 -clean -nosyncgomod -v 2

app_path="$(find build/bin -maxdepth 1 -name '*.app' -type d | head -n 1)"
if [[ -z "${app_path}" ]]; then
  echo "No .app bundle was produced under build/bin."
  find build/bin -maxdepth 3 -print
  exit 1
fi

mkdir -p "$(dirname "${OUTPUT_ZIP}")"
rm -f "${OUTPUT_ZIP}" "${OUTPUT_ZIP}.sha256"

ditto -c -k --sequesterRsrc --keepParent "${app_path}" "${OUTPUT_ZIP}"
shasum -a 256 "${OUTPUT_ZIP}" | tee "${OUTPUT_ZIP}.sha256"
ls -lh "${OUTPUT_ZIP}" "${OUTPUT_ZIP}.sha256"
