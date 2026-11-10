import { mount } from 'svelte';
import WebAppWorkspace from './WebAppWorkspace.svelte';

export function mountWebAppWorkspace(target, callbacks) {
  let controller;
  mount(WebAppWorkspace, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    update(data) {
      target.hidden = !data?.visible;
      controller?.update(data);
    },
  };
}
