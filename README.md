# Bashes

Bashes is a lightweight desktop application for managing remote servers, hosts and subsystems from a single local tool.

The project is now a Wails desktop app with a Go backend, a Vite frontend and an embedded `xterm.js` terminal connected to backend-managed SSH sessions.

**Status:** desktop preview. The app is usable for testing host management, JSON portability, SSH sessions and packaging, but it is still under active refactor.

## Goals

- Keep Bashes fast and practical for repeated server administration work.
- Run as a local desktop app on Linux, macOS and Windows.
- Avoid depending on an external terminal application or the system `ssh` command for core SSH sessions.
- Store hosts and subsystems in plain JSON so data can be backed up, inspected and moved between installations.
- Keep generated SSH keys and runtime data in user-writable platform application data directories.

## Features

- Host and subsystem management with stable IDs.
- Subsystem types for VM, LXC and Docker-like resources.
- JSON datastore with validation, atomic writes and backup of the previous store file.
- Import/migration support for legacy Bashes JSON exports.
- Wails desktop shell.
- Go backend bindings for CRUD operations and SSH session control.
- `xterm.js` terminal tabs for SSH sessions.
- Temporary tabs for selected resources before a connection is started.
- Double-click connection start from host/subsystem cards.
- Automatic terminal focus after connection.
- Terminal text selection copied to clipboard and right-click paste behavior.
- SSH authentication through password, SSH agent, default keys, explicit key path or generated Bashes keys.
- Ed25519 SSH key generation and public key installation on remote resources.
- Release builds for Linux amd64, macOS universal and Windows amd64.
- Experimental release jobs for Linux arm64 and Windows arm64.

## Runtime Data

Release builds save JSON data and generated keys outside the application bundle:

- Linux: `$XDG_DATA_HOME/bashes/hosts.json` or `~/.local/share/bashes/hosts.json`
- macOS: `~/Library/Application Support/Bashes/hosts.json`
- Windows: `%APPDATA%\Bashes\hosts.json`

Generated SSH keys are stored in a `keys` directory next to `hosts.json`.

Passwords and key passphrases are not saved in the JSON datastore.

## Download A Release

Tagged builds are published as GitHub Releases.

Download the package for your operating system:

- Linux amd64: `bashes-linux-amd64.tar.gz`
- macOS Apple Silicon and Intel: `bashes-darwin-universal.zip`
- Windows amd64: `bashes-windows-amd64.zip`
- Linux arm64: `bashes-linux-arm64.tar.gz` when the experimental job succeeds
- Windows arm64: `bashes-windows-arm64.zip` when the experimental job succeeds

On Linux, a raw ELF binary normally has a generic icon. The Linux archive includes `icons/bashes.png` and `install-desktop-entry.sh`; run that script from the extracted folder to install a user-local launcher with the Bashes icon.

macOS and Windows builds are currently unsigned unless the repository signing secrets are configured. See [docs/DESKTOP_TESTING.md](docs/DESKTOP_TESTING.md) for Gatekeeper, SmartScreen and desktop testing notes.

## Build From Source

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

Build the desktop app:

```bash
wails build -tags desktop
```

On Linux, Wails requires GTK/WebKitGTK development packages. Run `wails doctor` on the target desktop machine and install only the packages it reports as missing.

## Data CLI

Validate a Bashes JSON datastore:

```bash
go run ./cmd/bashes-data validate /path/to/hosts.json
```

Migrate an old Bashes JSON export to the current versioned format:

```bash
go run ./cmd/bashes-data migrate old-hosts.json hosts.json
```

The migrated file remains plain JSON and can be copied between Bashes installations.

## Development Notes

The current architecture is split into:

- `internal/domain`: JSON schema, validation and stable IDs.
- `internal/store`: JSON repository, legacy import, backup and atomic save.
- `internal/application`: host/subsystem service layer.
- `internal/remotessh`: SSH client and shell session primitives.
- `app.go`: Wails-bound backend API.
- `frontend`: Vite UI with `xterm.js`.
- `main_desktop.go`: Wails desktop entrypoint behind the `desktop` build tag.

More details are in:

- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)
- [docs/DESKTOP_TESTING.md](docs/DESKTOP_TESTING.md)
