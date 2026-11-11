import { mount } from 'svelte';
import ResourcePanel from './ResourcePanel.svelte';

export function mountResourcePanel(target, callbacks) {
  let controller;
  mount(ResourcePanel, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    open: (data) => controller?.open(data),
    close: () => controller?.close(),
    setBusy: (busy) => controller?.setBusy(busy),
    isOpen: () => controller?.isOpen() ?? false,
  };
}
