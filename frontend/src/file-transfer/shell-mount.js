import { mount } from 'svelte';
import FileTransferShell from './Shell.svelte';

export function mountFileTransferShell(target, callbacks) {
  let controller;
  mount(FileTransferShell, {
    target,
    props: {
      onready: (api) => { controller = api; },
      ...callbacks,
    },
  });

  return {
    open: (title) => controller?.open(title),
    hide: () => controller?.hide(),
    clampSize: () => controller?.clampSize(),
    isOpen: () => controller?.isOpen() ?? false,
    workspaceRoot: () => controller?.workspaceRoot() ?? null,
  };
}
