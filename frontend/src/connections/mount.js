import { mount } from 'svelte';
import ConnectPanel from './ConnectPanel.svelte';

export function mountConnectPanel(target, callbacks) {
  let controller;
  mount(ConnectPanel, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    open: (data) => controller?.open(data),
    close: () => controller?.close(),
    setBusy: (busy) => controller?.setBusy(busy),
    setStatus: (message, kind) => controller?.setStatus(message, kind),
    focusAuth: () => controller?.focusAuth(),
    isOpen: () => controller?.isOpen() ?? false,
  };
}
