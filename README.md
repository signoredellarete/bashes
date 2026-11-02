# Bashes

**A free, open-source, cross-platform SSH client and remote server manager.**

A lightweight alternative to MobaXterm, Termius and SecureCRT — for Linux, macOS and Windows.

[![License: GPL-3.0](https://img.shields.io/badge/License-GPLv3-blue.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/signoredellarete/bashes)](https://github.com/signoredellarete/bashes/releases)
[![Platforms](https://img.shields.io/badge/platforms-Linux%20%7C%20macOS%20%7C%20Windows-informational)](#install)

<!-- TODO: replace with a real screenshot (sidebar + tabbed terminal) -->
![Bashes main window](docs/screenshots/main-window.png)


Bashes keeps all your servers in one place: a searchable, taggable sidebar of hosts and subsystems (VMs, LXC containers, Docker), tabbed SSH terminals, SSH tunnels, key management, a dual-pane SFTP file manager, and one-click access to internal web applications through an automatic SSH tunnel. Portable connection and Web App data lives in a plain JSON file that is easy to inspect, back up and move between machines.

Built with [Wails](https://wails.io) (Go backend) and a Vite frontend, with the terminal powered by `xterm.js`. Lightweight native packages, no account, no telemetry, no paid tier.

## Why Bashes?

|                | Bashes                    | MobaXterm                            |
| -------------- | ------------------------- | ------------------------------------ |
| Price          | Free, GPL-3.0             | Freemium (Pro license for full features) |
| Source code    | Open source               | Proprietary                          |
| Platforms      | Linux, macOS, Windows     | Windows only                         |
| Data format    | Plain JSON, portable      | Proprietary                          |
| Footprint      | Lightweight native package | Full suite                           |

If you mainly need SSH sessions, tunnels, SFTP and a tidy way to organize many servers — without a license key — Bashes covers that on every desktop OS.

## Features

### Hosts and organization

- Hosts with name, address, port and user, plus subsystems attached to a host (VM, LXC and Docker resource types)
- **Tags on every card, with filtering.** Assign tags when adding or editing a connection, or by dragging a tag from the sidebar onto a card; remove them the same way. Filter the sidebar by tag with `ANY` or `ALL` matching, so a list of fifty servers narrows to the handful you need
- Searchable sidebar with a compact mode that maximizes terminal space
- Animated drag-and-drop ordering; dragging a subsystem moves its parent host block with all its children
- Contextual actions per resource: edit, add subsystem, key management, delete, connect
- Import hosts from the OS hosts file, honoring simple explicit `~/.ssh/config` blocks on Linux and macOS

### Web Apps

Web App cards sit in the sidebar alongside your hosts and hold an internal HTTP/HTTPS application you want to reach. Opening one launches a dedicated browser window on your computer, either pointed straight at the app or routed through an SSH tunnel to one of your saved hosts — so an intranet-only application opens as if you were sitting on the remote network. Bashes sets up the SOCKS proxy, the remote DNS resolution and the browser profile for you; there is no `ssh -D` to remember and no proxy setting to change by hand.

- Active tunnels show a `tun` chip and their loopback SOCKS port; the SSH connection can be stopped and restarted without changing the assigned port or closing the browser
- Isolated standard or private browser profiles, with optional cookie, local-storage and session retention, and cache cleanup when the browser closes
- Works with Firefox, Chrome, Chromium and Microsoft Edge (Safari is not used for proxied Web Apps)

### Terminal

- Tabbed SSH sessions, including several sessions to the same resource, with drag-and-drop tab ordering
- Local `localhost` shell tab on Linux and macOS, backed by a native PTY
- Double-click a card to connect; select-to-copy and right-click paste

### SSH and security

- Authentication via password, SSH agent, default keys, an explicit key path, Bashes-generated keys, or keys found in the system or a custom key directory
- Optional password storage in the OS credential store: macOS Keychain, Windows Credential Manager, or a Secret Service-compatible Linux keyring
- Ed25519 key generation and public key installation on a remote host or subsystem
- SSH tunnels without opening a terminal tab: SOCKS proxy (`-D`), local (`-L`) and remote (`-R`) forwarding

### File transfer

- Dual-pane SFTP file manager: local home on the left, remote home on the right
- Drag-and-drop or copy-paste transfers, with background progress and transfer speed
- Open local or remote documents and media in the OS default application; remote files are cached locally first, with progress

### Your data, portable

- Plain JSON datastore with validation, automatic backups and atomic writes
- Export and import the datastore from the Tools menu; passwords, passphrases and browser data are never written to it
- Startup and periodic update checks against GitHub Releases

## Install

Download the package for your OS from [GitHub Releases](https://github.com/signoredellarete/bashes/releases).

### Linux

The recommended install is the user-local script, which also updates an existing installation:

```bash
curl -fsSLO https://raw.githubusercontent.com/signoredellarete/bashes/main/scripts/install-bashes-linux.sh
bash install-bashes-linux.sh
```

It installs the app, desktop launcher and icon for the current user, and picks the right WebKitGTK build (4.0 or 4.1) automatically. To install a specific release:

```bash
BASHES_VERSION=v1.7.5 bash install-bashes-linux.sh
```

You can also extract `bashes-linux-<arch>.tar.gz` manually and run `./bashes`. See [docs/INSTALL_LINUX.md](docs/INSTALL_LINUX.md) for dependency handling and the `BASHES_INSTALL_SYSTEM_DEPS` options.

### macOS

Extract the zip and open `Bashes.app`. Current builds are unsigned unless signing secrets are configured; if Gatekeeper blocks the first launch:

```bash
xattr -dr com.apple.quarantine Bashes.app
```

### Windows

Extract the zip into a user-writable folder and run `bashes.exe`. Because builds are unsigned, SmartScreen may warn and Smart App Control may block the app.

## Documentation

- [Runtime files and data locations](docs/RUNTIME_FILES.md) — where Bashes stores data on each OS
- [Backup, restore, import and export](docs/BACKUP_RESTORE.md)
- [Building from source](docs/BUILDING.md)
- [Data CLI](docs/DATA_CLI.md) — validate and migrate the JSON datastore
- [Development notes](docs/DEVELOPMENT.md)
- [Desktop testing guide](docs/DESKTOP_TESTING.md)

## Building from source

Requires Go (see `go.mod`), Node.js and npm, Wails CLI v2, and the native Wails desktop dependencies for your OS.

```bash
npm install --prefix frontend
go test ./...
npm run build --prefix frontend
wails build -tags desktop
```

On Linux, run `wails doctor` and install only the missing GTK/WebKitGTK packages. Full details in [docs/BUILDING.md](docs/BUILDING.md).

## Contributing

Bug reports and feature requests are welcome in the [issue tracker](https://github.com/signoredellarete/bashes/issues). This project does not accept external pull requests. If Bashes is useful to you, a ⭐ helps others find it.

## License

[GPL-3.0](LICENSE)
