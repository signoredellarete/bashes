#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_icon="$repo_root/icons/bashes.png"
frontend_icon="$repo_root/frontend/src/assets/bashes.png"
wails_icon="$repo_root/build/appicon.png"
windows_icon="$repo_root/build/windows/icon.ico"
icon_sizes="16 22 24 32 48 64 128 256 512"
ico_sizes="16 24 32 48 64 128 256"

if ! command -v magick >/dev/null 2>&1; then
  echo "ImageMagick is required: magick command not found" >&2
  exit 1
fi

if [ ! -f "$source_icon" ]; then
  echo "Source icon not found: $source_icon" >&2
  exit 1
fi

mkdir -p "$(dirname "$frontend_icon")" "$(dirname "$wails_icon")" "$(dirname "$windows_icon")"

magick "$source_icon" -resize 1024x1024 "$frontend_icon"
magick "$source_icon" -resize 1024x1024 "$wails_icon"

for size in $icon_sizes; do
  target_dir="$repo_root/icons/hicolor/${size}x${size}/apps"
  mkdir -p "$target_dir"
  magick "$source_icon" -resize "${size}x${size}" "$target_dir/bashes.png"
done

tmp_dir="$(mktemp -d)"
cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT

ico_inputs=()
for size in $ico_sizes; do
  ico_file="$tmp_dir/bashes-${size}.png"
  magick "$source_icon" -resize "${size}x${size}" "$ico_file"
  ico_inputs+=("$ico_file")
done

magick "${ico_inputs[@]}" "$windows_icon"
