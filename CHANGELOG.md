# Changelog

All notable changes to Bashes are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.7.7] - 2026-11-06

### Fixed

- Fixed database imports from native Linux desktop dialogs.
- Kept sidebar scrollbars separate from connection cards across desktop platforms.

## [1.7.6] - 2026-11-04

### Fixed

- Fixed right-click terminal paste on macOS after extending a word selection.

## [1.7.5] - 2026-11-02

### Added

- Bundled terminal font choices with live Settings preview and JetBrains Mono as the default for new installations.
- Background indicators for file manager instances and active transfers.

### Changed

- Organized Settings into clearer visual groups.

### Fixed

- Corrected multiline right-click paste behavior in terminal editors.

## [1.7.4] - 2026-10-28

### Fixed

- Corrected the Add menu position and item visibility when the sidebar is collapsed.
- Let the Linux installer offer the required WebKitGTK 4.1 runtime on apt-based distributions.

## [1.7.3] - 2026-10-14

### Changed

- Applied a dark Web App workspace theme consistent with SSH terminal tabs.
- Improved Web App card and type-label contrast.

## [1.7.2] - 2026-10-14

### Added

- Integrated Web Apps into the shared session tab bar.
- Added temporary Web App tabs that become persistent when opened.

## [1.7.1] - 2026-10-14

### Fixed

- Made Web App tag selection on macOS use the same searchable chip editor as host forms.

## [1.7.0] - 2026-10-14

### Added

- Added direct and SSH SOCKS Web Apps with browser discovery, isolated profiles and configurable privacy retention.
- Added managed Web App tunnel status, assigned port display and stop/start controls.
- Added Windows SSH X11 forwarding with an on-demand bundled VcXsrv server.

## [1.6.8] - 2026-10-07

### Changed

- Redesigned SSH key management around searchable, grouped key sources.
- Made key management available for localhost, empty workspaces and the Tools menu.

### Added

- Added safe deletion for Bashes-managed keys, native directory selection and source rescanning.

## [1.6.7] - 2026-10-07

### Fixed

- Fixed SSH key generation on Windows filesystems without hard-link support.
- Prevented existing SSH keys from being overwritten.

## [1.6.6] - 2026-10-06

### Added

- Bundled WebKitGTK 4.0 and 4.1 builds in the Linux amd64 archive with automatic selection.

## [1.6.5] - 2026-10-05

### Added

- Reconnected dropped SSH sessions in the same tab.
- Added pasted SSH key-pair import, private-key reveal and WSL key discovery.
- Increased the maximum terminal font size to 36 px.

## [1.6.4] - 2026-09-05

### Fixed

- Fixed file manager scrolling on Linux.

### Changed

- Allowed remote browsing outside the user's home when server permissions permit it.

## [1.6.3] - 2026-09-04

### Added

- Opened local and remote files through double-click or the file manager context menu.
- Added transfer progress before remote files are opened and cleanup for stale cached files.

## [1.6.2] - 2026-09-04

### Fixed

- Preserved terminal response ordering so OSC color sequences do not leak into SSH output.

## [1.6.1] - 2026-09-01

### Added

- Remembered the last successful SSH tunnel configuration per host or subsystem.

### Changed

- Improved saved-password fields with a replaceable masked credential indicator.

## [1.6.0] - 2026-08-27

### Added

- Added portable connection tags, filtering and drag-and-drop assignment or removal.
- Added saved-password authentication and file manager refresh.

### Changed

- Improved host and terminal tab reordering, session headers, sidebar controls, tooltips and terminal layout.

Release history before 1.6.0 remains available in the Git tags and commit history.

[Unreleased]: https://github.com/signoredellarete/bashes/compare/v1.7.7...HEAD
[1.7.7]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.7
[1.7.6]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.6
[1.7.5]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.5
[1.7.4]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.4
[1.7.3]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.3
[1.7.2]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.2
[1.7.1]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.1
[1.7.0]: https://github.com/signoredellarete/bashes/releases/tag/v1.7.0
[1.6.8]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.8
[1.6.7]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.7
[1.6.6]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.6
[1.6.5]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.5
[1.6.4]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.4
[1.6.3]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.3
[1.6.2]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.2
[1.6.1]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.1
[1.6.0]: https://github.com/signoredellarete/bashes/releases/tag/v1.6.0
