<script>
  import { onMount } from 'svelte';
  import iconXRaw from '@tabler/icons/outline/x.svg?raw';
  import {
    fileTransferModalPosition,
    fileTransferModalSize,
  } from './shell-model.js';

  let { onready, onrequestclose, onbackground } = $props();

  const iconX = iconXRaw.replace('<svg ', '<svg class="ti" ');
  let visible = $state(false);
  let opened = $state(false);
  let title = $state('Files');
  let card;
  let header;
  let resizeHandle;
  let workspaceRoot;

  function open(nextTitle) {
    title = nextTitle || 'Files';
    visible = true;
    requestAnimationFrame(() => { opened = true; });
  }

  function hide() {
    opened = false;
    visible = false;
  }

  function resize(width, height) {
    if (!card) return;
    const size = fileTransferModalSize(width, height, window.innerWidth, window.innerHeight);
    card.style.width = `${size.width}px`;
    card.style.height = `${size.height}px`;
  }

  function clampSize() {
    if (!visible || !card?.style.width) return;
    resize(card.offsetWidth, card.offsetHeight);
  }

  function registerResize() {
    let resizeState = null;

    const start = (event) => {
      event.preventDefault();
      resizeState = {
        pointerId: event.pointerId,
        startX: event.clientX,
        startY: event.clientY,
        width: card.offsetWidth,
        height: card.offsetHeight,
      };
      resizeHandle.setPointerCapture(event.pointerId);
      document.body.classList.add('resizing-file-transfer');
    };
    const move = (event) => {
      if (!resizeState || resizeState.pointerId !== event.pointerId) return;
      resize(
        resizeState.width + event.clientX - resizeState.startX,
        resizeState.height + event.clientY - resizeState.startY,
      );
    };
    const stop = (event) => {
      if (!resizeState || resizeState.pointerId !== event.pointerId) return;
      resizeState = null;
      document.body.classList.remove('resizing-file-transfer');
    };

    resizeHandle.addEventListener('pointerdown', start);
    resizeHandle.addEventListener('pointermove', move);
    resizeHandle.addEventListener('pointerup', stop);
    resizeHandle.addEventListener('pointercancel', stop);
    return () => {
      resizeHandle.removeEventListener('pointerdown', start);
      resizeHandle.removeEventListener('pointermove', move);
      resizeHandle.removeEventListener('pointerup', stop);
      resizeHandle.removeEventListener('pointercancel', stop);
      document.body.classList.remove('resizing-file-transfer');
    };
  }

  function registerDrag() {
    let dragState = null;

    const start = (event) => {
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
    };
    const move = (event) => {
      if (!dragState || dragState.pointerId !== event.pointerId) return;
      const position = fileTransferModalPosition(
        event.clientX - dragState.offsetX,
        event.clientY - dragState.offsetY,
        dragState.width,
        dragState.height,
        window.innerWidth,
        window.innerHeight,
      );
      card.style.left = `${position.left}px`;
      card.style.top = `${position.top}px`;
    };
    const stop = (event) => {
      if (!dragState || dragState.pointerId !== event.pointerId) return;
      dragState = null;
      document.body.classList.remove('dragging-file-transfer');
    };

    header.addEventListener('pointerdown', start);
    header.addEventListener('pointermove', move);
    header.addEventListener('pointerup', stop);
    header.addEventListener('pointercancel', stop);
    return () => {
      header.removeEventListener('pointerdown', start);
      header.removeEventListener('pointermove', move);
      header.removeEventListener('pointerup', stop);
      header.removeEventListener('pointercancel', stop);
      document.body.classList.remove('dragging-file-transfer');
    };
  }

  onMount(() => {
    const cleanupResize = registerResize();
    const cleanupDrag = registerDrag();
    onready?.({
      open,
      hide,
      clampSize,
      isOpen: () => visible,
      workspaceRoot: () => workspaceRoot,
    });
    return () => {
      cleanupResize();
      cleanupDrag();
    };
  });
</script>

<section id="file-transfer-modal" class:open={opened} class="file-transfer-modal" hidden={!visible}>
  <div class="file-transfer-scrim"></div>
  <section bind:this={card} class="file-transfer-card">
    <header bind:this={header} class="file-transfer-header">
      <div>
        <p class="eyebrow">File Transfer</p>
        <h3 id="file-transfer-title">{title}</h3>
      </div>
      <div class="file-transfer-header-actions">
        <button id="file-transfer-background" class="secondary" type="button" title="Send transfer window to background" onclick={onbackground}>Background</button>
        <button class="close-panel" type="button" title="Close" onclick={onrequestclose}>{@html iconX}</button>
      </div>
    </header>
    <div bind:this={workspaceRoot} id="file-transfer-root" class="file-transfer-root"></div>
    <div bind:this={resizeHandle} id="file-transfer-resize-handle" class="file-transfer-resize-handle" title="Resize" aria-hidden="true"></div>
  </section>
</section>
