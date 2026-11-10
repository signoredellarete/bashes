import { mount } from 'svelte';
import AppOverlays from './AppOverlays.svelte';

export function mountAppOverlays(target, callbacks) {
  let controller;
  mount(AppOverlays, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    confirm: (options) => controller?.confirm(options),
    resolveConfirm: (confirmed) => controller?.resolveConfirm(confirmed),
    show: (options) => controller?.show(options),
    closeApp: () => controller?.closeApp(),
    isConfirmOpen: () => controller?.isConfirmOpen() ?? false,
    isAppOpen: () => controller?.isAppOpen() ?? false,
  };
}
