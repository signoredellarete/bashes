<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { tagHue } from '../resource-tags.js';

  let { onready, ontoggle, onclear, onmode, onrender } = $props();

  let tags = $state([]);
  let activeFilters = $state([]);
  let mode = $state('any');
  let renderVersion = $state(0);

  async function update(data = {}) {
    tags = data.tags ?? [];
    activeFilters = data.activeFilters ?? [];
    mode = data.mode === 'all' ? 'all' : 'any';
    const version = ++renderVersion;
    await tick();
    if (version === renderVersion) onrender?.();
  }

  function active(tag) {
    return activeFilters.includes(tag.key);
  }

  onMount(() => onready?.({ update }));
</script>

<section id="tag-filters" class="tag-filters" aria-label="Filter connections by tag" hidden={tags.length === 0}>
  <header>
    <span>Filter by tag</span>
    <div class="tag-filter-mode" aria-label="Tag matching mode">
      <button class:active={mode === 'any'} id="tag-mode-any" type="button" title="Match any selected tag" aria-pressed={mode === 'any'} onclick={() => onmode?.('any')}>ANY</button>
      <button class:active={mode === 'all'} id="tag-mode-all" type="button" title="Match all selected tags" aria-pressed={mode === 'all'} onclick={() => onmode?.('all')}>ALL</button>
    </div>
  </header>
  <div id="tag-filter-chips" class="tag-filter-chips">
    {#key renderVersion}
      {#each tags as tag (tag.key)}
        <button
          class:active={active(tag)}
          class="tag-filter-chip"
          type="button"
          data-tag-key={tag.key}
          data-tag-name={tag.name}
          title={`${tag.count} connection${tag.count === 1 ? '' : 's'} tagged ${tag.name}`}
          aria-pressed={active(tag)}
          style={`--tag-hue: ${tagHue(tag.name)}`}
          onclick={() => ontoggle?.(tag.key)}
        >
          <span class="tag-dot"></span>
          <span>{tag.name}</span>
          <small>{tag.count}</small>
        </button>
      {/each}
      {#if activeFilters.length > 0}
        <button class="clear-tag-filters" type="button" title="Clear tag filters" onclick={onclear}>
          {@html iconX}<span>Clear</span>
        </button>
      {/if}
    {/key}
  </div>
</section>
