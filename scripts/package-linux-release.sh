#!/usr/bin/env bash
set -euo pipefail

arch="${1:?usage: package-linux-release.sh <arch> [binary]}"
binary_source="${2:-build/bin/bashes}"
package_name="bashes-linux-${arch}"
package_dir="dist/${package_name}"
archive="dist/${package_name}.tar.gz"

if [ ! -f "$binary_source" ]; then
  echo "Bashes binary not found: $binary_source" >&2
  exit 1
fi

rm -rf "$package_dir" "$archive"
mkdir -p "$package_dir/icons"

cp "$binary_source" "$package_dir/bashes"
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
