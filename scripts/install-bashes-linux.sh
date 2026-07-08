#!/usr/bin/env bash
set -euo pipefail

repo="${BASHES_REPO:-signoredellarete/bashes}"
version="${BASHES_VERSION:-}"
install_dir="${BASHES_INSTALL_DIR:-$HOME/.local/opt/bashes}"
bin_dir="${BASHES_BIN_DIR:-$HOME/.local/bin}"
share_dir="${XDG_DATA_HOME:-$HOME/.local/share}"
applications_dir="$share_dir/applications"
icon_theme_dir="$share_dir/icons/hicolor"
icon_sizes="16 22 24 32 48 64 128 256 512"
binary_path="$install_dir/bashes"
bin_link="$bin_dir/bashes"
desktop_file="$applications_dir/bashes.desktop"
primary_icon_file="$icon_theme_dir/256x256/apps/bashes.png"

die() {
  printf 'Error: %s\n' "$*" >&2
  exit 1
}

have() {
  command -v "$1" >/dev/null 2>&1
}

require() {
  have "$1" || die "required command not found: $1"
}

download() {
  url="$1"
  output="$2"

  if have curl; then
    curl -fL --retry 3 --connect-timeout 20 "$url" -o "$output"
    return
  fi

  if have wget; then
    wget -q --tries=3 --timeout=20 "$url" -O "$output"
    return
  fi

  die "curl or wget is required to download Bashes"
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    aarch64|arm64)
      printf 'arm64'
      ;;
    *)
      die "unsupported Linux architecture: $(uname -m)"
      ;;
  esac
}

latest_version() {
  if [ -n "$version" ]; then
    printf '%s' "$version"
    return
  fi

  if have curl; then
    curl -fsSL "https://api.github.com/repos/$repo/releases/latest" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
      head -n 1
    return
  fi

  if have wget; then
    wget -qO- "https://api.github.com/repos/$repo/releases/latest" |
      sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' |
      head -n 1
    return
  fi

  die "curl or wget is required to find the latest Bashes release"
}

safe_mkdirs() {
  mkdir -p "$install_dir" "$bin_dir" "$applications_dir"
  for size in $icon_sizes; do
    mkdir -p "$icon_theme_dir/${size}x${size}/apps"
  done
}

install_icon_size() {
  size="$1"
  source_file="$2"
  target_file="$icon_theme_dir/${size}x${size}/apps/bashes.png"
  install -m 0644 "$source_file" "$target_file"
}

install_icons() {
  package_icon_root="$1"
  fallback_icon="$2"
  installed_any=false

  if [ -d "$package_icon_root/hicolor" ]; then
    for size in $icon_sizes; do
      source_file="$package_icon_root/hicolor/${size}x${size}/apps/bashes.png"
      if [ -f "$source_file" ]; then
        install_icon_size "$size" "$source_file"
        installed_any=true
      fi
    done
  fi

  if [ "$installed_any" = false ]; then
    if [ -f "$package_icon_root/bashes.png" ]; then
      install_icon_size 256 "$package_icon_root/bashes.png"
    else
      install_icon_size 256 "$fallback_icon"
    fi
  fi
}

install_desktop_entry() {
  cat > "$desktop_file" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Bashes
Comment=Remote server session manager
Exec=$binary_path
Icon=bashes
Terminal=false
Categories=Network;RemoteAccess;System;
StartupNotify=true
StartupWMClass=bashes
DESKTOP

  chmod 644 "$desktop_file"

  if have update-desktop-database; then
    update-desktop-database "$applications_dir" >/dev/null 2>&1 || true
  fi

  if have gtk-update-icon-cache; then
    gtk-update-icon-cache -q "$share_dir/icons/hicolor" >/dev/null 2>&1 || true
  fi
}

main() {
  [ "$(uname -s)" = "Linux" ] || die "this installer is only for Linux"
  require tar
  require mktemp
  require sed
  require head
  require find
  require install
  require mv
  require chmod
  require ln

  arch="$(detect_arch)"
  release_version="$(latest_version)"
  [ -n "$release_version" ] || die "could not determine the latest Bashes release"

  asset_name="bashes-linux-$arch.tar.gz"
  asset_url="https://github.com/$repo/releases/download/$release_version/$asset_name"
  icon_url="https://raw.githubusercontent.com/$repo/main/icons/bashes.png"
  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "$tmp_dir"' EXIT

  printf 'Installing Bashes %s for linux/%s\n' "$release_version" "$arch"
  download "$asset_url" "$tmp_dir/$asset_name"
  mkdir -p "$tmp_dir/extract"
  tar -xzf "$tmp_dir/$asset_name" -C "$tmp_dir/extract"

  package_binary="$(find "$tmp_dir/extract" -type f -name bashes -print -quit)"
  [ -n "$package_binary" ] || die "release archive does not contain the bashes binary"

  safe_mkdirs
  install -m 0755 "$package_binary" "$binary_path.new"
  mv -f "$binary_path.new" "$binary_path"
  printf '%s\n' "$release_version" > "$install_dir/VERSION"

  if [ ! -f "$tmp_dir/extract/bashes-linux-$arch/icons/bashes.png" ] && [ ! -d "$tmp_dir/extract/bashes-linux-$arch/icons/hicolor" ]; then
    download "$icon_url" "$tmp_dir/bashes.png"
  fi
  fallback_icon="$tmp_dir/bashes.png"
  [ -f "$fallback_icon" ] || fallback_icon="$tmp_dir/extract/bashes-linux-$arch/icons/bashes.png"
  install_icons "$tmp_dir/extract/bashes-linux-$arch/icons" "$fallback_icon"

  ln -sfn "$binary_path" "$bin_link"
  install_desktop_entry

  printf 'Installed binary: %s\n' "$binary_path"
  printf 'Installed launcher: %s\n' "$desktop_file"
  printf 'Installed icon: %s\n' "$primary_icon_file"

  case ":$PATH:" in
    *":$bin_dir:"*) ;;
    *) printf 'Note: %s is not in PATH; start Bashes from the desktop launcher or add it to PATH.\n' "$bin_dir" ;;
  esac
}

main "$@"
