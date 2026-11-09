# Plugin Manifest

This document records the initial Bashes 2 plugin package contract. Process
execution and third-party installation are not enabled yet.

Each plugin is stored in its own directory and contains a `manifest.json` plus
one or more platform executables:

```text
org.bashes.example/
  manifest.json
  bin/
    example-windows-amd64.exe
    example-windows-arm64.exe
```

Example manifest:

```json
{
  "schemaVersion": 1,
  "id": "org.bashes.example",
  "name": "Example plugin",
  "version": "1.0.0",
  "apiVersion": 1,
  "description": "A diagnostic fixture",
  "capabilities": ["diagnostic"],
  "permissions": ["runtime-data.write"],
  "executables": {
    "windows/amd64": "bin/example-windows-amd64.exe",
    "windows/arm64": "bin/example-windows-arm64.exe"
  }
}
```

The host accepts only the current manifest schema and API version. Executable
paths must be relative and must resolve inside the plugin directory, including
after symlink resolution. Unknown manifest fields and permissions are rejected.

Initial permissions are:

- `credential.read`
- `network.connect`
- `network.listen`
- `process.launch`
- `runtime-data.write`

Declaring a permission does not grant it. A later plugin supervisor milestone
will enforce enablement and user approval before exposing capabilities.

[Back to the Bashes 2 architecture plan](ARCHITECTURE_PLAN.md)
