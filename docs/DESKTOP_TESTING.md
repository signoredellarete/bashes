# Desktop Testing Guide

This guide is for testing the Wails version of Bashes on a machine with a graphical desktop environment.

## What Can Be Tested Today

Current status:

- Go JSON datastore, validation and legacy migration are implemented.
- Application service layer for hosts and subsystems is implemented.
- Wails desktop skeleton is present behind the `desktop` build tag.
- Frontend builds with Vite and includes `xterm.js`.
- Internal SSH client primitives exist, but the interactive SSH session is not yet wired to the frontend terminal.

The current desktop app is therefore useful to verify packaging, startup, UI rendering and backend bindings. Full terminal SSH interaction is a next milestone.

## Linux Dependencies

On a Linux desktop machine, Wails needs native WebKitGTK development libraries to compile the desktop shell.

Check the environment first:

```bash
wails doctor
```

On Ubuntu/Debian-like systems, Wails commonly reports:

```bash
sudo apt install build-essential libgtk-3-dev libwebkit2gtk-4.0-dev pkg-config npm
```

Important: on newer Ubuntu releases, `libwebkit2gtk-4.0-dev` may not be available. For example, Ubuntu 25.04 exposes `libwebkit2gtk-4.1-dev` and `libwebkitgtk-6.0-dev` in `apt-cache`, while Wails v2.12 still reports `libwebkit2gtk-4.0-dev`. In that case, do not force unrelated packages blindly. Check Wails documentation and distro package availability, then install the WebKitGTK package compatible with the Wails version being used.

## Risks Of Installing WebKitGTK Dev Packages

Installing `libwebkit2gtk-4.0-dev` or its distro equivalent is usually safe on a desktop development machine, but it is not a zero-impact operation:

- It can install many transitive packages: GTK headers, WebKitGTK headers, JavaScriptCore/WebKit libraries and build tooling.
- It increases disk usage significantly.
- It may upgrade shared GTK/WebKit libraries if the system has pending updates.
- On shared servers, package changes can affect other software that depends on the same system libraries.
- On headless servers, it adds GUI development dependencies that are not needed for backend tests.
- On newer distros, the exact package name may differ, so installing the wrong compatibility package can waste time or create dependency conflicts.

For this repository's current server, avoid installing it unless a GUI build on this host becomes explicitly necessary. Backend and frontend checks can be done headlessly.

## Recommended Desktop Test Flow

Clone and enter the repository:

```bash
git clone https://github.com/signoredellarete/bashes.git
cd bashes
```

Install normal project toolchains if they are not already available:

```bash
go version
node --version
npm --version
wails version
```

If Wails is missing:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

Install frontend dependencies:

```bash
npm install --prefix frontend
```

Run backend tests:

```bash
go test ./...
```

Build the frontend:

```bash
npm run build --prefix frontend
```

Check Wails dependencies:

```bash
wails doctor
```

Build the desktop app:

```bash
wails build -tags desktop
```

Run in development mode only after checking that the default Wails/Vite ports are not already in use:

```bash
ss -ltnp
wails dev -tags desktop
```

Stop the development process with `Ctrl+C` when done.

## Downloading A Release Build

Tagged builds are published as GitHub Releases.

1. Open the repository on GitHub.
2. Go to `Releases`.
3. Open the latest `Bashes v...` release.
4. Download the package for your OS:
   - Linux: `bashes-linux-amd64.tar.gz`
   - macOS: `bashes-darwin-universal.zip`
   - Windows: `bashes-windows-amd64.zip`
   - Linux ARM64, experimental when available: `bashes-linux-arm64.tar.gz`
   - Windows ARM64, experimental when available: `bashes-windows-arm64.zip`
5. Extract the archive and run the app from a graphical desktop session.

On Linux, the executable itself is an ELF binary and desktop environments normally show a generic icon for raw binaries. The release archive includes `icons/bashes.png` and `install-desktop-entry.sh`; run that script from the extracted folder to create a user-local launcher with the Bashes icon under `~/.local/share/applications`.

Release builds are created by pushing a tag matching `v*`.

## macOS Testing

The macOS build is produced on GitHub Actions with `wails build -platform darwin/universal`, so the same archive is intended for Apple Silicon and Intel Macs.

The current macOS package is unsigned and not notarized. On a Mac, Gatekeeper may block the first launch. For testing, extract the zip, then either:

```bash
xattr -dr com.apple.quarantine Bashes.app
open Bashes.app
```

or right-click `Bashes.app`, choose `Open`, and confirm the launch.

For regular distribution outside internal testing, the app must be signed with an Apple Developer ID certificate and notarized. The GitHub workflow supports this when these repository secrets are configured:

- `MACOS_CERTIFICATE_P12`: base64-encoded Developer ID Application certificate exported as `.p12`
- `MACOS_CERTIFICATE_PASSWORD`: password for the `.p12`
- `MACOS_CODESIGN_IDENTITY`: codesign identity, for example `Developer ID Application: ...`
- `APPLE_ID`: Apple developer account email
- `APPLE_TEAM_ID`: Apple team id
- `APPLE_APP_SPECIFIC_PASSWORD`: app-specific password for notarization

Without those secrets, the workflow still produces an unsigned test build.

## Windows Testing

The Windows build is produced on GitHub Actions with `wails build -platform windows/amd64`. The workflow also attempts an experimental `windows/arm64` artifact. The executable embeds `build/windows/icon.ico` through the Wails Windows resource step.

The current package is a plain zip containing `bashes.exe`. It is not signed, so Windows SmartScreen may warn on first launch. For testing, extract the zip into a normal user-writable folder and run `bashes.exe`.

For regular distribution, the executable should be Authenticode-signed. The GitHub workflow supports this when these repository secrets are configured:

- `WINDOWS_CERTIFICATE_PFX`: base64-encoded code-signing certificate exported as `.pfx`
- `WINDOWS_CERTIFICATE_PASSWORD`: password for the `.pfx`

Signing reduces security warnings, but SmartScreen reputation may still require a trusted certificate and download reputation over time.

## Runtime Data Paths

Release builds save JSON data and generated keys in a user-writable application data directory instead of writing beside the executable:

- macOS: `~/Library/Application Support/Bashes/hosts.json`
- Linux: `$XDG_DATA_HOME/bashes/hosts.json` or `~/.local/share/bashes/hosts.json`
- Windows: `%APPDATA%\Bashes\hosts.json`

Generated SSH keys are stored in a `keys` directory next to `hosts.json`. The datastore remains plain JSON and can still be copied or imported between installations.

## Downloading A GitHub Actions Build

The repository includes a GitHub Actions workflow at `.github/workflows/build-desktop.yml`.

It runs automatically on pushes to `main`, on tags matching `v*`, and can also be started manually from GitHub:

1. Open the repository on GitHub.
2. Go to `Actions`.
3. Select `Build Desktop App`.
4. Click `Run workflow`.
5. Wait for the desktop build jobs to finish.
6. Download `bashes-linux-amd64`, `bashes-darwin-universal`, `bashes-windows-amd64` or experimental ARM artifacts from the workflow run page.

The workflow uses `ubuntu-22.04` because Wails v2.12 expects `libwebkit2gtk-4.0-dev`, which is available there but may be missing on newer Ubuntu releases.

## Data Testing

Validate the current JSON data file:

```bash
go run ./cmd/bashes-data validate public/db/hosts.json
```

Migrate a legacy Bashes JSON export to the new versioned format:

```bash
go run ./cmd/bashes-data migrate old-hosts.json hosts.json
```

The migrated file remains plain JSON and can be copied between Bashes installations.

## Expected UI Behavior At This Stage

When the desktop shell starts:

- The left sidebar shows hosts from the bound backend if data exists.
- The desktop build uses the Wails `go.main.App` binding for host and subsystem data.
- `Add Host` opens a left slide-out panel.
- Selecting a host or subsystem exposes contextual actions in the session header: edit, add subsystem, keys, delete, connect.
- The UI writes host and subsystem changes through the Go backend.
- Browser-only frontend testing uses an in-memory demo store because Wails bindings are unavailable there.
- The main panel renders `xterm.js` terminal tabs, one per active SSH session.
- Clicking a host or subsystem in the sidebar focuses its active SSH tab when one exists.
- Clicking a host or subsystem without an active session creates a temporary tab; starting SSH turns it into a real session tab.
- Double-clicking a host or subsystem attempts an SSH connection using automatic credentials, then opens the connect panel if credentials are needed.
- Selecting terminal text copies it to the clipboard; right-clicking the terminal pastes clipboard text into the session.
- The Connect action opens an SSH panel and starts a backend-managed shell session.
- SSH authentication can use a session-only password, `SSH_AUTH_SOCK`, default `~/.ssh` keys, or an explicit key path.
- The `Keys` panel can generate Ed25519 keys under the platform data directory and install the selected public key on a selected remote resource.
- Passwords and key passphrases are not saved in the JSON datastore.
- Generated private keys under the platform data directory must be handled as secrets.

## Troubleshooting

If `wails build -tags desktop` fails with WebKitGTK errors, run:

```bash
wails doctor
```

Then install only the missing package set appropriate for that OS and distro release.

If frontend dependencies fail to install, clean only frontend-generated files:

```bash
rm -rf frontend/node_modules frontend/package-lock.json
npm install --prefix frontend
```

Do not remove project source files or legacy data files.
