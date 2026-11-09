export const RUNTIME_EVENT = Object.freeze({
  sshOutput: 'ssh:output',
  sshStatus: 'ssh:status',
  sshClosed: 'ssh:closed',
  sshTunnelClosed: 'ssh:tunnel-closed',
  webAppStatus: 'webapp:status',
  webAppClosed: 'webapp:closed',
  fileTransferJob: 'file-transfer:job',
  databaseExported: 'database:exported',
  databaseImported: 'database:imported',
  hostsFileImported: 'database:hosts-file-imported',
  hostsFilePreview: 'database:hosts-file-preview',
  openSettings: 'app:settings',
  openSSHKeys: 'app:ssh-keys',
  openAbout: 'app:about',
  updateCheck: 'app:update-check',
});

export function onRuntimeEvent(name, handler, runtime = globalThis.runtime) {
  if (typeof runtime?.EventsOn !== 'function') return () => {};
  const unsubscribe = runtime.EventsOn(name, handler);
  return typeof unsubscribe === 'function' ? unsubscribe : () => {};
}

export async function writeClipboardText(text, runtime = globalThis.runtime, clipboard = globalThis.navigator?.clipboard) {
  if (typeof runtime?.ClipboardSetText === 'function') {
    const written = await runtime.ClipboardSetText(text);
    if (written !== false) return;
  }
  if (typeof clipboard?.writeText === 'function') {
    await clipboard.writeText(text);
  }
}

export async function readClipboardText(runtime = globalThis.runtime, clipboard = globalThis.navigator?.clipboard) {
  if (typeof runtime?.ClipboardGetText === 'function') {
    return await runtime.ClipboardGetText();
  }
  if (typeof clipboard?.readText === 'function') {
    return await clipboard.readText();
  }
  return '';
}

export function openRuntimeURL(url, runtime = globalThis.runtime) {
  if (typeof runtime?.BrowserOpenURL !== 'function') return false;
  runtime.BrowserOpenURL(url);
  return true;
}

export async function resolveRuntimeFilePaths(files, runtime = globalThis.runtime) {
  if (typeof runtime?.CanResolveFilePaths !== 'function' || typeof runtime?.ResolveFilePaths !== 'function') {
    return [];
  }
  if (!await runtime.CanResolveFilePaths()) return [];
  const paths = await runtime.ResolveFilePaths(files);
  return Array.isArray(paths) ? paths.filter(Boolean) : [];
}
