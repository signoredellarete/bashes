import { mount } from 'svelte';
import TagFilters from './TagFilters.svelte';

export function mountTagFilters(target, callbacks) {
  let controller;
  let pendingData;
  mount(TagFilters, {
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
      controller?.update(data);
    },
  };
}
