<script>
  import { onMount, tick } from 'svelte';
  import SidebarRow from './SidebarRow.svelte';

  let { onready, onselect, onactivate, ontooltip, onhidetooltip, ontagpopover, onrender } = $props();
  let data = $state({ selectedId: '', tooltipDelay: 500, localRow: null, blocks: [], empty: false, canReorder: false });

  async function update(next) {
    data = next;
    await tick();
    onrender?.(data);
  }

  function setSelection(selectedId) {
    data = { ...data, selectedId };
  }

  onMount(() => onready?.({ update, setSelection }));
</script>

{#if data.localRow}
  <SidebarRow
    row={data.localRow}
    selectedId={data.selectedId}
    tooltipDelay={data.tooltipDelay}
    {onselect}
    {onactivate}
    {ontooltip}
    {onhidetooltip}
    {ontagpopover}
  />
{/if}
{#each data.blocks as block (block.id)}
  <div class:webapp-block={block.webApp} class="host-block" data-top-level-id={block.id}>
    {#each block.rows as row (row.id)}
      <SidebarRow
        {row}
        selectedId={data.selectedId}
        tooltipDelay={data.tooltipDelay}
        {onselect}
        {onactivate}
        {ontooltip}
        {onhidetooltip}
        {ontagpopover}
      />
    {/each}
  </div>
{/each}
{#if data.empty}
  <p class="host-filter-empty">No connection matches the current filters.</p>
{/if}
