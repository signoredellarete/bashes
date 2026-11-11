<script>
  import { onMount, tick } from 'svelte';
  import iconServer from '@tabler/icons/outline/server-2.svg?raw';
  import iconX from '@tabler/icons/outline/x.svg?raw';
  import TagEditor from '../tags/TagEditor.svelte';
  import { canonicalTag } from '../resource-tags.js';
  import { normalizeTagSelection } from '../tags/editor-model.js';

  let { onready, onsubmit, oncreatetag, onrequestclose } = $props();

  let visible = $state(false);
  let opened = $state(false);
  let busy = $state(false);
  let panel = $state({});
  let values = $state({ type: '', hostname: '', ip: '', port: 22, user: '', tags: [] });
  let catalog = $state([]);
  let editorKey = $state(0);
  let hostnameInput = $state();

  async function open(data = {}) {
    panel = data;
    values = { ...data.values, tags: normalizeTagSelection(data.values?.tags) };
    catalog = data.catalog ?? [];
    editorKey += 1;
    visible = true;
    await tick();
    requestAnimationFrame(() => {
      opened = true;
      hostnameInput?.focus();
    });
  }

  function close() {
    opened = false;
    visible = false;
  }

  function setBusy(value) {
    busy = Boolean(value);
  }

  async function createTag(name) {
    const created = await oncreatetag?.(name);
    if (!created) return '';
    const key = canonicalTag(created);
    if (!catalog.some((tag) => tag.key === key)) {
      catalog = [...catalog, { key, name: created, count: 0 }];
    }
    return created;
  }

  function submit(event) {
    event.preventDefault();
    if (busy) return;
    onsubmit?.({
      mode: panel.mode,
      parentId: panel.parentId,
      editId: panel.editId,
      input: {
        hostname: values.hostname.trim(),
        ip: values.ip.trim(),
        port: Number.parseInt(String(values.port), 10),
        user: values.user.trim(),
        tags: [...values.tags],
        ...(panel.subsystem ? { type: values.type } : {}),
      },
    });
  }

  onMount(() => onready?.({ open, close, setBusy, isOpen: () => visible }));
</script>

<section id="resource-panel" class:open={opened} class="slide-panel" hidden={!visible}>
  <button class="panel-scrim" type="button" aria-label="Close resource panel" onclick={onrequestclose}></button>
  <form id="resource-form" class="panel-card resource-panel-card tag-editor-panel" onsubmit={submit}>
    <header class="panel-header resource-panel-header">
      <span class="resource-panel-heading-icon">{@html iconServer}</span>
      <div class="resource-panel-heading">
        <h3>{panel.title}</h3>
        <p>SSH session</p>
      </div>
      <button class="close-panel" type="button" title="Close" aria-label="Close" onclick={onrequestclose}>{@html iconX}</button>
    </header>

    <div class="resource-form-body">
      <p class="parent-summary" hidden={!panel.subsystem}>{panel.parentSummary}</p>

      <label hidden={!panel.subsystem}>
        <span>Type</span>
        <select bind:value={values.type} name="type">
          <option value="vm">VM</option>
          <option value="lxc">LXC</option>
          <option value="docker">Docker</option>
        </select>
      </label>

      <label>
        <span>Display name</span>
        <input bind:this={hostnameInput} bind:value={values.hostname} name="hostname" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
      </label>
      <div class="resource-endpoint-grid">
        <label>
          <span>Hostname / IP</span>
          <input bind:value={values.ip} class="monospace-input" name="ip" autocomplete="off" autocapitalize="none" autocorrect="off" spellcheck="false" required />
        </label>
        <label>
          <span>Port</span>
          <input bind:value={values.port} class="monospace-input" name="port" type="number" min="1" max="65535" required />
        </label>
      </div>
      <label>
        <span>Username</span>
        <input bind:value={values.user} class="monospace-input" name="user" autocomplete="username" autocapitalize="none" autocorrect="off" spellcheck="false" required />
      </label>

      <div class="resource-form-separator"></div>

      {#key editorKey}
        <TagEditor
          tags={values.tags}
          {catalog}
          {busy}
          emptySummary="No tags on this system"
          countLabel="on this system"
          onchange={(tags) => { values.tags = tags; }}
          oncreate={createTag}
        />
      {/key}
    </div>

    <footer class="panel-actions resource-panel-actions">
      <span>{values.tags.length === 0 ? 'No tags on this system' : `${values.tags.length} tag${values.tags.length === 1 ? '' : 's'} on this system`}</span>
      <button class="secondary" type="button" onclick={onrequestclose}>Cancel</button>
      <button type="submit" disabled={busy}>{panel.submitLabel}</button>
    </footer>
  </form>
</section>
