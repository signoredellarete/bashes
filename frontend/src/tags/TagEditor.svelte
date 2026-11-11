<script>
  import iconPlus from '@tabler/icons/outline/plus.svg?raw';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import { canonicalTag, tagHue } from '../resource-tags.js';
  import {
    addTagSelection,
    normalizeTagSelection,
    removeTagSelection,
    tagEditorSuggestions,
  } from './editor-model.js';

  let {
    tags = [],
    catalog = [],
    busy = false,
    emptySummary = 'No tags',
    countLabel = '',
    onchange,
    oncreate,
  } = $props();

  let query = $state('');
  let creating = $state(false);
  let queryInput = $state();

  let suggestions = $derived(tagEditorSuggestions(catalog, tags, query));
  let selectedKeys = $derived(new Set(normalizeTagSelection(tags).map(canonicalTag)));
  let normalizedQuery = $derived(canonicalTag(query));
  let exactMatch = $derived((catalog ?? []).some((tag) => tag.key === normalizedQuery));

  function update(nextTags) {
    onchange?.(normalizeTagSelection(nextTags));
    query = '';
    requestAnimationFrame(() => queryInput?.focus());
  }

  function add(name) {
    update(addTagSelection(tags, name));
  }

  function remove(name) {
    update(removeTagSelection(tags, name));
  }

  async function create() {
    const name = query.trim();
    if (!name || creating || busy) return;
    creating = true;
    try {
      const created = await oncreate?.(name);
      if (created) add(created);
    } finally {
      creating = false;
    }
  }

  function handleKeydown(event) {
    if (event.key === 'Enter') {
      const name = query.trim();
      if (!name) return;
      event.preventDefault();
      const existing = (catalog ?? []).find((tag) => tag.key === canonicalTag(name));
      if (existing) add(existing.name);
      else void create();
    } else if (event.key === 'Backspace' && !event.currentTarget.value && tags.length > 0) {
      event.preventDefault();
      remove(tags.at(-1));
    }
  }
</script>

<section class="resource-tag-editor" aria-label="Tags">
  <header>
    <strong>Tags</strong>
    <span>type to search, Enter to add</span>
  </header>
  <div class="resource-tag-input">
    <div class="resource-selected-tags">
      {#each tags as tag (canonicalTag(tag))}
        <span class="resource-selected-tag" style={`--tag-hue: ${tagHue(tag)}`}>
          <span>{tag}</span>
          <button type="button" title={`Remove ${tag}`} aria-label={`Remove ${tag}`} onclick={() => remove(tag)}>{@html iconX}</button>
        </span>
      {/each}
    </div>
    <input
      bind:this={queryInput}
      bind:value={query}
      class="resource-tag-query"
      type="text"
      autocomplete="off"
      autocapitalize="none"
      autocorrect="off"
      spellcheck="false"
      placeholder={tags.length === 0 ? 'prod, milano...' : 'Add another...'}
      aria-label="Search or create tags"
      onkeydown={handleKeydown}
    />
  </div>
  <div class="resource-tag-suggestion-box">
    <span>{query ? 'Matching tags' : 'Existing tags'}</span>
    <div class="resource-tag-options">
      {#each suggestions as tag (tag.key)}
        <button
          class="resource-tag-option"
          type="button"
          title={`${tag.count} system${tag.count === 1 ? '' : 's'} tagged ${tag.name}`}
          style={`--tag-hue: ${tagHue(tag.name)}`}
          onclick={() => add(tag.name)}
        >{tag.name}</button>
      {/each}
      {#if normalizedQuery && !exactMatch && !selectedKeys.has(normalizedQuery)}
        <button class="resource-tag-create" type="button" disabled={busy || creating} onclick={create}>
          {@html iconPlus}<span>Create "{query.trim()}"</span>
        </button>
      {:else if suggestions.length === 0}
        <p class="resource-tag-options-empty">{catalog.length === selectedKeys.size ? 'All existing tags already added.' : 'No matching tags.'}</p>
      {/if}
    </div>
  </div>
</section>

