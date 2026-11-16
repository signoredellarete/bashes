const LAST_CHECK_KEY = 'bashes.updateCheck.lastAt';
const NOTIFIED_VERSION_KEY = 'bashes.updateCheck.notifiedLatest';

export function createUpdateController({
  checkForUpdate,
  storage,
  showModal,
  openURL,
  timers = globalThis,
  now = () => Date.now(),
  repositoryURL = 'https://github.com/signoredellarete/bashes',
}) {
  async function runAutomatic() {
    try {
      const info = await checkForUpdate();
      storage.setItem(LAST_CHECK_KEY, String(now()));
      if (!info?.updateAvailable) return;

      const latestVersion = String(info.latestVersion ?? '').trim();
      if (latestVersion && storage.getItem(NOTIFIED_VERSION_KEY) === latestVersion) return;
      show(info, false);
      if (latestVersion) storage.setItem(NOTIFIED_VERSION_KEY, latestVersion);
    } catch {
      // Automatic checks are retried at the next scheduled interval.
    }
  }

  function schedule({ startupDelayMs = 1200, intervalMs = 6 * 60 * 60 * 1000 } = {}) {
    timers.setTimeout(runAutomatic, startupDelayMs);
    timers.setInterval(runAutomatic, intervalMs);
  }

  function show(info, manual = false) {
    if (!info || (!info.updateAvailable && !manual)) return;
    showModal({
      kicker: 'Updates',
      title: info.updateAvailable ? 'Update Available' : 'Bashes Is Up To Date',
      message: info.message ?? '',
      details: [
        ['Current version', info.currentVersion ?? 'dev'],
        ['Latest version', info.latestVersion ?? 'unknown'],
      ],
      primaryLabel: info.updateAvailable ? 'Open Releases' : 'OK',
      primaryAction: info.updateAvailable ? () => openURL(info.releaseUrl ?? info.repoUrl) : null,
      secondaryLabel: info.updateAvailable ? 'Repository' : '',
      secondaryAction: info.updateAvailable ? () => openURL(info.repoUrl) : null,
    });
  }

  function showError(message) {
    showModal({
      kicker: 'Updates',
      title: 'Could Not Check Updates',
      message,
      primaryLabel: 'Repository',
      primaryAction: () => openURL(repositoryURL),
      secondaryLabel: 'OK',
    });
  }

  return { runAutomatic, schedule, show, showError };
}
