import { mount, unmount } from 'svelte';
import TagPopover from './TagPopover.svelte';

export function mountTagPopover(target, props) {
  const instance = mount(TagPopover, { target, props });
  return () => unmount(instance);
}
