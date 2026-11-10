<script>
  import { onMount, tick } from 'svelte';
  import iconCheck from '@tabler/icons/outline/check.svg?raw';
  import iconPencil from '@tabler/icons/outline/pencil.svg?raw';
  import iconTag from '@tabler/icons/outline/tag.svg?raw';
  import iconTags from '@tabler/icons/outline/tags.svg?raw';
  import iconTrash from '@tabler/icons/outline/trash.svg?raw';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { tagHue } from '../resource-tags.js';

  let { onready, oncreate, onrename, ondelete, ontoggle, onclear, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let tags = $state([]);
  let systemCount = $state(0);
  let activeFilters = $state([]);
  let busy = $state(false);
  let createName = $state('');
  let editingTagKey = $state(null);
  let renameValue = $state('');
  let createInput = $state();
  let renameInput = $state();

  function update(data = {}) {
    tags = data.tags ?? tags;
    systemCount = data.systemCount ?? systemCount;
    activeFilters = data.activeFilters ?? activeFilters;
    busy = data.busy ?? busy;
    if (editingTagKey && !tags.some((tag) => tag.key === editingTagKey)) cancelRename();
  }

  async function open(data) {
    update(data);
    editingTagKey = null;
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      createInput?.focus();
    });
  }

  function close() {
    opened = false;
    visible = false;
    editingTagKey = null;
  }

  function setBusy(value) {
    busy = Boolean(value);
  }

  async function createTag(event) {
    event.preventDefault();
    const name = createName.trim();
    if (!name || busy) return;
    if (await oncreate?.(name)) createName = '';
  }

  async function beginRename(tag) {
    editingTagKey = tag.key;
    renameValue = tag.name;
    await tick();
    renameInput?.focus();
    renameInput?.select();
  }

  function cancelRename() {
    editingTagKey = null;
    renameValue = '';
  }

  async function renameTag(event, tag) {
    event.preventDefault();
    const name = renameValue.trim();
    if (!name || busy) return;
    if (await onrename?.(tag, name)) cancelRename();
  }

  function selected(tag) {
    return activeFilters.includes(tag.key);
  }

  onMount(() => onready?.({
    open,
    close,
    update,
    setBusy,
    isOpen: () => visible,
  }));
</script>

<section id="tag-panel" class:open={opened} class="slide-panel tag-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close tag manager" onclick={onrequestclose}></button>
  <div class="panel-card tag-panel-card" role="dialog" aria-modal="true" aria-labelledby="tag-panel-title">
    <header class="panel-header tag-panel-header">
      <div class="tag-panel-heading">
        <span class="tag-panel-heading-icon">{@html iconTags}</span>
        <div>
          <h3 id="tag-panel-title">Manage tags</h3>
          <p id="tag-panel-subtitle">{tags.length} tag{tags.length === 1 ? '' : 's'} across {systemCount} system{systemCount === 1 ? '' : 's'}</p>
        </div>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close tag manager" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <form id="tag-create-form" class="tag-create-form" onsubmit={createTag}>
      <label class="tag-create-control">
        {@html iconTag}
        <input
          bind:this={createInput}
          bind:value={createName}
          name="name"
          maxlength="40"
          autocomplete="off"
          autocapitalize="none"
          autocorrect="off"
          spellcheck="false"
          placeholder="New tag name..."
          aria-label="New tag name"
          required
        />
      </label>
      <button id="tag-create-submit" type="submit" disabled={busy || createName.trim() === ''}>Create</button>
    </form>

    <div class="tag-list-heading">
      <span>All tags</span>
      <small id="tag-list-hint">Click a tag to filter</small>
    </div>
    <div id="tag-manager-list" class="tag-manager-list">
      {#if tags.length === 0}
        <p class="tag-manager-empty">Create the first tag to organize your connections.</p>
      {:else}
        {#each tags as tag (tag.key)}
          <div
            class:selected={selected(tag)}
            class="tag-manager-row"
            style={`--tag-hue: ${tagHue(tag.name)}`}
          >
            {#if editingTagKey === tag.key}
              <form class="tag-rename-form" onsubmit={(event) => renameTag(event, tag)}>
                <input bind:this={renameInput} bind:value={renameValue} maxlength="40" required aria-label={`Rename ${tag.name}`} />
                <button class="tag-row-action" type="submit" title="Save tag name" aria-label="Save tag name">{@html iconCheck}</button>
                <button class="tag-row-action" type="button" title="Cancel rename" aria-label="Cancel rename" onclick={cancelRename}>{@html iconX}</button>
              </form>
            {:else}
              <button
                class="tag-manager-select"
                type="button"
                aria-pressed={selected(tag)}
                title={`${selected(tag) ? 'Remove' : 'Apply'} ${tag.name} filter`}
                onclick={() => ontoggle?.(tag)}
              >
                <span class="tag-manager-swatch"></span>
                <strong>{tag.name}</strong>
                <small>{tag.count} system{tag.count === 1 ? '' : 's'}</small>
              </button>
              <div class="tag-manager-actions">
                <button class="tag-row-action" type="button" title={`Rename ${tag.name}`} aria-label={`Rename ${tag.name}`} onclick={() => beginRename(tag)}>{@html iconPencil}</button>
                <button class="tag-row-action danger-icon" type="button" title={`Delete ${tag.name}`} aria-label={`Delete ${tag.name}`} onclick={() => ondelete?.(tag)}>{@html iconTrash}</button>
              </div>
            {/if}
          </div>
        {/each}
      {/if}
    </div>

    <footer class="tag-panel-footer">
      <span id="tag-panel-status">{activeFilters.length > 0 ? `${activeFilters.length} active filter${activeFilters.length === 1 ? '' : 's'}` : 'No active filters'}</span>
      <div>
        <button id="tag-panel-clear" class="secondary" type="button" disabled={activeFilters.length === 0} onclick={onclear}>Clear filter</button>
        <button type="button" onclick={onrequestclose}>Done</button>
      </div>
    </footer>
  </div>
</section>
