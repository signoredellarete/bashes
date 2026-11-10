import { mount } from 'svelte';
import TagPanel from './TagPanel.svelte';

export function mountTagPanel(target, callbacks) {
  let controller;
  mount(TagPanel, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    open: (data) => controller?.open(data),
    close: () => controller?.close(),
    update: (data) => controller?.update(data),
    setBusy: (busy) => controller?.setBusy(busy),
    isOpen: () => controller?.isOpen() ?? false,
  };
}
