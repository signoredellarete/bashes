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
    fileConflict: (options) => controller?.fileConflict(options),
    resolveFileConflict: (action) => controller?.resolveFileConflict(action),
    show: (options) => controller?.show(options),
    closeApp: () => controller?.closeApp(),
    notify: (message, level) => controller?.notify(message, level),
    isConfirmOpen: () => controller?.isConfirmOpen() ?? false,
    isFileConflictOpen: () => controller?.isFileConflictOpen() ?? false,
    isAppOpen: () => controller?.isAppOpen() ?? false,
  };
}
