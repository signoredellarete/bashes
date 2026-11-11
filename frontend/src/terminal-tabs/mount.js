import { mount } from 'svelte';
import TerminalTabs from './TerminalTabs.svelte';

export function mountTerminalTabs(target, callbacks) {
  let controller;
  let pendingData;
  mount(TerminalTabs, {
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
    update(data) {
      pendingData = data;
      target.hidden = data.tabs.length === 0;
      controller?.update(data);
    },
  };
}
