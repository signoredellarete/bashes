# Bashes 2 Architecture Plan

This document defines the incremental path from the current Bashes 1.x codebase
to a modular Bashes 2. It is an implementation plan, not a promise to rewrite
the application from scratch.

## Product direction

Bashes remains a fast, local-first remote operations application with a
terminal-first interface. Version 2 must improve internal structure and
reliability without adopting the visually dense workspace of larger remote
administration suites.

The guiding product rule is:

> Rich architecture, focused interface.

### Goals

- Preserve the current lightweight, spacious UI and established workflows.
- Keep Go, Wails and xterm.js as the core technology choices.
- Separate domain logic, runtime lifecycle, native integrations and UI state.
- Make SSH, file transfer, tunnels, Web Apps and future protocols independent
  modules with explicit contracts.
- Support optional out-of-process plugins without making the base installation
  larger or less reliable.
- Keep the portable JSON datastore and migrate existing Bashes 1.x data.
- Improve testability, error contracts, cancellation and resource cleanup.

### Non-goals

- Reimplementing stable SSH or terminal protocol libraries.
- Replacing xterm.js or SVAR solely to make the implementation look uniform.
- Loading untrusted native libraries into the Bashes process.
- Turning every internal function into a generic plugin extension point.
- Redesigning the UI while architectural work is in progress.
- Adding new end-user protocols before the modular foundation is proven.

## Implementation status

Status on 2026-11-10 for branch `feature/v2-modular-architecture`:

| Phase | Status | Current result |
| --- | --- | --- |
| 0 - Baseline and contracts | Complete | Desktop adapters, error/event contracts and architecture guards are in place. |
| 1 - Backend application boundaries | Complete | Data and connection workflows are independent of Wails and native operations use injectable adapters. |
| 2 - Runtime manager | Complete | Native runtimes have backend ownership, common lifecycle state and deterministic cleanup tests. |
| 3 - Frontend modularization | In progress | Desktop APIs and shared application/resource state are extracted; feature UI still remains in `main.js`. |
| 4 - Transfer reliability | Not started | Planned after the application boundaries are stable. |
| 5 - Plugin foundation | In progress | Manifest, discovery, protocol and supervisor exist without user-facing enablement. |
| 6 - First optional plugin | Not started | Depends on the completed plugin permission and installation model. |
| 7 - Framework and release decision | Not started | Deferred until v1 feature parity is preserved by the modular core. |

- baseline Go, race, vet, frontend test and production frontend build checks pass;
- the frontend desktop binding and Wails runtime APIs are isolated behind tested
  adapters;
- structured application errors and desktop event names have explicit backend
  contracts;
- architecture tests prevent core Go packages and frontend features from adding
  direct Wails dependencies;
- application settings use a versioned, locked, atomic repository while keeping
  the Bashes 1.x JSON fields compatible;
- hosts file and SSH config import logic lives in an independent tested module;
- portable JSON export/import and hosts-file preview/import orchestration live
  in a datastore service; `App` only performs runtime cleanup and emits desktop
  events after replacement;
- local SSH key inventory, settings, generation, import, reading and deletion
  live in an independent tested service; remote key installation has a separate
  service for resource lookup, SSH execution and authentication persistence;
- SSH authentication method construction and OpenSSH/PEM/PPK parsing live in an
  independent builder with injectable home, filesystem, environment and agent
  socket dependencies;
- saved-password lifecycle and resource authentication preferences are owned by
  one tested service shared by SSH sessions, tunnels, transfers and Web Apps;
- SSH and local terminal startup, I/O, resize, keepalive and close events are
  orchestrated by a desktop-independent runtime service with typed events;
- SSH tunnel validation, listener setup, forwarding, keepalive and cleanup are
  orchestrated by a desktop-independent runtime service shared by the facade;
- SSH dialing, host-key policy, structured trust errors and accepted fingerprint
  persistence are owned by a tested connector shared by sessions, tunnels,
  transfers and Web App gateways;
- Web App browser detection, preference validation, persistence and runtime
  resolution live in an independent tested native service;
- Wails events, dialogs and external URL operations are isolated behind an
  injectable desktop runtime adapter; facade menu workflows run in tests
  without a WebView;
- runtime timestamps and Web App browser launch/cleanup operations use
  injectable adapters; deterministic tests cover clock-dependent filenames and
  transfer staging paths;
- terminal session lifecycle, SSH tunnel lifecycle and file transfer job state
  are owned by dedicated managers with idempotent cancellation and race-tested
  concurrent cleanup;
- terminal sessions, tunnels, file transfer sessions/jobs and Web App runtimes
  expose common `starting`, `running`, `stopping`, `stopped` and `failed`
  lifecycle transitions through typed snapshots and observer events;
- SFTP session ownership and shared Web App browser/tunnel associations use
  dedicated registries, so `App` no longer coordinates native resources through
  one global mutex;
- SSH keepalive behavior is shared by sessions, tunnels and Web App gateways
  through a tested native adapter;
- frontend application state and the hierarchical resource model are isolated
  from DOM rendering and covered by unit tests;
- the complete Settings panel is mounted as a Svelte feature with a tested pure
  model; terminal preview, cancel rollback and browser preferences keep their
  existing application contracts;
- shared confirmation and application-message overlays are mounted as one
  Svelte feature while preserving focus restoration, Escape handling and the
  existing modal call sites; error notifications and their lifecycle now use
  the same feature, with overlay styles isolated in one feature stylesheet;
- the Tag Manager panel owns its rendering and local edit state in Svelte;
  backend tag mutations and global filtering remain explicit application
  callbacks backed by a tested panel model;
- the sidebar tag-filter strip owns ANY/ALL controls, chips and clear state in
  Svelte while retaining the existing SortableJS drag hooks;
- the sidebar resource tree is rendered from a tested Svelte view model that
  preserves hierarchy, top-level order, filters, tunnel chips, tooltips and the
  existing SortableJS data hooks;
- the selected Web App workbench is a Svelte feature backed by a pure view
  model; browser launch and tunnel operations remain application callbacks;
- newly created hosts, nested systems and Web Apps enter the same selected and
  pending-tab state as a direct card click, while edits preserve the current
  selection; the sidebar minimally scrolls the new card into view;
- Add/Edit Host and Subsystem now use a Svelte panel with component-local form
  state and a reusable, tested tag editor; backend mutations remain application
  callbacks and the legacy JSON contract is unchanged;
- Add/Edit Web App now uses the same reusable tag editor inside a dedicated
  Svelte panel; gateway selection and conditional tunnel/browser fields are
  produced by a tested view model while persistence remains in the backend;
- the SSH Connect panel now owns authentication form state and key-source
  rendering in Svelte, backed by tested password/key/path normalization while
  session startup and host-key confirmation remain application callbacks;
- the SSH Tunnel panel now owns SOCKS/local/remote form state, command previews,
  per-host defaults and active status in Svelte; it shares the authentication
  fields with Connect while native tunnel lifecycle remains application-owned;
- Web App gateway authentication now uses the same shared credentials component
  and tested normalization as SSH sessions and tunnels, with browser launch and
  host-key confirmation still delegated to application callbacks;
- plugin manifest validation, discovery, handshake and bounded process
  supervision are tested; the supervisor is part of application shutdown but
  has no user-facing installation or enablement path yet;
- plugin package details are documented in
  [PLUGIN_MANIFEST.md](PLUGIN_MANIFEST.md).

The next implementation milestone is frontend modularization in Phase 3. The
SFTP engine remains a Phase 4 concern, where its extraction is coupled to
conflict, retry and rollback contracts rather than treated as an unrelated
facade refactor. Plugin process supervision is available
to the application shutdown path, but user-facing plugin installation and
enablement remain blocked on explicit permission and trust policy.

## Technology decisions

### Go remains the native backend

Go is a good fit for concurrent network sessions, streaming, cancellation and
cross-platform binaries. The current maintainability problems come from module
boundaries, not from the language. Protocol implementations remain behind Go
interfaces so they can be tested without Wails.

### Wails remains the desktop adapter

Business logic must not import Wails. A small desktop adapter will expose typed
commands and translate backend events for the frontend. This makes a future
Wails upgrade a packaging/integration project rather than another application
rewrite.

Bashes 2 initially stays on Wails v2 because the current Linux release supports
WebKitGTK 4.0 and 4.1. Wails v3 will be evaluated after the native boundaries are
separated; migration requires an explicit supported-distribution decision.

### xterm.js remains the terminal renderer

xterm.js is a mature renderer also used by MobaRust and VS Code. It will remain
responsible only for terminal display and input. PTY, SSH, session lifecycle and
terminal byte streams remain native responsibilities.

### Svelte becomes the frontend component boundary

The existing frontend mixes a large imperative `main.js` with isolated Svelte
components. Bashes 2 will move screens to small Svelte components incrementally,
preserving existing markup and styling. Pure state and protocol code remains in
framework-independent JavaScript modules. There will be no parallel permanent
implementation of the same screen.

## Target architecture

```text
Frontend
  app shell and routing
  feature components and stores
  typed desktop client and event client
                 |
          Wails adapter only
                 |
Application core
  use cases, lifecycle orchestration, policies
                 |
Domain model ----+---- ports/interfaces
                      | SSH/SFTP/local PTY
                      | JSON repository/keyring
                      | browser/OS integration
                      | plugin supervisor
                 |
Infrastructure adapters and optional plugin processes
```

Dependencies point inward. Domain and application packages must be usable by
unit tests and command-line tools without starting Wails or a WebView.

### Backend modules

The intended package ownership is:

| Area | Responsibility |
| --- | --- |
| `internal/domain` | Validated persistent models and value types only |
| `internal/application` | Use cases operating through interfaces |
| `internal/runtime` | Ownership of active sessions, tunnels, transfers and Web App runtimes |
| `internal/session` | Terminal session state machine and events |
| `internal/transfer` | Transfer jobs, conflict policy, progress, cancellation and retry |
| `internal/sshkeys` | Key inventory, generation, import and installation workflows |
| `internal/importer` | Database, hosts file and future import formats |
| `internal/settings` | Versioned application settings independent of host data |
| `internal/plugin` | Manifest validation, discovery and process supervision |
| `internal/adapters/*` | JSON, keyring, SSH/SFTP, browser and OS implementations |
| `internal/desktop` | Wails-facing facade and event translation |

These names are targets, not a requirement to move every file immediately.
A package is introduced only when one coherent responsibility can be moved with
tests.

### Frontend modules

```text
frontend/src/
  app/                 bootstrap, shell, global overlays
  api/                 desktop commands, events and error decoding
  state/               resource, session and UI stores
  features/
    resources/         sidebar, tags and resource editors
    terminal/          tabs, xterm lifecycle and focus
    connections/       authentication and connection panels
    transfers/         SVAR workspace and transfer queue
    tunnels/           tunnel forms and status
    keys/              SSH key management
    webapps/           Web App editor and runtime controls
    settings/          application settings
  shared/              reusable controls, icons and small utilities
```

Feature modules communicate through explicit state/actions, not by querying
unrelated DOM nodes. Terminal instances stay outside serializable UI state and
are owned by a dedicated terminal registry.

## Runtime ownership

The application core must be the authoritative owner of native resources.
Frontend tabs represent runtime state; they do not control its integrity.

Every long-lived runtime has:

- a stable ID and owning resource ID;
- an explicit state (`starting`, `running`, `stopping`, `stopped`, `failed`);
- a context and cancellation path;
- idempotent stop and bounded shutdown;
- structured progress/status events;
- a final reason and error code;
- cleanup triggered by resource deletion, database replacement and app exit.

The runtime manager exposes snapshots plus events. It must never require the
frontend to call several methods in the correct order to avoid a leak.

## Plugin model

Plugins run as separate processes. Go's native `.so` plugin mechanism is not
suitable because it is version-sensitive and not consistently available across
Windows, macOS and Linux.

### Package layout

An installed plugin is a directory containing:

```text
manifest.json
bin/<platform-architecture executable>
assets/                     optional static UI assets
LICENSE
```

The manifest declares a stable plugin ID, version, Bashes API range, supported
platforms, executable path, capabilities and requested permissions. Plugin
configuration is namespaced by plugin ID. Secrets are never stored in the
manifest or exported JSON.

### Communication

- A versioned, framed JSON protocol over stdin/stdout is the initial transport.
- Standard error is diagnostic output with bounded retention and redaction.
- Startup uses a `hello -> capabilities -> ready` handshake with a timeout.
- Every request has an ID, deadline and cancellation operation.
- Events are typed and bounded; binary or high-volume streams use a dedicated
  local channel rather than JSON/base64 messages.
- Unexpected exit affects only the plugin feature and produces a structured
  diagnostic; it must not terminate Bashes.

The first protocol version will remain intentionally small. It will cover
discovery, health, lifecycle and capability invocation before supporting custom
plugin UI.

### Security boundary

- Plugins are disabled until explicitly installed and enabled by the user.
- The core validates manifests and executable paths before launch.
- Plugins receive only declared configuration and capabilities.
- Credentials remain in the native credential service. A plugin receives a
  short-lived secret through an inherited pipe or scoped request only when a
  capability requires it; secrets never appear in arguments, logs or exports.
- Plugin UI, if introduced, uses a constrained host API and cannot call Wails
  bindings directly.
- Signing and repository trust are separate later milestones; the first local
  development implementation must label unsigned plugins clearly.

### Initial plugin candidates

The plugin host will be proven with a small diagnostic fixture. Only after the
contract and failure isolation are tested should X11, RDP or VNC be implemented.
X11 server packaging is the preferred first real plugin because it is optional,
platform-specific and adds substantial release weight.

## Persistence and compatibility

- Existing `hosts.json` remains importable and exportable.
- Each schema migration is explicit, one-way in memory and covered by fixtures.
- Runtime state, transfer history, browser cache and plugin processes are not
  part of the portable database.
- Plugin configuration is stored in a namespaced section with schema version;
  secrets are represented only by opaque credential references.
- Writes remain validated, locked, backed up and atomic.
- Bashes 1.x must remain able to run alongside v2 during development by using a
  separate opt-in data directory or a copied fixture database.

## Lessons adopted from MobaRust

Bashes should adopt the engineering patterns, not the denser interface:

- explicit state machines for sessions and transfers;
- typed native commands and events;
- backend-owned streaming and cancellation;
- bounded concurrency, recursion and event rates;
- temporary transfer destinations and explicit conflict policy;
- isolated experimental helpers with a readiness handshake;
- protocol fixtures that do not require production credentials or hosts;
- visible capability and transport status rather than optimistic UI state.

## Delivery phases

Checklist legend: `[x]` complete, `[-]` in progress, `[ ]` pending.

### Phase 0: Baseline and contracts

- [x] Record the supported platform/release matrix and current behavioral baseline.
- [x] Run existing Go, race, vet, frontend and production build checks.
- [x] Extract the frontend desktop API from `main.js` with no behavior change.
- [x] Centralize structured backend errors and desktop event names.
- [x] Add architecture checks that prevent core packages from importing Wails.

Exit condition: existing tests pass and `main.js` no longer owns the desktop
binding implementation.

### Phase 1: Backend application boundaries

- [x] Reduce `App` to a Wails facade composed from services.
- [x] Extract settings, imports and SSH key workflows with existing tests moved or
  expanded around their public contracts.
- [x] Introduce dependency injection for filesystem, dialogs, browser and clock
  operations where platform behavior matters.

Exit condition: resource management, imports, keys and settings are testable
without constructing the Wails facade.

Scope note: protocol engines that already have backend runtime ownership are not
rewritten merely to move files. The SFTP engine is extracted with the transfer
contracts in Phase 4; Web App runtime evolution follows the plugin boundary.

### Phase 2: Runtime manager

- [x] Introduce common lifecycle state and structured runtime events.
- [x] Move SSH/local sessions, tunnels, transfers and Web App runtimes behind the
  manager incrementally.
- [x] Add deterministic tests for concurrent start/stop, connection loss, deletion,
  database replacement and shutdown.

Exit condition: no native resource relies on frontend cleanup for correctness.

### Phase 3: Frontend modularization

- [x] Introduce state stores and the terminal registry.
- [x] Extract Settings, shared overlays, Tag Manager and sidebar tag filters.
- [x] Extract Add/Edit Host, Subsystem and Web App panels with one shared tag
  editor.
- [x] Extract Connect, SSH Tunnel and Web App gateway authentication with one
  shared credentials component.
- [x] Extract the selected Web App workspace and runtime controls.
- [-] Extract the sidebar resource tree, ordering and tag drag/drop behavior.
- [ ] Extract terminal tabs, xterm lifecycle, resizing and focus management.
- [-] Extract the file-transfer modal shell around the existing Svelte file
  manager and transfer queue.
- [ ] Extract SSH key inventory, generation, import and installation views.
- [-] Preserve UI dimensions, colors, shortcuts, focus and modal behavior with
  focused automated and desktop preview tests.
- [-] Split CSS by tokens, layout and feature while keeping one generated bundle.
- [ ] Reduce `main.js` to application bootstrap and feature orchestration only.

Exit condition: `main.js` is bootstrap code rather than the application.

### Phase 4: Transfer reliability

- [ ] Add structured SFTP errors, retry and ETA.
- [ ] Define explicit destination conflict policies.
- [ ] Add bounded recursive traversal and explicit symlink/special-file behavior.
- [ ] Add rollback/preservation rules for replacement.
- [ ] Design remote editing with revision/conflict checks before implementation.

Exit condition: interrupted or conflicting transfers have deterministic,
tested outcomes and useful diagnostics.

### Phase 5: Plugin foundation

- [x] Implement manifest schema and validation.
- [-] Implement discovery, enable/disable and compatibility reporting.
- [x] Implement process handshake, health, cancellation and shutdown.
- [x] Ship a repository-local fixture plugin and failure-isolation tests.
- [ ] Add a minimal plugin settings screen only after backend behavior is stable.

Exit condition: a crashing or incompatible fixture is contained and explained
without affecting SSH sessions or persisted data.

### Phase 6: First optional plugin

- [ ] Move the Windows X server experiment into its own plugin repository/package.
- [ ] Keep binaries out of the core Bashes archive.
- [ ] Verify installation, updates, process cleanup and diagnostics on Windows x64
  and ARM64.

### Phase 7: Framework and release decision

- [ ] Evaluate Wails v3 against the supported Linux distribution matrix.
- [ ] Benchmark startup, idle memory, terminal throughput and package size.
- [ ] Produce signed/installable package plans independently from plugin packages.
- [ ] Publish a v2 prerelease only after v1 feature parity and migration tests.

## Known collision points

Changes in these areas require focused regression tests:

- `App` method names are the current generated Wails API contract.
- `main.js` combines state, DOM identity, event registration and xterm ownership.
- Session IDs connect backend maps, frontend tabs and asynchronous output events.
- Resource deletion and database import currently stop several runtime types.
- Authentication defaults are shared by SSH, tunnels, transfers and Web Apps.
- File transfer has its own Svelte mount and Wails event listeners.
- Browser profiles and Web App tunnels share lifecycle state.
- Settings and host data use related filesystem locations but different export
  and secrecy requirements.
- macOS, Windows and Linux WebViews differ in clipboard, drag/drop, scrolling
  and keyboard behavior.
- Release packaging currently encodes platform-specific WebKit and icon work.

## Quality gates

Every phase must maintain:

- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `npm test --prefix frontend`
- `npm run build --prefix frontend`
- JSON migration and round-trip fixtures when persistence changes
- no secrets in logs, events, exports or process arguments
- bounded goroutines/processes and idempotent shutdown
- no visual redesign unless separately reviewed

Cross-platform preview artifacts are requested only at milestones that touch
desktop integration. Main remains the maintenance line for Bashes 1.x bug fixes;
those fixes are merged or cherry-picked into this branch with their tests.

## Definition of Bashes 2 readiness

Bashes 2 is ready for prerelease when:

- all Bashes 1.x supported workflows pass the platform test matrix;
- the Wails facade and frontend bootstrap contain no business logic;
- runtime ownership and shutdown are backend-authoritative;
- datastore migration is reversible through backup and documented export;
- the plugin fixture proves compatibility checks and crash isolation;
- core packages do not depend on plugin binaries;
- performance is no worse than Bashes 1.x for startup, idle memory and terminal
  input/output under the agreed tolerance;
- the UI retains the focused Bashes identity.

