<script>
  import { onMount, tick } from 'svelte';
  import iconArrowLeftRaw from '@tabler/icons/outline/arrow-left.svg?raw';
  import iconCopyRaw from '@tabler/icons/outline/copy.svg?raw';
  import iconEyeRaw from '@tabler/icons/outline/eye.svg?raw';
  import iconEyeOffRaw from '@tabler/icons/outline/eye-off.svg?raw';
  import iconFolderOpenRaw from '@tabler/icons/outline/folder-open.svg?raw';
  import iconKeyRaw from '@tabler/icons/outline/key.svg?raw';
  import iconPlusRaw from '@tabler/icons/outline/plus.svg?raw';
  import iconRefreshRaw from '@tabler/icons/outline/refresh.svg?raw';
  import iconSearchRaw from '@tabler/icons/outline/search.svg?raw';
  import iconTrashRaw from '@tabler/icons/outline/trash.svg?raw';
  import iconXRaw from '@tabler/icons/outline/x.svg?raw';
  import { keyChoice, keyGroups, keySource, keySourceLabel } from './model.js';

  let {
    onready, onrequestclose, onselect, ongenerate, onimport, oninstall, onreadprivate,
    oncopypublic, ondelete, onsavedirectory, onbrowse, onrescan,
  } = $props();

  const icon = (value) => value.replace('<svg ', '<svg class="ti" ');
  const iconArrowLeft = icon(iconArrowLeftRaw);
  const iconCopy = icon(iconCopyRaw);
  const iconEye = icon(iconEyeRaw);
  const iconEyeOff = icon(iconEyeOffRaw);
  const iconFolderOpen = icon(iconFolderOpenRaw);
  const iconKey = icon(iconKeyRaw);
  const iconPlus = icon(iconPlusRaw);
  const iconRefresh = icon(iconRefreshRaw);
  const iconSearch = icon(iconSearchRaw);
  const iconTrash = icon(iconTrashRaw);
  const iconX = icon(iconXRaw);

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let view = $state('main');
  let addMode = $state('generate');
  let search = $state('');
  let data = $state({ keys: [], sources: [], customDirectory: '', selectedValue: '', installTarget: null });
  let selectedValue = $state('');
  let publicKey = $state('');
  let privateKey = $state('');
  let privateVisible = $state(false);
  let customDirectory = $state('');
  let highlightValue = $state('');
  let statuses = $state({ action: null, import: null, directory: null, view: null, install: null });
  let selectedKey = $derived(keyChoice(data.keys, selectedValue));
  let groups = $derived(keyGroups(data.keys, search));

  async function update(next) {
    const selectionChanged = selectedValue !== next.selectedValue;
    data = next;
    selectedValue = next.selectedValue;
    customDirectory = next.customDirectory;
    if (selectionChanged) hidePrivate();
    await tick();
  }

  async function open(next) {
    visible = true;
    view = 'main';
    addMode = 'generate';
    search = '';
    statuses = { action: null, import: null, directory: null, view: null, install: null };
    resetAddForms();
    await update(next);
    requestAnimationFrame(() => { opened = true; });
  }

  function close() {
    opened = false;
    visible = false;
    view = 'main';
    hidePrivate();
    resetAddForms();
  }

  function select(value) {
    selectedValue = value;
    publicKey = '';
    hidePrivate();
    statuses.view = null;
    onselect?.(value);
  }

  function show(nextView) {
    view = nextView;
    hidePrivate();
    if (nextView === 'add') {
      addMode = 'generate';
      focus('add-name');
    }
  }

  function hidePrivate() {
    privateKey = '';
    privateVisible = false;
  }

  function togglePrivate() {
    if (privateVisible) {
      hidePrivate();
      statuses.view = null;
    } else if (selectedValue) {
      onreadprivate?.(selectedValue);
    }
  }

  function submitGenerate(event) {
    event.preventDefault();
    ongenerate?.({ name: new FormData(event.currentTarget).get('name')?.trim() ?? '' });
  }

  function submitImport(event) {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    onimport?.({
      name: values.get('name')?.trim() ?? '',
      privateKey: values.get('privateKey') ?? '',
      publicKey: values.get('publicKey') ?? '',
    });
  }

  function submitInstall(event) {
    event.preventDefault();
    const values = new FormData(event.currentTarget);
    oninstall?.({
      selectedValue,
      password: values.get('password') ?? '',
      trustHostKey: values.get('trustHostKey') === 'on',
    });
  }

  function submitDirectory(event) {
    event.preventDefault();
    onsavedirectory?.(customDirectory.trim());
  }

  function setStatus(area, message, kind = '') {
    statuses = { ...statuses, [area]: message ? { message, kind } : null };
  }

  function setPrivateKey(value) {
    privateKey = value;
    privateVisible = Boolean(value);
  }

  function highlight(value) {
    highlightValue = value;
    window.setTimeout(() => { if (highlightValue === value) highlightValue = ''; }, 1600);
  }

  function resetAddForms() {
    document.querySelector('#key-generate-form')?.reset();
    document.querySelector('#key-import-form')?.reset();
  }

  function resetInstall(trustHostKey = false) {
    const form = document.querySelector('#key-install-form');
    form?.reset();
    if (form) form.elements.trustHostKey.checked = trustHostKey;
  }

  function focus(field) {
    window.setTimeout(() => document.querySelector(`[data-key-focus="${field}"]`)?.focus(), 0);
  }

  onMount(() => onready?.({
    close, focus, highlight, hidePrivate, open, resetAddForms, resetInstall,
    setBusy: (value) => { busy = value; },
    setDirectory: (value) => { customDirectory = value ?? ''; },
    setPrivateKey,
    setPublicKey: (value) => { publicKey = value ?? ''; },
    setStatus,
    show,
    update,
  }));
</script>

<section id="keys-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close SSH keys" onclick={onrequestclose}></button>
  <div class="panel-card keys-panel-card" role="dialog" aria-modal="true" aria-labelledby="keys-panel-title">
    <header class="panel-header keys-panel-header">
      <div class="keys-panel-heading">
        <span class="keys-panel-heading-icon">{@html iconKey}</span>
        <div><h3 id="keys-panel-title">SSH Keys</h3><p>Keys available to Bashes</p></div>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close SSH keys" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <section class="keys-panel-view" hidden={view !== 'main'}>
      <div class="keys-toolbar">
        <label class="keys-search-control">{@html iconSearch}<input bind:value={search} type="search" autocomplete="off" placeholder="Search keys" aria-label="Search SSH keys" /></label>
        <button class="secondary keys-toolbar-action" type="button" disabled={busy} onclick={() => show('add')}>{@html iconPlus}<span>Add key</span></button>
        <button class="secondary keys-toolbar-action" type="button" disabled={busy} onclick={() => show('sources')}>{@html iconFolderOpen}<span>Sources</span></button>
      </div>
      {#if statuses.action}<p class="inline-status key-action-status" data-kind={statuses.action.kind}>{statuses.action.message}</p>{/if}
      <div class="key-list" aria-label="Available SSH keys">
        {#each groups as group (group.id)}
          <section class="key-list-group">
            <h4>{group.label}</h4>
            {#each group.keys as key (key.value)}
              <button
                class:selected={key.value === selectedValue}
                class:key-list-row-new={key.value === highlightValue}
                class="key-list-row"
                type="button"
                aria-pressed={key.value === selectedValue}
                onclick={() => select(key.value)}
              >
                {@html iconKey}<span class="key-list-row-text"><strong>{key.name}</strong><span>{key.privateKey || 'Managed key'}</span></span>
              </button>
            {/each}
          </section>
        {/each}
      </div>
      {#if groups.length === 0}<p class="keys-empty-state">{data.keys.length === 0 ? 'No SSH keys found.' : 'No keys match this search.'}</p>{/if}

      {#if selectedKey}
        <section class="key-details">
          <header class="key-details-header">
            <div><strong>{selectedKey.name}</strong><span>{keySourceLabel(selectedKey)}</span></div>
            <div class="key-detail-actions">
              <button class="secondary icon-action" type="button" title="Copy public key" aria-label="Copy public key" onclick={() => oncopypublic?.(publicKey)}>{@html iconCopy}</button>
              <button class="secondary icon-action" type="button" title={privateVisible ? 'Hide private key' : 'Show private key'} aria-label={privateVisible ? 'Hide private key' : 'Show private key'} aria-expanded={privateVisible} onclick={togglePrivate}>{@html privateVisible ? iconEyeOff : iconEye}</button>
              {#if keySource(selectedKey) === 'bashes'}<button class="secondary icon-action danger-icon" type="button" title="Delete managed key" aria-label="Delete managed key" onclick={() => ondelete?.(selectedValue)}>{@html iconTrash}</button>{/if}
            </div>
          </header>
          <p class="key-details-path" title={selectedKey.privateKey || selectedKey.publicKey || ''}>{selectedKey.privateKey || selectedKey.publicKey || ''}</p>
          <label><span>Public Key</span><textarea rows="5" readonly value={publicKey}></textarea></label>
          {#if privateVisible}<textarea rows="7" readonly aria-label="Private key" value={privateKey}></textarea>{/if}
          {#if statuses.view}<p class="inline-status" title={statuses.view.message} data-kind={statuses.view.kind}>{statuses.view.message}</p>{/if}

          {#if data.installTarget}
            <form id="key-install-form" class="key-install-form" onsubmit={submitInstall}>
              <p class="key-install-summary">Install selected key on {data.installTarget.label}</p>
              {#if statuses.install}<p class="inline-status" title={statuses.install.message} data-kind={statuses.install.kind}>{statuses.install.message}</p>{/if}
              <label><span>Remote Password</span><input data-key-focus="install-password" name="password" type="password" autocomplete="current-password" /></label>
              <label class="checkbox-row"><input name="trustHostKey" type="checkbox" /><span>Skip host key verification for this install (insecure)</span></label>
              <button type="submit" disabled={busy}>Install on selected host</button>
            </form>
          {/if}
        </section>
      {/if}
    </section>

    <section class="keys-panel-view keys-subview" hidden={view !== 'add'}>
      <header class="keys-subview-header">
        <button class="secondary keys-back-action" type="button" onclick={() => show('main')}>{@html iconArrowLeft}<span>Keys</span></button>
        <div><h4>Add key</h4><p>Create a key or import an existing pair.</p></div>
      </header>
      <div class="keys-mode-switch" role="tablist" aria-label="Add key method">
        <button class:secondary={addMode !== 'generate'} type="button" role="tab" aria-selected={addMode === 'generate'} onclick={() => { addMode = 'generate'; statuses.import = null; focus('add-name'); }}>Generate</button>
        <button class:secondary={addMode !== 'import'} type="button" role="tab" aria-selected={addMode === 'import'} onclick={() => { addMode = 'import'; statuses.import = null; focus('import-name'); }}>Import</button>
      </div>
      <form id="key-generate-form" class="key-add-form" hidden={addMode !== 'generate'} onsubmit={submitGenerate}>
        <header><strong>Generate an Ed25519 key</strong><span>The key pair is stored in Bashes' private data directory.</span></header>
        <label><span>Key Name <small>(optional)</small></span><input data-key-focus="add-name" name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Generated automatically" /></label>
        <button type="submit" disabled={busy}>Generate key</button>
      </form>
      <form id="key-import-form" class="key-add-form" hidden={addMode !== 'import'} onsubmit={submitImport}>
        <header><strong>Import a key pair</strong><span>Paste matching public and private SSH keys.</span></header>
        <label><span>Key Name <small>(optional)</small></span><input data-key-focus="import-name" name="name" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Generated automatically" /></label>
        <label><span>Private Key</span><textarea class="key-material-input" name="privateKey" rows="7" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required></textarea></label>
        <label><span>Public Key</span><textarea class="key-material-input" name="publicKey" rows="4" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required></textarea></label>
        {#if statuses.import}<p class="inline-status" title={statuses.import.message} data-kind={statuses.import.kind}>{statuses.import.message}</p>{/if}
        <button type="submit" disabled={busy}>Import key pair</button>
      </form>
    </section>

    <section class="keys-panel-view keys-subview" hidden={view !== 'sources'}>
      <header class="keys-subview-header">
        <button class="secondary keys-back-action" type="button" onclick={() => show('main')}>{@html iconArrowLeft}<span>Keys</span></button>
        <div><h4>Key sources</h4><p>Directories scanned for SSH private keys.</p></div>
      </header>
      <div class="key-source-list">
        {#each data.sources as source}
          <section class="key-source-row">
            <div><strong>{source.label || keySourceLabel(source)}</strong><span>{source.count ?? 0} key{source.count === 1 ? '' : 's'}</span></div>
            <p title={source.path || (source.source === 'custom' ? 'Not configured' : 'Not available')}>{source.path || (source.source === 'custom' ? 'Not configured' : 'Not available')}</p>
            {#if source.error}<p class="key-source-error">{source.error}</p>{/if}
          </section>
        {/each}
      </div>
      <form class="key-source-form" onsubmit={submitDirectory}>
        <label><span>Custom Keys Directory</span><input data-key-focus="directory" bind:value={customDirectory} name="directory" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" placeholder="Optional folder to scan" /></label>
        <div class="key-source-actions">
          <button class="secondary" type="button" onclick={onbrowse}>Browse</button>
          <button class="secondary" type="button" onclick={() => { customDirectory = ''; }}>Clear</button>
          <button type="submit" disabled={busy}>Save</button>
        </div>
      </form>
      {#if statuses.directory}<p class="inline-status" title={statuses.directory.message} data-kind={statuses.directory.kind}>{statuses.directory.message}</p>{/if}
      <button class="secondary keys-rescan-action" type="button" disabled={busy} onclick={() => onrescan?.(true)}>{@html iconRefresh}<span>Rescan sources</span></button>
    </section>
  </div>
</section>
