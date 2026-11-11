import { mount } from 'svelte';
import SSHKeysPanel from './SSHKeysPanel.svelte';

export function mountSSHKeysPanel(target, callbacks) {
  let controller;
  let pendingData;
  mount(SSHKeysPanel, {
    target,
    props: {
      onready: (api) => {
        controller = api;
        if (pendingData) controller.update(pendingData);
      },
      ...callbacks,
    },
  });
  return {
    close: () => controller?.close(),
    focus: (field) => controller?.focus(field),
    highlight: (value) => controller?.highlight(value),
    hidePrivate: () => controller?.hidePrivate(),
    open(data) {
      pendingData = data;
      controller?.open(data);
    },
    resetAddForms: () => controller?.resetAddForms(),
    resetInstall: (trust) => controller?.resetInstall(trust),
    setBusy: (value) => controller?.setBusy(value),
    setDirectory: (value) => controller?.setDirectory(value),
    setPrivateKey: (value) => controller?.setPrivateKey(value),
    setPublicKey: (value) => controller?.setPublicKey(value),
    setStatus: (area, message, kind) => controller?.setStatus(area, message, kind),
    show: (view) => controller?.show(view),
    update(data) {
      pendingData = data;
      controller?.update(data);
    },
  };
}
