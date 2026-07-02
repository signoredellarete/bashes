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
  let managerElement = null;
  let session = null;
  let status = 'Connecting...';
  let error = '';
  let busy = false;
  let dragging = false;
  let dragDepth = 0;
  let cleanupDragEvents = () => {};
  let cleanupDraggableObserver = () => {};

  onMount(() => {
    cleanupDragEvents = setupDragAndDrop();
    startSession();
  });

  async function startSession() {
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
  }

  onDestroy(() => {
    cleanupDragEvents();
    cleanupDraggableObserver();
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
    setupDraggableItems();
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
    scheduleDraggableRefresh();
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

  function localIcon(file, size) {
    const type = file?.type === 'folder' ? 'folder' : 'file';
    const ext = type === 'file' ? String(file?.ext || '').slice(0, 5).toUpperCase() : '';
    const large = size === 'big';
    const width = large ? 96 : 24;
    const height = large ? 96 : 24;
    const labelSize = large ? 17 : 0;
    const fileSvg =
      `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 96 96">` +
      '<path fill="#eef4f8" stroke="#7f93a5" stroke-width="3" d="M24 8h33l17 17v63H24z"/>' +
      '<path fill="#d9e6ee" d="M57 8v18h17z"/>' +
      '<path fill="#5c7488" d="M33 47h30v5H33zm0 12h30v5H33zm0 12h20v5H33z"/>' +
      (large && ext ? `<text x="48" y="39" fill="#2f3945" font-family="Arial, sans-serif" font-size="${labelSize}" font-weight="700" text-anchor="middle">${escapeSvg(ext)}</text>` : '') +
      '</svg>';
    const folderSvg =
      `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 96 96">` +
      '<path fill="#f2b84b" d="M8 24h31l8 10h41v10H8z"/>' +
      '<path fill="#f7ca6b" stroke="#c88a28" stroke-width="3" d="M8 35h80l-7 45H15z"/>' +
      '<path fill="#ffe09a" d="M17 43h61l-2 10H15z"/>' +
      '</svg>';
    return `data:image/svg+xml,${encodeURIComponent(type === 'folder' ? folderSvg : fileSvg)}`;
  }

  function escapeSvg(value) {
    return value.replace(/[&<>"']/g, (char) => {
      const entities = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' };
      return entities[char];
    });
  }

  function setupDraggableItems() {
    cleanupDraggableObserver();
    if (!managerElement) return;
    const observer = new MutationObserver(scheduleDraggableRefresh);
    observer.observe(managerElement, { childList: true, subtree: true });
    cleanupDraggableObserver = () => observer.disconnect();
    scheduleDraggableRefresh();
  }

  function scheduleDraggableRefresh() {
    requestAnimationFrame(() => {
      if (!managerElement) return;
      managerElement.querySelectorAll('[data-id^=":/local/"], [data-id^=":/remote/"]').forEach((item) => {
        item.draggable = true;
      });
    });
  }

  function setupDragAndDrop() {
    requestAnimationFrame(() => setupDraggableItems());
    const target = () => managerElement;
    const listeners = [
      ['dragstart', handleDragStart, false],
      ['dragenter', handleDragEnter, true],
      ['dragleave', handleDragLeave, true],
      ['dragover', handleDragOver, true],
      ['drop', handleDrop, true],
    ];

    const attached = [];
    requestAnimationFrame(() => {
      if (!target()) return;
      for (const [eventName, handler, capture] of listeners) {
        target().addEventListener(eventName, handler, { capture });
        attached.push([eventName, handler, capture]);
      }
    });

    return () => {
      if (target()) {
        for (const [eventName, handler, capture] of attached) {
          target().removeEventListener(eventName, handler, { capture });
        }
      }
    };
  }

  function handleDragStart(event) {
    if (!api) return;
    const id = transferIdFromElement(event.target);
    if (!isTransferItem(id)) return;
    const ids = selectedDragIds(id);
    event.dataTransfer.setData('application/x-bashes-transfer-ids', JSON.stringify(ids));
    event.dataTransfer.effectAllowed = 'copyMove';
    status = `Dragging ${ids.length} item${ids.length === 1 ? '' : 's'}`;
  }

  function handleDragEnter(event) {
    if (!hasDroppablePayload(event)) return;
    dragDepth += 1;
    dragging = true;
  }

  function handleDragLeave(event) {
    if (!hasDroppablePayload(event)) return;
    dragDepth = Math.max(0, dragDepth - 1);
    dragging = dragDepth > 0;
  }

  function handleDragOver(event) {
    if (!hasDroppablePayload(event)) return;
    event.preventDefault();
    event.dataTransfer.dropEffect = event.shiftKey ? 'move' : 'copy';
  }

  async function handleDrop(event) {
    if (!hasDroppablePayload(event)) return;
    event.preventDefault();
    event.stopPropagation();
    dragDepth = 0;
    dragging = false;
    if (!session?.sessionId) return;

    const target = dropTarget(event);
    if (!target) return;
    const ids = transferIdsFromData(event.dataTransfer);
    if (ids.length) {
      await runOperation(async () => {
        await copyItems(session.sessionId, ids, target, event.shiftKey);
        await refreshAfterTransfer(ids, target, event.shiftKey);
        status = `${event.shiftKey ? 'Moved' : 'Copied'} ${ids.length} item${ids.length === 1 ? '' : 's'}`;
      });
      return;
    }

    const files = [...(event.dataTransfer.files || [])];
    if (files.length) {
      await runOperation(async () => {
        for (const file of files) {
          await createItem(session.sessionId, target, {
            name: file.name,
            type: 'file',
            size: file.size,
            file,
          });
        }
        await provide(target);
        status = `Uploaded ${files.length} file${files.length === 1 ? '' : 's'}`;
      });
    }
  }

  async function refreshAfterTransfer(ids, target, move) {
    const folders = move ? unique([...ids.map(parentId), target]) : [target];
    for (const folder of folders) await provide(folder);
  }

  function hasDroppablePayload(event) {
    const types = [...(event.dataTransfer?.types || [])];
    return types.includes('application/x-bashes-transfer-ids') || types.includes('Files');
  }

  function transferIdsFromData(dataTransfer) {
    try {
      return JSON.parse(dataTransfer.getData('application/x-bashes-transfer-ids') || '[]');
    } catch {
      return [];
    }
  }

  function selectedDragIds(id) {
    const state = api.getState();
    const panel = state.panels.find((item) => item.selected?.includes(id));
    return panel?.selected?.length ? panel.selected : [id];
  }

  function dropTarget(event) {
    const state = api?.getState();
    if (!state) return null;
    const panelElement = event.target.closest('[data-panel]');
    const panelIndex = Number(panelElement?.dataset.panel ?? state.activePanel ?? 0);
    const panel = state.panels[panelIndex];
    const id = transferIdFromElement(event.target);
    const item = id ? api.getFile(id) : null;
    return item?.type === 'folder' ? id : panel?.path;
  }

  function transferIdFromElement(element) {
    const node = element?.closest?.('[data-id]');
    const raw = node?.dataset?.id;
    if (!raw?.startsWith(':')) return null;
    return raw.slice(1);
  }

  function isTransferItem(id) {
    return id?.startsWith('/local/') || id?.startsWith('/remote/');
  }
</script>

<div class="transfer-shell" class:busy class:dragging>
  <div class="transfer-status">
    <span>{status}</span>
    {#if busy}<strong>Working...</strong>{/if}
  </div>
  {#if error}
    <p class="transfer-error">{error}</p>
  {/if}
  <div class="transfer-manager" bind:this={managerElement}>
    <Willow fonts={false}>
      <Filemanager
        {data}
        mode="panels"
        preview={true}
        icons={localIcon}
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
