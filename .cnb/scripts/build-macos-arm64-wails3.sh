#!/bin/sh
set -e

export APP_NAME=panplayer115
export OUTPUT_ZIP="${OUTPUT_ZIP:-dist/PanPlayer115-darwin-arm64.zip}"

wails3 generate bindings
npm install --prefix frontend
npm run build --prefix frontend

/usr/local/bin/build.sh darwin arm64

app_dir="bin/PanPlayer115.app"
binary_path="bin/panplayer115-darwin-arm64"

rm -rf "$app_dir" "$OUTPUT_ZIP" "$OUTPUT_ZIP.sha256"
mkdir -p "$app_dir/Contents/MacOS" "$app_dir/Contents/Resources" "$(dirname "$OUTPUT_ZIP")"

cp "$binary_path" "$app_dir/Contents/MacOS/panplayer115"
chmod +x "$app_dir/Contents/MacOS/panplayer115"
cp build/darwin/Info.plist "$app_dir/Contents/Info.plist"
cp build/darwin/icons.icns "$app_dir/Contents/Resources/icons.icns"
if [ -f build/darwin/Assets.car ]; then
  cp build/darwin/Assets.car "$app_dir/Contents/Resources/Assets.car"
fi

cd bin
zip -qry "../$OUTPUT_ZIP" PanPlayer115.app
cd ..
sha256sum "$OUTPUT_ZIP" > "$OUTPUT_ZIP.sha256"
ls -lh "$OUTPUT_ZIP" "$OUTPUT_ZIP.sha256"
