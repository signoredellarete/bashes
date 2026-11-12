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
import {
  errorDetail,
  isAuthError,
  isCredentialStoreError,
  parseHostKeyMismatchError,
  parsePublicTunnelBindError,
  parseUnknownHostKeyError,
} from './ssh-errors.js';
import {
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
import { revealSidebarResource, sidebarListInset } from './sidebar-scroll.js';
import { loadTunnelPreference, saveTunnelPreference } from './tunnel-preferences.js';
import { tunnelLabel, tunnelPanelData } from './tunnels/model.js';
import { mountTunnelPanel } from './tunnels/mount.js';
import { terminalTabsData, terminalKindLabel } from './terminal-tabs/model.js';
import { mountTerminalTabs } from './terminal-tabs/mount.js';
import { createTerminalTabSorting } from './terminal-tabs/sorting.js';
import { createTerminalController } from './terminal/controller.js';
import {
  canonicalTag,
  mergeTagCatalog,
  resourceTags,
} from './resource-tags.js';
import { isWebAppResource } from './web-apps/model.js';
import { mountSidebarResources } from './sidebar/mount.js';
import { resourceTooltip, sidebarTreeData } from './sidebar/model.js';
import { createSidebarSorting } from './sidebar/sorting.js';
import {
  installInputForKey,
  keyChoice,
  keyChoiceLabel,
  keyChoiceValue,
  keySource,
  sshKeysPanelData,
} from './ssh-keys/model.js';
import { mountSSHKeysPanel } from './ssh-keys/mount.js';
import { mountFileTransferShell } from './file-transfer/shell-mount.js';
import { mountWebAppAuthPanel } from './web-apps/auth-mount.js';
import { webAppAuthPanelData } from './web-apps/auth-model.js';
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
let sidebarRenderFrame = 0;
let tooltipTimer = 0;
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

  <div id="resource-panel-root" class="overlay-root"></div>

  <div id="webapp-panel-root" class="overlay-root"></div>

  <div id="webapp-auth-panel-root" class="overlay-root"></div>

  <div id="connect-panel-root" class="overlay-root"></div>

  <div id="tunnel-panel-root" class="overlay-root"></div>

  <div id="settings-root" class="overlay-root"></div>
  <div id="app-overlays-root" class="overlay-root"></div>

  <div id="file-transfer-shell-root" class="overlay-root"></div>

  <div id="ssh-keys-panel-root" class="overlay-root"></div>

  <div id="tag-panel-root" class="overlay-root"></div>

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
const fileTransferShell = mountFileTransferShell(document.querySelector('#file-transfer-shell-root'), {
  onrequestclose: () => closeFileTransferModal(),
  onbackground: () => backgroundFileTransferModal(),
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
const webAppAuthPanel = mountWebAppAuthPanel(document.querySelector('#webapp-auth-panel-root'), {
  onsubmit: (submission) => submitWebAppAuth(submission),
  onrequestclose: () => closeWebAppAuthPanel(),
});
const connectPanel = mountConnectPanel(document.querySelector('#connect-panel-root'), {
  onsubmit: (submission) => submitConnect(submission),
  onrequestclose: () => closeConnectPanel(),
});
const tunnelPanel = mountTunnelPanel(document.querySelector('#tunnel-panel-root'), {
  onsubmit: (submission) => submitTunnel(submission),
  onstop: (resourceId) => stopResourceTunnel(resourceId),
  onrequestclose: () => closeTunnelPanel(),
});
const sshKeysPanel = mountSSHKeysPanel(document.querySelector('#ssh-keys-panel-root'), {
  onrequestclose: () => closeKeysPanel(),
  onselect: (value) => selectSSHKey(value),
  ongenerate: (input) => submitGenerateKey(input),
  onimport: (input) => submitImportKeyPair(input),
  oninstall: (input) => submitInstallKey(input),
  onreadprivate: (value) => showSelectedPrivateKey(value),
  oncopypublic: (value) => copySelectedPublicKey(value),
  ondelete: (value) => deleteSelectedManagedKey(value),
  onsavedirectory: (directory) => submitKeyDirectory(directory),
  onbrowse: () => browseKeyDirectory(),
  onrescan: (announce) => rescanKeySources(announce),
});
const sidebarResources = mountSidebarResources(document.querySelector('#hosts'), {
  onselect: (resourceId) => selectSidebarResource(resourceId),
  onactivate: (resourceId, type) => activateSidebarResource(resourceId, type),
  ontooltip: (anchor, delay) => showTooltip(anchor, delay),
  onhidetooltip: () => hideTooltip(),
  ontagpopover: (anchor, resourceId) => toggleTagPopover(anchor, resourceId),
  onrender: (data) => finishSidebarRender(data),
});
const sidebarSorting = createSidebarSorting({
  onReorder: (order) => persistHostOrder(order),
  onTagAssignment: (resourceId, tagName, assigned) => updateResourceTagAssignment(resourceId, tagName, assigned),
  onHideTooltip: () => hideTooltip(),
  onClearDragOutline: (kind) => clearDragOutline(kind),
  onClearAllDragOutlines: () => clearAllDragOutlines(),
  onSuppressTagFilter: () => { suppressTagFilterClickUntil = Date.now() + 120; },
});
const terminalTabs = mountTerminalTabs(document.querySelector('#session-tabs'), {
  onselect: (sessionId) => selectTerminalTab(sessionId),
  onactivate: (sessionId) => activateTerminalTab(sessionId),
  onclose: (sessionId) => stopSession(sessionId),
  ontooltip: (anchor) => showTooltip(anchor),
  onhidetooltip: () => hideTooltip(),
  onrender: (data) => finishTerminalTabsRender(data),
});
const terminalTabSorting = createTerminalTabSorting({
  onReorder: (order) => reorderSessionTabs(order),
  onClearDragOutline: () => clearDragOutline('tabs'),
  onFocus: () => focusActiveTerminal(),
});

const stack = document.querySelector('#terminal-stack');
const hostsScroll = document.querySelector('#hosts-scroll');
const terminalController = createTerminalController({
  stack,
  sessions: () => state.sessions,
  activeSessionId: () => state.activeSessionId,
  settings: () => state,
  writeSession: (sessionId, data) => apiWriteSSHSession(sessionId, data),
  resizeSession: (sessionId, columns, rows) => apiResizeSSHSession(sessionId, columns, rows),
  readClipboard,
  writeClipboard,
  openLink: (event, url) => openTerminalLink(event, url),
  closeSession: (sessionId) => removeSessionFromUI(sessionId),
  reconnectSession: (sessionId) => reconnectClosedSession(sessionId),
  inputError: (kind, error) => writeNotice(`${terminalKindLabel(kind)} input error: ${error?.message ?? error}`),
});

window.addEventListener('resize', () => {
  scheduleTerminalFit();
  fileTransferShell.clampSize();
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
  const inventory = await apiGetSSHKeyInventory();
  state.keys = inventory.keys ?? [];
  state.keySources = inventory.sources ?? [];
  const data = keysPanelData();
  state.selectedKeyValue = data.selectedValue;
  sshKeysPanel.update(data);
  await renderSelectedKey();
}

async function loadKeySettings() {
  state.keySettings = await apiGetSSHKeySettings();
  renderKeysPanel();
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
      scheduleTerminalFit();
    } catch (error) {
      const message = connectErrorMessage(error, selected);
      connectPanel.setStatus(message, 'error');
      writeNotice(message);
      if (isAuthError(error)) connectPanel.focusAuth();
    }
  });
}

async function submitTunnel({ resourceId, input }) {
  const selected = findResource(resourceId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const active = tunnelForResource(selected.id);
  if (active) {
    writeNotice(`Tunnel already active on ${active.localAddress}.`);
    renderTunnelStatus();
    return;
  }
  await withBusy(async () => {
    try {
      const tunnel = await startSSHTunnelWithHostKeyPrompt(input, selected);
      state.tunnels.set(tunnel.tunnelId, tunnel);
      saveTunnelPreference(localStorage, selected.id, input);
      tunnelPanel.markPasswordSaved(input.savePassword === true);
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
        scheduleTerminalFit();
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
      scheduleTerminalFit();
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
          webAppAuthPanel.setStatus(message, 'error');
          webAppAuthPanel.focusAuth();
        }
        return false;
      }
      if (gateway && !promptAuth) {
        webAppAuthPanel.setStatus(connectErrorMessage(error, gateway), 'error');
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
  let hasSavedPassword = false;
  if (gateway.auth?.method === 'password') {
    try {
      hasSavedPassword = await apiHasSavedPassword(gateway.id);
    } catch {
      // The keyring error is reported if the user explicitly requests password storage.
    }
  }
  webAppAuthPanel.open(webAppAuthPanelData({
    webApp,
    gateway,
    keys: state.keys,
    hasSavedPassword,
    message,
  }));
}

async function submitWebAppAuth({ webAppId, gatewayId, auth }) {
  const webApp = findResource(webAppId)?.resource;
  const gateway = findResource(gatewayId)?.resource;
  if (!webApp || !gateway) return;
  webAppAuthPanel.setStatus('', '');
  try {
    const opened = await launchWebApp(webApp, auth, { promptAuth: false });
    if (opened) closeWebAppAuthPanel();
  } catch (error) {
    const message = connectErrorMessage(error, gateway);
    webAppAuthPanel.setStatus(message, 'error');
    if (isAuthError(error)) webAppAuthPanel.focusAuth();
  }
}

function closeWebAppAuthPanel() {
  webAppAuthPanel.close();
  restoreTerminalFocusAfterOverlay();
}

async function submitGenerateKey(input) {
  await withBusy(async () => {
    const key = await apiGenerateSSHKey(input);
    state.selectedKeyValue = keyChoiceValue({ ...key, source: 'bashes' });
    await loadKeys();
    sshKeysPanel.resetAddForms();
    sshKeysPanel.show('main');
    showKeyActionStatus(`Generated ${key.name}. The new key is selected.`, 'success');
    sshKeysPanel.highlight(state.selectedKeyValue);
  });
}

async function submitImportKeyPair(input) {
  setKeyImportStatus('Validating and importing key pair ...', 'pending');
  await withBusy(async () => {
    try {
      const key = await apiImportSSHKeyPair(input);
      state.selectedKeyValue = keyChoiceValue(key);
      await loadKeys();
      sshKeysPanel.resetAddForms();
      sshKeysPanel.show('main');
      showKeyActionStatus(`Imported ${key.name}. The new key is selected.`, 'success');
      sshKeysPanel.highlight(state.selectedKeyValue);
    } catch (error) {
      const message = `Could not import SSH key pair: ${error?.message ?? error}`;
      setKeyImportStatus(message, 'error');
      writeNotice(message);
    }
  });
}

async function submitKeyDirectory(directory) {
  await withBusy(async () => {
    try {
      state.keySettings = await apiSaveSSHKeySettings({ customDirectory: directory });
      await loadKeys();
      const customCount = state.keys.filter((key) => keySource(key) === 'custom').length;
      if (state.keySettings.customDirectory && customCount === 0) {
        setKeyDirectoryStatus('Saved, but no SSH keys were found in that directory.', 'pending');
      } else if (state.keySettings.customDirectory) {
        setKeyDirectoryStatus(`Saved. Found ${customCount} custom key${customCount === 1 ? '' : 's'}.`, 'success');
      } else {
        setKeyDirectoryStatus('Custom keys directory cleared.', 'success');
      }
    } catch (error) {
      const message = `Could not save custom keys directory: ${error?.message ?? error}`;
      setKeyDirectoryStatus(message, 'error');
      writeNotice(message);
      sshKeysPanel.focus('directory');
    }
  });
}

async function submitInstallKey(input) {
  const selected = findResource(state.selectedId)?.resource;
  const selectedKey = keyChoice(state.keys, input.selectedValue);
  if (!selected || isLocalResource(selected)) {
    setKeyInstallStatus('Select a host or subsystem before installing a key.', 'error');
    writeNotice('Select a host or subsystem before installing a key.');
    return;
  }
  if (!selectedKey) {
    setKeyInstallStatus('Select or generate an SSH key before installing it.', 'error');
    writeNotice('Select or generate an SSH key before installing it.');
    return;
  }

  const keyLabel = keyChoiceLabel(selectedKey);
  setKeyInstallStatus(`Installing key ${keyLabel} on ${selected.hostname} ...`, 'pending');
  await withBusy(async () => {
    try {
      await installSSHKeyWithHostKeyPrompt({
        resourceId: selected.id,
        ...installInputForKey(selectedKey),
        password: input.password,
        trustHostKey: input.trustHostKey,
      }, selected);
      sshKeysPanel.resetInstall(trustHostKeyFromPreference(selected));
      await refreshHosts();
      setKeyInstallStatus(`Installed key ${keyLabel} on ${selected.hostname}.`, 'success');
      writeNotice(`Installed key ${keyLabel} on ${selected.hostname}.`);
    } catch (error) {
      const message = keyInstallErrorMessage(error, selected);
      setKeyInstallStatus(message, 'error');
      writeNotice(message);
      sshKeysPanel.focus('install-password');
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

async function stopResourceTunnel(resourceId) {
  const tunnel = tunnelForResource(resourceId);
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
    session.element?.remove();
    const view = terminalController.createTerminal(sessionID, 'ssh', true);
    session.terminal = view.terminal;
    session.fitAddon = view.fitAddon;
    session.element = view.element;
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
  terminalController.dispose(session);
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
  closeTagPopover();
  clearDragOutline('hosts');
  sidebarSorting.destroy();
  renderTagFilters();
  void sidebarResources.update(sidebarTreeData({
    hosts: state.hosts,
    webApps: state.webApps,
    topLevelOrder: state.topLevelOrder,
    query: filter,
    activeTags: state.activeTagFilters,
    tagFilterMode: state.tagFilterMode,
    localShellSupported: state.localShellSupported,
    tunnels: state.tunnels.values(),
    webAppRuntimes: state.webAppRuntimes,
    selectedId: state.selectedId,
    sidebarCollapsed: state.sidebarCollapsed,
  }));
}

function finishSidebarRender(data) {
  if (sidebarRenderFrame) cancelAnimationFrame(sidebarRenderFrame);
  sidebarRenderFrame = requestAnimationFrame(() => {
    sidebarRenderFrame = 0;
    syncSidebarScrollbarInset();
    sidebarSorting.initialize({ container: document.querySelector('#hosts'), canReorder: data.canReorder });
  });
}

function selectSidebarResource(resourceId) {
  const selected = findResource(resourceId);
  if (selected) selectResource(selected.resource);
}

function activateSidebarResource(resourceId, type) {
  const selected = findResource(resourceId);
  if (!selected) return;
  state.selectedId = resourceId;
  if (type === 'webapp') {
    selectResource(selected.resource);
    launchWebApp(selected.resource);
  } else {
    createPendingTab(selected.resource);
    quickConnect(selected.resource);
  }
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
    onDragStart: () => sidebarSorting.beginTagSorting('assigned'),
    onDragEnd: () => sidebarSorting.finishTagSorting(),
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

async function persistHostOrder(order) {
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
  const pane = terminalController.createPendingPane(sessionID, isLocalResource(resource)
    ? 'Ready to start local shell'
    : `Ready to connect to ${resource.user}@${resource.ip || resource.hostname}:${resource.port}`);

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

  let view;
  try {
    view = terminalController.createTerminal(sessionID, kind);

    const nextSession = {
      id: sessionID,
      resourceId: resource.id,
      title: sessionTitle(resource.hostname, ordinal),
      ordinal,
      target: resourceTarget(resource),
      kind,
      terminal: view.terminal,
      fitAddon: view.fitAddon,
      element: view.element,
      closed: false,
    };
    if (replacedSession) {
      state.sessions = replaceSession(state.sessions, replaceSessionID, nextSession);
      state.pendingSSHOutput.delete(replaceSessionID);
      forgetSessionFocus(replaceSessionID);
      if (state.lastSessionByResource.get(replacedSession.resourceId) === replaceSessionID) {
        state.lastSessionByResource.delete(replacedSession.resourceId);
      }
      terminalController.dispose(replacedSession);
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
    terminalController.dispose(view);
    throw error;
  }
}

function renderTabs() {
  hideTooltip();
  clearDragOutline('tabs');
  terminalTabSorting.destroy();
  const empty = document.querySelector('#empty-terminal');
  const sessions = [...state.sessions.values()];
  empty.hidden = Boolean(state.activeSessionId);
  const resourceTooltips = new Map(sessions.flatMap((session) => {
    const resource = findResource(session.resourceId)?.resource;
    return resource ? [[session.resourceId, resourceTooltip(resource)]] : [];
  }));
  terminalTabs.update(terminalTabsData({
    sessions,
    activeSessionId: state.activeSessionId,
    resourceTooltips,
  }));

  terminalController.showActivePanes();
}

function finishTerminalTabsRender(data) {
  terminalTabSorting.initialize(document.querySelector('#session-tabs'), data.tabs.length > 1);
}

function selectTerminalTab(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session) return;
  if (!session.pending) clearPendingTabs(session.resourceId);
  setActiveSession(session.id);
  renderTabs();
  renderSelection();
  scheduleTerminalFit();
  focusActiveTerminal();
}

function activateTerminalTab(sessionID) {
  const session = state.sessions.get(sessionID);
  if (!session) return;
  if (session.kind === 'webapp' && session.pending) {
    const webApp = findResource(session.resourceId)?.resource;
    if (webApp) launchWebApp(webApp);
  } else if (session.closed) {
    reconnectClosedSession(session.id);
  }
}

function reorderSessionTabs(order) {
  state.sessions = orderSessions(state.sessions, order);
  renderTabs();
}

function schedulePointerDragOutline(event) {
  if (event.button !== 0) return;
  if (event.target.closest?.('.resource-tag, .tag-filter-chip')) return;
  const tab = event.target.closest?.('.session-tab');
  if (terminalTabSorting.isActive() && tab) {
    scheduleDragOutline('tabs', tab.closest('.session-tab-wrap'));
    return;
  }
  const host = event.target.closest?.('.host-block');
  if (sidebarSorting.hostSortingActive() && host) scheduleDragOutline('hosts', host);
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
  const preference = loadTunnelPreference(localStorage, selected.id);
  let hasSavedPassword = false;
  if (selected.auth?.method === 'password') {
    try {
      hasSavedPassword = await apiHasSavedPassword(selected.id);
    } catch {
      // The keyring error is reported if the user explicitly requests password storage.
    }
  }
  tunnelPanel.open(tunnelPanelData({
    resource: selected,
    keys: state.keys,
    preference,
    hasSavedPassword,
    tunnel: tunnelForResource(selected.id),
  }));
}

function closeTunnelPanel() {
  tunnelPanel.close();
  restoreTerminalFocusAfterOverlay();
}

async function openKeysPanel({ allowInstall = true } = {}) {
  state.keyInstallEnabled = allowInstall;

  await loadKeySettings();
  await loadKeys();
  sshKeysPanel.open(keysPanelData());
  await renderSelectedKey();
}

function closeKeysPanel() {
  sshKeysPanel.close();
  state.keyInstallEnabled = true;
  restoreTerminalFocusAfterOverlay();
}

async function openFileTransferModal() {
  if (!FILE_TRANSFER_ENABLED) return;
  const selected = findResource(state.selectedId)?.resource;
  if (!selected) return;
  if (isLocalResource(selected)) return;

  const root = fileTransferShell.workspaceRoot();
  if (!root) return;
  fileTransferShell.open(`${selected.user}@${selected.ip || selected.hostname}:${selected.port}`);

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
  fileTransferShell.hide();
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

function renderSelection() {
  sidebarResources.setSelection(state.selectedId);

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
  renderKeysPanel();
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

function keysPanelData() {
  const selected = findResource(state.selectedId)?.resource;
  const installTarget = selected && !isLocalResource(selected) && !isWebAppResource(selected) ? selected : null;
  return sshKeysPanelData({
    keys: state.keys,
    sources: state.keySources,
    settings: state.keySettings,
    selectedValue: state.selectedKeyValue,
    installTarget,
    installEnabled: state.keyInstallEnabled,
  });
}

function renderKeysPanel() {
  const data = keysPanelData();
  state.selectedKeyValue = data.selectedValue;
  sshKeysPanel.update(data);
}

async function selectSSHKey(value) {
  state.selectedKeyValue = value;
  renderKeysPanel();
  await renderSelectedKey();
}

async function browseKeyDirectory() {
  try {
    const directory = await apiChooseSSHKeyDirectory();
    if (directory) sshKeysPanel.setDirectory(directory);
  } catch (error) {
    const message = `Could not choose SSH keys directory: ${error?.message ?? error}`;
    setKeyDirectoryStatus(message, 'error');
    writeNotice(message);
  }
}

async function rescanKeySources(announce = true) {
  if (announce) setKeyDirectoryStatus('Scanning key sources ...', 'pending');
  try {
    await loadKeys();
    if (announce) setKeyDirectoryStatus('Key sources rescanned.', 'success');
  } catch (error) {
    const message = `Could not rescan key sources: ${error?.message ?? error}`;
    setKeyDirectoryStatus(message, 'error');
    writeNotice(message);
  }
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
  sshKeysPanel.hidePrivate();
  setKeyViewStatus('', '');
  const selectedKey = keyChoice(state.keys, state.selectedKeyValue);
  if (!selectedKey) {
    sshKeysPanel.setPublicKey('');
    return;
  }
  try {
    const publicKey = keySource(selectedKey) !== 'bashes'
      ? await apiReadSSHPublicKeyPath(selectedKey.publicKey || selectedKey.privateKey)
      : await apiReadSSHPublicKey(selectedKey.name);
    sshKeysPanel.setPublicKey(publicKey);
  } catch (error) {
    sshKeysPanel.setPublicKey('');
    const message = `Could not read SSH key ${keyChoiceLabel(selectedKey)}: ${error?.message ?? error}`;
    setKeyViewStatus(message, 'error');
    writeNotice(message);
  }
}

async function showSelectedPrivateKey(value) {
  const selectedKey = keyChoice(state.keys, value);
  if (!selectedKey) {
    setKeyViewStatus('Select an SSH key first.', 'error');
    return;
  }

  setKeyViewStatus('Reading private key ...', 'pending');
  try {
    const privateKey = keySource(selectedKey) === 'bashes'
      ? await apiReadSSHPrivateKey(selectedKey.name)
      : await apiReadSSHPrivateKeyPath(selectedKey.privateKey);
    sshKeysPanel.setPrivateKey(privateKey);
    setKeyViewStatus('', '');
  } catch (error) {
    sshKeysPanel.hidePrivate();
    const message = `Could not read private key ${keyChoiceLabel(selectedKey)}: ${error?.message ?? error}`;
    setKeyViewStatus(message, 'error');
    writeNotice(message);
  }
}

async function copySelectedPublicKey(publicKey) {
  const value = String(publicKey ?? '').trim();
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

async function deleteSelectedManagedKey(value) {
  const selectedKey = keyChoice(state.keys, value);
  if (!selectedKey || keySource(selectedKey) !== 'bashes') return;
  const usages = resourcesUsingManagedKey(selectedKey.name);
  const usageMessage = usages.length > 0
    ? `\n\nUsed by: ${usages.join(', ')}. Update those connections after deletion.`
    : '';
  const confirmed = await openConfirmModal({
    kicker: 'SSH Keys',
    title: `Delete ${selectedKey.name}?`,
    message: `The local public and private key files will be removed. Keys already installed on remote systems are not removed.${usageMessage}`,
    confirmLabel: 'Delete key',
  });
  if (!confirmed) return;

  await withBusy(async () => {
    try {
      await apiDeleteSSHKey(selectedKey.name);
      await loadKeys();
      showKeyActionStatus(`Deleted ${selectedKey.name}.`, 'success');
    } catch (error) {
      const message = `Could not delete SSH key: ${error?.message ?? error}`;
      setKeyViewStatus(message, 'error');
      writeNotice(message);
    }
  });
}

function showKeyActionStatus(message, kind) {
  sshKeysPanel.setStatus('action', message, kind);
}

function setKeyDirectoryStatus(message, kind) {
  sshKeysPanel.setStatus('directory', message, kind);
}

function setKeyImportStatus(message, kind) {
  sshKeysPanel.setStatus('import', message, kind);
}

function setKeyViewStatus(message, kind) {
  sshKeysPanel.setStatus('view', message, kind);
}

function setKeyInstallStatus(message, kind) {
  sshKeysPanel.setStatus('install', message, kind);
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
  const selected = findResource(state.selectedId)?.resource;
  const tunnel = selected && !isLocalResource(selected) ? tunnelForResource(selected.id) : null;
  tunnelPanel.setTunnel(tunnel);
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
  webAppAuthPanel.setBusy(true);
  connectPanel.setBusy(true);
  tunnelPanel.setBusy(true);
  sshKeysPanel.setBusy(true);
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
    webAppAuthPanel.setBusy(false);
    connectPanel.setBusy(false);
    tunnelPanel.setBusy(false);
    sshKeysPanel.setBusy(false);
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
  terminalController.applySettings(settings);
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
  terminalController.focusActive();
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

function scheduleTerminalFit() {
  terminalController.scheduleFit();
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
