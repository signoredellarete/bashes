import { mount } from 'svelte';
import WebAppPanel from './WebAppPanel.svelte';

export function mountWebAppPanel(target, callbacks) {
  let controller;
  mount(WebAppPanel, {
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
