export function createMenuActions({
  showModal,
  openRuntimeURL,
  openWindow,
  importHosts,
  refreshHosts,
  withBusy,
  writeNotice,
  restoreFocus,
}) {
  function openURL(url) {
    if (!url) return;
    if (!openRuntimeURL(url)) openWindow(url);
  }

  function showHostsPreview(result) {
    const hosts = result?.hosts ?? [];
    showModal({
      kicker: 'Tools',
      title: 'Import From Hosts File',
      message: hosts.length > 0
        ? `${hosts.length} host${hosts.length === 1 ? '' : 's'} will be imported.`
        : 'No new remote hosts were found.',
      details: [
        ['Hosts file', result?.path ?? ''],
        ['Default SSH user', result?.user ?? ''],
        ['Skipped duplicates', String(result?.skipped ?? 0)],
        ...hosts.map((host, index) => [
          `Host ${index + 1}`,
          `${host.user}@${host.hostname}:${host.port}${host.ip ? ` (${host.ip})` : ''}`,
        ]),
      ],
      primaryLabel: hosts.length > 0 ? 'Import' : 'OK',
      primaryAction: hosts.length > 0 ? applyHostsImport : null,
      secondaryLabel: hosts.length > 0 ? 'Cancel' : '',
    });
  }

  async function applyHostsImport() {
    await withBusy(async () => {
      const result = await importHosts();
      await refreshHosts();
      const imported = result?.imported ?? 0;
      const skipped = result?.skipped ?? 0;
      writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
      restoreFocus();
    });
  }

  function showAbout(info = {}) {
    showModal({
      kicker: 'About',
      title: 'Bashes',
      message: 'Fast remote server session manager.',
      details: [
        ['Version', info.version ?? 'dev'],
        ['Platform', `${info.platform ?? 'unknown'}/${info.arch ?? 'unknown'}`],
        ['Data file', info.dataPath ?? ''],
      ],
      primaryLabel: 'README',
      primaryAction: () => openURL(info.readmeUrl ?? 'https://github.com/signoredellarete/bashes#readme'),
      secondaryLabel: 'Releases',
      secondaryAction: () => openURL(info.releasesUrl ?? 'https://github.com/signoredellarete/bashes/releases'),
    });
  }

  return { applyHostsImport, openURL, showAbout, showHostsPreview };
}
