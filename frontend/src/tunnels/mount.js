import { mount } from 'svelte';
import TunnelPanel from './TunnelPanel.svelte';

export function mountTunnelPanel(target, callbacks) {
  let controller;
  mount(TunnelPanel, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    open: (data) => controller?.open(data),
    close: () => controller?.close(),
    setBusy: (busy) => controller?.setBusy(busy),
    setTunnel: (tunnel) => controller?.setTunnel(tunnel),
    markPasswordSaved: (saved) => controller?.markPasswordSaved(saved),
    isOpen: () => controller?.isOpen() ?? false,
  };
}
