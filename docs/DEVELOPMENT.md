# Development Notes

This repository is being migrated from the legacy local PHP app to a Wails desktop app.

## Current Architecture

- Go backend core:
  - `internal/domain`: JSON schema, validation and stable IDs.
  - `internal/store`: JSON repository with legacy import and atomic save.
  - `internal/application`: host/subsystem service layer.
  - `internal/remotessh`: internal SSH client primitives based on Go, not external `ssh`.
  - `internal/desktop`: Wails-facing adapter.
- Frontend:
  - `frontend`: Vite app with `xterm.js` mounted as the terminal surface.
- Wails entrypoint:
  - `main_desktop.go` is guarded by the `desktop` build tag so normal headless tests do not require Linux WebKitGTK development packages.

## Local Tooling on This Server

The current server does not provide Go, Node or Wails globally. Local toolchains are kept inside ignored directories:

- Go: `.tools/go`
- Node LTS: `.tools/node-v24.18.0-linux-x64`
- Wails CLI: `.tools/bin/wails`
- Go/npm caches: `.cache/`

Do not commit `.tools/`, `.cache/`, `frontend/node_modules/`, `frontend/dist/` or Wails build outputs.

## Headless Verification

Run Go tests:

```bash
GOCACHE=/home/ubuntu/git/bashes/.cache/go-build \
GOMODCACHE=/home/ubuntu/git/bashes/.cache/go-mod \
/home/ubuntu/git/bashes/.tools/go/bin/go test ./...
```

Validate or migrate JSON data without a desktop:

```bash
GOCACHE=/home/ubuntu/git/bashes/.cache/go-build \
GOMODCACHE=/home/ubuntu/git/bashes/.cache/go-mod \
/home/ubuntu/git/bashes/.tools/go/bin/go run ./cmd/bashes-data validate public/db/hosts.json
```

Build frontend without starting a server:

```bash
PATH=/home/ubuntu/git/bashes/.tools/node-v24.18.0-linux-x64/bin:$PATH \
npm_config_cache=/home/ubuntu/git/bashes/.cache/npm \
/home/ubuntu/git/bashes/.tools/node-v24.18.0-linux-x64/bin/npm run build --prefix frontend
```

## Wails Status

The local Wails CLI is `v2.12.0`.

`wails doctor` reports that this server is missing the required Linux package `libwebkit2gtk-4.0-dev`. Because this host is shared with other applications, do not install system packages unless that becomes explicitly necessary.

When building on a Linux desktop environment with Wails dependencies installed, use:

```bash
PATH=/home/ubuntu/git/bashes/.tools/node-v24.18.0-linux-x64/bin:/home/ubuntu/git/bashes/.tools/go/bin:/home/ubuntu/git/bashes/.tools/bin:$PATH \
/home/ubuntu/git/bashes/.tools/bin/wails build -tags desktop
```
