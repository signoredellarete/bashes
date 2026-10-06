#!/usr/bin/env bash
set -euo pipefail

arch="${1:?usage: package-linux-release.sh <arch> [webkit40-binary] [webkit41-binary]}"
webkit40_binary="${2:-build/bin/bashes}"
webkit41_binary="${3:-}"
package_name="bashes-linux-${arch}"
package_dir="dist/${package_name}"
archive="dist/${package_name}.tar.gz"

if [ ! -f "$webkit40_binary" ]; then
  echo "Bashes binary not found: $webkit40_binary" >&2
  exit 1
fi
if [ -n "$webkit41_binary" ] && [ ! -f "$webkit41_binary" ]; then
  echo "Bashes binary not found: $webkit41_binary" >&2
  exit 1
fi

rm -rf "$package_dir" "$archive"
mkdir -p "$package_dir/icons"

if [ -n "$webkit41_binary" ]; then
  mkdir -p "$package_dir/bin"
  cp "$webkit40_binary" "$package_dir/bin/bashes-webkit40"
  cp "$webkit41_binary" "$package_dir/bin/bashes-webkit41"
  chmod 755 "$package_dir/bin/bashes-webkit40" "$package_dir/bin/bashes-webkit41"
  cp scripts/select-bashes-linux-binary.sh "$package_dir/select-bashes-linux-binary.sh"
  chmod 755 "$package_dir/select-bashes-linux-binary.sh"
  cat > "$package_dir/bashes" <<'LAUNCHER'
#!/usr/bin/env bash
set -euo pipefail

app_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
binary="$("$app_dir/select-bashes-linux-binary.sh" "$app_dir")"
exec "$binary" "$@"
LAUNCHER
else
  cp "$webkit40_binary" "$package_dir/bashes"
fi
chmod 755 "$package_dir/bashes"
cp icons/bashes.png "$package_dir/icons/bashes.png"
if [ -d icons/hicolor ]; then
  cp -R icons/hicolor "$package_dir/icons/"
fi

cat > "$package_dir/bashes.desktop.template" <<'DESKTOP'
[Desktop Entry]
Type=Application
Name=Bashes
Comment=Remote server session manager
Exec=@APPDIR@/bashes
Icon=@APPDIR@/icons/bashes.png
Terminal=false
Categories=Network;RemoteAccess;
StartupNotify=true
StartupWMClass=bashes
DESKTOP

cat > "$package_dir/install-desktop-entry.sh" <<'INSTALLER'
#!/usr/bin/env bash
set -euo pipefail

app_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
desktop_dir="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
desktop_file="$desktop_dir/bashes.desktop"

mkdir -p "$desktop_dir"

cat > "$desktop_file" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Bashes
Comment=Remote server session manager
Exec=${app_dir}/bashes
Icon=${app_dir}/icons/bashes.png
Terminal=false
Categories=Network;RemoteAccess;
StartupNotify=true
StartupWMClass=bashes
DESKTOP

chmod 644 "$desktop_file"

if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$desktop_dir" >/dev/null 2>&1 || true
fi

echo "Installed desktop entry: $desktop_file"
INSTALLER

chmod 755 "$package_dir/install-desktop-entry.sh"

tar -C dist -czf "$archive" "$package_name"
