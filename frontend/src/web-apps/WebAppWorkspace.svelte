<script>
  import { onMount } from 'svelte';
  import iconExternalLink from '@tabler/icons/outline/external-link.svg?raw';
  import iconPlugConnectedX from '@tabler/icons/outline/plug-connected-x.svg?raw';
  import iconWorld from '@tabler/icons/outline/world-www.svg?raw';
  import iconPlayerPlay from '@tabler/icons/filled/player-play.svg?raw';

  let { onready, onopen, ontoggletunnel } = $props();

  let name = $state('');
  let url = $state('');
  let mode = $state('');
  let status = $state('');
  let tunnelAvailable = $state(false);
  let tunnelActive = $state(false);
  let busy = $state(false);

  function update(data = {}) {
    name = data.name ?? '';
    url = data.url ?? '';
    mode = data.mode ?? '';
    status = data.status ?? '';
    tunnelAvailable = Boolean(data.tunnelAvailable);
    tunnelActive = Boolean(data.tunnelActive);
    busy = Boolean(data.busy);
  }

  onMount(() => onready?.({ update }));
</script>

<div class="webapp-workspace-icon">{@html iconWorld}</div>
<h3 id="webapp-workspace-name">{name}</h3>
<p id="webapp-workspace-url">{url}</p>
<p id="webapp-workspace-mode">{mode}</p>
<p id="webapp-workspace-status" class="webapp-workspace-status">{status}</p>
<div class="webapp-workspace-actions">
  <button id="webapp-workspace-open" type="button" onclick={onopen}>{@html iconExternalLink}<span>Open Web App</span></button>
  <button
    id="webapp-workspace-tunnel-toggle"
    class="secondary"
    type="button"
    hidden={!tunnelAvailable}
    disabled={busy || !tunnelAvailable}
    title={tunnelActive ? 'Stop Tunnel' : 'Start Tunnel'}
    aria-label={tunnelActive ? 'Stop Tunnel' : 'Start Tunnel'}
    onclick={ontoggletunnel}
  >
    {@html tunnelActive ? iconPlugConnectedX : iconPlayerPlay}
    <span>{tunnelActive ? 'Stop Tunnel' : 'Start Tunnel'}</span>
  </button>
</div>
