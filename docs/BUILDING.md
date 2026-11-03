# Building From Source

This document contains the complete local build and verification commands for Bashes.

Required tools:

- Go matching `go.mod`
- Node.js and npm
- Wails CLI v2
- Native Wails desktop dependencies for the target OS

Install frontend dependencies:

```bash
npm install --prefix frontend
```

Run tests:

```bash
go test ./...
```

Build the frontend:

```bash
npm run build --prefix frontend
```

Regenerate app icons after replacing `icons/bashes.png`:

```bash
scripts/generate-icons.sh
```

`icons/bashes.png` is the source image for the in-app logo, Wails app icon, Linux hicolor icons and Windows `.ico` file.

Build the desktop app:

```bash
wails build -tags desktop
```

On Linux, run `wails doctor` and install only the missing GTK/WebKitGTK packages required by Wails for your distribution.

[← Back to README](../README.md)
