# Web Apps Implementation Plan

Status: implemented on `feature/web-apps`; automated checks pass and cross-platform artifacts are pending manual acceptance.

## Goal

Add top-level `WebApp` entries that open internal web applications in a Bashes-managed external browser. A WebApp can use the current network directly (for example, through a VPN) or an SSH SOCKS5 tunnel with remote DNS resolution.

The implementation must remain cross-platform, preserve existing host/session behavior, keep the JSON database portable, and avoid bundling a browser.

## Product Decisions

- WebApps are first-level items, alongside hosts. They are not SSH subsystems.
- Cards reuse the host card dimensions and interactions, use a `WebApp` badge, and have a very light blue background derived from `#495479`.
- The main `+` command becomes a small choice menu: `Host` or `Web App`.
- A double click opens the WebApp. A single click selects it and shows a lightweight WebApp workspace; it must never create an SSH terminal tab.
- Connection mode is `Direct` or `SSH Tunnel`. Tunnel fields are absent from the form while Direct is selected.
- Browser mode is `Standard` or `Private`. Persistence fields are absent while Private is selected.
- Standard defaults:
  - keep cookies and sign-ins: enabled;
  - keep local storage: enabled;
  - restore browser session: enabled;
  - clear cache when the managed browser closes: enabled.
- SOCKS port selection defaults to `Automatic`; `Choose manually` reveals a validated port field.
- Browser preference defaults to `Automatic`, with priority Firefox, Chrome, Chromium, then Edge. Safari is not supported for managed proxy sessions.
- If no compatible browser is installed, opening `Add Web App` shows a modal with OS-appropriate browser names and official download links. It must not redirect to Settings.
- Settings exposes `Web App Browser` only to choose among detected browsers or a validated custom executable.

## Data Model and JSON Migration

Keep hosts and WebApps as separate domain types. Do not add browser-only fields to `Host` or `Endpoint`.

Add to `domain.Store`:

```text
WebApps      []WebApp
TopLevelOrder []string
```

`WebApp` contains only portable configuration:

```text
ID, Name, URL, Tags
ConnectionMode: direct | ssh
GatewayResourceID (required only for ssh)
PortMode: automatic | manual
LocalSOCKSPort (manual mode only)
BrowserMode: standard | private
KeepCookies, KeepLocalStorage, RestoreSession, ClearCacheOnClose
```

Rules:

- Accept only absolute `http` and `https` URLs. Reject control characters and URL credentials.
- `GatewayResourceID` may reference a host or subsystem, but never localhost or another WebApp.
- Automatic ports are runtime values and are never persisted.
- Browser profiles, cache, cookies, credentials, PIDs and runtime tunnel state are never stored in `hosts.json` or exported.
- Increment the store schema to version 2. Add an explicit v1-to-v2 migration that creates an empty WebApp list and initializes `TopLevelOrder` with the existing host order. Never reject an existing v1 database.
- Import/export includes WebApp definitions and tags. Existing v1 imports must remain supported.
- Tag rename/delete and catalog normalization must include WebApps.
- Deleting a gateway resource referenced by WebApps must be blocked with a clear list of affected WebApps; do not cascade-delete them.

Use `TopLevelOrder` rather than replacing `Hosts` with a polymorphic resource tree. This limits migration risk and lets drag and drop order hosts and WebApps together.

## Backend Design

### Application service

Add focused methods instead of expanding the generic endpoint input:

- `ListWorkspace()` returning hosts, WebApps and top-level order;
- `AddWebApp`, `UpdateWebApp`, `DeleteWebApp`;
- `ReorderTopLevel(ids)`;
- WebApp tag updates through the existing tag contract.

Keep `ListHosts`, `AddHost`, `AddSubsystem`, `UpdateResource` and `ReorderHosts` behavior compatible until the frontend no longer depends on `ReorderHosts`.

### Browser package

Create an isolated package such as `internal/webappbrowser`; do not add OS process logic to `app.go`.

Define a small adapter contract:

- detect compatible installations;
- return stable browser ID, display name, executable and capabilities;
- start a Bashes-owned profile in Standard or Private mode;
- open another URL in the existing managed instance;
- wait for process termination;
- clean selected profile data safely.

Use OS-specific files for discovery and command construction. Execute binaries directly with argument arrays; never invoke a shell. Unit-test generated arguments for Linux, macOS and Windows.

Firefox profiles use Bashes-owned profile directories and Firefox proxy preferences. Chromium-family profiles use a unique `--user-data-dir`, `--proxy-server=socks5://127.0.0.1:<port>`, and host resolver rules that prevent local DNS leakage. Enterprise policies that override proxy flags must produce an explicit launch error.

Before building the UI, complete a technical spike on all three platforms proving:

1. a new managed instance starts without reusing the user's normal browser process;
2. a second WebApp with the same runtime group opens as a new tab;
3. internal DNS is resolved through SOCKS;
4. Private data is removed after exit;
5. only directories owned by Bashes are touched.

If an adapter cannot satisfy these contracts, mark that browser unsupported rather than adding browser-specific behavior to the frontend.

### Profiles and cleanup

Store managed profiles under the Bashes data directory, for example `webapps/profiles/<group-id>`. Private profiles go under the runtime/temp area.

Runtime groups are keyed by browser, connection mode, gateway, browser mode and persistence policy:

- matching groups reuse one browser process and open additional tabs;
- different gateways, modes or policies use separate processes/profiles;
- Direct WebApps with matching policies may share a direct profile;
- cache cleanup runs only after the last process using a profile exits;
- startup cleanup removes stale cache/private profiles left by crashes;
- cleanup uses an allow-listed set of paths below the Bashes profile root and rejects every path outside it.

Selective cleanup is adapter-owned because Firefox and Chromium use different files. When disabled, remove only the relevant Bashes-owned cookie, local-storage or session data. Default cache cleanup must not delete cookies or local storage.

### WebApp runtime manager

Create a manager separate from `App.sessions`, `App.tunnels` and file transfers. It owns browser processes, temporary profiles and WebApp-only SOCKS tunnels.

- Direct mode launches the managed browser without a proxy.
- SSH mode resolves the gateway resource, reuses its authentication preference/keyring password, starts a loopback-only SOCKS5 listener, then launches the browser.
- Automatic port mode binds `127.0.0.1:0` and uses the assigned port. Do not relax validation of the existing manual tunnel form.
- Extract reusable SSH dial/SOCKS primitives from `StartSSHTunnel`, but do not register WebApp tunnels in the normal tunnel map. Otherwise they would create `tun` chips and disable the existing Tunnel UI.
- Reuse structured authentication and host-key errors. If credentials or trust confirmation are needed, return them to a WebApp-specific prompt that reuses the existing auth field helpers.
- Monitor SSH keepalive and browser exit. On tunnel loss, attempt bounded reconnection only while the managed browser is alive; if credentials are unavailable, stop retrying and show an actionable error.
- On Bashes shutdown, stop WebApp tunnels and only browser processes started by Bashes. Never terminate the user's normal browser.

Expose narrow Wails methods such as `DetectWebAppBrowsers`, `LaunchWebApp`, `StopWebAppRuntime` and `ListWebAppRuntimes`. Keep profile paths and process details private to Go.

## Frontend Design

Move feature code into `frontend/src/web-apps/` rather than continuing to grow `main.js`. Keep pure modules for form state, validation, card ordering and runtime grouping so they can be unit-tested headlessly.

### Form

Use one Add/Edit form with these sections:

1. Name, URL and tags.
2. Browser mode: Standard / Private.
3. Standard-only persistence checkboxes.
4. Connection: Direct / SSH Tunnel.
5. Tunnel-only gateway selector and Automatic / Manual port control.

Hidden conditional fields must not submit stale values. Switching to Direct clears gateway/port fields; switching to Private ignores persistence flags without overwriting the saved Standard defaults until Save.

### Cards and workspace

- Merge hosts and WebApps according to `TopLevelOrder` before rendering.
- Keep localhost first and immovable.
- A dragged host still carries all subsystems; a WebApp moves as one top-level item.
- Existing search, tags, tag filtering, tag drag/drop, compact sidebar initials and delayed tooltip behavior must include WebApps.
- Use `WebApp` as the badge text and a restrained light-blue card treatment; selected, hover and drag outlines must remain consistent with host cards.
- Single click activates a non-terminal WebApp preview containing name, URL, connection mode and an Open command. It hides, but does not destroy or resize, active xterm sessions.
- Double click calls Open immediately.
- The contextual toolbar for a WebApp shows Open, Edit and Delete only. SSH Keys, Files, Tunnel, Connect and Disconnect remain host-only.
- Returning to a terminal tab restores terminal mode and focus using the existing focus helper.

### Browser availability

Run detection when Add Web App is requested and when Settings opens. Detection must not block normal startup.

- No compatible browser: show the informational modal and official links.
- One browser: Automatic selects it silently.
- Multiple browsers: Automatic uses the documented priority; Settings can override it.
- A configured browser removed later: show the same unavailable-browser error and do not fall back silently to an unrelated executable.

## Known Collision Points

| Area | Existing behavior | Required protection |
| --- | --- | --- |
| `internal/domain/model.go` | Schema v1 validates only hosts/subsystems | Explicit v1-to-v2 migration; old JSON must load unchanged |
| `internal/application/service.go` | Tags, delete and reorder traverse only hosts | Include WebApps without changing endpoint semantics; block deletion of referenced gateways |
| `frontend/src/main.js` selection | Selected resources can create temporary terminal tabs | WebApps use workspace mode and never enter SSH session helpers |
| Host drag/drop | `ReorderHosts` receives only host IDs | Replace frontend call with top-level ordering while preserving host/subsystem blocks |
| Header actions | Controls assume an SSH resource | Render a separate WebApp action set; never enable SSH actions for WebApps |
| Tunnel UI/state | Frontend assumes one visible tunnel per resource | Keep managed WebApp tunnels private from normal tunnel listing/chips |
| SSH authentication | Connect and Tunnel forms own credential prompts | Extract reusable auth fields/error handling; do not duplicate password/key logic |
| `settings.json` | `SSHKeySettings` currently owns the whole file | Add one read-modify-write app-settings repository so browser settings cannot erase key settings |
| Shutdown/import/delete | Active sessions, tunnels and transfers are stopped centrally | Add WebApp runtime cleanup without changing existing stop ordering |
| Browser launch | Current open-file helpers use the default OS application | Add dedicated browser adapters; do not overload file opening |
| Export/backup | `hosts.json` is portable; key/profile data is separate | Export definitions only; document that profiles contain secrets and are excluded |
| Packaging | No browser is bundled | Detect installed browsers; update README dependencies without increasing release size |

## Implementation Sequence

Keep the user-facing feature disabled until all phases pass their gates.

1. **Domain and migration**
   - Add schema v2, WebApp validation, top-level ordering and service APIs.
   - Gate: migration, JSON round-trip, tag and ordering tests pass with v1 fixtures.
2. **Browser adapter spike**
   - Implement detection, isolated profile launch, new-tab reuse and cleanup behind interfaces.
   - Gate: headless command tests plus manual Linux/macOS/Windows proof; unsupported adapters are removed.
3. **Direct WebApps**
   - Add cards, Add/Edit form, workspace preview, browser Settings and no-browser modal.
   - Gate: existing host/session/tab/drag/tag tests remain green; Direct mode works on all targets.
4. **Managed SSH mode**
   - Extract SOCKS primitives, add automatic loopback port, auth/host-key prompts and runtime sharing.
   - Gate: remote DNS and multi-service navigation work without OS proxy changes; normal Tunnel UI is unaffected.
5. **Profile retention**
   - Implement Standard/Private lifecycle and selective cleanup.
   - Gate: cookies/local storage/session choices and default cache cleanup are verified per adapter, including crash recovery.
6. **Hardening and delivery**
   - Add runtime error states, shutdown cleanup, documentation and full cross-platform regression matrix.
   - Produce feature-branch artifacts only. Merge and release only after explicit approval.

## Verification Matrix

Automated checks:

- domain validation, v1 migration, import/export and atomic JSON persistence;
- WebApp CRUD, tags, gateway deletion protection and mixed top-level reorder;
- browser detection with injected filesystem/PATH data;
- exact browser arguments and URL validation for each OS;
- profile grouping, process ownership, auto-port allocation and cleanup path containment;
- direct/tunnel runtime sharing, keepalive failure and shutdown cleanup;
- frontend conditional fields, menu choice, card filtering/order and workspace-mode transitions;
- existing Go tests, race tests, vet, frontend tests, packaging tests and version checks.

Manual acceptance on Linux, macOS, Windows amd64 and Windows arm64:

- Direct WebApp through VPN;
- SSH WebApp using internal DNS and links to additional internal services;
- two apps on the same gateway open as tabs in one managed browser;
- different gateways or Standard/Private policies open isolated instances;
- automatic and manual port behavior;
- Firefox preference and Chromium-family fallback;
- no-browser modal and official links;
- persistence choices across restart, Private cleanup, cache cleanup after normal exit and crash;
- no regressions in SSH sessions, normal tunnels, tags, drag/drop, file manager, localhost or JSON restore.

## References

- Firefox command-line profiles, tabs and private windows: https://firefox-source-docs.mozilla.org/browser/CommandLineParameters.html
- Chromium SOCKS5 and remote DNS configuration: https://www.chromium.org/developers/design-documents/network-stack/socks-proxy/

## Definition of Done

The feature is complete only when existing databases migrate without user action, WebApps work in Direct and SSH modes on every supported desktop target, browser data is confined to Bashes-owned profiles, cache cleanup is demonstrable, internal DNS does not leak locally in tunnel mode, and every existing Bashes workflow remains covered by passing regression tests.
