import { mount } from 'svelte';
import SettingsPanel from './SettingsPanel.svelte';

export function mountSettingsPanel(target, callbacks) {
  let controller;
  mount(SettingsPanel, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    open: (data) => controller?.open(data),
    close: () => controller?.close(),
    update: (data) => controller?.update(data),
    isOpen: () => controller?.isOpen() ?? false,
  };
}
