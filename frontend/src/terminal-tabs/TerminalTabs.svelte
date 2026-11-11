<script>
  import { onMount, tick } from 'svelte';
  import iconX from '@tabler/icons/outline/x.svg?raw';

  let { onready, onselect, onactivate, onclose, ontooltip, onhidetooltip, onrender } = $props();
  let data = $state({ activeSessionId: '', tabs: [] });

  async function update(next) {
    data = next;
    await tick();
    onrender?.(data);
  }

  function close(event, tab) {
    event.stopPropagation();
    onclose?.(tab.id);
  }

  onMount(() => onready?.({ update }));
</script>

{#each data.tabs as tab (tab.id)}
  <div class:active={tab.active} class="session-tab-wrap" data-session-id={tab.id}>
    <button
      class:active={tab.active}
      class:closed={tab.closed}
      class:pending={tab.pending}
      class="session-tab"
      type="button"
      data-tooltip={tab.tooltip}
      data-tooltip-placement="bottom"
      onclick={() => onselect?.(tab.id)}
      ondblclick={() => onactivate?.(tab.id)}
      onmouseenter={(event) => ontooltip?.(event.currentTarget)}
      onfocus={(event) => ontooltip?.(event.currentTarget)}
      onmouseleave={onhidetooltip}
      onblur={onhidetooltip}
    >
      <span>{tab.label}</span>
      <strong>{tab.title}</strong>
    </button>
    <button
      class="tab-close"
      type="button"
      title={`Close ${tab.title}`}
      aria-label={`Close ${tab.title}`}
      onclick={(event) => close(event, tab)}
    >{@html iconX}</button>
  </div>
{/each}
