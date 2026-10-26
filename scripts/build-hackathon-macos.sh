#!/usr/bin/env bash
# Build a separately identified local Hackathon Edition app bundle.
set -euo pipefail

if [[ "$(uname -s)" != Darwin ]]; then
  echo 'Build on macOS.' >&2
  exit 1
fi
if (( $# != 1 )) || [[ "$1" != /* ]] || [[ -e "$1" ]]; then
  echo 'Usage: scripts/build-hackathon-macos.sh /absolute/new/output-directory' >&2
  exit 2
fi

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="$1"
case "$output/" in
  "$repo/"*) echo 'Keep build output outside the source workspace.' >&2; exit 2 ;;
esac
app="$output/YourMove Hackathon Edition.app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$repo/release/macos/Info.plist" "$app/Contents/Info.plist"
cp "$repo/release/macos/YourMove.icns" "$app/Contents/Resources/YourMove.icns"
plutil -lint "$app/Contents/Info.plist"

cd "$repo"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -trimpath -buildvcs=false -tags desktop -o "$app/Contents/MacOS/YourMoveHackathon" .
codesign --force --sign - "$app"
codesign --verify --strict "$app"
echo "Built: $app"
echo "Bundle identifier: $(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "$app/Contents/Info.plist")"
