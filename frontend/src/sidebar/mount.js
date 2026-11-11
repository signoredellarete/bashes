import { mount } from 'svelte';
import SidebarResources from './SidebarResources.svelte';

export function mountSidebarResources(target, callbacks) {
  let controller;
  mount(SidebarResources, {
    target,
    props: { onready: (api) => { controller = api; }, ...callbacks },
  });
  return {
    update: (data) => controller?.update(data),
    setSelection: (id) => controller?.setSelection(id),
  };
}
