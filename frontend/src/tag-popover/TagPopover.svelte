<script>
  import { onDestroy, onMount } from 'svelte';
  import { Popup } from '@svar-ui/svelte-core';
  import Sortable from 'sortablejs';
  import { tagHue } from '../resource-tags.js';

  let {
    anchor,
    resourceId,
    tags = [],
    oncancel,
    onDragStart,
    onDragEnd,
  } = $props();

  let tagList;
  let sortable;

  onMount(() => {
    sortable = Sortable.create(tagList, {
      animation: 150,
      draggable: '.resource-tag',
      sort: false,
      group: { name: 'bashes-resource-tags', pull: 'clone', put: false, revertClone: true },
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      ghostClass: 'tag-drag-ghost',
      chosenClass: 'tag-drag-chosen',
      fallbackClass: 'tag-drag-mirror',
      onStart: () => onDragStart?.(),
      onEnd: () => onDragEnd?.(),
    });
  });

  onDestroy(() => sortable?.destroy());
</script>

<Popup parent={anchor} at="right-start" width="auto" css="bashes-tag-popover-shell" {oncancel}>
  <div
    bind:this={tagList}
    class="resource-tags tag-popover-tags"
    data-resource-id={resourceId}
    aria-label="Assigned tags"
  >
    {#each tags as tag (tag.key)}
      <span
        class="resource-tag"
        data-tag-key={tag.key}
        data-tag-name={tag.name}
        title={`Tag: ${tag.name}`}
        style={`--tag-hue:${tagHue(tag.name)}`}
      >{tag.name}</span>
    {/each}
  </div>
</Popup>
