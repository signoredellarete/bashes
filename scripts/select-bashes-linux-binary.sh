#!/usr/bin/env bash
set -euo pipefail

package_dir="${1:-$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)}"
ldd_command="${BASHES_LDD:-ldd}"

if ! command -v "$ldd_command" >/dev/null 2>&1; then
  printf 'Cannot select a Bashes binary: %s was not found.\n' "$ldd_command" >&2
  exit 1
fi

diagnostics=""
for variant in webkit41 webkit40; do
  candidate="$package_dir/bin/bashes-$variant"
  [ -x "$candidate" ] || continue

  set +e
  output="$("$ldd_command" "$candidate" 2>&1)"
  status=$?
  set -e
  if [ "$status" -eq 0 ] && [[ "$output" != *"not found"* ]]; then
    printf '%s\n' "$candidate"
    exit 0
  fi

  missing="$(printf '%s\n' "$output" | sed -n '/not found/p')"
  if [ -z "$missing" ]; then
    missing="$output"
  fi
  diagnostics+="${variant}: ${missing:-dependency check failed}"$'\n'
done

printf 'No compatible Bashes binary was found. Install WebKitGTK 4.1 or 4.0 and retry.\n' >&2
if [ -n "$diagnostics" ]; then
  printf '%s' "$diagnostics" >&2
fi
exit 1
