#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp_dir="$(mktemp -d)"
package_name="bashes-linux-amd64"
package_dir="$root_dir/dist/$package_name"
archive="$root_dir/dist/$package_name.tar.gz"

cleanup() {
  rm -rf "$tmp_dir" "$package_dir" "$archive"
}
trap cleanup EXIT

fail() {
  printf 'Test failed: %s\n' "$*" >&2
  exit 1
}

mkdir -p "$tmp_dir/bin"
cat > "$tmp_dir/bashes-webkit40" <<'BINARY'
#!/usr/bin/env bash
printf 'webkit40 %s\n' "$*"
BINARY
cat > "$tmp_dir/bashes-webkit41" <<'BINARY'
#!/usr/bin/env bash
printf 'webkit41 %s\n' "$*"
BINARY
cat > "$tmp_dir/bin/ldd" <<'LDD'
#!/usr/bin/env bash
case "$1" in
  *webkit41)
    if [ "${TEST_WEBKIT41_MISSING:-false}" = true ]; then
      printf 'libwebkit2gtk-4.1.so.0 => not found\n'
    else
      printf 'libwebkit2gtk-4.1.so.0 => /lib/libwebkit2gtk-4.1.so.0\n'
    fi
    ;;
  *webkit40)
    if [ "${TEST_WEBKIT40_MISSING:-false}" = true ]; then
      printf 'libwebkit2gtk-4.0.so.37 => not found\n'
    else
      printf 'libwebkit2gtk-4.0.so.37 => /lib/libwebkit2gtk-4.0.so.37\n'
    fi
    ;;
esac
LDD
cat > "$tmp_dir/bin/uname" <<'UNAME'
#!/usr/bin/env bash
case "${1:-}" in
  -s) printf 'Linux\n' ;;
  -m) printf 'x86_64\n' ;;
  *) printf 'Linux\n' ;;
esac
UNAME
cat > "$tmp_dir/bin/curl" <<'CURL'
#!/usr/bin/env bash
url=""
output=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      shift
      output="$1"
      ;;
    https://*) url="$1" ;;
  esac
  shift
done
[ -n "$output" ] || exit 1
case "$url" in
  */SHA256SUMS)
    checksum="$(sha256sum "$TEST_RELEASE_ARCHIVE" | awk '{print $1}')"
    printf '%s  bashes-linux-amd64.tar.gz\n' "$checksum" > "$output"
    ;;
  *.tar.gz) cp "$TEST_RELEASE_ARCHIVE" "$output" ;;
  *) exit 1 ;;
esac
CURL
chmod 755 \
  "$tmp_dir/bashes-webkit40" \
  "$tmp_dir/bashes-webkit41" \
  "$tmp_dir/bin/ldd" \
  "$tmp_dir/bin/uname" \
  "$tmp_dir/bin/curl"

cd "$root_dir"
scripts/package-linux-release.sh amd64 "$tmp_dir/bashes-webkit40" "$tmp_dir/bashes-webkit41"
archive_files="$(tar -tzf "$archive")"
grep -qx "$package_name/bin/bashes-webkit40" <<< "$archive_files" || fail "WebKitGTK 4.0 binary missing"
grep -qx "$package_name/bin/bashes-webkit41" <<< "$archive_files" || fail "WebKitGTK 4.1 binary missing"
tar -xzf "$archive" -C "$tmp_dir"

selected="$(PATH="$tmp_dir/bin:$PATH" "$tmp_dir/$package_name/select-bashes-linux-binary.sh" "$tmp_dir/$package_name")"
[ "${selected##*/}" = bashes-webkit41 ] || fail "WebKitGTK 4.1 was not preferred"

selected="$(PATH="$tmp_dir/bin:$PATH" TEST_WEBKIT41_MISSING=true "$tmp_dir/$package_name/select-bashes-linux-binary.sh" "$tmp_dir/$package_name")"
[ "${selected##*/}" = bashes-webkit40 ] || fail "WebKitGTK 4.0 fallback was not selected"

output="$(PATH="$tmp_dir/bin:$PATH" TEST_WEBKIT41_MISSING=true "$tmp_dir/$package_name/bashes" argument)"
[ "$output" = "webkit40 argument" ] || fail "package launcher did not execute the selected binary"

if PATH="$tmp_dir/bin:$PATH" TEST_WEBKIT41_MISSING=true TEST_WEBKIT40_MISSING=true \
  "$tmp_dir/$package_name/select-bashes-linux-binary.sh" "$tmp_dir/$package_name" >/dev/null 2>&1; then
  fail "selector accepted binaries with unresolved dependencies"
fi

mkdir -p "$tmp_dir/home"
PATH="$tmp_dir/bin:$PATH" \
HOME="$tmp_dir/home" \
XDG_DATA_HOME="$tmp_dir/share" \
BASHES_VERSION="vtest" \
BASHES_INSTALL_DIR="$tmp_dir/install" \
BASHES_BIN_DIR="$tmp_dir/user-bin" \
TEST_RELEASE_ARCHIVE="$archive" \
TEST_WEBKIT41_MISSING=true \
scripts/install-bashes-linux.sh >/dev/null

output="$("$tmp_dir/install/bashes" installed)"
[ "$output" = "webkit40 installed" ] || fail "installer did not install the compatible binary"
[ "$(cat "$tmp_dir/install/VERSION")" = vtest ] || fail "installer did not write the version marker"
[ -f "$tmp_dir/share/applications/bashes.desktop" ] || fail "installer did not create the desktop entry"

printf 'Linux package tests passed.\n'
