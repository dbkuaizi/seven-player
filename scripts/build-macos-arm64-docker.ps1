param(
    [string]$Image = "panplayer-wails3-cross",
    [string]$Output = "dist/PanPlayer115-darwin-arm64.zip"
)

$ErrorActionPreference = "Stop"

$Root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root

$GoPath = (& go env GOPATH).Trim()
$env:PATH = (Join-Path $GoPath "bin") + [IO.Path]::PathSeparator + $env:PATH

if (-not (Get-Command wails3 -ErrorAction SilentlyContinue)) {
    go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.80
}

wails3 generate bindings
npm install --prefix frontend
npm run build --prefix frontend

docker build -t $Image -f build/docker/Dockerfile.cross build/docker
docker run --rm --mount "type=bind,source=$Root,target=/app" -e APP_NAME=panplayer115 $Image /usr/local/bin/build.sh darwin arm64

$packageScript = @'
set -e
APP_DIR="bin/PanPlayer115.app"
ZIP_PATH="${OUTPUT_ZIP:-dist/PanPlayer115-darwin-arm64.zip}"
BIN_PATH="bin/panplayer115-darwin-arm64"

rm -rf "$APP_DIR" "$ZIP_PATH" "$ZIP_PATH.sha256"
mkdir -p "$APP_DIR/Contents/MacOS" "$APP_DIR/Contents/Resources" "$(dirname "$ZIP_PATH")"

cp "$BIN_PATH" "$APP_DIR/Contents/MacOS/panplayer115"
chmod +x "$APP_DIR/Contents/MacOS/panplayer115"
cp build/darwin/Info.plist "$APP_DIR/Contents/Info.plist"
cp build/darwin/icons.icns "$APP_DIR/Contents/Resources/icons.icns"
if [ -f build/darwin/Assets.car ]; then
  cp build/darwin/Assets.car "$APP_DIR/Contents/Resources/Assets.car"
fi

cd bin
zip -qry "../$ZIP_PATH" PanPlayer115.app
cd ..
sha256sum "$ZIP_PATH" > "$ZIP_PATH.sha256"
ls -lh "$ZIP_PATH" "$ZIP_PATH.sha256"
'@

docker run --rm --mount "type=bind,source=$Root,target=/app" -e OUTPUT_ZIP=$Output --entrypoint /bin/sh $Image -lc $packageScript
