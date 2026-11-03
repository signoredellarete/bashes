# Linux Installation Details

This document covers Linux package compatibility, optional dependency installation and manual archive use.

The recommended Linux install method is the user-local installer script:

```bash
curl -fsSLO https://raw.githubusercontent.com/signoredellarete/bashes/main/scripts/install-bashes-linux.sh
bash install-bashes-linux.sh
```

The same command updates an existing installation to the latest GitHub release. It installs the app, desktop launcher and icon for the current user.

The Linux amd64 archive contains builds for both WebKitGTK 4.0 and 4.1. The installer automatically checks their runtime dependencies and installs the compatible build, preferring WebKitGTK 4.1 when both are available. The commands above do not change and the user does not need to select a WebKitGTK variant.

To install a specific release:

```bash
BASHES_VERSION=v1.7.5 bash install-bashes-linux.sh
```

The script currently supports Linux `amd64` and `arm64`, matching the release assets. For the dual-runtime amd64 package, on Debian, Ubuntu and derivatives that use `apt`, the installer offers to install the `libwebkit2gtk-4.1-0` runtime through `sudo` when WebKitGTK is missing. It does not install development packages. `BASHES_INSTALL_SYSTEM_DEPS` defaults to `ask`; set it to `yes` for non-interactive installation or `no` to disable dependency installation. The equivalent boolean values `1`/`true` and `0`/`false` are also accepted. On other distributions, the installer reports the missing libraries so they can be installed with the system package manager.

You can still use the release archive manually: extract `bashes-linux-<arch>.tar.gz` into a folder you want to keep and run `./bashes`. On amd64, this launcher performs the same automatic WebKitGTK selection without requiring the binaries to be renamed.

Supported release architectures:

- Linux amd64: `bashes-linux-amd64.tar.gz`
- Linux arm64: `bashes-linux-arm64.tar.gz`

[← Back to README](../README.md)
