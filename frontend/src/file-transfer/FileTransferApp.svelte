<script>
  import { onDestroy, onMount } from 'svelte';
  import { Filemanager, Willow } from '@svar-ui/svelte-filemanager';
  import {
    closeFileTransfer,
    copyItems,
    createItem,
    deleteItems,
    listFiles,
    renameItem,
    startFileTransfer,
  } from './api.js';

  export let resource;

  const roots = [
    { id: '/local', type: 'folder', lazy: true },
    { id: '/remote', type: 'folder', lazy: true },
  ];

  let api = null;
  let data = roots;
  let session = null;
  let status = 'Connecting...';
  let error = '';
  let busy = false;

  onMount(async () => {
    try {
      session = await startFileTransfer({
        resourceId: resource.id,
        trustHostKey: true,
      });
      status = `Local: ${session.localRoot} | Remote: ${session.remoteRoot}`;
      await loadInitial();
    } catch (err) {
      error = String(err?.message ?? err);
      status = 'Connection failed';
    }
  });

  onDestroy(() => {
    if (session?.sessionId) {
      closeFileTransfer(session.sessionId).catch(() => {});
    }
  });

  function init(instance) {
    api = instance;
    api.intercept('request-data', handleRequestData);
    api.intercept('create-file', handleCreateFile);
    api.intercept('rename-file', handleRenameFile);
    api.intercept('delete-files', handleDeleteFiles);
    api.intercept('copy-files', handleCopyFiles);
    api.intercept('move-files', handleMoveFiles);
    api.intercept('download-file', handleDownloadFile);
    api.intercept('open-file', handleOpenFile);
    loadInitial();
  }

  async function loadInitial() {
    if (!api || !session?.sessionId) return;
    await provide('/local');
    await provide('/remote');
  }

  async function handleRequestData(event) {
    await provide(event.id);
    return false;
  }

  async function handleCreateFile(event) {
    await runOperation(async () => {
      const created = await createItem(session.sessionId, event.parent, event.file);
      await provide(event.parent);
      await api.exec('select-file', { id: created.id });
      status = `Created ${event.file.name}`;
    });
    return false;
  }

  async function handleRenameFile(event) {
    await runOperation(async () => {
      const parent = parentId(event.id);
      await renameItem(session.sessionId, event.id, event.name);
      await provide(parent);
      status = `Renamed ${basename(event.id)} to ${event.name}`;
    });
    return false;
  }

  async function handleDeleteFiles(event) {
    await runOperation(async () => {
      const parents = unique(event.ids.map(parentId));
      await deleteItems(session.sessionId, event.ids);
      for (const parent of parents) await provide(parent);
      status = `Deleted ${event.ids.length} item${event.ids.length === 1 ? '' : 's'}`;
    });
    return false;
  }

  async function handleCopyFiles(event) {
    await runTransfer(event, false);
    return false;
  }

  async function handleMoveFiles(event) {
    await runTransfer(event, true);
    return false;
  }

  async function runTransfer(event, move) {
    await runOperation(async () => {
      const parents = unique([...event.ids.map(parentId), event.target]);
      await copyItems(session.sessionId, event.ids, event.target, move);
      for (const parent of parents) await provide(parent);
      status = `${move ? 'Moved' : 'Copied'} ${event.ids.length} item${event.ids.length === 1 ? '' : 's'}`;
    });
  }

  async function handleDownloadFile(event) {
    await runOperation(async () => {
      if (event.id.startsWith('/remote/')) {
        await copyItems(session.sessionId, [event.id], '/local', false);
        await provide('/local');
        status = `Copied ${basename(event.id)} to local root`;
      } else {
        status = 'Local files are already on this machine';
      }
    });
    return false;
  }

  function handleOpenFile() {
    status = 'Preview is not enabled for this transfer view';
    return false;
  }

  async function provide(id) {
    if (!api || !session?.sessionId) return;
    const files = await listFiles(session.sessionId, id);
    await api.exec('provide-data', { id, data: files });
  }

  async function runOperation(operation) {
    if (!session?.sessionId) return;
    busy = true;
    error = '';
    try {
      await operation();
    } catch (err) {
      error = String(err?.message ?? err);
    } finally {
      busy = false;
    }
  }

  function parentId(id) {
    const index = id.lastIndexOf('/');
    return index <= 0 ? id : id.slice(0, index);
  }

  function basename(id) {
    return id.slice(id.lastIndexOf('/') + 1);
  }

  function unique(values) {
    return [...new Set(values)];
  }
</script>

<div class="transfer-shell" class:busy>
  <div class="transfer-status">
    <span>{status}</span>
    {#if busy}<strong>Working...</strong>{/if}
  </div>
  {#if error}
    <p class="transfer-error">{error}</p>
  {/if}
  <div class="transfer-manager">
    <Willow fonts={false}>
      <Filemanager
        {data}
        mode="panels"
        preview={true}
        panels={[
          { path: '/local', selected: [] },
          { path: '/remote', selected: [] },
        ]}
        activePanel={0}
        {init}
      />
    </Willow>
  </div>
</div>
