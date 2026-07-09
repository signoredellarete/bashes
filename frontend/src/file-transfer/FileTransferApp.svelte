<script>
  import { onDestroy, onMount } from 'svelte';
  import { Filemanager, WillowDark } from '@svar-ui/svelte-filemanager';
  import {
    cancelJob,
    closeFileTransfer,
    createItem,
    deleteItems,
    listJobs,
    listFiles,
    renameItem,
    resolveDroppedFilePaths,
    startCopyJob,
    startFileTransfer,
    startUploadJob,
  } from './api.js';

  export let resource;

  const roots = [
    { id: '/local', type: 'folder', lazy: true },
    { id: '/remote', type: 'folder', lazy: true },
  ];

  let api = null;
  let data = roots;
  let managerElement = null;
  let transferShell = null;
  let session = null;
  let status = 'Connecting...';
  let error = '';
  let busy = false;
  let needsPassword = false;
  let password = '';
  let trustHostKey = true;
  let passwordInput = null;
  let dragging = false;
  let dragDepth = 0;
  let cleanupDragEvents = () => {};
  let cleanupDraggableObserver = () => {};
  let cleanupJobEvents = () => {};
  let jobs = [];
  let jobRefreshTargets = new Map();
  let pointerDrag = null;
  let activeJobState = false;

  $: publishActiveTransferState(hasActiveJobs(jobs));

  onMount(() => {
    cleanupJobEvents = registerJobEvents();
    if (resource?.auth?.method === 'password') {
      showPasswordPrompt('');
      return;
    }
    startSession();
  });

  async function startSession(authInput = {}) {
    busy = true;
    error = '';
    try {
      session = await startFileTransfer({
        resourceId: resource.id,
        trustHostKey: true,
        ...authInput,
      });
      needsPassword = false;
      password = '';
      status = `Local: ${session.localRoot} | Remote: ${session.remoteRoot}`;
      await loadJobs();
      await loadInitial();
    } catch (err) {
      const message = String(err?.message ?? err);
      if (isAuthError(message)) {
        showPasswordPrompt(authInput.password ? 'Authentication failed. Check the password and try again.' : 'Enter the SSH password to open file transfer.');
      } else {
        error = message;
        status = 'Connection failed';
      }
    } finally {
      busy = false;
    }
  }

  onDestroy(() => {
    cleanupDragEvents();
    cleanupDraggableObserver();
    cleanupJobEvents();
    publishActiveTransferState(false);
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
    cleanupDragEvents();
    cleanupDragEvents = setupDragAndDrop();
    setupDraggableItems();
    loadInitial();
  }

  async function submitPassword(event) {
    event.preventDefault();
    if (!password.trim()) {
      error = 'Password is required.';
      return;
    }
    await startSession({
      password,
      trustHostKey,
    });
  }

  function showPasswordPrompt(message) {
    needsPassword = true;
    status = 'Password required';
    error = message;
    requestAnimationFrame(() => passwordInput?.focus());
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
      if (event.file?.file) {
        const paths = await resolveDroppedFilePaths([event.file.file]);
        const job = await startUploadJob(session.sessionId, paths, event.parent, false);
        trackJob(job, [event.parent]);
        status = `Uploading ${event.file.name}`;
        return;
      }
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
      if (!canTransferToTarget(event.ids, event.target)) {
        status = 'Choose a different destination';
        return;
      }
      const parents = unique([...event.ids.map(parentId), event.target]);
      const job = await startCopyJob(session.sessionId, event.ids, event.target, move);
      trackJob(job, parents);
      status = `${move ? 'Moving' : 'Copying'} ${event.ids.length} item${event.ids.length === 1 ? '' : 's'}`;
    });
  }

  async function handleDownloadFile(event) {
    await runOperation(async () => {
      if (event.id.startsWith('/remote/')) {
        const job = await startCopyJob(session.sessionId, [event.id], '/local', false);
        trackJob(job, ['/local']);
        status = `Downloading ${basename(event.id)} to local root`;
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

  function registerJobEvents() {
    const eventsOn = globalThis.runtime?.EventsOn;
    if (!eventsOn) return () => {};
    const off = eventsOn('file-transfer:job', (job) => {
      if (!job || job.resourceId !== resource.id) return;
      upsertJob(job);
      handleJobRefresh(job);
      updateStatusFromJob(job);
    });
    return typeof off === 'function' ? off : () => {};
  }

  async function loadJobs() {
    if (!session?.sessionId) return;
    try {
      jobs = await listJobs(session.sessionId);
    } catch {
      jobs = [];
    }
  }

  function trackJob(job, refreshTargets = []) {
    upsertJob(job);
    if (job?.jobId && refreshTargets.length) {
      jobRefreshTargets.set(job.jobId, unique(refreshTargets));
    }
  }

  function upsertJob(job) {
    if (!job?.jobId) return;
    const next = jobs.filter((item) => item.jobId !== job.jobId);
    next.unshift(job);
    jobs = next.slice(0, 8);
  }

  async function handleJobRefresh(job) {
    if (job.status !== 'completed' && job.status !== 'failed' && job.status !== 'canceled') return;
    const targets = jobRefreshTargets.get(job.jobId) ?? [job.targetId];
    jobRefreshTargets.delete(job.jobId);
    for (const target of targets.filter(Boolean)) {
      try {
        await provide(target);
      } catch {
        // The status event is still useful even if a folder refresh fails.
      }
    }
  }

  function dismissJob(jobId) {
    jobs = jobs.filter((job) => job.jobId !== jobId);
  }

  function updateStatusFromJob(job) {
    if (job.status === 'completed') {
      status = `${job.move ? 'Moved' : 'Copied'} ${job.sourceIds?.length || job.sourcePaths?.length || 1} item${(job.sourceIds?.length || job.sourcePaths?.length || 1) === 1 ? '' : 's'}`;
      return;
    }
    if (job.status === 'failed') {
      error = job.error || 'File transfer failed.';
      status = 'Transfer failed';
      return;
    }
    if (job.status === 'canceled') {
      status = 'Transfer canceled';
      return;
    }
    status = `${job.move ? 'Moving' : 'Copying'} ${formatBytes(job.transferredBytes)}${job.totalBytes ? ` / ${formatBytes(job.totalBytes)}` : ''}`;
  }

  async function cancelTransferJob(jobId) {
    await runOperation(async () => {
      await cancelJob(jobId);
      status = 'Cancel requested';
    });
  }

  function hasActiveJobs(items) {
    return items.some((job) => job.status === 'queued' || job.status === 'running');
  }

  function publishActiveTransferState(active) {
    if (active === activeJobState) return;
    activeJobState = active;
    transferShell?.dispatchEvent(new CustomEvent('bashes-file-transfer-active', {
      bubbles: true,
      detail: { resourceId: resource.id, active },
    }));
  }

  function progressValue(job) {
    if (job?.status === 'completed' && !job?.totalBytes) return 100;
    if (!job?.totalBytes) return 0;
    return Math.min(100, Math.round((job.transferredBytes / job.totalBytes) * 100));
  }

  function progressBarValue(job) {
    if (job?.status === 'completed' && !job?.totalBytes) return 1;
    return job?.transferredBytes || 0;
  }

  function progressBarMax(job) {
    return job?.totalBytes || 1;
  }

  function formatBytes(value) {
    const bytes = Number(value || 0);
    if (bytes < 1024) return `${bytes} B`;
    const units = ['KB', 'MB', 'GB', 'TB'];
    let current = bytes / 1024;
    let index = 0;
    while (current >= 1024 && index < units.length - 1) {
      current /= 1024;
      index += 1;
    }
    return `${current >= 10 ? current.toFixed(1) : current.toFixed(2)} ${units[index]}`;
  }

  function parentId(id) {
    const index = id.lastIndexOf('/');
    return index <= 0 ? id : id.slice(0, index);
  }

  function basename(id) {
    return id.slice(id.lastIndexOf('/') + 1);
  }

  function displayName(value) {
    const clean = String(value ?? '').replaceAll('\\', '/');
    const trimmed = clean.endsWith('/') ? clean.slice(0, -1) : clean;
    const name = trimmed.slice(trimmed.lastIndexOf('/') + 1);
    return name || trimmed || 'Transfer';
  }

  function unique(values) {
    return [...new Set(values)];
  }

  function isAuthError(message) {
    const normalized = message.toLowerCase();
    return normalized.includes('authenticate') ||
      normalized.includes('authentication') ||
      normalized.includes('no supported methods') ||
      normalized.includes('permission denied');
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
      const selectors = [
        '[data-id^=":/local/"]',
        '[data-id^=":/remote/"]',
        '[data-row-id^=":/local/"]',
        '[data-row-id^=":/remote/"]',
      ].join(', ');
      managerElement.querySelectorAll(selectors).forEach((item) => {
        item.draggable = true;
        item.setAttribute('draggable', 'true');
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
      ['pointerdown', handlePointerDown, true],
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
      cleanupPointerDrag();
    };
  }

  function handlePointerDown(event) {
    if (event.button !== 0 || event.target.closest('button, input, textarea, select, a')) return;
    const id = transferIdFromElement(event.target);
    if (!isTransferItem(id)) return;
    pointerDrag = {
      active: false,
      id,
      ids: selectedDragIds(id),
      pointerId: event.pointerId,
      shiftKey: event.shiftKey,
      startX: event.clientX,
      startY: event.clientY,
    };
    window.addEventListener('pointermove', handlePointerMove, true);
    window.addEventListener('pointerup', handlePointerUp, true);
    window.addEventListener('pointercancel', handlePointerCancel, true);
  }

  function handlePointerMove(event) {
    if (!pointerDrag || pointerDrag.pointerId !== event.pointerId) return;
    const distance = Math.hypot(event.clientX - pointerDrag.startX, event.clientY - pointerDrag.startY);
    if (!pointerDrag.active && distance < 8) return;
    if (!pointerDrag.active) {
      pointerDrag.active = true;
      dragging = true;
      dragDepth = 1;
      status = `Dragging ${pointerDrag.ids.length} item${pointerDrag.ids.length === 1 ? '' : 's'}`;
      document.body.classList.add('dragging-file-transfer-item');
    }
    pointerDrag.shiftKey = event.shiftKey;
    event.preventDefault();
    event.stopPropagation();
  }

  async function handlePointerUp(event) {
    if (!pointerDrag || pointerDrag.pointerId !== event.pointerId) return;
    const drag = pointerDrag;
    cleanupPointerDrag();
    if (!drag.active) return;
    event.preventDefault();
    event.stopPropagation();
    const targetElement = document.elementFromPoint(event.clientX, event.clientY);
    const target = dropTargetFromElement(targetElement);
    if (!target || !session?.sessionId) return;
    if (!canTransferToTarget(drag.ids, target)) {
      status = 'Choose a different destination';
      return;
    }
    await runOperation(async () => {
      const parents = unique([...drag.ids.map(parentId), target]);
      const job = await startCopyJob(session.sessionId, drag.ids, target, drag.shiftKey);
      trackJob(job, parents);
      status = `${drag.shiftKey ? 'Moving' : 'Copying'} ${drag.ids.length} item${drag.ids.length === 1 ? '' : 's'}`;
    });
  }

  function handlePointerCancel(event) {
    if (!pointerDrag || pointerDrag.pointerId !== event.pointerId) return;
    cleanupPointerDrag();
  }

  function cleanupPointerDrag() {
    pointerDrag = null;
    dragging = false;
    dragDepth = 0;
    document.body.classList.remove('dragging-file-transfer-item');
    window.removeEventListener('pointermove', handlePointerMove, true);
    window.removeEventListener('pointerup', handlePointerUp, true);
    window.removeEventListener('pointercancel', handlePointerCancel, true);
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
        if (!canTransferToTarget(ids, target)) {
          status = 'Choose a different destination';
          return;
        }
        const parents = unique([...ids.map(parentId), target]);
        const job = await startCopyJob(session.sessionId, ids, target, event.shiftKey);
        trackJob(job, parents);
        status = `${event.shiftKey ? 'Moving' : 'Copying'} ${ids.length} item${ids.length === 1 ? '' : 's'}`;
      });
      return;
    }

    const files = [...(event.dataTransfer.files || [])];
    if (files.length) {
      await runOperation(async () => {
        const paths = await resolveDroppedFilePaths(files);
        const job = await startUploadJob(session.sessionId, paths, target, false);
        trackJob(job, [target]);
        status = `Uploading ${files.length} file${files.length === 1 ? '' : 's'}`;
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
    return dropTargetFromElement(event.target);
  }

  function dropTargetFromElement(element) {
    const state = api?.getState();
    if (!state) return null;
    const panelElement = element?.closest?.('[data-panel]');
    if (!panelElement) return null;
    const panelIndex = Number(panelElement.dataset.panel);
    const panel = state.panels[panelIndex];
    if (!panel) return null;
    const id = transferIdFromElement(element);
    const item = id ? api.getFile(id) : null;
    return item?.type === 'folder' ? id : panel?.path;
  }

  function canTransferToTarget(ids, target) {
    if (!ids?.length || !target) return false;
    return ids.some((id) => {
      if (!id || id === target) return false;
      if (transferScope(id) === transferScope(target) && parentId(id) === target) return false;
      if (transferScope(id) === transferScope(target) && target.startsWith(`${id}/`)) return false;
      return true;
    });
  }

  function transferScope(id) {
    const parts = String(id || '').split('/');
    return parts.length > 1 ? parts[1] : '';
  }

  function transferIdFromElement(element) {
    const node = element?.closest?.('[data-id], [data-row-id]');
    const raw = node?.dataset?.id || node?.dataset?.rowId;
    if (!raw?.startsWith(':')) return null;
    return raw.slice(1);
  }

  function isTransferItem(id) {
    return id?.startsWith('/local/') || id?.startsWith('/remote/');
  }
</script>

<div class="transfer-shell" class:busy class:dragging bind:this={transferShell}>
  <div class="transfer-status">
    <span>{status}</span>
    {#if busy}<strong>Working...</strong>{/if}
  </div>
  {#if error}
    <p class="transfer-error">{error}</p>
  {/if}
  {#if jobs.length}
    <div class="transfer-jobs" aria-live="polite">
      {#each jobs.slice(0, 3) as job (job.jobId)}
        <article class:active={job.status === 'queued' || job.status === 'running'} class="transfer-job">
          <div class="transfer-job-header">
            <span>{job.status}</span>
            <strong>{progressValue(job)}%</strong>
          </div>
          <div class="transfer-job-current" title={job.current || job.sourceIds?.[0] || job.sourcePaths?.[0] || job.targetId}>
            {displayName(job.current || job.sourceIds?.[0] || job.sourcePaths?.[0] || job.targetId)}
          </div>
          <progress value={progressBarValue(job)} max={progressBarMax(job)}></progress>
          <div class="transfer-job-footer">
            <span>{formatBytes(job.transferredBytes)}{job.totalBytes ? ` / ${formatBytes(job.totalBytes)}` : ''}</span>
            {#if job.status === 'queued' || job.status === 'running'}
              <button type="button" onclick={() => cancelTransferJob(job.jobId)}>Cancel</button>
            {:else}
              {#if job.error}<span class="transfer-job-error" title={job.error}>{job.error}</span>{/if}
              <button type="button" onclick={() => dismissJob(job.jobId)}>Close</button>
            {/if}
          </div>
        </article>
      {/each}
    </div>
  {/if}
  {#if needsPassword && !session}
    <form class="transfer-auth" onsubmit={submitPassword}>
      <label>
        <span>User</span>
        <input value={resource.user} autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" readonly />
      </label>
      <label>
        <span>Password</span>
        <input bind:this={passwordInput} bind:value={password} type="password" autocomplete="current-password" disabled={busy} />
      </label>
      <label class="transfer-auth-check">
        <input bind:checked={trustHostKey} type="checkbox" disabled={busy} />
        <span>Trust host key for this transfer</span>
      </label>
      <button type="submit" disabled={busy}>{busy ? 'Connecting...' : 'Connect'}</button>
    </form>
  {:else if session}
    <div class="transfer-manager" bind:this={managerElement}>
      <WillowDark fonts={false}>
        <Filemanager
          {data}
          mode="panels"
          preview={false}
          icons={localIcon}
          panels={[
            { path: '/local', selected: [] },
            { path: '/remote', selected: [] },
          ]}
          activePanel={0}
          {init}
        />
      </WillowDark>
    </div>
  {:else}
    <div class="transfer-auth transfer-auth-placeholder">
      <p>Opening file transfer...</p>
    </div>
  {/if}
</div>
