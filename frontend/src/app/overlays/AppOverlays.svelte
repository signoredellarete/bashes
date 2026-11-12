<script>
  import { onDestroy, onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';

  let { onready, onclosed } = $props();

  let confirmVisible = $state(false);
  let confirmOpened = $state(false);
  let confirmKicker = $state('Confirm');
  let confirmTitle = $state('');
  let confirmMessage = $state('');
  let confirmLabel = $state('Confirm');
  let confirmDanger = $state(true);
  let confirmResolver = null;
  let confirmCancelButton;

  let appVisible = $state(false);
  let appOpened = $state(false);
  let appKicker = $state('Bashes');
  let appTitle = $state('Bashes');
  let appMessage = $state('');
  let appDetails = $state([]);
  let appPrimaryLabel = $state('OK');
  let appSecondaryLabel = $state('');
  let appPrimaryAction = null;
  let appSecondaryAction = null;
  let toastSequence = 0;
  let toasts = $state([]);
  const toastTimers = new Map();

  async function confirm(options = {}) {
    if (confirmResolver) confirmResolver(false);

    confirmKicker = options.kicker ?? 'Confirm';
    confirmTitle = options.title ?? '';
    confirmMessage = options.message ?? '';
    confirmLabel = options.confirmLabel ?? 'Confirm';
    confirmDanger = options.danger !== false;
    confirmVisible = true;
    await tick();
    requestAnimationFrame(() => {
      confirmOpened = true;
      confirmCancelButton?.focus();
    });

    return new Promise((resolve) => {
      confirmResolver = resolve;
    });
  }

  function resolveConfirm(confirmed) {
    if (!confirmVisible) return;
    confirmOpened = false;
    confirmVisible = false;
    onclosed?.();

    const resolve = confirmResolver;
    confirmResolver = null;
    resolve?.(confirmed);
  }

  async function show(options = {}) {
    appKicker = options.kicker ?? 'Bashes';
    appTitle = options.title ?? 'Bashes';
    appMessage = options.message ?? '';
    appDetails = options.details ?? [];
    appPrimaryLabel = options.primaryLabel ?? 'OK';
    appSecondaryLabel = options.secondaryLabel ?? '';
    appPrimaryAction = options.primaryAction;
    appSecondaryAction = options.secondaryAction;
    appVisible = true;
    await tick();
    requestAnimationFrame(() => {
      appOpened = true;
    });
  }

  function closeApp() {
    if (!appVisible) return;
    appOpened = false;
    appVisible = false;
    appPrimaryAction = null;
    appSecondaryAction = null;
    onclosed?.();
  }

  function runAppAction(action) {
    closeApp();
    action?.();
  }

  function notify(message, level = 'info') {
    if (level !== 'error') return;
    const id = ++toastSequence;
    toasts = [{ id, message: String(message ?? ''), level }, ...toasts];
    toastTimers.set(id, window.setTimeout(() => dismissToast(id), 10000));
  }

  function dismissToast(id) {
    const timer = toastTimers.get(id);
    if (timer) window.clearTimeout(timer);
    toastTimers.delete(id);
    toasts = toasts.filter((toast) => toast.id !== id);
  }

  onMount(() => onready?.({
    confirm,
    resolveConfirm,
    show,
    closeApp,
    notify,
    isConfirmOpen: () => confirmVisible,
    isAppOpen: () => appVisible,
  }));

  onDestroy(() => {
    for (const timer of toastTimers.values()) window.clearTimeout(timer);
  });
</script>

<section id="confirm-modal" class:open={confirmOpened} class="confirm-modal" hidden={!confirmVisible}>
  <button class="confirm-scrim" type="button" aria-label="Cancel" onclick={() => resolveConfirm(false)}></button>
  <div class="confirm-card" role="dialog" aria-modal="true" aria-labelledby="confirm-title" aria-describedby="confirm-message">
    <header>
      <p class="eyebrow" id="confirm-kicker">{confirmKicker}</p>
      <h3 id="confirm-title">{confirmTitle}</h3>
    </header>
    <p id="confirm-message">{confirmMessage}</p>
    <footer class="confirm-actions">
      <button bind:this={confirmCancelButton} id="confirm-cancel" class="secondary" type="button" onclick={() => resolveConfirm(false)}>Cancel</button>
      <button id="confirm-accept" class:danger={confirmDanger} type="button" onclick={() => resolveConfirm(true)}>{confirmLabel}</button>
    </footer>
  </div>
</section>

<div id="toast-stack" class="toast-stack" aria-live="polite" aria-relevant="additions">
  {#each toasts as toast (toast.id)}
    <article class={`toast toast-${toast.level}`}>
      <p>{toast.message}</p>
      <button type="button" title="Dismiss" aria-label="Dismiss" onclick={() => dismissToast(toast.id)}>{@html iconX}</button>
    </article>
  {/each}
</div>

<section id="app-modal" class:open={appOpened} class="app-modal" hidden={!appVisible}>
  <button class="app-modal-scrim" type="button" aria-label="Close" onclick={closeApp}></button>
  <div class="app-modal-card" role="dialog" aria-modal="true" aria-labelledby="app-modal-title" aria-describedby="app-modal-message">
    <header class="app-modal-header">
      <div>
        <p class="eyebrow" id="app-modal-kicker">{appKicker}</p>
        <h3 id="app-modal-title">{appTitle}</h3>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={closeApp}>{@html iconX}</button>
    </header>
    <p id="app-modal-message">{appMessage}</p>
    <dl id="app-modal-details" hidden={appDetails.length === 0}>
      {#each appDetails as detail}
        <dt>{detail[0]}</dt>
        <dd>{detail[1]}</dd>
      {/each}
    </dl>
    <footer class="app-modal-actions">
      <button
        id="app-modal-secondary"
        class="secondary"
        type="button"
        hidden={!appSecondaryLabel}
        onclick={() => runAppAction(appSecondaryAction)}
      >{appSecondaryLabel}</button>
      <button
        id="app-modal-primary"
        type="button"
        hidden={!appPrimaryLabel}
        onclick={() => runAppAction(appPrimaryAction)}
      >{appPrimaryLabel}</button>
    </footer>
  </div>
</section>
