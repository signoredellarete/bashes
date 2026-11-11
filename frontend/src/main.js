import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import Sortable from 'sortablejs';
import iconArrowsTransferUp from '@tabler/icons/outline/arrows-transfer-up.svg?raw';
import iconArrowLeft from '@tabler/icons/outline/arrow-left.svg?raw';
import iconCheck from '@tabler/icons/outline/check.svg?raw';
import iconCopy from '@tabler/icons/outline/copy.svg?raw';
import iconChevronLeft from '@tabler/icons/outline/chevron-left.svg?raw';
import iconChevronDown from '@tabler/icons/outline/chevron-down.svg?raw';
import iconEye from '@tabler/icons/outline/eye.svg?raw';
import iconEyeOff from '@tabler/icons/outline/eye-off.svg?raw';
import iconExternalLink from '@tabler/icons/outline/external-link.svg?raw';
import iconFileText from '@tabler/icons/outline/file-text.svg?raw';
import iconFolder from '@tabler/icons/outline/folder.svg?raw';
import iconFolderOpen from '@tabler/icons/outline/folder-open.svg?raw';
import iconKey from '@tabler/icons/outline/key.svg?raw';
import iconMinus from '@tabler/icons/outline/minus.svg?raw';
import iconPencil from '@tabler/icons/outline/pencil.svg?raw';
import iconPlugConnectedX from '@tabler/icons/outline/plug-connected-x.svg?raw';
import iconPlus from '@tabler/icons/outline/plus.svg?raw';
import iconRefresh from '@tabler/icons/outline/refresh.svg?raw';
import iconSearch from '@tabler/icons/outline/search.svg?raw';
import iconServer from '@tabler/icons/outline/server-2.svg?raw';
import iconSubtask from '@tabler/icons/outline/subtask.svg?raw';
import iconTag from '@tabler/icons/outline/tag.svg?raw';
import iconTags from '@tabler/icons/outline/tags.svg?raw';
import iconTrash from '@tabler/icons/outline/trash.svg?raw';
import iconWorld from '@tabler/icons/outline/world-www.svg?raw';
import iconX from '@tabler/icons/outline/x.svg?raw';
import iconPlayerPlay from '@tabler/icons/filled/player-play.svg?raw';
import '@xterm/xterm/css/xterm.css';
import bashesLogo from './assets/bashes.png';
import { createBackendClient } from './api/backend.js';
import {
  onRuntimeEvent,
  openRuntimeURL,
  readClipboardText as readClipboard,
  RUNTIME_EVENT,
  writeClipboardText as writeClipboard,
} from './api/runtime.js';
import { externalHttpURL } from './external-links.js';
import { CUSTOM_KEY_PATH_HELP } from './connections/auth-model.js';
import { connectPanelData } from './connections/connect-model.js';
import { mountConnectPanel } from './connections/mount.js';
import {
  LOCAL_RESOURCE_ID,
  findWorkspaceResource,
  isLocalResource,
  localResource,
  nestedResourceIDs,
  resourceIDsForSelection,
  resourceName,
  resourceTarget,
} from './features/resources/model.js';
import { createAppState } from './state/app-state.js';
import { mountAppOverlays } from './app/overlays/mount.js';
import { mountResourcePanel } from './resources/mount.js';
import { resourcePanelData } from './resources/panel-model.js';
import { mountSettingsPanel } from './settings/mount.js';
import { settingsPanelData as createSettingsPanelData } from './settings/model.js';
import { mountTagPanel } from './tags/mount.js';
import { mountTagFilters } from './tags/filters-mount.js';
import { tagFilterData } from './tags/filter-model.js';
import { tagPanelData as createTagPanelData } from './tags/panel-model.js';
import { passwordFieldValue, placeCaretAfterSavedPassword, showSavedPassword } from './password-field.js';
import {
  errorDetail,
  isAuthError,
  isCredentialStoreError,
  parseHostKeyMismatchError,
  parsePublicTunnelBindError,
  parseUnknownHostKeyError,
} from './ssh-errors.js';
import {
  closedSessionShortcut,
  lastFocusedSessionId as lastFocusedSessionIdFromState,
  pendingSessionForResource as pendingSessionForResourceFromState,
  preferredSessionForResource as preferredSessionForResourceFromState,
  realSessionsForResource as realSessionsForResourceFromState,
  rememberFocus,
  orderSessions,
  replaceSession,
  sessionsForResource as sessionsForResourceFromState,
} from './session-state.js';
import {
  clampNumber,
  DEFAULT_TERMINAL_FONT_FAMILY,
  DEFAULT_TERMINAL_FONT_SIZE,
  DEFAULT_TERMINAL_SCROLLBACK,
  MAX_TERMINAL_FONT_SIZE,
  MIN_TERMINAL_FONT_SIZE,
  TERMINAL_FONT_FAMILY_OPTIONS,
  loadTerminalSettings,
  persistTerminalSettings,
  terminalAppearancePreview,
} from './terminal-settings.js';
import { repeatedKeyData } from './terminal-keyboard.js';
import { createOrderedWriter } from './ordered-writer.js';
import { pasteIntoTerminal } from './terminal-paste.js';
import { revealSidebarResource, sidebarListInset } from './sidebar-scroll.js';
import { loadTunnelPreference, saveTunnelPreference } from './tunnel-preferences.js';
import {
  canonicalTag,
  filterResourceTree,
  mergeTagCatalog,
  resourceTags,
  tagHue,
  visibleResourceTags,
} from './resource-tags.js';
import { isWebAppResource, webAppMatchesFilters } from './web-apps/model.js';
import { mountWebAppPanel } from './web-apps/panel-mount.js';
import { webAppPanelData } from './web-apps/panel-model.js';
import { mountWebAppWorkspace } from './web-apps/workspace-mount.js';
import { webAppWorkspaceData } from './web-apps/workspace-model.js';
import './terminal-fonts.css';
import './styles.css';
import './app/overlays/overlays.css';
import './tags/filters.css';
import './tags/editor.css';
import './tags/panel.css';
import './web-apps/panel.css';
import './web-apps/workspace.css';

const terminalSettings = loadTerminalSettings(localStorage);

const state = createAppState({ terminalSettings, storage: localStorage });

let searchRenderFrame = 0;
let terminalFitFrame = 0;
let tooltipTimer = 0;
let hostSortable = null;
let tabSortable = null;
let tagCatalogSortable = null;
let tagRemoveSortable = null;
let activeTagDropTarget = null;
let resourceTagSortables = [];
let suppressTagFilterClickUntil = 0;
let tagPopover = null;
let tagPopoverRequest = 0;
const dragOutlineTimers = { hosts: 0, tabs: 0 };
const dragOutlineTargets = { hosts: null, tabs: null };

const FILE_TRANSFER_ENABLED = true;
const TOOLTIP_DELAY = 500;
const DRAG_OUTLINE_DELAY = 300;
const backend = createBackendClient({
  demoMode: import.meta.env.DEV && globalThis.__BASHES_DEMO__ === true,
  getSessionTerminal: (sessionID) => state.sessions.get(sessionID)?.terminal,
  getWebAppRuntime: (webAppID) => state.webAppRuntimes.get(webAppID),
  listWebAppRuntimes: () => state.webAppRuntimes.values(),
});
const {
  addHost: apiAddHost,
  addSubsystem: apiAddSubsystem,
  addWebApp: apiAddWebApp,
  checkForUpdate: apiCheckForUpdate,
  chooseSSHKeyDirectory: apiChooseSSHKeyDirectory,
  createTag: apiCreateTag,
  deleteResource: apiDeleteResource,
  deleteSSHKey: apiDeleteSSHKey,
  deleteTag: apiDeleteTag,
  deleteWebApp: apiDeleteWebApp,
  generateSSHKey: apiGenerateSSHKey,
  getAppInfo: apiGetAppInfo,
  getSSHKeyInventory: apiGetSSHKeyInventory,
  getSSHKeySettings: apiGetSSHKeySettings,
  getWebAppBrowserSettings: apiGetWebAppBrowserSettings,
  hasSavedPassword: apiHasSavedPassword,
  importFromHostsFile: apiImportFromHostsFile,
  importSSHKeyPair: apiImportSSHKeyPair,
  installSSHKey: apiInstallSSHKey,
  launchWebApp: apiLaunchWebApp,
  listSSHTunnels: apiListSSHTunnels,
  listTags: apiListTags,
  listWebAppBrowsers: apiListWebAppBrowsers,
  listWebAppRuntimes: apiListWebAppRuntimes,
  listWorkspace: apiListWorkspace,
  readSSHPrivateKey: apiReadSSHPrivateKey,
  readSSHPrivateKeyPath: apiReadSSHPrivateKeyPath,
  readSSHPublicKey: apiReadSSHPublicKey,
  readSSHPublicKeyPath: apiReadSSHPublicKeyPath,
  renameTag: apiRenameTag,
  reorderHosts: apiReorderHosts,
  reorderTopLevel: apiReorderTopLevel,
  resizeSSHSession: apiResizeSSHSession,
  saveSSHKeySettings: apiSaveSSHKeySettings,
  saveWebAppBrowserSettings: apiSaveWebAppBrowserSettings,
  setResourceTags: apiSetResourceTags,
  startLocalSession: apiStartLocalSession,
  startSSHSession: apiStartSSHSession,
  startSSHTunnel: apiStartSSHTunnel,
  startWebAppTunnel: apiStartWebAppTunnel,
  stopSSHSession: apiStopSSHSession,
  stopSSHTunnel: apiStopSSHTunnel,
  stopWebAppTunnel: apiStopWebAppTunnel,
  supportsLocalShell: apiSupportsLocalShell,
  updateResource: apiUpdateResource,
  updateWebApp: apiUpdateWebApp,
  writeSSHSession: apiWriteSSHSession,
} = backend;
const customKeyPathHelp = CUSTOM_KEY_PATH_HELP;
const TABLER_ICONS = Object.freeze({
  'arrows-transfer-up': iconArrowsTransferUp,
  'arrow-left': iconArrowLeft,
  check: iconCheck,
  copy: iconCopy,
  'chevron-left': iconChevronLeft,
  'chevron-down': iconChevronDown,
  eye: iconEye,
  'eye-off': iconEyeOff,
  'external-link': iconExternalLink,
  'file-text': iconFileText,
  folder: iconFolder,
  'folder-open': iconFolderOpen,
  key: iconKey,
  minus: iconMinus,
  pencil: iconPencil,
  'player-play-filled': iconPlayerPlay,
  'plug-connected-x': iconPlugConnectedX,
  plus: iconPlus,
  refresh: iconRefresh,
  search: iconSearch,
  'server-2': iconServer,
  subtask: iconSubtask,
  tag: iconTag,
  tags: iconTags,
  trash: iconTrash,
  'world-www': iconWorld,
  x: iconX,
});
const iconElement = (icon, className = '') => {
  const template = document.createElement('template');
  template.innerHTML = TABLER_ICONS[icon].trim();
  const element = template.content.firstElementChild;
  element.classList.add('ti', `ti-${icon}`, ...className.split(/\s+/).filter(Boolean));
  element.setAttribute('aria-hidden', 'true');
  element.setAttribute('focusable', 'false');
  return element;
};
const iconMarkup = (icon, className = '') => iconElement(icon, className).outerHTML;
const ICON_CLOSE = iconMarkup('x');

function authFieldsMarkup(context) {
  return `
    <label>
      <span>Authentication</span>
      <select name="authMethod">
        <option value="agent">Agent / system keys</option>
        <option value="password">Password</option>
        <option value="key">Existing key</option>
        <option value="path">Key file</option>
      </select>
    </label>
    <label data-auth-field="key">
      <span>Existing Key</span>
      <select name="keyName"></select>
    </label>
    <label data-auth-field="password">
      <span>Password</span>
      <input name="password" type="password" autocomplete="current-password" />
    </label>
    <label class="checkbox-row" data-auth-field="password">
      <input name="savePassword" type="checkbox" />
      <span>Save password in system keyring</span>
    </label>
    <label data-auth-field="path">
      <span class="field-label-with-help">
        Custom Key Path
        <span class="field-help-trigger" role="button" tabindex="0" aria-label="Custom key path help" aria-expanded="false">?</span>
        <span class="field-help-popover">${customKeyPathHelp}</span>
      </span>
      <input name="privateKeyPath" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="optional private key path" />
    </label>
    <label data-auth-field="key-passphrase">
      <span>Key Passphrase</span>
      <input name="privateKeyPassphrase" type="password" autocomplete="off" />
    </label>
    <label class="checkbox-row">
      <input name="trustHostKey" type="checkbox" />
      <span>Skip host key verification for this ${context} (insecure)</span>
    </label>
  `;
}

const app = document.querySelector('#app');

app.innerHTML = `
  <aside class="sidebar">
    <header class="brand">
      <img src="${bashesLogo}" alt="" />
      <div>
        <h1>Bashes</h1>
        <span>Remote sessions</span>
      </div>
      <button id="toggle-sidebar" class="sidebar-toggle" type="button" title="Compact sidebar" aria-label="Compact sidebar">
        ${iconMarkup('chevron-left')}
      </button>
    </header>

    <div class="toolbar">
      <label class="search-control" aria-label="Search connections">
        ${iconMarkup('search')}
        <input id="search" type="search" placeholder="Search hosts, users, tags" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" />
      </label>
      <button id="open-add-menu" class="toolbar-icon-button" type="button" title="Add" aria-label="Add host or Web App" aria-expanded="false">
        ${iconMarkup('plus')}
      </button>
      <div id="add-resource-menu" class="add-resource-menu" hidden>
        <button type="button" data-add-resource="host">${iconMarkup('server-2')}<span>Host</span></button>
        <button type="button" data-add-resource="webapp">${iconMarkup('world-www')}<span>Web App</span></button>
      </div>
      <button id="open-tag-manager" class="toolbar-icon-button secondary" type="button" title="Tags" aria-label="Tags">
        ${iconMarkup('tags')}
      </button>
    </div>

    <div id="tag-filters-root"></div>

    <div id="hosts-scroll" class="hosts-scroll">
      <section id="hosts" class="hosts" aria-label="Hosts"></section>
    </div>
    <div id="tag-remove-zone" class="tag-remove-zone" aria-hidden="true">
      ${iconMarkup('trash')}
      <span>Remove tag</span>
    </div>
  </aside>

  <main class="workspace">
    <section class="session-header">
      <h2 id="session-title">No session selected</h2>
      <div class="session-actions" aria-label="Connection actions">
        <div class="session-action-toolbar">
          <div class="session-action-group">
            <button id="edit-resource" class="secondary icon-action" type="button" title="Edit connection" aria-label="Edit connection" disabled>${iconMarkup('pencil')}</button>
            <button id="header-add-subsystem" class="secondary icon-action" type="button" title="Add subsystem" aria-label="Add subsystem" disabled>${iconMarkup('subtask')}</button>
          </div>
          <div class="session-action-group">
            <button id="open-keys-panel" class="secondary icon-action" type="button" title="SSH keys" aria-label="SSH keys" disabled>${iconMarkup('key')}</button>
            <button id="open-file-transfer" class="secondary icon-action" type="button" title="Files" aria-label="Files" disabled>${iconMarkup('folder')}</button>
            <button id="open-tunnel-panel" class="secondary icon-action" type="button" title="SSH tunnel" aria-label="SSH tunnel" disabled>${iconMarkup('arrows-transfer-up')}</button>
          </div>
          <div class="session-action-group">
            <button id="disconnect" class="secondary icon-action danger-icon" type="button" title="Disconnect" aria-label="Disconnect" disabled>${iconMarkup('plug-connected-x')}</button>
            <button id="delete-resource" class="secondary icon-action danger-icon" type="button" title="Delete connection" aria-label="Delete connection" disabled>${iconMarkup('trash')}</button>
          </div>
        </div>
        <button id="connect" class="connect-action" type="button" disabled><span class="connect-action-icon">${iconMarkup('player-play-filled')}</span><span class="connect-action-label">Connect</span></button>
      </div>
    </section>

    <section class="workbench">
      <section class="terminal-shell" aria-label="Terminal sessions">
        <div id="session-tabs" class="session-tabs" hidden></div>
        <div id="terminal-stack" class="terminal-stack">
          <div id="empty-terminal" class="empty-terminal">No active terminal session</div>
          <section id="webapp-workspace" class="webapp-workspace" hidden></section>
        </div>
      </section>
    </section>

    <footer class="workspace-footer">
      <p id="app-status" class="app-status" aria-live="polite"></p>
      <button id="message-log-button" class="status-log-button" type="button" title="Show message log">${iconMarkup('file-text')}<span>Log</span></button>
      <div class="terminal-font-controls" aria-label="Terminal font size">
        <button id="decrease-terminal-font" type="button" title="Decrease terminal font size" aria-label="Decrease terminal font size">${iconMarkup('minus')}</button>
        <span aria-hidden="true">A</span>
        <button id="increase-terminal-font" type="button" title="Increase terminal font size" aria-label="Increase terminal font size">${iconMarkup('plus')}</button>
      </div>
    </footer>
  </main>

  <div id="resource-panel-root"></div>

  <div id="webapp-panel-root"></div>

  <section id="webapp-auth-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-webapp-auth></div>
    <form id="webapp-auth-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow">Web App Tunnel</p>
          <h3>Connect Gateway</h3>
        </div>
        <button class="close-panel" type="button" data-close-webapp-auth title="Close">${ICON_CLOSE}</button>
      </header>
      <p id="webapp-auth-summary" class="parent-summary"></p>
      <p id="webapp-auth-status" class="inline-status" hidden></p>
      ${authFieldsMarkup('Web App tunnel')}
      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-webapp-auth>Cancel</button>
        <button type="submit">Open Web App</button>
      </footer>
    </form>
  </section>

  <div id="connect-panel-root"></div>

  <section id="tunnel-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-tunnel></div>
    <form id="tunnel-form" class="panel-card">
      <header class="panel-header">
        <div>
          <p class="eyebrow">SSH Tunnel</p>
          <h3 id="tunnel-title">SOCKS Proxy</h3>
        </div>
        <button class="close-panel" type="button" data-close-tunnel title="Close">${ICON_CLOSE}</button>
      </header>

      <p id="tunnel-summary" class="parent-summary"></p>
      <p id="tunnel-status" class="tunnel-status" hidden></p>

      <label>
        <span>Mode</span>
        <select name="type">
          <option value="socks">SOCKS proxy (-D)</option>
          <option value="local">Local forward (-L)</option>
          <option value="remote">Remote forward (-R)</option>
        </select>
      </label>
      <div class="form-grid">
        <label>
          <span id="tunnel-bind-label">Bind</span>
          <input name="localHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" value="127.0.0.1" />
        </label>
        <label>
          <span id="tunnel-port-label">Port</span>
          <input name="localPort" type="number" min="1" max="65535" value="1080" required />
        </label>
      </div>
      <div id="tunnel-target-fields" class="form-grid" hidden>
        <label>
          <span id="tunnel-target-host-label">Target Host</span>
          <input name="remoteHost" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" value="127.0.0.1" />
        </label>
        <label>
          <span id="tunnel-target-port-label">Target Port</span>
          <input name="remotePort" type="number" min="1" max="65535" value="80" />
        </label>
      </div>
      ${authFieldsMarkup('tunnel')}

      <footer class="panel-actions">
        <button class="secondary" type="button" data-close-tunnel>Close</button>
        <button id="stop-tunnel" class="secondary" type="button" disabled>Stop</button>
        <button id="start-tunnel" type="submit">Start Tunnel</button>
      </footer>
    </form>
  </section>

  <div id="settings-root"></div>
  <div id="app-overlays-root"></div>

  <section id="file-transfer-modal" class="file-transfer-modal" hidden>
    <div class="file-transfer-scrim"></div>
    <section class="file-transfer-card">
      <header class="file-transfer-header">
        <div>
          <p class="eyebrow">File Transfer</p>
          <h3 id="file-transfer-title">Files</h3>
        </div>
        <div class="file-transfer-header-actions">
          <button id="file-transfer-background" class="secondary" type="button" title="Send transfer window to background">Background</button>
          <button class="close-panel" type="button" data-close-file-transfer title="Close">${ICON_CLOSE}</button>
        </div>
      </header>
      <div id="file-transfer-root" class="file-transfer-root"></div>
      <div id="file-transfer-resize-handle" class="file-transfer-resize-handle" title="Resize" aria-hidden="true"></div>
    </section>
  </section>

  <section id="keys-panel" class="slide-panel" hidden>
    <div class="panel-scrim" data-close-keys></div>
    <section class="panel-card keys-panel-card" role="dialog" aria-modal="true" aria-labelledby="keys-panel-title">
      <header class="panel-header keys-panel-header">
        <div class="keys-panel-heading">
          <span class="keys-panel-heading-icon">${iconMarkup('key')}</span>
          <div>
            <h3 id="keys-panel-title">SSH Keys</h3>
            <p>Keys available to Bashes</p>
          </div>
        </div>
        <button class="close-panel" type="button" data-close-keys title="Close" aria-label="Close SSH keys">${ICON_CLOSE}</button>
      </header>

      <section id="key-main-view" class="keys-panel-view">
        <div class="keys-toolbar">
          <label class="keys-search-control">
            ${iconMarkup('search')}
            <input id="key-search" type="search" autocomplete="off" placeholder="Search keys" aria-label="Search SSH keys" />
          </label>
          <button id="open-key-add" class="secondary keys-toolbar-action" type="button">${iconMarkup('plus')}<span>Add key</span></button>
          <button id="open-key-sources" class="secondary keys-toolbar-action" type="button">${iconMarkup('folder-open')}<span>Sources</span></button>
        </div>
        <p class="inline-status key-action-status" id="key-action-status" hidden></p>
        <select id="key-select" class="key-native-select" aria-hidden="true" tabindex="-1"></select>
        <div id="key-list" class="key-list" aria-label="Available SSH keys"></div>

        <section id="key-details" class="key-details" hidden>
          <header class="key-details-header">
            <div>
              <strong id="key-details-name"></strong>
              <span id="key-details-source"></span>
            </div>
            <div class="key-detail-actions">
              <button id="copy-public-key" class="secondary icon-action" type="button" title="Copy public key" aria-label="Copy public key">${iconMarkup('copy')}</button>
              <button id="toggle-private-key" class="secondary icon-action" type="button" title="Show private key" aria-label="Show private key" aria-expanded="false">${iconMarkup('eye')}</button>
              <button id="delete-managed-key" class="secondary icon-action danger-icon" type="button" title="Delete managed key" aria-label="Delete managed key">${iconMarkup('trash')}</button>
            </div>
          </header>
          <p id="key-details-path" class="key-details-path"></p>
          <label>
            <span>Public Key</span>
            <textarea id="public-key" rows="5" readonly></textarea>
          </label>
          <textarea id="private-key" rows="7" readonly hidden aria-label="Private key"></textarea>
          <p class="inline-status" id="key-view-status" hidden></p>

          <form id="key-install-form" class="key-install-form">
            <p class="key-install-summary" id="key-install-summary">Select a host or subsystem to install the key.</p>
            <p class="inline-status" id="key-install-status" hidden></p>
            <label>
              <span>Remote Password</span>
              <input name="password" type="password" autocomplete="current-password" />
            </label>
            <label class="checkbox-row">
              <input name="trustHostKey" type="checkbox" />
              <span>Skip host key verification for this install (insecure)</span>
            </label>
            <button type="submit">Install on selected host</button>
          </form>
        </section>
        <p id="key-empty-state" class="keys-empty-state" hidden>No SSH keys found.</p>
      </section>

      <section id="key-add-view" class="keys-panel-view keys-subview" hidden>
        <header class="keys-subview-header">
          <button class="secondary keys-back-action" type="button" data-key-back>${iconMarkup('arrow-left')}<span>Keys</span></button>
          <div>
            <h4>Add key</h4>
            <p>Create a key or import an existing pair.</p>
          </div>
        </header>
        <div class="keys-mode-switch" role="tablist" aria-label="Add key method">
          <button id="show-key-generate" type="button" role="tab" aria-selected="true">Generate</button>
          <button id="show-key-import" class="secondary" type="button" role="tab" aria-selected="false">Import</button>
        </div>
        <form id="key-generate-form" class="key-add-form">
          <header>
            <strong>Generate an Ed25519 key</strong>
            <span>The key pair is stored in Bashes' private data directory.</span>
          </header>
          <label>
            <span>Key Name <small>(optional)</small></span>
            <input name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Generated automatically" />
          </label>
          <button type="submit">Generate key</button>
        </form>

        <form id="key-import-form" class="key-add-form" hidden>
          <header>
            <strong>Import a key pair</strong>
            <span>Paste matching public and private SSH keys.</span>
          </header>
          <label>
            <span>Key Name <small>(optional)</small></span>
            <input name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Generated automatically" />
          </label>
          <label>
            <span>Private Key</span>
            <textarea class="key-material-input" name="privateKey" rows="7" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required></textarea>
          </label>
          <label>
            <span>Public Key</span>
            <textarea class="key-material-input" name="publicKey" rows="4" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required></textarea>
          </label>
          <p class="inline-status" id="key-import-status" hidden></p>
          <button type="submit">Import key pair</button>
        </form>
      </section>

      <section id="key-sources-view" class="keys-panel-view keys-subview" hidden>
        <header class="keys-subview-header">
          <button class="secondary keys-back-action" type="button" data-key-back>${iconMarkup('arrow-left')}<span>Keys</span></button>
          <div>
            <h4>Key sources</h4>
            <p>Directories scanned for SSH private keys.</p>
          </div>
        </header>
        <div id="key-source-list" class="key-source-list"></div>
        <form id="key-directory-form" class="key-source-form">
          <label>
            <span>Custom Keys Directory</span>
            <input name="directory" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Optional folder to scan" />
          </label>
          <div class="key-source-actions">
            <button id="browse-key-directory" class="secondary" type="button">Browse</button>
            <button id="clear-key-directory" class="secondary" type="button">Clear</button>
            <button type="submit">Save</button>
          </div>
        </form>
        <p class="inline-status" id="key-directory-status" hidden></p>
        <button id="rescan-key-sources" class="secondary keys-rescan-action" type="button">${iconMarkup('refresh')}<span>Rescan sources</span></button>
      </section>
    </section>
  </section>

  <div id="tag-panel-root"></div>

  <div id="app-tooltip" class="app-tooltip" role="tooltip" hidden></div>
  <div id="edit-context-menu" class="edit-context-menu" hidden>
    <button type="button" data-edit-command="cut">Cut</button>
    <button type="button" data-edit-command="copy">Copy</button>
    <button type="button" data-edit-command="paste">Paste</button>
  </div>
`;

const settingsPanel = mountSettingsPanel(document.querySelector('#settings-root'), {
  onpreview: (values) => previewTerminalAppearance(values),
  onsave: (values) => submitSettings(values),
  onrequestclose: () => closeSettingsPanel(),
});
const appOverlays = mountAppOverlays(document.querySelector('#app-overlays-root'), {
  onclosed: () => restoreTerminalFocusAfterOverlay(),
});
const tagPanel = mountTagPanel(document.querySelector('#tag-panel-root'), {
  oncreate: (name) => createManagedTag(name),
  onrename: (tag, name) => renameManagedTag(tag, name),
  ondelete: (tag) => deleteManagedTag(tag),
  ontoggle: (tag) => toggleTagPanelTag(tag),
  onclear: () => clearTagPanelSelection(),
  onrequestclose: () => closeTagPanel(),
});
const webAppWorkspace = mountWebAppWorkspace(document.querySelector('#webapp-workspace'), {
  onopen: () => openSelectedWebApp(),
  ontoggletunnel: () => toggleSelectedWebAppTunnel(),
});
const tagFilters = mountTagFilters(document.querySelector('#tag-filters-root'), {
  ontoggle: (tag) => {
    if (Date.now() >= suppressTagFilterClickUntil) toggleTagFilter(tag);
  },
  onclear: () => clearTagFilters(),
  onmode: (mode) => setTagFilterMode(mode),
  onrender: () => initializeTagSorting(),
});
const resourcePanel = mountResourcePanel(document.querySelector('#resource-panel-root'), {
  onsubmit: (submission) => submitResource(submission),
  oncreatetag: (name) => createResourceTag(name),
  onrequestclose: () => closeResourcePanel(),
});
const webAppPanel = mountWebAppPanel(document.querySelector('#webapp-panel-root'), {
  onsubmit: (submission) => submitWebApp(submission),
  oncreatetag: (name) => createResourceTag(name),
  onrequestclose: () => closeWebAppPanel(),
});
const connectPanel = mountConnectPanel(document.querySelector('#connect-panel-root'), {
  onsubmit: (submission) => submitConnect(submission),
  onrequestclose: () => closeConnectPanel(),
});

const stack = document.querySelector('#terminal-stack');
const hostsScroll = document.querySelector('#hosts-scroll');

window.addEventListener('resize', () => {
  scheduleTerminalFit();
  clampFileTransferModalSize();
  syncSidebarScrollbarInset();
});

if (globalThis.ResizeObserver) {
  const terminalResizeObserver = new ResizeObserver(() => scheduleTerminalFit());
  terminalResizeObserver.observe(stack);
  const hostsResizeObserver = new ResizeObserver(() => syncSidebarScrollbarInset());
  hostsResizeObserver.observe(hostsScroll);
}

registerSSHEvents();
registerAppEvents();

const searchInput = document.querySelector('#search');
document.querySelector('#toggle-sidebar').addEventListener('click', () => toggleSidebar());
searchInput.addEventListener('input', () => scheduleHostRender());
['beforeinput', 'keydown', 'keyup'].forEach((eventName) => {
  searchInput.addEventListener(eventName, (event) => event.stopPropagation());
});
document.querySelector('#open-add-menu').addEventListener('click', () => toggleAddResourceMenu());
document.querySelectorAll('[data-add-resource]').forEach((button) => {
  button.addEventListener('click', () => {
    closeAddResourceMenu();
    if (button.dataset.addResource === 'webapp') openWebAppPanel();
    else openResourcePanel('host');
  });
});
document.querySelector('#open-tag-manager').addEventListener('click', () => openTagPanel());
document.querySelector('#open-keys-panel').addEventListener('click', () => openKeysPanel({ allowInstall: true }));
document.querySelector('#open-tunnel-panel').addEventListener('click', () => openTunnelPanel());
document.querySelector('#open-file-transfer').addEventListener('click', () => openFileTransferModal());
document.querySelector('#edit-resource').addEventListener('click', () => {
  const selected = findResource(state.selectedId);
  if (selected?.type === 'webapp') openWebAppPanel(selected.resource);
  else openEditPanel();
});
document.querySelector('#header-add-subsystem').addEventListener('click', () => {
  const selected = findResource(state.selectedId);
  if (selected && selected.type !== 'webapp' && !isLocalResource(selected.resource)) openResourcePanel('subsystem', selected.resource.id);
});
document.querySelector('#connect').addEventListener('click', () => {
  const selected = findResource(state.selectedId);
  if (selected?.type === 'webapp') openSelectedWebApp();
  else openConnectPanel();
});
document.querySelector('#disconnect').addEventListener('click', () => disconnectActiveSession());
document.querySelector('#delete-resource').addEventListener('click', () => deleteSelectedResource());
document.querySelector('#message-log-button').addEventListener('click', () => showMessageLog());
document.querySelector('#decrease-terminal-font').addEventListener('click', () => adjustTerminalFontSize(-1));
document.querySelector('#increase-terminal-font').addEventListener('click', () => adjustTerminalFontSize(1));
document.querySelector('#webapp-auth-form').addEventListener('submit', (event) => submitWebAppAuth(event));
registerAuthChoiceSync(document.querySelector('#webapp-auth-form'));
document.querySelector('#tunnel-form').addEventListener('submit', (event) => submitTunnel(event));
document.querySelector('#tunnel-form').addEventListener('input', () => updateTunnelSummary());
document.querySelector('#tunnel-form').elements.type.addEventListener('change', () => updateTunnelMode());
registerAuthChoiceSync(document.querySelector('#tunnel-form'));
document.querySelector('#stop-tunnel').addEventListener('click', () => stopSelectedTunnel());
document.querySelector('#key-generate-form').addEventListener('submit', (event) => submitGenerateKey(event));
document.querySelector('#key-import-form').addEventListener('submit', (event) => submitImportKeyPair(event));
document.querySelector('#key-directory-form').addEventListener('submit', (event) => submitKeyDirectory(event));
document.querySelector('#key-install-form').addEventListener('submit', (event) => submitInstallKey(event));
document.querySelector('#key-select').addEventListener('change', () => renderSelectedKey());
document.querySelector('#toggle-private-key').addEventListener('click', () => toggleSelectedPrivateKey());
document.querySelector('#copy-public-key').addEventListener('click', () => copySelectedPublicKey());
document.querySelector('#delete-managed-key').addEventListener('click', () => deleteSelectedManagedKey());
document.querySelector('#key-search').addEventListener('input', () => renderKeyBrowser());
document.querySelector('#open-key-add').addEventListener('click', () => showKeyPanelView('add'));
document.querySelector('#open-key-sources').addEventListener('click', () => showKeyPanelView('sources'));
document.querySelectorAll('[data-key-back]').forEach((element) => {
  element.addEventListener('click', () => showKeyPanelView('main'));
});
document.querySelector('#show-key-generate').addEventListener('click', () => showKeyAddMode('generate'));
document.querySelector('#show-key-import').addEventListener('click', () => showKeyAddMode('import'));
document.querySelector('#browse-key-directory').addEventListener('click', () => browseKeyDirectory());
document.querySelector('#clear-key-directory').addEventListener('click', () => {
  document.querySelector('#key-directory-form').elements.directory.value = '';
});
document.querySelector('#rescan-key-sources').addEventListener('click', () => rescanKeySources());
document.querySelectorAll('[data-close-webapp-auth]').forEach((element) => {
  element.addEventListener('click', () => closeWebAppAuthPanel());
});
document.querySelectorAll('[data-close-tunnel]').forEach((element) => {
  element.addEventListener('click', () => closeTunnelPanel());
});
document.querySelectorAll('[data-close-keys]').forEach((element) => {
  element.addEventListener('click', () => closeKeysPanel());
});
document.querySelectorAll('[data-close-file-transfer]').forEach((element) => {
  element.addEventListener('click', () => closeFileTransferModal());
});
document.querySelector('#file-transfer-background').addEventListener('click', () => backgroundFileTransferModal());
registerFileTransferModalResize();
registerFileTransferModalDrag();
document.querySelectorAll('.field-help-trigger').forEach((button) => {
  button.addEventListener('click', (event) => toggleFieldHelp(event));
  button.addEventListener('keydown', (event) => {
    if (event.key === 'Enter' || event.key === ' ') {
      toggleFieldHelp(event);
    }
  });
});
document.querySelector('#edit-context-menu').addEventListener('click', (event) => runEditContextCommand(event));
document.addEventListener('contextmenu', (event) => openEditContextMenu(event));
document.addEventListener('pointerdown', (event) => {
  if (!event.target.closest?.('#edit-context-menu')) hideEditContextMenu();
  if (!event.target.closest?.('.field-label-with-help')) closeFieldHelpPopovers();
  schedulePointerDragOutline(event);
  if (!event.target.closest?.('#open-add-menu, #add-resource-menu')) closeAddResourceMenu();
});
window.addEventListener('pointerup', () => clearAllDragOutlines());
window.addEventListener('pointercancel', () => clearAllDragOutlines());
window.addEventListener('blur', () => clearAllDragOutlines());
window.addEventListener('keydown', (event) => {
  if (event.key === 'Escape' && appOverlays.isConfirmOpen()) {
    resolveConfirmModal(false);
  } else if (event.key === 'Escape' && appOverlays.isAppOpen()) {
    closeAppModal();
  } else if (event.key === 'Escape' && !document.querySelector('#settings-panel').hidden) {
    closeSettingsPanel();
  } else if (event.key === 'Escape' && tagPanel.isOpen()) {
    closeTagPanel();
  } else if (event.key === 'Escape') {
    closeFieldHelpPopovers();
    hideEditContextMenu();
  }
});

await loadCapabilities();
await loadHosts();
await loadKeySettings();

function toggleFieldHelp(event) {
  event.preventDefault();
  event.stopPropagation();

  const button = event.currentTarget;
  const willOpen = button.getAttribute('aria-expanded') !== 'true';
  closeFieldHelpPopovers(button);
  button.setAttribute('aria-expanded', willOpen ? 'true' : 'false');
}

function closeFieldHelpPopovers(except = null) {
  document.querySelectorAll('.field-help-trigger[aria-expanded="true"]').forEach((button) => {
    if (button !== except) {
      button.setAttribute('aria-expanded', 'false');
    }
  });
}
await loadKeys();
await loadTunnels();
applySidebarState();
schedulePeriodicUpdateCheck();
window.setTimeout(() => {
  openInitialLocalSession().catch((error) => {
    writeNotice(`Could not initialize local terminal UI: ${error?.message ?? error}`, 'error');
  });
}, 0);

async function loadCapabilities() {
  state.localShellSupported = await apiSupportsLocalShell();
}

async function loadHosts() {
  return await withBusy(async () => {
    await refreshHosts();
  });
}

async function refreshHosts() {
  const [workspace, tags, runtimes] = await Promise.all([apiListWorkspace(), apiListTags(), apiListWebAppRuntimes()]);
  state.hosts = workspace.hosts ?? [];
  state.webApps = workspace.webApps ?? [];
  state.topLevelOrder = workspace.topLevelOrder ?? state.hosts.map((host) => host.id);
  state.tags = tags;
  state.webAppRuntimes = new Map((runtimes ?? []).map((runtime) => [runtime.webAppId, runtime]));
  const activeSession = state.sessions.get(state.activeSessionId);
  if (activeSession && findResource(activeSession.resourceId)) {
    state.selectedId = activeSession.resourceId;
  } else if (state.selectedId && !findResource(state.selectedId)) {
    state.selectedId = null;
  }
  if (!state.selectedId && !activeSession && state.localShellSupported) {
    state.selectedId = LOCAL_RESOURCE_ID;
  } else if (!state.selectedId && !activeSession && state.topLevelOrder.length > 0) {
    state.selectedId = state.topLevelOrder[0];
  }
  renderHosts(searchInput.value);
  renderSelection();
}

async function loadKeys() {
  const selectedValue = document.querySelector('#key-select')?.value || '';
  const inventory = await apiGetSSHKeyInventory();
  state.keys = inventory.keys ?? [];
  state.keySources = inventory.sources ?? [];
  renderKeyOptions();
  const select = document.querySelector('#key-select');
  if (select && selectedValue && [...select.options].some((option) => option.value === selectedValue)) {
    select.value = selectedValue;
  }
  await renderSelectedKey();
  renderKeySources();
}

async function loadKeySettings() {
  state.keySettings = await apiGetSSHKeySettings();
  renderKeySettings();
}

async function loadWebAppBrowserSettings() {
  const [browsers, settings] = await Promise.all([apiListWebAppBrowsers(), apiGetWebAppBrowserSettings()]);
  state.webAppBrowsers = browsers ?? [];
  state.webAppBrowserSettings = settings ?? { preferredBrowserId: '', customBrowserPath: '' };
}

async function loadTunnels() {
  const tunnels = await apiListSSHTunnels();
  state.tunnels = new Map(tunnels.map((tunnel) => [tunnel.tunnelId, tunnel]));
  renderHosts(searchInput.value);
}

async function submitResource({ mode, parentId, editId, input }) {
  return await withBusy(async () => {
    let createdResourceID = '';
    if (mode === 'edit') {
      const resource = findResource(editId)?.resource;
      await apiUpdateResource(editId, input);
      for (const session of sessionsForResource(editId)) {
        session.title = input.hostname;
        session.target = `${input.user}@${input.ip || input.hostname}:${input.port}`;
      }
      writeNotice(`Updated ${resource?.hostname ?? 'resource'}.`);
    } else if (mode === 'subsystem') {
      const subsystem = await apiAddSubsystem(parentId, input);
      createdResourceID = subsystem.id;
      writeNotice(`Added ${subsystem.type} ${subsystem.hostname}.`);
    } else {
      const host = await apiAddHost(input);
      createdResourceID = host.id;
      writeNotice(`Added host ${host.hostname}.`);
    }
    closeResourcePanel();
    await refreshHosts();
    const createdResource = createdResourceID ? findResource(createdResourceID)?.resource : null;
    if (createdResource) {
      selectResource(createdResource);
      revealSidebarResource(hostsScroll, createdResource.id);
    }
    renderTabs();
  });
}

async function createResourceTag(name) {
  const created = await withBusy(() => apiCreateTag(name));
  if (!created) return '';
  state.tags = [...state.tags, created];
  return created;
}

async function submitWebApp({ editId, input }) {
  await withBusy(async () => {
    let createdWebAppID = '';
    if (editId) {
      await apiUpdateWebApp(editId, input);
      const tab = webAppTabForResource(editId);
      if (tab) {
        tab.title = input.name;
        tab.target = input.connectionMode === 'tunnel' ? `${input.url} via SSH` : input.url;
      }
      writeNotice(`Updated ${input.name}.`);
    } else {
      const webApp = await apiAddWebApp(input);
      createdWebAppID = webApp.id;
      writeNotice(`Added Web App ${input.name}.`);
    }
    closeWebAppPanel();
    await refreshHosts();
    const createdWebApp = createdWebAppID ? findResource(createdWebAppID)?.resource : null;
    if (createdWebApp) {
      selectResource(createdWebApp);
      revealSidebarResource(hostsScroll, createdWebApp.id);
    }
    renderTabs();
  });
}

function toggleAddResourceMenu() {
  const menu = document.querySelector('#add-resource-menu');
  const button = document.querySelector('#open-add-menu');
  const open = menu.hidden;
  menu.hidden = !open;
  button.setAttribute('aria-expanded', String(open));
}

function closeAddResourceMenu() {
  document.querySelector('#add-resource-menu').hidden = true;
  document.querySelector('#open-add-menu').setAttribute('aria-expanded', 'false');
}

async function openWebAppPanel(webApp = null) {
  try {
    await loadWebAppBrowserSettings();
  } catch (error) {
    writeNotice(`Could not detect Web App browsers: ${errorDetail(error)}`, 'error');
    return;
  }
  if (!webApp && state.webAppBrowsers.length === 0 && !state.webAppBrowserSettings.customBrowserPath) {
    showAppModal({
      kicker: 'Web App Browser',
      title: 'A supported browser is required',
      message: 'Install Firefox, Chrome, Chromium, or Microsoft Edge before adding a Web App.',
      primaryLabel: 'Firefox',
      primaryAction: () => openExternalURL('https://www.mozilla.org/firefox/new/'),
      secondaryLabel: 'Chrome',
      secondaryAction: () => openExternalURL('https://www.google.com/chrome/'),
    });
    return;
  }

  webAppPanel.open(webAppPanelData({
    webApp,
    hosts: state.hosts,
    catalog: resourceTagCatalog(),
  }));
}

function closeWebAppPanel() {
  webAppPanel.close();
  restoreTerminalFocusAfterOverlay();
}

async function submitConnect({ resourceId, input }) {
  const selected = findResource(resourceId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) {
    await quickConnect(selected);
    closeConnectPanel();
    return;
  }
  await withBusy(async () => {
    const target = `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
    connectPanel.setStatus('', '');
    writeNotice(`Connecting to ${target} ...`);
    try {
      const sessionID = await startSSHSessionWithHostKeyPrompt({
        ...input,
        resourceId: selected.id,
      }, selected, 'Trust and connect');
      await attachStartedSession(sessionID, selected);
      closeConnectPanel();
      await refreshHosts();
      resizeActiveSession();
    } catch (error) {
      const message = connectErrorMessage(error, selected);
      connectPanel.setStatus(message, 'error');
      writeNotice(message);
      if (isAuthError(error)) connectPanel.focusAuth();
    }
  });
}

async function submitTunnel(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const active = tunnelForResource(selected.id);
  if (active) {
    writeNotice(`Tunnel already active on ${active.localAddress}.`);
    renderTunnelStatus();
    return;
  }

  const form = event.currentTarget;
  await withBusy(async () => {
    const auth = authInputFromForm(form);
    const tunnelInput = {
      resourceId: selected.id,
      type: form.elements.type.value,
      localHost: form.elements.localHost.value.trim(),
      localPort: Number.parseInt(form.elements.localPort.value, 10),
      remoteHost: form.elements.remoteHost.value.trim(),
      remotePort: Number.parseInt(form.elements.remotePort.value, 10),
      ...auth,
      trustHostKey: form.elements.trustHostKey.checked,
    };
    try {
      const tunnel = await startSSHTunnelWithHostKeyPrompt(tunnelInput, selected);
      state.tunnels.set(tunnel.tunnelId, tunnel);
      saveTunnelPreference(localStorage, selected.id, tunnelInput);
      form.dataset.hadSavedPassword = String(auth.savePassword === true);
      showSavedPassword(form.elements.password, auth.savePassword === true);
      await refreshHosts();
      renderHosts(searchInput.value);
      renderTunnelStatus();
      writeNotice(`${tunnelLabel(tunnel.type)} active on ${tunnel.localAddress}.`);
    } catch (error) {
      writeNotice(connectErrorMessage(error, selected));
    }
  });
}

async function quickConnect(resource, { failureSessionID = '' } = {}) {
  return await withConnectionBusy(resource.id, async () => {
    try {
      if (isLocalResource(resource)) {
        writeNotice('Starting local shell ...');
        const sessionID = await apiStartLocalSession({
          cols: 120,
          rows: 32,
        });
        await attachStartedSession(sessionID, resource, 'local', failureSessionID);
        resizeActiveSession();
        return sessionID;
      }

      writeNotice(`Connecting to ${resource.user}@${resource.ip || resource.hostname}:${resource.port} ...`);
      const sessionID = await startSSHSessionWithHostKeyPrompt({
        resourceId: resource.id,
        ...authInputFromPreference(resource),
        trustHostKey: trustHostKeyFromPreference(resource),
        cols: 120,
        rows: 32,
      }, resource, 'Trust and connect');
      await attachStartedSession(sessionID, resource, 'ssh', failureSessionID);
      await refreshHosts();
      resizeActiveSession();
      return sessionID;
    } catch (error) {
      const message = connectErrorMessage(error, resource);
      writeNotice(message);
      if (isLocalResource(resource)) {
        const pending = pendingSessionForResource(resource.id);
        if (pending) removeSessionFromUI(pending.id);
        return null;
      }
      if (isAuthError(error)) {
        await openConnectPanel(message, 'error');
      } else {
        const pending = pendingSessionForResource(resource.id);
        const failedSession = pending ?? state.sessions.get(failureSessionID);
        if (failedSession) markSessionConnectionFailed(failedSession.id, resource, message);
      }
      return null;
    }
  });
}

async function openSelectedWebApp() {
  const selected = findResource(state.selectedId);
  if (selected?.type !== 'webapp') return;
  await launchWebApp(selected.resource);
}

async function toggleSelectedWebAppTunnel() {
  const selected = findResource(state.selectedId);
  if (selected?.type !== 'webapp') return;
  const runtime = state.webAppRuntimes.get(selected.resource.id);
  if (!runtime?.tunnel) return;

  await withBusy(async () => {
    const updated = runtime.tunnelActive
      ? await apiStopWebAppTunnel(selected.resource.id)
      : await apiStartWebAppTunnel(selected.resource.id);
    state.webAppRuntimes.set(selected.resource.id, updated);
    writeNotice(`${updated.tunnelActive ? 'Started' : 'Stopped'} Web App tunnel on 127.0.0.1:${updated.socksPort}.`);
    renderHosts(searchInput.value);
    renderSelection();
  });
}

async function launchWebApp(webApp, auth = null, { promptAuth = true } = {}) {
  const gateway = webApp.connectionMode === 'tunnel'
    ? findResource(webApp.gatewayResourceId)?.resource
    : null;
  const input = {
    webAppId: webApp.id,
    ...(gateway ? auth ?? authInputFromPreference(gateway) : {}),
    trustHostKey: gateway ? auth?.trustHostKey ?? trustHostKeyFromPreference(gateway) : false,
  };
  return await withBusy(async () => {
    try {
      const runtime = await launchWebAppWithHostKeyPrompt(input, gateway);
      state.webAppRuntimes.set(webApp.id, runtime);
      promoteWebAppTab(webApp);
      writeNotice(`Opened ${webApp.name} in ${runtime.browser}.`);
      renderHosts(searchInput.value);
      renderTabs();
      renderSelection();
      return true;
    } catch (error) {
      const detail = errorDetail(error);
      if (/supported browser|selected browser|browser.*not installed/i.test(detail)) {
        showAppModal({
          kicker: 'Web App Browser',
          title: 'A supported browser is required',
          message: detail,
          primaryLabel: 'Firefox',
          primaryAction: () => openExternalURL('https://www.mozilla.org/firefox/new/'),
          secondaryLabel: 'Chrome',
          secondaryAction: () => openExternalURL('https://www.google.com/chrome/'),
        });
        return false;
      }
      if (gateway && isAuthError(error)) {
        const message = connectErrorMessage(error, gateway);
        if (promptAuth) await openWebAppAuthPanel(webApp, gateway, message);
        else {
          setWebAppAuthStatus(message, 'error');
          focusConnectPasswordInput(document.querySelector('#webapp-auth-form'));
        }
        return false;
      }
      if (gateway && !promptAuth) {
        setWebAppAuthStatus(connectErrorMessage(error, gateway), 'error');
        return false;
      }
      writeNotice(gateway ? connectErrorMessage(error, gateway) : `Could not open ${webApp.name}: ${detail}`, 'error');
      return false;
    }
  });
}

async function launchWebAppWithHostKeyPrompt(input, gateway) {
  try {
    return await apiLaunchWebApp(input);
  } catch (error) {
    if (!gateway || input.acceptHostKey || input.replaceHostKey) throw error;
    const retryInput = await confirmHostKeyChange(error, gateway, 'Trust and open Web App');
    if (!retryInput) throw error;
    return await apiLaunchWebApp({ ...input, ...retryInput });
  }
}

async function openWebAppAuthPanel(webApp, gateway, message = '') {
  await loadKeys();
  state.webAppAuthTarget = { webApp, gateway };
  const panel = document.querySelector('#webapp-auth-panel');
  const form = document.querySelector('#webapp-auth-form');
  form.reset();
  renderKeyOptions(form.elements.keyName, true);
  applyConnectDefaults(form, gateway);
  await applySavedPasswordDefault(form, gateway);
  form.elements.trustHostKey.checked = trustHostKeyFromPreference(gateway);
  document.querySelector('#webapp-auth-summary').textContent =
    `${webApp.name} via ${gateway.user}@${gateway.ip || gateway.hostname}:${gateway.port}`;
  setWebAppAuthStatus(message, message ? 'error' : '');
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  focusConnectPasswordInput(form);
}

async function submitWebAppAuth(event) {
  event.preventDefault();
  const target = state.webAppAuthTarget;
  if (!target) return;
  const form = event.currentTarget;
  setWebAppAuthStatus('', '');
  try {
    const opened = await launchWebApp(target.webApp, {
      ...authInputFromForm(form),
      trustHostKey: form.elements.trustHostKey.checked,
    }, { promptAuth: false });
    if (opened) closeWebAppAuthPanel();
  } catch (error) {
    const message = connectErrorMessage(error, target.gateway);
    setWebAppAuthStatus(message, 'error');
    if (isAuthError(error)) focusConnectPasswordInput(form);
  }
}

function setWebAppAuthStatus(message, kind) {
  const status = document.querySelector('#webapp-auth-status');
  status.textContent = message;
  status.title = message;
  status.dataset.kind = kind;
  status.hidden = !message;
}

function closeWebAppAuthPanel() {
  const panel = document.querySelector('#webapp-auth-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  state.webAppAuthTarget = null;
  setWebAppAuthStatus('', '');
  restoreTerminalFocusAfterOverlay();
}

async function submitGenerateKey(event) {
  event.preventDefault();
  const form = event.currentTarget;

  await withBusy(async () => {
    const key = await apiGenerateSSHKey({ name: form.elements.name.value.trim() });
    form.reset();
    await loadKeys();
    document.querySelector('#key-select').value = keyChoiceValue({ ...key, source: 'bashes' });
    await renderSelectedKey();
    showKeyPanelView('main');
    showKeyActionStatus(`Generated ${key.name}. The new key is selected.`, 'success');
    highlightKeyRow(keyChoiceValue(key));
  });
}

async function submitImportKeyPair(event) {
  event.preventDefault();
  const form = event.currentTarget;
  setKeyImportStatus('Validating and importing key pair ...', 'pending');
  await withBusy(async () => {
    try {
      const key = await apiImportSSHKeyPair({
        name: form.elements.name.value.trim(),
        privateKey: form.elements.privateKey.value,
        publicKey: form.elements.publicKey.value,
      });
      form.reset();
      await loadKeys();
      document.querySelector('#key-select').value = keyChoiceValue(key);
      await renderSelectedKey();
      showKeyPanelView('main');
      showKeyActionStatus(`Imported ${key.name}. The new key is selected.`, 'success');
      highlightKeyRow(keyChoiceValue(key));
    } catch (error) {
      const message = `Could not import SSH key pair: ${error?.message ?? error}`;
      setKeyImportStatus(message, 'error');
      writeNotice(message);
    }
  });
}

async function submitKeyDirectory(event) {
  event.preventDefault();
  const form = event.currentTarget;
  const directory = form.elements.directory.value.trim();

  await withBusy(async () => {
    try {
      state.keySettings = await apiSaveSSHKeySettings({ customDirectory: directory });
      renderKeySettings();
      await loadKeys();
      const customCount = state.keys.filter((key) => keySource(key) === 'custom').length;
      if (state.keySettings.customDirectory && customCount === 0) {
        setKeyDirectoryStatus('Saved, but no SSH keys were found in that directory.', 'pending');
      } else if (state.keySettings.customDirectory) {
        setKeyDirectoryStatus(`Saved. Found ${customCount} custom key${customCount === 1 ? '' : 's'}.`, 'success');
      } else {
        setKeyDirectoryStatus('Custom keys directory cleared.', 'success');
      }
      renderKeySources();
    } catch (error) {
      const message = `Could not save custom keys directory: ${error?.message ?? error}`;
      setKeyDirectoryStatus(message, 'error');
      writeNotice(message);
      window.setTimeout(() => form.elements.directory.focus(), 0);
    }
  });
}

async function submitInstallKey(event) {
  event.preventDefault();
  const selected = findResource(state.selectedId)?.resource;
  const keyChoice = selectedKeyChoice(document.querySelector('#key-select'));
  if (!selected || isLocalResource(selected)) {
    setKeyInstallStatus('Select a host or subsystem before installing a key.', 'error');
    writeNotice('Select a host or subsystem before installing a key.');
    return;
  }
  if (!keyChoice) {
    setKeyInstallStatus('Select or generate an SSH key before installing it.', 'error');
    writeNotice('Select or generate an SSH key before installing it.');
    return;
  }

  const form = event.currentTarget;
  const keyLabel = keyChoiceLabel(keyChoice);
  setKeyInstallStatus(`Installing key ${keyLabel} on ${selected.hostname} ...`, 'pending');
  await withBusy(async () => {
    try {
      await installSSHKeyWithHostKeyPrompt({
        resourceId: selected.id,
        ...installInputFromKeyChoice(keyChoice),
        password: form.elements.password.value,
        trustHostKey: form.elements.trustHostKey.checked,
      }, selected);
      form.reset();
      form.elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
      await refreshHosts();
      setKeyInstallStatus(`Installed key ${keyLabel} on ${selected.hostname}.`, 'success');
      writeNotice(`Installed key ${keyLabel} on ${selected.hostname}.`);
    } catch (error) {
      const message = keyInstallErrorMessage(error, selected);
      setKeyInstallStatus(message, 'error');
      writeNotice(message);
      window.setTimeout(() => form.elements.password.focus(), 0);
    }
  });
}

async function deleteSelectedResource() {
  const selected = findResource(state.selectedId);
  if (!selected) return;
  if (isLocalResource(selected.resource)) return;
  if (!(await confirmDeleteResource(selected))) return;

  await withBusy(async () => {
    if (selected.type === 'webapp') {
      const tab = webAppTabForResource(selected.resource.id);
      if (tab) removeSessionFromUI(tab.id);
      await apiDeleteWebApp(selected.resource.id);
      writeNotice(`Deleted ${selected.resource.name}.`);
      state.selectedId = state.activeSessionId
        ? state.sessions.get(state.activeSessionId)?.resourceId ?? null
        : null;
      await refreshHosts();
      return;
    }
    const resourceIDs = resourceIDsForSelection(selected);
    for (const resourceID of resourceIDs) {
      for (const session of sessionsForResource(resourceID)) {
        await stopSession(session.id);
      }
      for (const tunnel of tunnelsForResource(resourceID)) {
        await stopTunnel(tunnel.tunnelId);
      }
    }
    await apiDeleteResource(selected.resource.id);
    writeNotice(`Deleted ${selected.resource.hostname}.`);
    if (!state.activeSessionId) state.selectedId = selected.parent?.id ?? null;
    await refreshHosts();
  });
}

function confirmDeleteResource(selected) {
  if (selected.type === 'webapp') {
    return openConfirmModal({
      kicker: 'Delete',
      title: `Delete ${selected.resource.name}?`,
      message: 'This action cannot be undone.',
      confirmLabel: 'Delete',
    });
  }
  const subsystemCount = nestedResourceIDs(selected.resource.subsystems ?? []).length;
  const extra = subsystemCount > 0
    ? `\n\nThis will also delete ${subsystemCount} subsystem${subsystemCount === 1 ? '' : 's'}.`
    : '';
  return openConfirmModal({
    kicker: 'Delete',
    title: `Delete ${selected.resource.hostname}?`,
    message: `This action cannot be undone.${extra}`,
    confirmLabel: 'Delete',
  });
}

function openConfirmModal({ kicker = 'Confirm', title, message, confirmLabel = 'Confirm' }) {
  return appOverlays.confirm({ kicker, title, message, confirmLabel });
}

function resolveConfirmModal(confirmed) {
  appOverlays.resolveConfirm(confirmed);
}

function showAppModal(options) {
  appOverlays.show(options);
}

function closeAppModal() {
  appOverlays.closeApp();
}

async function stopSelectedTunnel() {
  const selected = findResource(state.selectedId)?.resource;
  const tunnel = selected ? tunnelForResource(selected.id) : null;
  if (!tunnel) return;
  await withBusy(async () => {
    await stopTunnel(tunnel.tunnelId);
    renderTunnelStatus();
    writeNotice(`Stopped tunnel on ${tunnel.localAddress}.`);
  });
}

async function stopTunnel(tunnelID) {
  await apiStopSSHTunnel(tunnelID);
  state.tunnels.delete(tunnelID);
  renderHosts(searchInput.value);
}

async function disconnectActiveSession() {
  if (!state.activeSessionId) return;
  await stopSession(state.activeSessionId);
}

async function stopSession(sessionID) {
  const session = state.sessions.get(sessionID);
  removeSessionFromUI(sessionID);
  if (session?.kind === 'webapp') return;
  if (session && !session.pending && !session.closed) {
    await apiStopSSHSession(sessionID);
    writeNotice(`${terminalKindLabel(session.kind)} session disconnected: ${session.title}.`);
  }
}

function markSessionClosed(sessionID, message = '') {
  const session = state.sessions.get(sessionID);
  if (!session) {
    state.pendingSSHOutput.delete(sessionID);
    return;
  }

  flushPendingSSHOutput(sessionID);
  session.closed = true;
  session.pending = false;
  if (session.terminal) {
    session.terminal.options.disableStdin = true;
    const reason = String(message || `${terminalKindLabel(session.kind)} session closed`).trim();
    writeClosedSessionNotice(session.terminal, `[Session closed: ${terminalSafeText(reason)}]`);
  }
  if (session.element) session.element.classList.add('closed-pane');
  if (state.lastSessionByResource.get(session.resourceId) === sessionID) {
    state.lastSessionByResource.delete(session.resourceId);
  }
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
}

function markSessionConnectionFailed(sessionID, resource, message) {
  const session = state.sessions.get(sessionID);
  if (!session) return;

  if (!session.terminal) {
    session.element.textContent = '';
    const terminal = new Terminal(terminalOptions(true));
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(session.element);
    installTerminalKeyRepeatFallback(terminal, sessionID);
    session.terminal = terminal;
    session.fitAddon = fitAddon;
  }

  session.kind = 'ssh';
  session.closed = true;
  session.pending = false;
  session.title = resource.hostname;
  session.target = resourceTarget(resource);
  session.terminal.options.disableStdin = true;
  session.element.classList.remove('pending-pane');
  session.element.classList.add('closed-pane');
  writeClosedSessionNotice(session.terminal, `\x1b[31m[Connection failed]\x1b[0m ${terminalSafeText(message)}`);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function writeClosedSessionNotice(terminal, message) {
  terminal.write(`\r\n${message}\r\n\x1b[90m[Ctrl+D close tab | Ctrl+R reconnect]\x1b[0m\r\n`);
  terminal.scrollToBottom();
}

function terminalSafeText(value) {
  return String(value ?? '').replace(/[\x00-\x08\x0b-\x1f\x7f]/g, '').replace(/\s+/g, ' ').trim();
}

function removeSessionFromUI(sessionID) {
  const session = state.sessions.get(sessionID);
  state.sessions.delete(sessionID);
  state.pendingSSHOutput.delete(sessionID);
  forgetSessionFocus(sessionID);
  if (session && state.lastSessionByResource.get(session.resourceId) === sessionID) {
    state.lastSessionByResource.delete(session.resourceId);
  }
  if (session?.terminal) session.terminal.dispose();
  if (session?.element) session.element.remove();
  if (state.activeSessionId === sessionID) {
    setActiveSession(lastFocusedSessionID());
  }
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function resourceTagCatalog() {
  return mergeTagCatalog(state.tags, [...state.hosts, ...state.webApps]);
}

function scheduleHostRender() {
  if (searchRenderFrame) cancelAnimationFrame(searchRenderFrame);
  searchRenderFrame = requestAnimationFrame(() => {
    searchRenderFrame = 0;
    renderHosts(searchInput.value);
    renderSelection();
  });
}

function renderHosts(filter = '') {
  const container = document.querySelector('#hosts');
  closeTagPopover();
  clearDragOutline('hosts');
  hostSortable?.destroy();
  hostSortable = null;
  destroyTagSorting();
  const query = filter.trim().toLowerCase();
  renderTagFilters();
  const hasFilters = query !== '' || state.activeTagFilters.length > 0;
  const canReorder = !hasFilters;
  const elements = [];

  if (state.localShellSupported) {
    elements.push(resourceRow(localResource(), 'local', 0).element);
  }
  const filteredHosts = filterResourceTree(state.hosts, {
    query,
    activeTags: state.activeTagFilters,
    mode: state.tagFilterMode,
  });
  const filteredWebApps = state.webApps.filter((webApp) => webAppMatchesFilters(webApp, {
    query,
    activeTags: state.activeTagFilters,
    mode: state.tagFilterMode,
  }));
  const items = new Map();
  for (const node of filteredHosts) {
    const block = document.createElement('div');
    block.className = 'host-block';
    block.dataset.topLevelId = node.resource.id;
    block.append(...resourceRows(node, 'host', 0).map((row) => row.element));
    items.set(node.resource.id, block);
  }
  for (const webApp of filteredWebApps) {
    const block = document.createElement('div');
    block.className = 'host-block webapp-block';
    block.dataset.topLevelId = webApp.id;
    block.append(resourceRow(webApp, 'webapp', 0).element);
    items.set(webApp.id, block);
  }
  for (const id of state.topLevelOrder) {
    if (!items.has(id)) continue;
    elements.push(items.get(id));
    items.delete(id);
  }
  elements.push(...items.values());

  if (hasFilters && filteredHosts.length === 0 && filteredWebApps.length === 0) {
    const empty = document.createElement('p');
    empty.className = 'host-filter-empty';
    empty.textContent = 'No connection matches the current filters.';
    elements.push(empty);
  }
  container.replaceChildren(...elements);
  requestAnimationFrame(syncSidebarScrollbarInset);
  if (canReorder && state.topLevelOrder.length > 1) initializeHostSorting(container);
}

function resourceRows(node, type, depth) {
  const resource = node.resource;
  const rows = [resourceRow(resource, type, depth)];
  for (const child of node.children) {
    rows.push(...resourceRows(child, child.resource.type, depth + 1));
  }
  return rows;
}

function resourceRow(resource, type, depth = 0) {
  const row = document.createElement('div');
  row.className = `host-row ${depth > 0 ? 'child' : ''}`;
  if (isLocalResource(resource)) row.classList.add('local-row');
  if (type === 'webapp') row.classList.add('webapp-row');
  row.dataset.id = resource.id;
  row.style.setProperty('--tree-offset', `${depth * 18}px`);
  const name = resourceName(resource);
  const target = resourceTarget(resource);
  const tags = resourceTags(resource);
  const tooltip = resourceTooltip(resource);
  const tunnel = tunnelForResource(resource.id);
  const webAppRuntime = type === 'webapp' ? state.webAppRuntimes.get(resource.id) : null;
  const webAppTunnelActive = Boolean(webAppRuntime?.tunnelActive);

  const selectButton = document.createElement('button');
  selectButton.type = 'button';
  selectButton.className = 'host-select';
  if (tunnel || webAppTunnelActive) selectButton.classList.add('tunnel-active');
  if (tags.length) selectButton.classList.add('has-tags');
  selectButton.setAttribute('aria-label', tooltip);
  selectButton.dataset.tooltip = tooltip;
  selectButton.dataset.tooltipPlacement = 'right';
  selectButton.innerHTML = `
    <span class="type"></span>
    <span class="compact-name" aria-hidden="true"></span>
    <span class="details">
      <span class="resource-name-line">
        <strong></strong>
      </span>
      <small></small>
    </span>
    <span class="resource-tags" aria-label="Connection tags"></span>
  `;
  if (tunnel || webAppTunnelActive) {
    const chip = document.createElement('span');
    chip.className = 'smart-chip tunnel-chip';
    chip.textContent = 'tun';
    chip.setAttribute('aria-label', webAppTunnelActive
      ? `SOCKS tunnel on 127.0.0.1:${webAppRuntime.socksPort}`
      : tunnelLabel(tunnel.type));
    selectButton.querySelector('.resource-name-line').append(chip);
  }
  selectButton.querySelector('.type').textContent = type === 'webapp' ? 'WebApp' : type;
  selectButton.querySelector('.compact-name').textContent = compactResourceName(name);
  selectButton.querySelector('strong').textContent = name;
  selectButton.querySelector('small').textContent = target;
  renderResourceTagPills(selectButton.querySelector('.resource-tags'), tags, resource.id);
  if (!isLocalResource(resource)) {
    const tagDropTarget = document.createElement('span');
    tagDropTarget.className = 'resource-tag-drop-target';
    tagDropTarget.dataset.resourceId = resource.id;
    tagDropTarget.setAttribute('aria-hidden', 'true');
    selectButton.append(tagDropTarget);
  }
  selectButton.addEventListener('click', () => {
    selectResource(resource);
  });
  selectButton.addEventListener('dblclick', () => {
    state.selectedId = resource.id;
    if (type === 'webapp') {
      selectResource(resource);
      launchWebApp(resource);
    } else {
      createPendingTab(resource);
      quickConnect(resource);
    }
  });
  selectButton.addEventListener('mouseenter', () => showTooltip(selectButton, state.sidebarCollapsed ? 0 : TOOLTIP_DELAY));
  selectButton.addEventListener('focus', () => showTooltip(selectButton, state.sidebarCollapsed ? 0 : TOOLTIP_DELAY));
  selectButton.addEventListener('mouseleave', () => hideTooltip());
  selectButton.addEventListener('blur', () => hideTooltip());
  row.append(selectButton);

  return {
    element: row,
  };
}

function renderTagFilters() {
  const data = tagFilterData({
    catalog: state.tags,
    hosts: state.hosts,
    webApps: state.webApps,
    activeFilters: state.activeTagFilters,
    mode: state.tagFilterMode,
  });
  state.activeTagFilters = data.activeFilters;
  tagFilters.update(data);
}

function renderResourceTagPills(container, tags, resourceID) {
  if (!container) return;
  container.dataset.resourceId = resourceID;
  if (tags.length === 0) return;
  const { shown, hidden } = visibleResourceTags(tags);
  container.dataset.visibleCount = String(shown.length + (hidden.length > 0 ? 1 : 0));
  const pills = shown.map((tag) => {
    const pill = document.createElement('span');
    pill.className = `resource-tag${state.activeTagFilters.includes(tag.key) ? ' active' : ''}`;
    pill.dataset.tagKey = tag.key;
    pill.dataset.tagName = tag.name;
    pill.textContent = tag.name;
    pill.title = `Tag: ${tag.name}`;
    applyTagPalette(pill, tag.name);
    return pill;
  });
  if (hidden.length > 0) {
    const overflow = document.createElement('span');
    overflow.className = 'resource-tag overflow';
    overflow.dataset.resourceId = resourceID;
    overflow.textContent = `+${hidden.length}`;
    overflow.title = hidden.map((tag) => tag.name).join(', ');
    overflow.setAttribute('role', 'button');
    overflow.setAttribute('tabindex', '0');
    overflow.setAttribute('aria-label', `Show all ${tags.length} tags`);
    overflow.setAttribute('aria-expanded', 'false');
    overflow.addEventListener('click', (event) => {
      event.stopPropagation();
      toggleTagPopover(overflow, resourceID);
    });
    overflow.addEventListener('keydown', (event) => {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      event.stopPropagation();
      toggleTagPopover(overflow, resourceID);
    });
    pills.push(overflow);
  }
  container.replaceChildren(...pills);
}

function toggleTagPopover(anchor, resourceID) {
  if (tagPopover?.resourceID === resourceID) {
    closeTagPopover();
    return;
  }
  void openTagPopover(anchor, resourceID);
}

async function openTagPopover(anchor, resourceID) {
  const found = findResource(resourceID);
  const tags = resourceTags(found?.resource);
  if (!anchor?.isConnected || tags.length <= 3) return;

  closeTagPopover();
  const request = tagPopoverRequest;
  const root = document.createElement('div');
  root.className = 'tag-popover-root';
  document.body.append(root);
  const entry = { anchor, resourceID, root, unmount: null };
  tagPopover = entry;
  anchor.setAttribute('aria-expanded', 'true');

  const { mountTagPopover } = await import('./tag-popover/mount.js');
  if (request !== tagPopoverRequest || tagPopover !== entry || !anchor.isConnected) {
    root.remove();
    return;
  }
  entry.unmount = mountTagPopover(root, {
    anchor,
    resourceId: resourceID,
    tags,
    oncancel: () => closeTagPopover(),
    onDragStart: () => beginTagSorting('assigned'),
    onDragEnd: () => finishTagSorting(),
  });
}

function closeTagPopover() {
  tagPopoverRequest += 1;
  if (!tagPopover) return;
  const { anchor, root, unmount } = tagPopover;
  tagPopover = null;
  if (anchor?.isConnected) anchor.setAttribute('aria-expanded', 'false');
  unmount?.();
  root.remove();
}

function restoreTagPopover(resourceID) {
  const anchor = [...document.querySelectorAll('.resource-tag.overflow')]
    .find((element) => element.dataset.resourceId === resourceID);
  if (anchor) void openTagPopover(anchor, resourceID);
}

function applyTagPalette(element, name) {
  element.style.setProperty('--tag-hue', String(tagHue(name)));
}

function initializeTagSorting() {
  const groupName = 'bashes-resource-tags';
  const catalog = document.querySelector('#tag-filter-chips');
  const removeZone = document.querySelector('#tag-remove-zone');

  if (catalog.querySelector('.tag-filter-chip')) {
    tagCatalogSortable = Sortable.create(catalog, {
      animation: 150,
      draggable: '.tag-filter-chip',
      sort: false,
      group: { name: groupName, pull: 'clone', put: false, revertClone: true },
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      ghostClass: 'tag-drag-ghost',
      chosenClass: 'tag-drag-chosen',
      fallbackClass: 'tag-drag-mirror',
      onStart() {
        beginTagSorting('catalog');
      },
      onMove(event) {
        setActiveTagDropTarget(event.to.closest?.('.resource-tag-drop-target'));
        return true;
      },
      onEnd() {
        suppressTagFilterClickUntil = Date.now() + 120;
        finishTagSorting();
      },
    });
  }

  document.querySelectorAll('.resource-tag-drop-target').forEach((target) => {
    resourceTagSortables.push(Sortable.create(target, {
      sort: false,
      group: {
        name: groupName,
        pull: false,
        put: (_to, from) => from.el?.id === 'tag-filter-chips',
      },
      onAdd(event) {
        const resourceID = event.to.dataset.resourceId;
        const tagName = event.item.dataset.tagName;
        event.item.remove();
        finishTagSorting();
        void updateResourceTagAssignment(resourceID, tagName, true);
      },
    }));
  });

  document.querySelectorAll('.resource-tags').forEach((container) => {
    if (!container.querySelector('.resource-tag:not(.overflow)')) return;
    resourceTagSortables.push(Sortable.create(container, {
      animation: 150,
      draggable: '.resource-tag:not(.overflow)',
      sort: false,
      group: { name: groupName, pull: 'clone', put: false, revertClone: true },
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      ghostClass: 'tag-drag-ghost',
      chosenClass: 'tag-drag-chosen',
      fallbackClass: 'tag-drag-mirror',
      onStart() {
        beginTagSorting('assigned');
      },
      onMove(event) {
        setActiveTagDropTarget(event.to.closest?.('.tag-remove-zone'));
        return true;
      },
      onEnd() {
        finishTagSorting();
      },
    }));
  });

  tagRemoveSortable = Sortable.create(removeZone, {
    sort: false,
    group: {
      name: groupName,
      pull: false,
      put: (_to, from) => from.el?.classList.contains('resource-tags'),
    },
    onAdd(event) {
      const resourceID = event.from.dataset.resourceId;
      const tagName = event.item.dataset.tagName;
      event.item.remove();
      finishTagSorting();
      void updateResourceTagAssignment(resourceID, tagName, false);
    },
  });
}

function destroyTagSorting() {
  tagCatalogSortable?.destroy();
  tagCatalogSortable = null;
  tagRemoveSortable?.destroy();
  tagRemoveSortable = null;
  resourceTagSortables.forEach((sortable) => sortable.destroy());
  resourceTagSortables = [];
  finishTagSorting();
}

function beginTagSorting(kind) {
  hideTooltip();
  clearAllDragOutlines();
  document.documentElement.classList.add('sorting-tags', `sorting-${kind}-tag`);
}

function finishTagSorting() {
  setActiveTagDropTarget(null);
  document.documentElement.classList.remove('sorting-tags', 'sorting-catalog-tag', 'sorting-assigned-tag');
}

function setActiveTagDropTarget(target) {
  if (activeTagDropTarget === target) return;
  activeTagDropTarget?.classList.remove('active');
  activeTagDropTarget = target;
  activeTagDropTarget?.classList.add('active');
}

async function updateResourceTagAssignment(resourceID, tagName, assigned) {
  const found = findResource(resourceID);
  const normalizedName = String(tagName ?? '').trim();
  if (!found || isLocalResource(found.resource) || !normalizedName) return;

  const currentTags = resourceTags(found.resource).map((tag) => tag.name);
  const key = canonicalTag(normalizedName);
  const alreadyAssigned = currentTags.some((tag) => canonicalTag(tag) === key);
  if (assigned === alreadyAssigned) return;
  const restorePopover = tagPopover?.resourceID === resourceID;

  const nextTags = assigned
    ? [...currentTags, normalizedName]
    : currentTags.filter((tag) => canonicalTag(tag) !== key);
  await withBusy(async () => {
    await apiSetResourceTags(resourceID, nextTags);
    await refreshHosts();
    if (restorePopover) restoreTagPopover(resourceID);
    writeNotice(`${assigned ? 'Added' : 'Removed'} tag ${normalizedName} ${assigned ? 'to' : 'from'} ${resourceName(found.resource)}.`);
  });
}

function toggleTagFilter(tag) {
  tag = canonicalTag(tag);
  state.activeTagFilters = state.activeTagFilters.includes(tag)
    ? state.activeTagFilters.filter((active) => active !== tag)
    : [...state.activeTagFilters, tag];
  renderHosts(searchInput.value);
  renderSelection();
}

function clearTagFilters() {
  state.activeTagFilters = [];
  renderHosts(searchInput.value);
  renderSelection();
}

function setTagFilterMode(mode) {
  state.tagFilterMode = mode === 'all' ? 'all' : 'any';
  renderHosts(searchInput.value);
  renderSelection();
}

function openTagPanel() {
  tagPanel.open(tagPanelData());
}

function closeTagPanel() {
  tagPanel.close();
  restoreTerminalFocusAfterOverlay();
}

function renderTagPanel() {
  tagPanel.update(tagPanelData());
}

function tagPanelData() {
  return createTagPanelData({
    catalog: state.tags,
    hosts: state.hosts,
    webApps: state.webApps,
    activeFilters: state.activeTagFilters,
    busy: state.busy,
  });
}

async function createManagedTag(name) {
  let created = false;
  await withBusy(async () => {
    await apiCreateTag(name);
    created = true;
    await refreshHosts();
    renderTagPanel();
  });
  return created;
}

async function renameManagedTag(tag, newName) {
  let renamed = false;
  await withBusy(async () => {
    await apiRenameTag(tag.name, newName);
    const replacement = canonicalTag(newName);
    state.activeTagFilters = state.activeTagFilters.map((active) => active === tag.key ? replacement : active);
    renamed = true;
    await refreshHosts();
    renderTagPanel();
  });
  return renamed;
}

async function deleteManagedTag(tag) {
  const accepted = await openConfirmModal({
    kicker: 'Tags',
    title: `Delete ${tag.name}?`,
    message: tag.count > 0
      ? `This tag will be removed from ${tag.count} system${tag.count === 1 ? '' : 's'}.`
      : 'This tag is not assigned to any system.',
    confirmLabel: 'Delete',
  });
  if (!accepted) return;

  await withBusy(async () => {
    await apiDeleteTag(tag.name);
    state.activeTagFilters = state.activeTagFilters.filter((active) => active !== tag.key);
    await refreshHosts();
    renderTagPanel();
  });
}

function toggleTagPanelTag(tag) {
  toggleTagFilter(tag.key);
  renderTagPanel();
}

function clearTagPanelSelection() {
  clearTagFilters();
  renderTagPanel();
}

function initializeHostSorting(container) {
  hostSortable = Sortable.create(container, {
    animation: 170,
    easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)',
    draggable: '.host-block',
    filter: '.resource-tag, .resource-tag-drop-target',
    preventOnFilter: false,
    forceFallback: true,
    fallbackOnBody: true,
    fallbackTolerance: 4,
    ghostClass: 'host-drag-ghost',
    chosenClass: 'host-drag-chosen',
    fallbackClass: 'host-drag-mirror',
    onUnchoose() {
      clearDragOutline('hosts');
    },
    onStart() {
      hideTooltip();
      document.documentElement.classList.add('sorting-hosts');
    },
    onEnd() {
      clearDragOutline('hosts');
      document.documentElement.classList.remove('sorting-hosts');
      persistHostOrder(container);
    },
  });
}

async function persistHostOrder(container) {
  const order = [...container.querySelectorAll(':scope > .host-block')].map((block) => block.dataset.topLevelId);
  const previousOrder = state.topLevelOrder;
  if (order.length !== previousOrder.length || order.every((id, index) => id === previousOrder[index])) return;

  await withBusy(async () => {
    await apiReorderTopLevel(order);
    state.topLevelOrder = [...order];
    writeNotice('Connection order updated.');
  });
  renderHosts(searchInput.value);
  renderSelection();
}

function compactResourceName(name) {
  const clean = String(name ?? '').trim();
  if (!clean) return '?';
  const parts = clean.split(/[-_\s.]+/).filter(Boolean);
  if (parts.length >= 2) {
    return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
  }
  return clean.slice(0, 2).toUpperCase();
}

function toggleSidebar() {
  state.sidebarCollapsed = !state.sidebarCollapsed;
  localStorage.setItem('bashes.sidebarCollapsed', String(state.sidebarCollapsed));
  applySidebarState();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function applySidebarState() {
  app.classList.toggle('sidebar-collapsed', state.sidebarCollapsed);
  const toggle = document.querySelector('#toggle-sidebar');
  toggle.title = state.sidebarCollapsed ? 'Expand sidebar' : 'Compact sidebar';
  toggle.setAttribute('aria-label', toggle.title);
  toggle.setAttribute('aria-expanded', String(!state.sidebarCollapsed));
  hideTooltip();
  requestAnimationFrame(syncSidebarScrollbarInset);
}

function syncSidebarScrollbarInset() {
  const inset = sidebarListInset(hostsScroll.offsetWidth, hostsScroll.clientWidth);
  hostsScroll.style.setProperty('--hosts-scroll-inset', `${inset}px`);
}

function showTooltip(anchor, delay = TOOLTIP_DELAY) {
  hideTooltip();
  if (delay <= 0) {
    renderTooltip(anchor);
    return;
  }

  tooltipTimer = window.setTimeout(() => {
    tooltipTimer = 0;
    renderTooltip(anchor);
  }, delay);
}

function renderTooltip(anchor) {
  if (!anchor.isConnected) return;
  const tooltip = document.querySelector('#app-tooltip');
  const text = anchor.dataset.tooltip;
  if (!tooltip || !text) return;

  const rect = anchor.getBoundingClientRect();
  tooltip.textContent = text;
  tooltip.hidden = false;
  const tooltipRect = tooltip.getBoundingClientRect();
  const placement = anchor.dataset.tooltipPlacement || 'right';
  const left = placement === 'bottom'
    ? rect.left + rect.width / 2 - tooltipRect.width / 2
    : rect.right + 10;
  const top = placement === 'bottom'
    ? rect.bottom + 8
    : rect.top + rect.height / 2 - tooltipRect.height / 2;
  tooltip.style.left = `${Math.round(clampNumber(left, 8, Math.max(8, window.innerWidth - tooltipRect.width - 8)))}px`;
  tooltip.style.top = `${Math.round(clampNumber(top, 8, Math.max(8, window.innerHeight - tooltipRect.height - 8)))}px`;
}

function hideTooltip() {
  if (tooltipTimer) {
    window.clearTimeout(tooltipTimer);
    tooltipTimer = 0;
  }
  const tooltip = document.querySelector('#app-tooltip');
  if (tooltip) tooltip.hidden = true;
}

function resourceTooltip(resource) {
  const tags = resourceTags(resource);
  return `${resourceName(resource)} - ${resourceTarget(resource)}${tags.length ? ` - ${tags.map((tag) => tag.name).join(', ')}` : ''}`;
}

function webAppTabForResource(resourceID) {
  return [...state.sessions.values()].find((session) => session.kind === 'webapp' && session.resourceId === resourceID);
}

function createWebAppTab(webApp) {
  clearPendingTabs(webApp.id);
  const existing = webAppTabForResource(webApp.id);
  if (existing) {
    focusSession(existing.id);
    return existing.id;
  }

  const tabID = `webapp-${webApp.id}`;
  state.sessions.set(tabID, {
    id: tabID,
    resourceId: webApp.id,
    title: webApp.name,
    target: resourceTarget(webApp),
    kind: 'webapp',
    pending: !state.webAppRuntimes.has(webApp.id),
    closed: false,
  });
  setActiveSession(tabID);
  renderTabs();
  renderSelection();
  return tabID;
}

function promoteWebAppTab(webApp) {
  const tabID = createWebAppTab(webApp);
  const tab = state.sessions.get(tabID);
  if (!tab) return;
  tab.pending = false;
  tab.title = webApp.name;
  tab.target = resourceTarget(webApp);
}

function createPendingTab(resource) {
  clearPendingTabs(resource.id);
  const existing = pendingSessionForResource(resource.id);
  if (existing) {
    focusSession(existing.id);
    return existing.id;
  }

  const sessionID = `pending-${resource.id}`;
  const pane = document.createElement('section');
  pane.className = 'terminal-pane pending-pane';
  pane.dataset.sessionId = sessionID;
  pane.textContent = isLocalResource(resource)
    ? 'Ready to start local shell'
    : `Ready to connect to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  stack.append(pane);

  state.sessions.set(sessionID, {
    id: sessionID,
    resourceId: resource.id,
    title: `${resource.hostname} new`,
    target: resourceTarget(resource),
    element: pane,
    pending: true,
    closed: false,
  });
  setActiveSession(sessionID);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  return sessionID;
}

function clearPendingTabs(exceptResourceId = '') {
  for (const session of [...state.sessions.values()]) {
    if (!session.pending || session.resourceId === exceptResourceId) continue;
    if (session.element) session.element.remove();
    state.sessions.delete(session.id);
    forgetSessionFocus(session.id);
    if (state.activeSessionId === session.id) setActiveSession(lastFocusedSessionID());
  }
}

function selectResource(resource) {
  state.selectedId = resource.id;
  if (isWebAppResource(resource)) {
    createWebAppTab(resource);
    return;
  }
  const session = preferredSessionForResource(resource.id);
  if (session) {
    if (!session.pending) clearPendingTabs(resource.id);
    focusSession(session.id);
  } else {
    createPendingTab(resource);
  }
}

async function attachStartedSession(sessionID, resource, kind = 'ssh', replaceSessionID = '') {
  try {
    createSession(sessionID, resource, kind, replaceSessionID);
  } catch (error) {
    await apiStopSSHSession(sessionID).catch(() => {});
    const attachError = new Error(`Terminal UI initialization failed: ${error?.message ?? error}`);
    attachError.stage = 'terminal-ui';
    throw attachError;
  }
}

function createSession(sessionID, resource, kind = 'ssh', replaceSessionID = '') {
  const pending = pendingSessionForResource(resource.id);
  if (pending?.pending) {
    pending.element.remove();
    state.sessions.delete(pending.id);
  }
  const replacedSession = replaceSessionID ? state.sessions.get(replaceSessionID) : null;
  const ordinal = replacedSession?.ordinal ?? realSessionsForResource(resource.id).length + 1;

  const pane = document.createElement('section');
  pane.className = 'terminal-pane';
  pane.dataset.sessionId = sessionID;
  stack.append(pane);

  let terminal;
  try {
    terminal = new Terminal(terminalOptions());
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(pane);
  const writeInput = createOrderedWriter((data) => apiWriteSSHSession(sessionID, data));
  installTerminalKeyRepeatFallback(terminal, sessionID, writeInput);
  terminal.onData((data) => {
    if (state.sessions.get(sessionID)?.closed) return;
    writeInput(data).catch((error) => {
      writeNotice(`${terminalKindLabel(kind)} input error: ${error?.message ?? error}`);
    });
  });
  terminal.onSelectionChange(() => {
    if (!state.terminalCopyOnSelect) return;
    const selected = terminal.getSelection();
    if (!selected) return;
    writeClipboard(selected).catch(() => {});
  });
  pane.addEventListener('contextmenu', (event) => {
    event.preventDefault();
    event.stopPropagation();
    readClipboard()
      .then((text) => {
        if (state.sessions.get(sessionID)?.closed) return undefined;
        pasteIntoTerminal(terminal, text);
        return undefined;
      })
      .catch(() => {});
  });

    const nextSession = {
      id: sessionID,
      resourceId: resource.id,
      title: sessionTitle(resource.hostname, ordinal),
      ordinal,
      target: resourceTarget(resource),
      kind,
      terminal,
      fitAddon,
      element: pane,
      closed: false,
    };
    if (replacedSession) {
      state.sessions = replaceSession(state.sessions, replaceSessionID, nextSession);
      state.pendingSSHOutput.delete(replaceSessionID);
      forgetSessionFocus(replaceSessionID);
      if (state.lastSessionByResource.get(replacedSession.resourceId) === replaceSessionID) {
        state.lastSessionByResource.delete(replacedSession.resourceId);
      }
      replacedSession.terminal?.dispose();
      replacedSession.element?.remove();
    } else {
      state.sessions.set(sessionID, nextSession);
    }
    flushPendingSSHOutput(sessionID);
    setActiveSession(sessionID);
    renderTabs();
    renderSelection();
    scheduleTerminalFit();
    focusActiveTerminal();
  } catch (error) {
    state.sessions.delete(sessionID);
    terminal?.dispose();
    pane.remove();
    throw error;
  }
}

function terminalOptions(disableStdin = false) {
  return {
    cursorBlink: !disableStdin,
    convertEol: true,
    disableStdin,
    fontFamily: state.terminalFontFamily,
    fontSize: state.terminalFontSize,
    rightClickSelectsWord: false,
    scrollback: state.terminalScrollback,
    linkHandler: {
      activate: (event, url) => openTerminalLink(event, url),
    },
    theme: {
      background: '#101418',
      foreground: '#d7dde5',
      cursor: '#f5c542',
      selectionBackground: '#3d4a58',
    },
  };
}

function installTerminalKeyRepeatFallback(terminal, sessionID, writeInput) {
  terminal.attachCustomKeyEventHandler((event) => {
    const session = state.sessions.get(sessionID);
    if (session?.closed && event.type === 'keydown') {
      const shortcut = closedSessionShortcut(event);
      event.preventDefault();
      if (shortcut === 'close') {
        window.setTimeout(() => removeSessionFromUI(sessionID), 0);
      } else if (shortcut === 'reconnect') {
        window.setTimeout(() => reconnectClosedSession(sessionID), 0);
      }
      return false;
    }

    if (event.type !== 'keydown' || !event.repeat || event.isComposing) {
      return true;
    }

    const applicationCursor = Boolean(terminal.modes?.applicationCursorKeysMode);
    const data = repeatedKeyData(event, applicationCursor);
    if (!data) return true;

    event.preventDefault();
    writeInput(data).catch((error) => {
      const session = state.sessions.get(sessionID);
      writeNotice(`${terminalKindLabel(session?.kind)} input error: ${error?.message ?? error}`);
    });
    return false;
  });
}

function renderTabs() {
  const tabs = document.querySelector('#session-tabs');
  hideTooltip();
  clearDragOutline('tabs');
  tabSortable?.destroy();
  tabSortable = null;
  const empty = document.querySelector('#empty-terminal');
  const sessions = [...state.sessions.values()];
  tabs.hidden = sessions.length === 0;
  empty.hidden = Boolean(state.activeSessionId);

  tabs.replaceChildren(...sessions.map((session) => {
    const tab = document.createElement('button');
    tab.type = 'button';
    tab.className = `session-tab ${session.id === state.activeSessionId ? 'active' : ''} ${session.closed ? 'closed' : ''} ${session.pending ? 'pending' : ''}`;
    tab.innerHTML = '<span></span><strong></strong>';
    tab.querySelector('span').textContent = session.kind === 'webapp'
      ? 'webapp'
      : session.closed ? 'closed' : session.pending ? 'new' : terminalKindLabel(session.kind);
    tab.querySelector('strong').textContent = session.title;
    const resource = findResource(session.resourceId)?.resource;
    tab.dataset.tooltip = resource ? resourceTooltip(resource) : `${session.title} - ${session.target}`;
    tab.dataset.tooltipPlacement = 'bottom';
    tab.addEventListener('mouseenter', () => showTooltip(tab));
    tab.addEventListener('focus', () => showTooltip(tab));
    tab.addEventListener('mouseleave', () => hideTooltip());
    tab.addEventListener('blur', () => hideTooltip());
    tab.addEventListener('click', () => {
      if (!session.pending) clearPendingTabs(session.resourceId);
      setActiveSession(session.id);
      renderTabs();
      renderSelection();
      scheduleTerminalFit();
      focusActiveTerminal();
    });
    tab.addEventListener('dblclick', () => {
      if (session.kind === 'webapp' && session.pending) {
        const webApp = findResource(session.resourceId)?.resource;
        if (webApp) launchWebApp(webApp);
      } else if (session.closed) {
        reconnectClosedSession(session.id);
      }
    });
    const close = document.createElement('button');
    close.type = 'button';
    close.className = 'tab-close';
    close.append(iconElement('x'));
    close.title = `Close ${session.title}`;
    close.addEventListener('click', (event) => {
      event.stopPropagation();
      stopSession(session.id);
    });

    const wrapper = document.createElement('div');
    wrapper.className = `session-tab-wrap ${session.id === state.activeSessionId ? 'active' : ''}`;
    wrapper.dataset.sessionId = session.id;
    wrapper.append(tab, close);
    return wrapper;
  }));

  if (sessions.length > 1) initializeTabSorting(tabs);

  document.querySelectorAll('.terminal-pane').forEach((pane) => {
    pane.hidden = pane.dataset.sessionId !== state.activeSessionId;
  });
}

function initializeTabSorting(tabs) {
  tabSortable = Sortable.create(tabs, {
    animation: 170,
    easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)',
    direction: 'horizontal',
    draggable: '.session-tab-wrap',
    handle: '.session-tab',
    forceFallback: true,
    fallbackOnBody: true,
    fallbackTolerance: 4,
    ghostClass: 'tab-drag-ghost',
    chosenClass: 'tab-drag-chosen',
    fallbackClass: 'tab-drag-mirror',
    onUnchoose() {
      clearDragOutline('tabs');
    },
    onStart() {
      document.documentElement.classList.add('sorting-tabs');
    },
    onEnd() {
      clearDragOutline('tabs');
      document.documentElement.classList.remove('sorting-tabs');
      const order = [...tabs.querySelectorAll(':scope > .session-tab-wrap')].map((tab) => tab.dataset.sessionId);
      state.sessions = orderSessions(state.sessions, order);
      renderTabs();
      focusActiveTerminal();
    },
  });
}

function schedulePointerDragOutline(event) {
  if (event.button !== 0) return;
  if (event.target.closest?.('.resource-tag, .tag-filter-chip')) return;
  const tab = event.target.closest?.('.session-tab');
  if (tabSortable && tab) {
    scheduleDragOutline('tabs', tab.closest('.session-tab-wrap'));
    return;
  }
  const host = event.target.closest?.('.host-block');
  if (hostSortable && host) scheduleDragOutline('hosts', host);
}

function scheduleDragOutline(kind, target) {
  clearDragOutline(kind);
  dragOutlineTargets[kind] = target;
  target?.classList.add('drag-outline-source');
  dragOutlineTimers[kind] = window.setTimeout(() => {
    dragOutlineTimers[kind] = 0;
    document.documentElement.classList.add(`${kind}-drag-outline-ready`);
  }, DRAG_OUTLINE_DELAY);
}

function clearDragOutline(kind) {
  window.clearTimeout(dragOutlineTimers[kind]);
  dragOutlineTimers[kind] = 0;
  dragOutlineTargets[kind]?.classList.remove('drag-outline-source');
  dragOutlineTargets[kind] = null;
  document.documentElement.classList.remove(`${kind}-drag-outline-ready`);
}

function clearAllDragOutlines() {
  clearDragOutline('hosts');
  clearDragOutline('tabs');
}

function openResourcePanel(mode, hostID = '') {
  resourcePanel.open(resourcePanelData({
    mode,
    parent: mode === 'subsystem' ? findResource(hostID)?.resource : null,
    catalog: resourceTagCatalog(),
  }));
}

function openEditPanel() {
  const selected = findResource(state.selectedId);
  if (!selected) return;
  resourcePanel.open(resourcePanelData({
    mode: 'edit',
    resourceType: selected.type,
    parent: selected.parent,
    resource: selected.resource,
    catalog: resourceTagCatalog(),
  }));
}

function closeResourcePanel() {
  resourcePanel.close();
  restoreTerminalFocusAfterOverlay();
}

async function openConnectPanel(statusMessage = '', statusKind = '') {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) {
    await quickConnect(selected);
    return;
  }

  await loadKeys();
  let hasSavedPassword = false;
  if (selected.auth?.method === 'password') {
    try {
      hasSavedPassword = await apiHasSavedPassword(selected.id);
    } catch {
      // The keyring error is reported if the user explicitly requests password storage.
    }
  }
  const realSessionCount = realSessionsForResource(selected.id).length;
  connectPanel.open(connectPanelData({
    resource: selected,
    keys: state.keys,
    sessionCount: realSessionCount,
    hasSavedPassword,
    statusMessage,
    statusKind,
    forcePassword: statusKind === 'error'
      && isAuthMessage(statusMessage)
      && (!selected.auth?.method || selected.auth.method === 'password'),
  }));
}

function focusConnectPasswordInput(form) {
  window.setTimeout(() => {
    requestAnimationFrame(() => {
      const method = form.elements.authMethod?.value || 'agent';
      const target = method === 'password'
        ? form.elements.password
        : method === 'key'
          ? form.elements.keyName
          : method === 'path'
            ? form.elements.privateKeyPath
            : form.elements.authMethod;
      target?.focus();
      if (!(method === 'password' && placeCaretAfterSavedPassword(target))) {
        target?.select?.();
      }
    });
  }, 0);
}

function closeConnectPanel() {
  connectPanel.close();
  restoreTerminalFocusAfterOverlay();
}

async function openTunnelPanel() {
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  await loadKeys();
  await loadTunnels();
  const panel = document.querySelector('#tunnel-panel');
  const form = document.querySelector('#tunnel-form');
  form.reset();
  const preference = loadTunnelPreference(localStorage, selected.id);
  form.elements.type.value = preference?.type || 'socks';
  form.elements.localHost.value = preference?.localHost || '127.0.0.1';
  form.elements.localPort.value = String(preference?.localPort || 1080);
  form.elements.remoteHost.value = preference?.remoteHost || '127.0.0.1';
  form.elements.remotePort.value = String(preference?.remotePort || 80);
  form.elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
  renderKeyOptions(form.elements.keyName, true);
  applyConnectDefaults(form, selected);
  await applySavedPasswordDefault(form, selected);
  updateTunnelMode();
  renderTunnelStatus();
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
  form.elements.localPort.focus();
}

function closeTunnelPanel() {
  const panel = document.querySelector('#tunnel-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  restoreTerminalFocusAfterOverlay();
}

function updateTunnelSummary() {
  const selected = findResource(state.selectedId)?.resource;
  const form = document.querySelector('#tunnel-form');
  const summary = document.querySelector('#tunnel-summary');
  if (!selected || !form || !summary) return;

  const type = form.elements.type.value || 'socks';
  const bind = form.elements.localHost.value.trim() || '127.0.0.1';
  const port = form.elements.localPort.value || '1080';
  const targetHost = form.elements.remoteHost.value.trim() || '127.0.0.1';
  const targetPort = form.elements.remotePort.value || (type === 'remote' ? '3000' : '80');
  const login = `${selected.user}@${selected.ip || selected.hostname}`;

  if (type === 'local') {
    summary.textContent = `ssh -L ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
    return;
  }
  if (type === 'remote') {
    summary.textContent = `ssh -R ${bind}:${port}:${targetHost}:${targetPort} ${login}`;
    return;
  }
  summary.textContent = `ssh -D ${bind}:${port} ${login}`;
}

function updateTunnelMode() {
  const form = document.querySelector('#tunnel-form');
  if (!form) return;

  const type = form.elements.type.value || 'socks';
  const title = document.querySelector('#tunnel-title');
  const bindLabel = document.querySelector('#tunnel-bind-label');
  const portLabel = document.querySelector('#tunnel-port-label');
  const targetFields = document.querySelector('#tunnel-target-fields');
  const targetHostLabel = document.querySelector('#tunnel-target-host-label');
  const targetPortLabel = document.querySelector('#tunnel-target-port-label');

  targetFields.hidden = type === 'socks';
  if (type === 'local') {
    title.textContent = 'Local Forward';
    bindLabel.textContent = 'Local Bind';
    portLabel.textContent = 'Local Port';
    targetHostLabel.textContent = 'Remote Host';
    targetPortLabel.textContent = 'Remote Port';
    if (!form.elements.localPort.value || form.elements.localPort.value === '1080') form.elements.localPort.value = '8080';
    if (!form.elements.remotePort.value) form.elements.remotePort.value = '80';
  } else if (type === 'remote') {
    title.textContent = 'Remote Forward';
    bindLabel.textContent = 'Remote Bind';
    portLabel.textContent = 'Remote Port';
    targetHostLabel.textContent = 'Local Host';
    targetPortLabel.textContent = 'Local Port';
    if (!form.elements.localPort.value || form.elements.localPort.value === '1080') form.elements.localPort.value = '8080';
    if (!form.elements.remotePort.value) form.elements.remotePort.value = '3000';
  } else {
    title.textContent = 'SOCKS Proxy';
    bindLabel.textContent = 'Bind';
    portLabel.textContent = 'Port';
    if (!form.elements.localPort.value) form.elements.localPort.value = '1080';
  }
  updateTunnelSummary();
}

async function openKeysPanel({ allowInstall = true } = {}) {
  const selected = findResource(state.selectedId)?.resource;
  state.keyInstallEnabled = allowInstall;

  await loadKeySettings();
  await loadKeys();
  setKeyDirectoryStatus('', '');
  setKeyImportStatus('', '');
  setKeyViewStatus('', '');
  setKeyInstallStatus('', '');
  showKeyActionStatus('', '');
  hideSelectedPrivateKey();
  showKeyPanelView('main');
  document.querySelector('#key-search').value = '';
  renderKeyBrowser();
  renderKeyInstallSummary();
  document.querySelector('#key-install-form').elements.trustHostKey.checked = trustHostKeyFromPreference(selected);
  const panel = document.querySelector('#keys-panel');
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));
}

function closeKeysPanel() {
  const panel = document.querySelector('#keys-panel');
  panel.classList.remove('open');
  panel.hidden = true;
  document.querySelector('#key-import-form').reset();
  setKeyImportStatus('', '');
  setKeyViewStatus('', '');
  showKeyActionStatus('', '');
  hideSelectedPrivateKey();
  showKeyPanelView('main');
  state.keyInstallEnabled = true;
  restoreTerminalFocusAfterOverlay();
}

async function openFileTransferModal() {
  if (!FILE_TRANSFER_ENABLED) return;
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const panel = document.querySelector('#file-transfer-modal');
  const title = document.querySelector('#file-transfer-title');
  const root = document.querySelector('#file-transfer-root');
  title.textContent = `${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
  panel.hidden = false;
  requestAnimationFrame(() => panel.classList.add('open'));

  let workspace = state.fileTransferWorkspaces.get(selected.id);
  if (!workspace) {
    const { mountFileTransfer } = await import('./file-transfer/mount.js');
    const element = document.createElement('div');
    element.className = 'file-transfer-workspace';
    element.dataset.resourceId = selected.id;
    element.addEventListener('bashes-file-transfer-active', (event) => {
      workspace.active = Boolean(event.detail?.active);
      renderFileTransferBackgroundIndicator();
    });
    root.append(element);
    workspace = {
      active: false,
      element,
      resource: selected,
      unmount: mountFileTransfer(element, { resource: selected }),
    };
    state.fileTransferWorkspaces.set(selected.id, workspace);
  }

  showFileTransferWorkspace(selected.id);
}

function closeFileTransferModal() {
  const workspace = state.fileTransferWorkspaces.get(state.activeFileTransferResourceId);
  if (workspace?.active) {
    backgroundFileTransferModal();
    writeNotice('File transfer continues in background.');
    return;
  }
  if (workspace) destroyFileTransferWorkspace(state.activeFileTransferResourceId);
  hideFileTransferModal();
}

function backgroundFileTransferModal() {
  if (state.activeFileTransferResourceId) {
    state.backgroundFileTransferResourceIds.add(state.activeFileTransferResourceId);
  }
  hideFileTransferModal();
  renderFileTransferBackgroundIndicator();
}

function hideFileTransferModal() {
  const panel = document.querySelector('#file-transfer-modal');
  panel.classList.remove('open');
  panel.hidden = true;
  restoreTerminalFocusAfterOverlay();
}

function showFileTransferWorkspace(resourceId) {
  for (const [id, workspace] of state.fileTransferWorkspaces) {
    workspace.element.hidden = id !== resourceId;
  }
  state.activeFileTransferResourceId = resourceId;
  state.backgroundFileTransferResourceIds.delete(resourceId);
  renderFileTransferBackgroundIndicator();
}

function destroyFileTransferWorkspace(resourceId) {
  if (!resourceId) return;
  const workspace = state.fileTransferWorkspaces.get(resourceId);
  if (!workspace) return;
  workspace.unmount();
  workspace.element.remove();
  state.fileTransferWorkspaces.delete(resourceId);
  state.backgroundFileTransferResourceIds.delete(resourceId);
  if (state.activeFileTransferResourceId === resourceId) {
    state.activeFileTransferResourceId = null;
  }
  renderFileTransferBackgroundIndicator();
}

function renderFileTransferBackgroundIndicator() {
  const button = document.querySelector('#open-file-transfer');
  if (!button) return;

  const backgrounded = state.backgroundFileTransferResourceIds.size > 0;
  const transferActive = [...state.backgroundFileTransferResourceIds].some(
    (resourceId) => state.fileTransferWorkspaces.get(resourceId)?.active,
  );
  const label = transferActive
    ? 'File transfer running in background'
    : backgrounded ? 'Files in background' : 'Files';

  button.classList.toggle('file-transfer-backgrounded', backgrounded);
  button.classList.toggle('file-transfer-running', transferActive);
  button.title = label;
  button.setAttribute('aria-label', label);
}

function closeAllFileTransferWorkspaces() {
  for (const resourceId of [...state.fileTransferWorkspaces.keys()]) {
    destroyFileTransferWorkspace(resourceId);
  }
  hideFileTransferModal();
}

function registerFileTransferModalResize() {
  const card = document.querySelector('.file-transfer-card');
  const handle = document.querySelector('#file-transfer-resize-handle');
  if (!card || !handle) return;

  let resizeState = null;
  handle.addEventListener('pointerdown', (event) => {
    event.preventDefault();
    resizeState = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      width: card.offsetWidth,
      height: card.offsetHeight,
    };
    handle.setPointerCapture(event.pointerId);
    document.body.classList.add('resizing-file-transfer');
  });

  handle.addEventListener('pointermove', (event) => {
    if (!resizeState || resizeState.pointerId !== event.pointerId) return;
    resizeFileTransferModal(
      resizeState.width + event.clientX - resizeState.startX,
      resizeState.height + event.clientY - resizeState.startY,
    );
  });

  const stopResize = (event) => {
    if (!resizeState || resizeState.pointerId !== event.pointerId) return;
    resizeState = null;
    document.body.classList.remove('resizing-file-transfer');
  };
  handle.addEventListener('pointerup', stopResize);
  handle.addEventListener('pointercancel', stopResize);
}

function registerFileTransferModalDrag() {
  const card = document.querySelector('.file-transfer-card');
  const header = document.querySelector('.file-transfer-header');
  if (!card || !header) return;

  let dragState = null;
  header.addEventListener('pointerdown', (event) => {
    if (event.target.closest('button, input, select, textarea, a')) return;
    event.preventDefault();
    const rect = card.getBoundingClientRect();
    dragState = {
      pointerId: event.pointerId,
      offsetX: event.clientX - rect.left,
      offsetY: event.clientY - rect.top,
      width: rect.width,
      height: rect.height,
    };
    card.style.position = 'fixed';
    card.style.left = `${rect.left}px`;
    card.style.top = `${rect.top}px`;
    card.style.margin = '0';
    header.setPointerCapture(event.pointerId);
    document.body.classList.add('dragging-file-transfer');
  });

  header.addEventListener('pointermove', (event) => {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    const maxLeft = Math.max(18, window.innerWidth - dragState.width - 18);
    const maxTop = Math.max(18, window.innerHeight - dragState.height - 18);
    card.style.left = `${clamp(event.clientX - dragState.offsetX, 18, maxLeft)}px`;
    card.style.top = `${clamp(event.clientY - dragState.offsetY, 18, maxTop)}px`;
  });

  const stopDrag = (event) => {
    if (!dragState || dragState.pointerId !== event.pointerId) return;
    dragState = null;
    document.body.classList.remove('dragging-file-transfer');
  };
  header.addEventListener('pointerup', stopDrag);
  header.addEventListener('pointercancel', stopDrag);
}

function resizeFileTransferModal(width, height) {
  const card = document.querySelector('.file-transfer-card');
  if (!card) return;
  const minWidth = Math.min(720, window.innerWidth - 36);
  const minHeight = Math.min(460, window.innerHeight - 36);
  const maxWidth = window.innerWidth - 36;
  const maxHeight = window.innerHeight - 36;
  card.style.width = `${clamp(width, minWidth, maxWidth)}px`;
  card.style.height = `${clamp(height, minHeight, maxHeight)}px`;
}

function clampFileTransferModalSize() {
  const panel = document.querySelector('#file-transfer-modal');
  const card = document.querySelector('.file-transfer-card');
  if (!panel || panel.hidden || !card || !card.style.width) return;
  resizeFileTransferModal(card.offsetWidth, card.offsetHeight);
}

function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

function renderSelection() {
  document.querySelectorAll('.host-row').forEach((row) => {
    row.classList.toggle('selected', row.dataset.id === state.selectedId);
  });

  const selected = findResource(state.selectedId);
  const activeSession = state.sessions.get(state.activeSessionId);
  const title = document.querySelector('#session-title');
  const edit = document.querySelector('#edit-resource');
  const addSubsystem = document.querySelector('#header-add-subsystem');
  const keys = document.querySelector('#open-keys-panel');
  const fileTransfer = document.querySelector('#open-file-transfer');
  const tunnel = document.querySelector('#open-tunnel-panel');
  const connect = document.querySelector('#connect');
  const disconnect = document.querySelector('#disconnect');
  const remove = document.querySelector('#delete-resource');

  const webAppSelected = selected?.type === 'webapp';
  if (activeSession) {
    title.textContent = activeSession.title;
  } else if (selected) {
    title.textContent = resourceName(selected.resource);
  } else {
    title.textContent = 'No session selected';
  }
  renderKeyInstallSummary();
  renderWorkbench(selected);

  if (!selected) {
    edit.disabled = true;
    addSubsystem.disabled = true;
    addSubsystem.hidden = false;
    keys.hidden = false;
    keys.closest('.session-action-group').hidden = false;
    keys.disabled = state.busy;
    fileTransfer.disabled = true;
    fileTransfer.hidden = !FILE_TRANSFER_ENABLED;
    tunnel.disabled = true;
    tunnel.hidden = false;
    connect.disabled = true;
    disconnect.disabled = true;
    disconnect.hidden = false;
    remove.disabled = true;
    return;
  }

  const localSelected = isLocalResource(selected.resource);
  const connectionPending = state.connectingResourceIds.has(selected.resource.id);
  addSubsystem.hidden = webAppSelected;
  edit.disabled = state.busy || !selected || localSelected;
  addSubsystem.disabled = state.busy || !selected || localSelected || webAppSelected;
  keys.hidden = webAppSelected;
  keys.closest('.session-action-group').hidden = webAppSelected;
  keys.disabled = state.busy;
  fileTransfer.hidden = webAppSelected || !FILE_TRANSFER_ENABLED;
  fileTransfer.disabled = state.busy || !selected || localSelected || webAppSelected || !FILE_TRANSFER_ENABLED;
  tunnel.hidden = webAppSelected;
  tunnel.disabled = state.busy || !selected || localSelected || webAppSelected;
  disconnect.hidden = webAppSelected;
  const connectLabel = webAppSelected
    ? 'Open'
    : realSessionsForResource(selected.resource.id).length > 0 ? 'New Session' : 'Connect';
  connect.querySelector('.connect-action-icon').replaceChildren(iconElement(webAppSelected ? 'external-link' : 'player-play-filled'));
  connect.querySelector('.connect-action-label').textContent = connectLabel;
  connect.title = connectLabel;
  connect.setAttribute('aria-label', connectLabel);
  connect.disabled = state.busy || !selected || connectionPending;
  disconnect.disabled = state.busy || !activeSession || activeSession.closed || webAppSelected;
  remove.disabled = state.busy || !selected || localSelected;
  renderTunnelStatus();
}

function renderWorkbench(selected) {
  const activeTab = state.sessions.get(state.activeSessionId);
  const webApp = activeTab?.kind === 'webapp' && selected?.type === 'webapp' ? selected.resource : null;
  stack.classList.toggle('webapp-active', Boolean(webApp));
  const gateway = webApp?.connectionMode === 'tunnel' ? findResource(webApp.gatewayResourceId)?.resource : null;
  webAppWorkspace.update(webAppWorkspaceData({
    webApp,
    gateway,
    runtime: webApp ? state.webAppRuntimes.get(webApp.id) : null,
    busy: state.busy,
  }));
}

function renderKeyOptions(select = document.querySelector('#key-select'), includeEmpty = false) {
  if (!select) return;
  const options = [];
  if (includeEmpty) {
    const empty = document.createElement('option');
    empty.value = '';
    empty.textContent = 'Use password/agent or custom path';
    options.push(empty);
  }

  const bashesGroup = keyOptionGroup('Bashes keys', state.keys.filter((key) => keySource(key) === 'bashes'));
  const systemGroup = keyOptionGroup('System keys (~/.ssh)', state.keys.filter((key) => keySource(key) === 'system'));
  const customGroup = keyOptionGroup('Custom keys directory', state.keys.filter((key) => keySource(key) === 'custom'));
  if (bashesGroup) options.push(bashesGroup);
  if (systemGroup) options.push(systemGroup);
  const wslDistributions = [...new Set(state.keys
    .filter((key) => keySource(key) === 'wsl')
    .map((key) => key.distribution || 'WSL'))];
  for (const distribution of wslDistributions) {
    const group = keyOptionGroup(
      `WSL keys (${distribution})`,
      state.keys.filter((key) => keySource(key) === 'wsl' && (key.distribution || 'WSL') === distribution),
    );
    if (group) options.push(group);
  }
  if (customGroup) options.push(customGroup);

  select.replaceChildren(...options);
}

function showKeyPanelView(view) {
  const views = {
    main: document.querySelector('#key-main-view'),
    add: document.querySelector('#key-add-view'),
    sources: document.querySelector('#key-sources-view'),
  };
  for (const [name, element] of Object.entries(views)) {
    element.hidden = name !== view;
  }
  hideSelectedPrivateKey();
  if (view === 'add') {
    showKeyAddMode('generate');
    window.setTimeout(() => document.querySelector('#key-generate-form').elements.name.focus(), 0);
  } else if (view === 'sources') {
    renderKeySources();
  } else {
    renderKeyBrowser();
  }
}

function showKeyAddMode(mode) {
  const generate = mode === 'generate';
  const generateButton = document.querySelector('#show-key-generate');
  const importButton = document.querySelector('#show-key-import');
  document.querySelector('#key-generate-form').hidden = !generate;
  document.querySelector('#key-import-form').hidden = generate;
  generateButton.classList.toggle('secondary', !generate);
  importButton.classList.toggle('secondary', generate);
  generateButton.setAttribute('aria-selected', String(generate));
  importButton.setAttribute('aria-selected', String(!generate));
  setKeyImportStatus('', '');
  const form = document.querySelector(generate ? '#key-generate-form' : '#key-import-form');
  window.setTimeout(() => form.elements.name.focus(), 0);
}

function keySourceLabel(key) {
  const source = keySource(key);
  if (source === 'bashes') return 'Managed by Bashes';
  if (source === 'system') return 'System';
  if (source === 'wsl') return key.distribution ? `WSL: ${key.distribution}` : 'WSL';
  if (source === 'custom') return 'Custom directory';
  return 'External';
}

function renderKeyBrowser() {
  const container = document.querySelector('#key-list');
  const empty = document.querySelector('#key-empty-state');
  const query = document.querySelector('#key-search')?.value.trim().toLocaleLowerCase() ?? '';
  const selectedValue = document.querySelector('#key-select')?.value ?? '';
  const keys = state.keys.filter((key) => {
    if (!query) return true;
    return [key.name, key.privateKey, keySourceLabel(key)]
      .some((value) => String(value ?? '').toLocaleLowerCase().includes(query));
  });
  const groupOrder = ['bashes', 'system', 'wsl', 'custom'];
  const groups = new Map();
  for (const key of keys) {
    const group = keySource(key) === 'wsl' ? `wsl:${key.distribution || 'WSL'}` : keySource(key);
    if (!groups.has(group)) groups.set(group, []);
    groups.get(group).push(key);
  }

  const fragments = [];
  for (const source of groupOrder) {
    for (const [group, groupKeys] of groups) {
      if (group !== source && !group.startsWith(`${source}:`)) continue;
      const section = document.createElement('section');
      section.className = 'key-list-group';
      const heading = document.createElement('h4');
      heading.textContent = keySourceLabel(groupKeys[0]);
      section.append(heading);
      for (const key of groupKeys) {
        const value = keyChoiceValue(key);
        const row = document.createElement('button');
        row.className = 'key-list-row';
        row.type = 'button';
        row.dataset.keyValue = value;
        row.classList.toggle('selected', value === selectedValue);
        row.setAttribute('aria-pressed', String(value === selectedValue));
        row.append(iconElement('key'));
        const text = document.createElement('span');
        text.className = 'key-list-row-text';
        const name = document.createElement('strong');
        name.textContent = key.name;
        const path = document.createElement('span');
        path.textContent = key.privateKey || 'Managed key';
        text.append(name, path);
        row.append(text);
        row.addEventListener('click', async () => {
          const select = document.querySelector('#key-select');
          select.value = value;
          await renderSelectedKey();
        });
        section.append(row);
      }
      fragments.push(section);
    }
  }
  container.replaceChildren(...fragments);
  empty.hidden = keys.length !== 0;
  empty.textContent = state.keys.length === 0 ? 'No SSH keys found.' : 'No keys match this search.';
}

function renderKeySources() {
  const container = document.querySelector('#key-source-list');
  if (!container) return;
  const rows = state.keySources.map((source) => {
    const row = document.createElement('section');
    row.className = 'key-source-row';
    const heading = document.createElement('div');
    const label = document.createElement('strong');
    label.textContent = source.label || keySourceLabel({ source: source.source, distribution: source.distribution });
    const count = document.createElement('span');
    count.textContent = `${source.count ?? 0} key${source.count === 1 ? '' : 's'}`;
    heading.append(label, count);
    const path = document.createElement('p');
    path.textContent = source.path || (source.source === 'custom' ? 'Not configured' : 'Not available');
    path.title = path.textContent;
    row.append(heading, path);
    if (source.error) {
      const error = document.createElement('p');
      error.className = 'key-source-error';
      error.textContent = source.error;
      row.append(error);
    }
    return row;
  });
  container.replaceChildren(...rows);
}

async function browseKeyDirectory() {
  try {
    const directory = await apiChooseSSHKeyDirectory();
    if (directory) document.querySelector('#key-directory-form').elements.directory.value = directory;
  } catch (error) {
    const message = `Could not choose SSH keys directory: ${error?.message ?? error}`;
    setKeyDirectoryStatus(message, 'error');
    writeNotice(message);
  }
}

async function rescanKeySources() {
  setKeyDirectoryStatus('Scanning key sources ...', 'pending');
  try {
    await loadKeys();
    renderKeySources();
    setKeyDirectoryStatus('Key sources rescanned.', 'success');
  } catch (error) {
    const message = `Could not rescan key sources: ${error?.message ?? error}`;
    setKeyDirectoryStatus(message, 'error');
    writeNotice(message);
  }
}

function keyOptionGroup(label, keys) {
  if (keys.length === 0) return null;
  const group = document.createElement('optgroup');
  group.label = label;
  for (const key of keys) {
    const option = document.createElement('option');
    option.value = keyChoiceValue(key);
    option.textContent = keyOptionLabel(key);
    option.title = key.privateKey || key.name;
    group.append(option);
  }
  return group;
}

function keySource(key) {
  return key?.source || 'bashes';
}

function keyChoiceValue(key) {
  if (keySource(key) !== 'bashes') return `path:${key.privateKey || ''}`;
  return `bashes:${key.name || ''}`;
}

function keyOptionLabel(key) {
  if (keySource(key) !== 'bashes') return `${key.name} - ${key.privateKey}`;
  return key.name;
}

function registerAuthChoiceSync(form) {
  if (!form?.elements?.authMethod) return;

  form.elements.authMethod.addEventListener('change', () => {
    updateAuthFields(form);
  });

  form.elements.privateKeyPath?.addEventListener('input', () => {
    if (form.elements.privateKeyPath.value.trim()) {
      form.elements.authMethod.value = 'path';
      form.elements.keyName.value = '';
      updateAuthFields(form);
    }
  });

  form.elements.keyName?.addEventListener('change', () => {
    if (form.elements.keyName.value) {
      form.elements.authMethod.value = 'key';
      form.elements.privateKeyPath.value = '';
      updateAuthFields(form);
    }
  });

  updateAuthFields(form);
}

function updateAuthFields(form) {
  const method = form.elements.authMethod?.value || 'agent';
  form.querySelectorAll('[data-auth-field]').forEach((field) => {
    const type = field.dataset.authField;
    field.hidden = !(type === method || (type === 'key-passphrase' && (method === 'key' || method === 'path')));
  });
}

function authInputFromForm(form) {
  const method = form.elements.authMethod?.value || 'agent';
  if (method === 'password') {
    const savePassword = form.elements.savePassword.checked;
    return {
      password: passwordFieldValue(form.elements.password, form.dataset.hadSavedPassword === 'true'),
      managePassword: savePassword || form.dataset.hadSavedPassword === 'true',
      savePassword,
      keyName: '',
      privateKeyPath: '',
      privateKeyPassphrase: '',
    };
  }

  if (method === 'path') {
    const privateKeyPath = form.elements.privateKeyPath.value.trim();
    return {
      keyName: '',
      privateKeyPath,
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }

  if (method === 'agent') {
    return {
      keyName: '',
      privateKeyPath: '',
      privateKeyPassphrase: '',
    };
  }

  const keyChoice = selectedKeyChoice(form.elements.keyName);
  if (keyChoice && keyChoice.source !== 'bashes') {
    return {
      keyName: '',
      privateKeyPath: keyChoice.privateKey,
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }
  if (keyChoice?.source === 'bashes') {
    return {
      keyName: keyChoice.name,
      privateKeyPath: '',
      privateKeyPassphrase: form.elements.privateKeyPassphrase.value,
    };
  }

  return {
    keyName: '',
    privateKeyPath: '',
    privateKeyPassphrase: '',
  };
}

function applyConnectDefaults(form, resource) {
  const auth = resource?.auth;
  form.elements.authMethod.value = 'agent';
  form.elements.keyName.value = '';
  form.elements.privateKeyPath.value = '';
  form.elements.password.value = '';
  showSavedPassword(form.elements.password, false);
  form.elements.savePassword.checked = false;
  form.elements.privateKeyPassphrase.value = '';
  if (!auth) {
    updateAuthFields(form);
    return;
  }

  if (auth.method === 'password') {
    form.elements.authMethod.value = 'password';
  }
  if (auth.method === 'key' && auth.keyName) {
    const value = `bashes:${auth.keyName}`;
    const hasKey = [...form.elements.keyName.options].some((option) => option.value === value);
    if (hasKey) {
      form.elements.authMethod.value = 'key';
      form.elements.keyName.value = value;
      form.elements.privateKeyPath.value = '';
    }
  }
  if (auth.method === 'path' && auth.privateKeyPath) {
    const value = `path:${auth.privateKeyPath}`;
    const hasPath = [...form.elements.keyName.options].some((option) => option.value === value);
    if (hasPath) {
      form.elements.authMethod.value = 'key';
      form.elements.keyName.value = value;
      form.elements.privateKeyPath.value = '';
    } else {
      form.elements.authMethod.value = 'path';
      form.elements.keyName.value = '';
      form.elements.privateKeyPath.value = auth.privateKeyPath;
    }
  }
  updateAuthFields(form);
}

async function applySavedPasswordDefault(form, resource) {
  if (!form?.elements?.savePassword) return;
  form.elements.savePassword.checked = false;
  form.dataset.hadSavedPassword = 'false';
  showSavedPassword(form.elements.password, false);
  if (resource?.auth?.method !== 'password') return;
  try {
    const hasSavedPassword = await apiHasSavedPassword(resource.id);
    form.elements.savePassword.checked = hasSavedPassword;
    form.dataset.hadSavedPassword = String(hasSavedPassword);
    showSavedPassword(form.elements.password, hasSavedPassword);
  } catch {
    // The keyring error is reported if the user explicitly requests password storage.
  }
}

function selectedKeyChoice(select) {
  if (!select?.value) return null;
  const key = state.keys.find((item) => keyChoiceValue(item) === select.value);
  if (key) return key;
  if (select.value.startsWith('path:')) {
    const privateKey = select.value.slice('path:'.length);
    return { source: 'path', name: privateKey.split(/[\\/]/).pop(), privateKey, publicKey: `${privateKey}.pub` };
  }
  if (select.value.startsWith('bashes:')) {
    const name = select.value.slice('bashes:'.length);
    return { source: 'bashes', name };
  }
  return null;
}

function installInputFromKeyChoice(keyChoice) {
  if (keyChoice.source !== 'bashes') {
    return {
      keyName: '',
      privateKeyPath: keyChoice.privateKey,
      publicKeyPath: keyChoice.publicKey || keyChoice.privateKey,
    };
  }
  return { keyName: keyChoice.name };
}

function keyChoiceLabel(keyChoice) {
  if (!keyChoice) return '';
  return keyChoice.source === 'bashes' ? keyChoice.name : keyChoice.privateKey;
}

function authInputFromPreference(resource) {
  const auth = resource?.auth;
  if (!auth) return {};

  if (auth.method === 'key' && auth.keyName) {
    return { keyName: auth.keyName };
  }
  if (auth.method === 'path' && auth.privateKeyPath) {
    return { privateKeyPath: auth.privateKeyPath };
  }
  return {};
}

function trustHostKeyFromPreference(resource) {
	return false;
}

async function renderSelectedKey() {
  const select = document.querySelector('#key-select');
  const output = document.querySelector('#public-key');
  if (!select || !output) return;
  hideSelectedPrivateKey();
  setKeyViewStatus('', '');
  const keyChoice = selectedKeyChoice(select);
  const details = document.querySelector('#key-details');
  details.hidden = !keyChoice;
  renderKeyBrowser();
  if (!keyChoice) {
    output.value = '';
    return;
  }
  document.querySelector('#key-details-name').textContent = keyChoice.name;
  document.querySelector('#key-details-source').textContent = keySourceLabel(keyChoice);
  const path = keyChoice.privateKey || keyChoice.publicKey || '';
  const pathElement = document.querySelector('#key-details-path');
  pathElement.textContent = path;
  pathElement.title = path;
  document.querySelector('#delete-managed-key').hidden = keyChoice.source !== 'bashes';
  try {
    output.value = keyChoice.source !== 'bashes'
      ? await apiReadSSHPublicKeyPath(keyChoice.publicKey || keyChoice.privateKey)
      : await apiReadSSHPublicKey(keyChoice.name);
  } catch (error) {
    output.value = '';
    const message = `Could not read SSH key ${keyChoiceLabel(keyChoice)}: ${error?.message ?? error}`;
    setKeyViewStatus(message, 'error');
    writeNotice(message);
  }
}

function hideSelectedPrivateKey() {
  const output = document.querySelector('#private-key');
  const toggle = document.querySelector('#toggle-private-key');
  if (output) {
    output.value = '';
    output.hidden = true;
  }
  if (toggle) {
    toggle.replaceChildren(iconElement('eye'));
    toggle.title = 'Show private key';
    toggle.setAttribute('aria-label', 'Show private key');
    toggle.setAttribute('aria-expanded', 'false');
  }
}

async function toggleSelectedPrivateKey() {
  const select = document.querySelector('#key-select');
  const output = document.querySelector('#private-key');
  const toggle = document.querySelector('#toggle-private-key');
  if (!select || !output || !toggle) return;
  if (!output.hidden) {
    hideSelectedPrivateKey();
    setKeyViewStatus('', '');
    return;
  }

  const keyChoice = selectedKeyChoice(select);
  if (!keyChoice) {
    setKeyViewStatus('Select an SSH key first.', 'error');
    return;
  }

  setKeyViewStatus('Reading private key ...', 'pending');
  try {
    output.value = keyChoice.source === 'bashes'
      ? await apiReadSSHPrivateKey(keyChoice.name)
      : await apiReadSSHPrivateKeyPath(keyChoice.privateKey);
    output.hidden = false;
    toggle.replaceChildren(iconElement('eye-off'));
    toggle.title = 'Hide private key';
    toggle.setAttribute('aria-label', 'Hide private key');
    toggle.setAttribute('aria-expanded', 'true');
    setKeyViewStatus('', '');
  } catch (error) {
    hideSelectedPrivateKey();
    const message = `Could not read private key ${keyChoiceLabel(keyChoice)}: ${error?.message ?? error}`;
    setKeyViewStatus(message, 'error');
    writeNotice(message);
  }
}

async function copySelectedPublicKey() {
  const value = document.querySelector('#public-key').value.trim();
  if (!value) {
    setKeyViewStatus('No public key is available to copy.', 'error');
    return;
  }
  try {
    await writeClipboard(value);
    setKeyViewStatus('Public key copied.', 'success');
  } catch (error) {
    const message = `Could not copy public key: ${error?.message ?? error}`;
    setKeyViewStatus(message, 'error');
    writeNotice(message);
  }
}

function resourcesUsingManagedKey(name) {
  const resources = [];
  const visit = (resource) => {
    if (resource?.auth?.method === 'key' && resource.auth.keyName === name) {
      resources.push(resource.hostname);
    }
    for (const child of resource?.subsystems ?? []) visit(child);
  };
  for (const host of state.hosts) visit(host);
  return resources;
}

async function deleteSelectedManagedKey() {
  const select = document.querySelector('#key-select');
  const keyChoice = selectedKeyChoice(select);
  if (!keyChoice || keyChoice.source !== 'bashes') return;
  const usages = resourcesUsingManagedKey(keyChoice.name);
  const usageMessage = usages.length > 0
    ? `\n\nUsed by: ${usages.join(', ')}. Update those connections after deletion.`
    : '';
  const confirmed = await openConfirmModal({
    kicker: 'SSH Keys',
    title: `Delete ${keyChoice.name}?`,
    message: `The local public and private key files will be removed. Keys already installed on remote systems are not removed.${usageMessage}`,
    confirmLabel: 'Delete key',
  });
  if (!confirmed) return;

  await withBusy(async () => {
    try {
      await apiDeleteSSHKey(keyChoice.name);
      await loadKeys();
      showKeyActionStatus(`Deleted ${keyChoice.name}.`, 'success');
    } catch (error) {
      const message = `Could not delete SSH key: ${error?.message ?? error}`;
      setKeyViewStatus(message, 'error');
      writeNotice(message);
    }
  });
}

function showKeyActionStatus(message, kind) {
  const status = document.querySelector('#key-action-status');
  if (!status) return;
  status.textContent = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function highlightKeyRow(value) {
  requestAnimationFrame(() => {
    const row = [...document.querySelectorAll('.key-list-row')]
      .find((item) => item.dataset.keyValue === value);
    if (!row) return;
    row.classList.add('key-list-row-new');
    row.scrollIntoView({ block: 'nearest' });
    window.setTimeout(() => row.classList.remove('key-list-row-new'), 1600);
  });
}

function renderKeyInstallSummary() {
  const form = document.querySelector('#key-install-form');
  const summary = document.querySelector('#key-install-summary');
  if (!form || !summary) return;
  const selected = findResource(state.selectedId)?.resource;
  const remoteSelected = state.keyInstallEnabled && selected && !isLocalResource(selected) && !isWebAppResource(selected);
  form.hidden = !remoteSelected;
  if (!remoteSelected) {
    form.reset();
    setKeyInstallStatus('', '');
    return;
  }
  summary.textContent = `Install selected key on ${selected.user}@${selected.ip || selected.hostname}:${selected.port}`;
}

function renderKeySettings() {
  const form = document.querySelector('#key-directory-form');
  if (!form) return;
  form.elements.directory.value = state.keySettings?.customDirectory || '';
}

function setKeyDirectoryStatus(message, kind) {
  const status = document.querySelector('#key-directory-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function setKeyImportStatus(message, kind) {
  const status = document.querySelector('#key-import-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function setKeyViewStatus(message, kind) {
  const status = document.querySelector('#key-view-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

function setKeyInstallStatus(message, kind) {
  const status = document.querySelector('#key-install-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  status.hidden = !message;
  status.dataset.kind = kind;
}

async function startSSHSessionWithHostKeyPrompt(input, resource, confirmLabel = 'Trust and continue') {
  try {
    return await apiStartSSHSession(input);
  } catch (error) {
    if (input.acceptHostKey || input.replaceHostKey) throw error;
    const retryInput = await confirmHostKeyChange(error, resource, confirmLabel);
    if (!retryInput) throw error;
    return await apiStartSSHSession({ ...input, ...retryInput });
  }
}

async function startSSHTunnelWithHostKeyPrompt(input, resource) {
	try {
		return await apiStartSSHTunnel(input);
	} catch (error) {
		const publicBind = parsePublicTunnelBindError(error);
		if (publicBind && !input.allowPublicBind) {
			const warning = input.type === 'socks'
				? `This exposes an unauthenticated SOCKS proxy on ${publicBind.host}. Anyone who can reach this address may use the tunnel.`
				: `This exposes the tunnel listener on ${publicBind.host} instead of loopback.`;
			const confirmed = await openConfirmModal({
				kicker: 'Public Tunnel',
				title: `Bind tunnel to ${publicBind.host}?`,
				message: `${warning}\n\nContinue only if this network exposure is intentional.`,
				confirmLabel: 'Bind Publicly',
			});
			if (!confirmed) throw error;
			return await startSSHTunnelWithHostKeyPrompt({ ...input, allowPublicBind: true }, resource);
		}
		if (input.acceptHostKey || input.replaceHostKey) throw error;
		const retryInput = await confirmHostKeyChange(error, resource, 'Trust and start tunnel');
		if (!retryInput) throw error;
		return await apiStartSSHTunnel({ ...input, ...retryInput });
  }
}

async function installSSHKeyWithHostKeyPrompt(input, resource) {
  try {
    return await apiInstallSSHKey(input);
  } catch (error) {
    if (input.acceptHostKey || input.replaceHostKey) throw error;
    const retryInput = await confirmHostKeyChange(error, resource, 'Trust and install key');
    if (!retryInput) throw error;
    return await apiInstallSSHKey({ ...input, ...retryInput });
  }
}

async function confirmHostKeyChange(error, resource, confirmLabel) {
  const hostKey = parseUnknownHostKeyError(error);
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  if (hostKey) {
    const confirmed = await openConfirmModal({
      kicker: 'SSH Host Key',
      title: `Trust ${resource.hostname}?`,
      message: `Verify the server fingerprint before continuing.\n\n${target}\nFingerprint: ${hostKey.fingerprint}`,
      confirmLabel,
    });
    return confirmed ? { acceptHostKey: true } : null;
  }

  const mismatch = parseHostKeyMismatchError(error);
  if (!mismatch) return null;
  const confirmed = await openConfirmModal({
    kicker: 'SSH Host Key Changed',
    title: `Replace trusted key for ${resource.hostname}?`,
    message: `The server identity changed. Verify the new fingerprint before continuing.\n\n${target}\nTrusted: ${mismatch.expected}\nNew: ${mismatch.actual}`,
    confirmLabel: 'Replace Trusted Key',
  });
  return confirmed ? { replaceHostKey: true } : null;
}

function connectErrorMessage(error, resource) {
	const detail = errorDetail(error);
	if (error?.stage === 'terminal-ui') {
		return `Could not open the terminal interface for ${resource.hostname}: ${detail}`;
	}
  if (isLocalResource(resource)) {
    return `Could not start local shell: ${detail || 'unknown error'}`;
  }
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  const unknownHostKey = parseUnknownHostKeyError(error);
  if (unknownHostKey) {
    return `Could not verify ${target}: SSH host key was not trusted.`;
  }
  const mismatch = parseHostKeyMismatchError(error);
  if (mismatch) {
    return `Could not verify ${target}: SSH host key changed. Expected ${mismatch.expected}, got ${mismatch.actual}.`;
  }
  if (isCredentialStoreError(error)) {
    return `Could not access the system keyring for ${resource.hostname}: ${detail}`;
  }
  if (isAuthError(error)) {
    return `Could not authenticate to ${target}. Enter the remote password or configure a valid SSH key.`;
  }
  if (/host key|knownhosts|known host/i.test(detail)) {
    return `Could not verify ${target}. Use "Skip host key verification" only if you accept the security risk.`;
  }
  if (/timeout|deadline|i\/o timeout|operation timed out|context canceled|context deadline exceeded/i.test(detail)) {
    return `Could not connect to ${target}: connection timed out.`;
  }
  if (/no route to host|network is unreachable|host is down/i.test(detail)) {
    return `Could not connect to ${target}: host is unreachable.`;
  }
  if (/connection refused/i.test(detail)) {
    return `Could not connect to ${target}: connection refused.`;
  }
  return `Could not connect to ${target}: ${detail || 'unknown error'}`;
}

function isAuthMessage(message) {
  return /authenticate|authentication|password|SSH key|permission denied/i.test(String(message ?? ''));
}

function keyInstallErrorMessage(error, resource) {
	const detail = errorDetail(error);
  const target = `${resource.user}@${resource.ip || resource.hostname}:${resource.port}`;
  const unknownHostKey = parseUnknownHostKeyError(error);
  if (unknownHostKey) {
    return `Could not verify ${target}: SSH host key was not trusted.`;
  }
  const mismatch = parseHostKeyMismatchError(error);
  if (mismatch) {
    return `Could not verify ${target}: SSH host key changed. Expected ${mismatch.expected}, got ${mismatch.actual}.`;
  }
  if (isAuthError(error)) {
    return `Could not connect to ${target} to install the SSH key. Enter the remote password or configure a valid existing key.`;
  }
  if (/host key|knownhosts|known host/i.test(detail)) {
    return `Could not verify ${target}. Use "Skip host key verification" only if you accept the security risk.`;
  }
  if (/timeout|deadline|i\/o timeout/i.test(detail)) {
    return `Could not reach ${target} to install the SSH key: connection timed out.`;
  }
  if (/connection refused/i.test(detail)) {
    return `Could not reach ${target} to install the SSH key: connection refused.`;
  }
  return `Could not install the SSH key on ${target}: ${detail || 'unknown error'}`;
}

function renderTunnelStatus() {
  const status = document.querySelector('#tunnel-status');
  const stop = document.querySelector('#stop-tunnel');
  const start = document.querySelector('#start-tunnel');
  if (!status || !stop || !start) return;

  const selected = findResource(state.selectedId)?.resource;
  const tunnel = selected && !isLocalResource(selected) ? tunnelForResource(selected.id) : null;
  status.hidden = !tunnel;
  status.textContent = tunnel
    ? `${tunnelLabel(tunnel.type)} active on ${tunnel.localAddress} -> ${tunnel.forwardTarget || tunnel.target}`
    : '';
  stop.disabled = state.busy || !tunnel;
  start.disabled = state.busy || Boolean(tunnel);
}

function tunnelLabel(type) {
  if (type === 'local') return 'Local tunnel';
  if (type === 'remote') return 'Remote tunnel';
  return 'SOCKS tunnel';
}

function findResource(id) {
  return findWorkspaceResource(state, id);
}

function sessionsForResource(resourceId) {
	return sessionsForResourceFromState(state.sessions, resourceId);
}

function realSessionsForResource(resourceId) {
	return realSessionsForResourceFromState(state.sessions, resourceId);
}

function pendingSessionForResource(resourceId) {
	return pendingSessionForResourceFromState(state.sessions, resourceId);
}

function preferredSessionForResource(resourceId) {
	return preferredSessionForResourceFromState(state.sessions, state.lastSessionByResource, resourceId);
}

function sessionTitle(hostname, ordinal) {
  return ordinal > 1 ? `${hostname} #${ordinal}` : hostname;
}

function terminalKindLabel(kind = 'ssh') {
  return kind === 'local' ? 'local' : 'ssh';
}

async function openInitialLocalSession() {
  if (!state.localShellSupported || state.sessions.size > 0) return;
  await quickConnect(localResource());
}

function tunnelForResource(resourceId) {
  if (resourceId === LOCAL_RESOURCE_ID) return null;
  return [...state.tunnels.values()].find((tunnel) => tunnel.resourceId === resourceId);
}

function tunnelsForResource(resourceId) {
  if (resourceId === LOCAL_RESOURCE_ID) return [];
  return [...state.tunnels.values()].filter((tunnel) => tunnel.resourceId === resourceId);
}

function focusSession(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session) return;
  if (!session.pending) clearPendingTabs(session.resourceId);
  setActiveSession(session.id);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

async function reconnectClosedSession(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session?.closed) return;
  const resource = isLocalResource(session.resourceId)
    ? localResource()
    : findResource(session.resourceId)?.resource;
  if (!resource) {
    writeNotice(`Cannot reconnect ${session.title}: resource no longer exists.`);
    return;
  }
  const replacementSessionID = await quickConnect(resource, { failureSessionID: sessionID });
  if (replacementSessionID && state.sessions.has(sessionID)) {
    removeSessionFromUI(sessionID);
  }
}

function setActiveSession(sessionID) {
  const session = sessionID ? state.sessions.get(sessionID) : null;
  state.activeSessionId = session?.id ?? null;
  if (session) {
    state.selectedId = session.resourceId;
    rememberSessionFocus(session.id);
    if (session.kind !== 'webapp' && !session.pending && !session.closed) {
      state.lastSessionByResource.set(session.resourceId, session.id);
    }
  }
  return session;
}

function rememberSessionFocus(sessionID) {
	state.sessionFocusHistory = rememberFocus(state.sessionFocusHistory, sessionID);
}

function forgetSessionFocus(sessionID) {
  state.sessionFocusHistory = state.sessionFocusHistory.filter((id) => id !== sessionID);
}

function lastFocusedSessionID() {
	return lastFocusedSessionIdFromState(state.sessionFocusHistory, state.sessions);
}

async function withBusy(task) {
  if (state.busy) {
    writeNotice('Operation in progress, please wait.', 'warning');
    return;
  }

  state.busy = true;
  renderSelection();
  tagPanel.setBusy(true);
  resourcePanel.setBusy(true);
  webAppPanel.setBusy(true);
  connectPanel.setBusy(true);
  try {
    return await task();
  } catch (error) {
    writeNotice(`Error: ${error?.message ?? error}`, 'error');
    return null;
  } finally {
    state.busy = false;
    renderSelection();
    tagPanel.setBusy(false);
    resourcePanel.setBusy(false);
    webAppPanel.setBusy(false);
    connectPanel.setBusy(false);
  }
}

async function withConnectionBusy(resourceID, task) {
  if (state.connectingResourceIds.has(resourceID)) {
    writeNotice('A connection attempt is already in progress for this node.', 'warning');
    return null;
  }

  state.connectingResourceIds.add(resourceID);
  renderSelection();
  try {
    return await task();
  } finally {
    state.connectingResourceIds.delete(resourceID);
    renderSelection();
  }
}

function schedulePeriodicUpdateCheck() {
  const startupDelayMs = 1200;
  const periodicIntervalMs = 6 * 60 * 60 * 1000;
  window.setTimeout(runAutomaticUpdateCheck, startupDelayMs);
  window.setInterval(runAutomaticUpdateCheck, periodicIntervalMs);
}

async function runAutomaticUpdateCheck() {
  try {
    const info = await apiCheckForUpdate();
    localStorage.setItem('bashes.updateCheck.lastAt', String(Date.now()));
    if (!info?.updateAvailable) return;

    const latestVersion = String(info.latestVersion ?? '').trim();
    const notifiedVersion = localStorage.getItem('bashes.updateCheck.notifiedLatest');
    if (latestVersion && notifiedVersion === latestVersion) return;

    showUpdateModal(info, false);
    if (latestVersion) {
      localStorage.setItem('bashes.updateCheck.notifiedLatest', latestVersion);
    }
  } catch {
    // Keep update failures silent and retry at the next scheduled check.
  }
}

async function openSettingsPanel() {
  try {
    await loadWebAppBrowserSettings();
  } catch (error) {
    writeNotice(`Could not detect Web App browsers: ${errorDetail(error)}`, 'error');
  }
  await settingsPanel.open(settingsPanelData());
}

function closeSettingsPanel({ restoreTerminalSettings = true } = {}) {
  if (restoreTerminalSettings) applyTerminalSettings(state, { focus: false });
  settingsPanel.close();
  restoreTerminalFocusAfterOverlay();
}

function renderSettingsForm() {
  settingsPanel.update(settingsPanelData());
}

function settingsPanelData() {
  return createSettingsPanelData(state, TERMINAL_FONT_FAMILY_OPTIONS, {
    min: MIN_TERMINAL_FONT_SIZE,
    max: MAX_TERMINAL_FONT_SIZE,
  });
}

async function submitSettings(values) {
  state.terminalFontSize = clampNumber(
    values.terminalFontSize,
    MIN_TERMINAL_FONT_SIZE,
    MAX_TERMINAL_FONT_SIZE,
    DEFAULT_TERMINAL_FONT_SIZE,
  );
  state.terminalScrollback = clampNumber(values.terminalScrollback, 1000, 500000, DEFAULT_TERMINAL_SCROLLBACK);
  state.terminalFontFamily = values.terminalFontFamily || DEFAULT_TERMINAL_FONT_FAMILY;
  state.terminalCopyOnSelect = values.terminalCopyOnSelect;
  try {
    state.webAppBrowserSettings = await apiSaveWebAppBrowserSettings({
      preferredBrowserId: values.webAppBrowser,
      customBrowserPath: values.customBrowserPath.trim(),
    });
  } catch (error) {
    writeNotice(`Could not save browser settings: ${errorDetail(error)}`, 'error');
    return false;
  }
  saveTerminalSettings();
  applyTerminalSettings();
  writeNotice('Settings saved.');
  closeSettingsPanel({ restoreTerminalSettings: false });
  return true;
}

function previewTerminalAppearance(values) {
  const preview = terminalAppearancePreview(
    state,
    values.terminalFontSize,
    values.terminalFontFamily,
  );
  applyTerminalSettings(preview, { focus: false });
}

function applyTerminalSettings(settings = state, { focus = true } = {}) {
  for (const session of state.sessions.values()) {
    if (!session.terminal) continue;
    session.terminal.options.fontSize = settings.terminalFontSize;
    session.terminal.options.fontFamily = settings.terminalFontFamily;
    session.terminal.options.scrollback = settings.terminalScrollback;
  }
  scheduleTerminalFit();
  if (focus) focusActiveTerminal();
}

function saveTerminalSettings() {
	persistTerminalSettings(localStorage, state);
}

function adjustTerminalFontSize(delta) {
  state.terminalFontSize = clampNumber(
    state.terminalFontSize + delta,
    MIN_TERMINAL_FONT_SIZE,
    MAX_TERMINAL_FONT_SIZE,
    DEFAULT_TERMINAL_FONT_SIZE,
  );
  saveTerminalSettings();
  applyTerminalSettings();
  if (settingsPanel.isOpen()) renderSettingsForm();
}

function focusActiveTerminal() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.terminal) return;
  requestAnimationFrame(() => session.terminal.focus());
}

function restoreTerminalFocusAfterOverlay() {
  window.setTimeout(() => {
    requestAnimationFrame(() => {
      if (hasOpenBlockingOverlay()) return;
      focusActiveTerminal();
    });
  }, 0);
}

function hasOpenBlockingOverlay() {
  return Boolean(document.querySelector(
    '.slide-panel:not([hidden]), .confirm-modal:not([hidden]), .app-modal:not([hidden]), .file-transfer-modal:not([hidden])',
  ));
}

function fitActiveTerminal() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.fitAddon || !session.terminal) return;
  session.fitAddon.fit();
}

function scheduleTerminalFit() {
  if (terminalFitFrame) cancelAnimationFrame(terminalFitFrame);
  terminalFitFrame = requestAnimationFrame(() => {
    terminalFitFrame = 0;
    fitActiveTerminal();
    resizeActiveSession();
  });
}

function resizeActiveSession() {
  const session = state.sessions.get(state.activeSessionId);
  if (!session?.terminal || session.pending || session.closed) return;
  apiResizeSSHSession(session.id, session.terminal.cols, session.terminal.rows).catch(() => {});
}

function openEditContextMenu(event) {
  const target = editableContextTarget(event.target);
  if (!target) return;

  event.preventDefault();
  event.stopPropagation();
  state.editContextTarget = target;
  const menu = document.querySelector('#edit-context-menu');
  if (!menu) return;

  const canWrite = !target.readOnly && !target.disabled;
  menu.querySelector('[data-edit-command="cut"]').disabled = !canWrite;
  menu.querySelector('[data-edit-command="paste"]').disabled = !canWrite;
  menu.hidden = false;

  const rect = menu.getBoundingClientRect();
  const left = Math.min(event.clientX, window.innerWidth - rect.width - 8);
  const top = Math.min(event.clientY, window.innerHeight - rect.height - 8);
  menu.style.left = `${Math.max(8, left)}px`;
  menu.style.top = `${Math.max(8, top)}px`;
}

function editableContextTarget(target) {
  const element = target?.closest?.('input, textarea, [contenteditable="true"]');
  if (!element || element.closest('.terminal-pane')) return null;
  if (element.matches('input')) {
    const type = (element.getAttribute('type') || 'text').toLowerCase();
    if (['button', 'checkbox', 'color', 'file', 'hidden', 'radio', 'range', 'reset', 'submit'].includes(type)) return null;
  }
  return element;
}

async function runEditContextCommand(event) {
  const button = event.target.closest?.('[data-edit-command]');
  if (!button || button.disabled || !state.editContextTarget) return;
  event.preventDefault();
  event.stopPropagation();

  const target = state.editContextTarget;
  const command = button.dataset.editCommand;
  try {
    if (command === 'copy') {
      await writeClipboard(selectedEditableText(target));
    } else if (command === 'cut') {
      await writeClipboard(selectedEditableText(target));
      replaceEditableSelection(target, '');
    } else if (command === 'paste') {
      const text = await readClipboard();
      if (text) replaceEditableSelection(target, text);
    }
  } finally {
    target.focus();
    hideEditContextMenu();
  }
}

function hideEditContextMenu() {
  const menu = document.querySelector('#edit-context-menu');
  if (menu) menu.hidden = true;
  state.editContextTarget = null;
}

function selectedEditableText(target) {
  if (target.isContentEditable) {
    return String(globalThis.getSelection?.() ?? '');
  }
  const start = target.selectionStart ?? 0;
  const end = target.selectionEnd ?? start;
  return target.value.slice(start, end);
}

function replaceEditableSelection(target, text) {
  if (target.isContentEditable) {
    document.execCommand('insertText', false, text);
    target.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: 'insertText', data: text }));
    return;
  }
  const start = target.selectionStart ?? target.value.length;
  const end = target.selectionEnd ?? start;
  const before = target.value.slice(0, start);
  const after = target.value.slice(end);
  target.value = `${before}${text}${after}`;
  const next = start + text.length;
  target.setSelectionRange(next, next);
  target.dispatchEvent(new InputEvent('input', { bubbles: true, inputType: text ? 'insertText' : 'deleteByCut', data: text }));
}

function writeNotice(message, level = '') {
  const status = document.querySelector('#app-status');
  if (!status) return;
  status.textContent = message;
  status.title = message;
  const resolvedLevel = level || noticeLevel(message);
  recordNotice(message, resolvedLevel);
  notify(message, resolvedLevel);
}

function noticeLevel(message) {
  const text = String(message ?? '');
  if (/error|could not|failed|unable|invalid|mismatch|required|denied|refused|timeout|unreachable/i.test(text)) {
    return 'error';
  }
  if (/updated|added|imported|installed|generated|connected|started|stopped|active|saved/i.test(text)) {
    return 'success';
  }
  return 'info';
}

function recordNotice(message, level) {
  state.messageLog.unshift({
    time: new Date().toLocaleTimeString(),
    level,
    message: String(message ?? ''),
  });
  state.messageLog = state.messageLog.slice(0, 50);
}

function notify(message, level = 'info') {
  appOverlays.notify(message, level);
}

function showMessageLog() {
  const entries = state.messageLog.length > 0
    ? state.messageLog.map((entry) => [
      `${entry.time} ${entry.level.toUpperCase()}`,
      entry.message,
    ])
    : [['Log', 'No messages yet.']];
  showAppModal({
    kicker: 'Tools',
    title: 'Message Log',
    message: 'Latest Bashes messages.',
    details: entries,
  });
}

function registerSSHEvents() {
  onRuntimeEvent(RUNTIME_EVENT.sshOutput, (event) => {
    writeSSHOutput(event.sessionId, decodeSSHOutput(event));
  });
  onRuntimeEvent(RUNTIME_EVENT.sshStatus, (event) => {
    if (event?.message) writeNotice(event.message);
  });
  onRuntimeEvent(RUNTIME_EVENT.sshClosed, (event) => {
    const session = state.sessions.get(event.sessionId);
    if (!session) {
      state.pendingSSHOutput.delete(event.sessionId);
      return;
    }
    markSessionClosed(event.sessionId, event?.message);
    if (event?.message) writeNotice(event.message);
  });
  onRuntimeEvent(RUNTIME_EVENT.sshTunnelClosed, (event) => {
    if (event?.sessionId) {
      state.tunnels.delete(event.sessionId);
      renderHosts(searchInput.value);
      renderTunnelStatus();
    }
    if (event?.message) writeNotice(event.message);
  });
}

function registerAppEvents() {
  onRuntimeEvent(RUNTIME_EVENT.webAppClosed, (event) => {
    for (const [webAppID, runtime] of state.webAppRuntimes) {
      if (runtime.runtimeId === event?.runtimeId) state.webAppRuntimes.delete(webAppID);
    }
    renderHosts(searchInput.value);
    renderSelection();
    if (event?.message && /failed|error|closed:/i.test(event.message)) writeNotice(event.message, 'error');
  });
  onRuntimeEvent(RUNTIME_EVENT.webAppStatus, (event) => {
    if (event?.webAppId && event?.runtimeId) {
      const current = state.webAppRuntimes.get(event.webAppId) ?? {};
      state.webAppRuntimes.set(event.webAppId, { ...current, ...event });
      renderHosts(searchInput.value);
      renderSelection();
    }
    if (event?.message) writeNotice(event.message, event.error ? 'error' : undefined);
  });

  onRuntimeEvent(RUNTIME_EVENT.databaseExported, (event) => {
    if (event?.message) writeNotice(`${event.message} ${event.path ?? ''}`.trim());
  });
  onRuntimeEvent(RUNTIME_EVENT.databaseImported, async (event) => {
    clearAllSessionsFromUI();
    closeAllFileTransferWorkspaces();
    state.selectedId = null;
    await refreshHosts();
    await loadTunnels();
    writeNotice(event?.message ?? 'Database imported.');
  });
  onRuntimeEvent(RUNTIME_EVENT.hostsFileImported, async (event) => {
    await refreshHosts();
    const imported = event?.imported ?? 0;
    const skipped = event?.skipped ?? 0;
    writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
    restoreTerminalFocusAfterOverlay();
  });
  onRuntimeEvent(RUNTIME_EVENT.hostsFilePreview, (event) => {
    showHostsFileImportPreview(event);
  });
  onRuntimeEvent(RUNTIME_EVENT.openSettings, () => {
    openSettingsPanel();
  });
  onRuntimeEvent(RUNTIME_EVENT.openSSHKeys, () => {
    openKeysPanel({ allowInstall: false });
  });
  onRuntimeEvent(RUNTIME_EVENT.openAbout, (info) => showAboutModal(info));
  onRuntimeEvent(RUNTIME_EVENT.updateCheck, (event) => {
    if (event?.error) {
      showUpdateErrorModal(event.error);
      return;
    }
    if (event?.info) showUpdateModal(event.info, Boolean(event.manual));
  });
}

function clearAllSessionsFromUI() {
  for (const sessionID of [...state.sessions.keys()]) {
    removeSessionFromUI(sessionID);
  }
  state.pendingSSHOutput.clear();
  state.lastSessionByResource.clear();
  state.sessionFocusHistory = [];
  state.activeSessionId = null;
  renderTabs();
  renderSelection();
}

function showHostsFileImportPreview(result) {
  const hosts = result?.hosts ?? [];
  const details = [
    ['Hosts file', result?.path ?? ''],
    ['Default SSH user', result?.user ?? ''],
    ['Skipped duplicates', String(result?.skipped ?? 0)],
    ...hosts.map((host, index) => [
      `Host ${index + 1}`,
      `${host.user}@${host.hostname}:${host.port}${host.ip ? ` (${host.ip})` : ''}`,
    ]),
  ];

  showAppModal({
    kicker: 'Tools',
    title: 'Import From Hosts File',
    message: hosts.length > 0
      ? `${hosts.length} host${hosts.length === 1 ? '' : 's'} will be imported.`
      : 'No new remote hosts were found.',
    details,
    primaryLabel: hosts.length > 0 ? 'Import' : 'OK',
    primaryAction: hosts.length > 0 ? () => applyHostsFileImport() : null,
    secondaryLabel: hosts.length > 0 ? 'Cancel' : '',
  });
}

async function applyHostsFileImport() {
  await withBusy(async () => {
    const result = await apiImportFromHostsFile();
    await refreshHosts();
    const imported = result?.imported ?? 0;
    const skipped = result?.skipped ?? 0;
    writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
    restoreTerminalFocusAfterOverlay();
  });
}

function showAboutModal(info) {
  const appInfo = info ?? {};
  showAppModal({
    kicker: 'About',
    title: 'Bashes',
    message: 'Fast remote server session manager.',
    details: [
      ['Version', appInfo.version ?? 'dev'],
      ['Platform', `${appInfo.platform ?? 'unknown'}/${appInfo.arch ?? 'unknown'}`],
      ['Data file', appInfo.dataPath ?? ''],
    ],
    primaryLabel: 'README',
    primaryAction: () => openExternalURL(appInfo.readmeUrl ?? 'https://github.com/signoredellarete/bashes#readme'),
    secondaryLabel: 'Releases',
    secondaryAction: () => openExternalURL(appInfo.releasesUrl ?? 'https://github.com/signoredellarete/bashes/releases'),
  });
}

function showUpdateModal(info, manual = false) {
  if (!info) return;
  if (!info.updateAvailable && !manual) return;
  showAppModal({
    kicker: 'Updates',
    title: info.updateAvailable ? 'Update Available' : 'Bashes Is Up To Date',
    message: info.message ?? '',
    details: [
      ['Current version', info.currentVersion ?? 'dev'],
      ['Latest version', info.latestVersion ?? 'unknown'],
    ],
    primaryLabel: info.updateAvailable ? 'Open Releases' : 'OK',
    primaryAction: info.updateAvailable ? () => openExternalURL(info.releaseUrl ?? info.repoUrl) : null,
    secondaryLabel: info.updateAvailable ? 'Repository' : '',
    secondaryAction: info.updateAvailable ? () => openExternalURL(info.repoUrl) : null,
  });
}

function showUpdateErrorModal(message) {
  showAppModal({
    kicker: 'Updates',
    title: 'Could Not Check Updates',
    message,
    primaryLabel: 'Repository',
    primaryAction: () => openExternalURL('https://github.com/signoredellarete/bashes'),
    secondaryLabel: 'OK',
  });
}

function openExternalURL(url) {
  if (!url) return;
  if (!openRuntimeURL(url)) window.open(url, '_blank', 'noopener');
}

function openTerminalLink(event, value) {
  const url = externalHttpURL(value);
  if (!url) {
    writeNotice('Blocked an unsupported terminal link.', 'error');
    return;
  }
  event?.preventDefault?.();
  openExternalURL(url);
}

function writeSSHOutput(sessionID, data) {
  if (!sessionID || !data || data.length === 0) return;
  const session = state.sessions.get(sessionID);
  if (session?.terminal) {
    session.terminal.write(data);
    return;
  }

  const current = state.pendingSSHOutput.get(sessionID);
  const chunks = Array.isArray(current) ? current : current ? [current] : [];
  chunks.push(data);
  state.pendingSSHOutput.set(sessionID, trimPendingSSHOutput(chunks));
}

function flushPendingSSHOutput(sessionID) {
  const pending = state.pendingSSHOutput.get(sessionID);
  if (!pending) return;
  state.pendingSSHOutput.delete(sessionID);
  const session = state.sessions.get(sessionID);
  if (!session?.terminal) return;

  const chunks = Array.isArray(pending) ? pending : [pending];
  chunks.forEach((chunk) => session.terminal.write(chunk));
}

function decodeSSHOutput(event) {
  if (event?.bytes) return base64ToBytes(event.bytes);
  return event?.data ?? '';
}

function base64ToBytes(value) {
  const binary = atob(String(value));
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function trimPendingSSHOutput(chunks) {
  const limit = 262144;
  let total = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  while (chunks.length > 1 && total > limit) {
    const removed = chunks.shift();
    total -= removed.length;
  }
  if (total <= limit) return chunks;

  const first = chunks[0];
  const start = first.length - limit;
  chunks[0] = typeof first === 'string' ? first.slice(start) : first.slice(start);
  return chunks;
}
