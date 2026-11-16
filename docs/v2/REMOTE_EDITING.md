# Remote Editing Design

This document defines the reliability contract for editing remote files in a
local application. It deliberately separates the design from implementation.

## Scope

- Only regular remote files are editable. Folders, symbolic links, devices,
  sockets, and pipes are rejected.
- The existing SFTP session and transfer job infrastructure performs downloads,
  uploads, progress reporting, cancellation, and structured errors.
- Temporary files live in Bashes' private runtime cache and are removed by the
  existing cache retention policy.

## Revision contract

Before download, Bashes records a revision containing the normalized remote
path, size, modification time, and server file identity where available. A
content hash is calculated only when timestamp and size cannot distinguish two
revisions; large files must not be hashed unconditionally.

The working copy is downloaded atomically and opened through the operating
system's default application. Bashes watches only files opened by the current
runtime and debounces repeated editor save events.

## Saving

Before every upload, Bashes reads the current remote revision:

- if it matches the downloaded revision, the working copy is uploaded to a
  temporary sibling and atomically promoted;
- if it changed, Bashes does not overwrite it automatically and asks the user
  to download the remote version, overwrite explicitly, or save the local copy
  under another name;
- if it was deleted, recreating it requires explicit confirmation;
- a failed upload preserves both the remote file and the local working copy.

After a successful save, the recorded revision is replaced with the new remote
revision. Multiple save events for the same file are serialized.

## Lifecycle

Closing the Files window does not discard dirty working copies. Closing Bashes
must wait for or cancel active uploads using the existing transfer shutdown
contract. Runtime cache cleanup removes only clean, inactive copies; unresolved
or failed edits remain recoverable until the user dismisses them.

This design must be implemented behind a small backend service and must not add
editor-specific behavior to `main.js` or the SVAR component.

[Back to the architecture plan](ARCHITECTURE_PLAN.md)
