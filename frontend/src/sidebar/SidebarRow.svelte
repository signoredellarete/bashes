<script>
  let { row, selectedId, tooltipDelay, onselect, onactivate, ontooltip, onhidetooltip, ontagpopover } = $props();

  function showTooltip(event) {
    ontooltip?.(event.currentTarget, tooltipDelay);
  }

  function toggleTagPopover(event) {
    event.preventDefault();
    event.stopPropagation();
    ontagpopover?.(event.currentTarget, row.id);
  }

  function handleOverflowKey(event) {
    if (event.key !== 'Enter' && event.key !== ' ') return;
    toggleTagPopover(event);
  }
</script>

<div
  class:child={row.depth > 0}
  class:local-row={row.local}
  class:webapp-row={row.webApp}
  class:selected={row.id === selectedId}
  class="host-row"
  data-id={row.id}
  style={`--tree-offset: ${row.depth * 18}px`}
>
  <button
    class:tunnel-active={row.tunnelActive}
    class:has-tags={row.tags.length > 0}
    class="host-select"
    type="button"
    aria-label={row.tooltip}
    data-tooltip={row.tooltip}
    data-tooltip-placement="right"
    onclick={() => onselect?.(row.id)}
    ondblclick={() => onactivate?.(row.id, row.type)}
    onmouseenter={showTooltip}
    onfocus={showTooltip}
    onmouseleave={onhidetooltip}
    onblur={onhidetooltip}
  >
    <span class="type">{row.typeLabel}</span>
    <span class="compact-name" aria-hidden="true">{row.compactName}</span>
    <span class="details">
      <span class="resource-name-line">
        <strong>{row.name}</strong>
        {#if row.tunnelActive}
          <span class="smart-chip tunnel-chip" aria-label={row.tunnelLabel}>tun</span>
        {/if}
      </span>
      <small>{row.target}</small>
    </span>
    <span
      class="resource-tags"
      aria-label="Connection tags"
      data-resource-id={row.id}
      data-visible-count={row.visibleTagCount}
    >
      {#each row.shownTags as tag (tag.key)}
        <span
          class:active={tag.active}
          class="resource-tag"
          data-tag-key={tag.key}
          data-tag-name={tag.name}
          title={`Tag: ${tag.name}`}
          style={`--tag-hue: ${tag.hue}`}
        >{tag.name}</span>
      {/each}
      {#if row.hiddenTags.length > 0}
        <span
          class="resource-tag overflow"
          data-resource-id={row.id}
          title={row.hiddenTags.map((tag) => tag.name).join(', ')}
          role="button"
          tabindex="0"
          aria-label={`Show all ${row.tags.length} tags`}
          aria-expanded="false"
          onclick={toggleTagPopover}
          onkeydown={handleOverflowKey}
        >+{row.hiddenTags.length}</span>
      {/if}
    </span>
    {#if !row.local}
      <span class="resource-tag-drop-target" data-resource-id={row.id} aria-hidden="true"></span>
    {/if}
  </button>
</div>
