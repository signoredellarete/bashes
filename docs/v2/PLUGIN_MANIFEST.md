# Plugin Manifest

This document records the initial Bashes 2 plugin package contract. Third-party
installation and user-facing plugin enablement are not enabled yet.

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

Declaring a permission does not grant it. A later plugin enablement milestone
will enforce user approval before exposing capabilities.

Discovery reads plugin directories without executing them. Invalid manifests,
unsupported platforms and incompatible API versions remain visible as
unavailable candidates with an explanatory error. Before a process can become
ready it must answer a versioned `hello` request with the same plugin ID,
version, API version and a subset of the capabilities declared here.

The native supervisor now has a repository-tested process boundary. It starts
only previously validated candidates, passes a reduced environment without SSH
agent or arbitrary application variables, limits JSON frames and retained
diagnostics, requires a bounded handshake, serializes requests and escalates a
bounded shutdown to process termination. This contract is not an operating
system sandbox: installing and enabling a plugin remains an explicit trust
decision that will be implemented before the supervisor is exposed in Bashes.

`cmd/bashes-plugin-fixture` is the repository-local diagnostic implementation.
It supports only `hello`, `diagnostic.echo` and `shutdown`; it exists to test
the transport and lifecycle contract and is not included in desktop releases.

[Back to the Bashes 2 architecture plan](ARCHITECTURE_PLAN.md)
