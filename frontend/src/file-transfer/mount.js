import { mount, unmount } from 'svelte';
import FileTransferApp from './FileTransferApp.svelte';

export function mountFileTransfer(target, props) {
  const instance = mount(FileTransferApp, { target, props });
  return () => unmount(instance);
}
