# Backup And Restore

This document explains how Bashes stores, exports, imports and restores its portable JSON datastore.

The important portable file is:

```text
hosts.json
```

It contains hosts, subsystems, Web App definitions, mixed sidebar ordering, the tag catalog and assignments, saved connection metadata, and authentication preferences that are safe to store in the JSON datastore. Passwords, key passphrases, browser profiles, cookies and browser storage are never written to this file.

When requested from the password login form, Bashes stores SSH passwords in the operating system credential store: macOS Keychain, Windows Credential Manager, or a Secret Service-compatible keyring on Linux. These credentials are local to that operating-system account and are not included in database exports or backups of `hosts.json`.

On Linux, password storage requires an active Secret Service-compatible provider. SSH password login remains available without it by leaving the save option disabled.

`settings.json` is intentionally separate from `hosts.json`: it stores local machine preferences such as the optional custom SSH keys directory and preferred Web App browser. Those paths are usually OS-specific and are not part of the portable database.

## Manual Backup

Close Bashes, then copy `hosts.json` from the runtime data directory for your operating system.

If you use Bashes-generated SSH keys and want to move them too, also copy the `keys/` directory. Treat private keys as secrets.

## Export

Use:

```text
Tools > Export Database...
```

This writes a validated, normalized JSON copy of the current datastore to the file selected in the native save dialog. Hosts, Web Apps, ordering and tags are included; saved passwords, key passphrases, generated private keys and browser profile data are not.

## Import

Use:

```text
Tools > Import Database...
```

The selected JSON file is loaded, normalized and validated before it replaces the current datastore. Bashes creates `hosts.json.bak` from the previous datastore before saving the imported data. Active SSH sessions, tunnels, Web App browser runtimes and file transfers are closed after a successful import so the UI matches the imported data.

## Import From Hosts File

Use:

```text
Tools > Import from hosts file
```

Bashes reads the operating system hosts file and imports non-local entries as top-level SSH hosts. Existing hostnames or IP addresses already present in Bashes are skipped.

Default connection values:

- Port: `22`
- User: the current local username

On Linux and macOS, Bashes also reads `~/.ssh/config` and applies simple explicit `Host` blocks when they match a hosts file entry. Only `HostName`, `User` and `Port` are used. Wildcards, `Host *`, `Match` and `Include` are ignored.

Hosts file locations:

- Linux and macOS: `/etc/hosts`
- Windows: `%SystemRoot%\System32\drivers\etc\hosts`

## Automatic Backup

Before every successful save, Bashes copies the previous datastore to:

```text
hosts.json.bak
```

This backup is overwritten on each save. It is meant as a quick rollback file, not as a full backup history.

## Restore

Close Bashes, then replace the current `hosts.json` with your backup copy.

If restoring Bashes-generated keys, restore the `keys/` directory next to `hosts.json`.

On Linux and macOS, private keys should be readable only by your user:

```bash
chmod 700 keys
chmod 600 keys/*
chmod 644 keys/*.pub
```

Then start Bashes again. The app loads the JSON file on startup and validates it before use.

[← Back to README](../README.md)
