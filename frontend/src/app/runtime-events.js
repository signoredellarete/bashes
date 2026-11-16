export function registerRuntimeEvents({
  on,
  events,
  state,
  decodeSSHOutput,
  writeSSHOutput,
  markSessionClosed,
  renderHosts,
  renderTunnelStatus,
  renderSelection,
  writeNotice,
  clearSessions,
  closeFileTransfers,
  refreshHosts,
  loadTunnels,
  restoreFocus,
  showHostsFilePreview,
  openSettings,
  openSSHKeys,
  showAbout,
  showUpdate,
  showUpdateError,
}) {
  on(events.sshOutput, (event) => {
    writeSSHOutput(event.sessionId, decodeSSHOutput(event));
  });
  on(events.sshStatus, (event) => {
    if (event?.message) writeNotice(event.message);
  });
  on(events.sshClosed, (event) => {
    if (!state.sessions.has(event.sessionId)) {
      state.pendingSSHOutput.delete(event.sessionId);
      return;
    }
    markSessionClosed(event.sessionId, event?.message);
    if (event?.message) writeNotice(event.message);
  });
  on(events.sshTunnelClosed, (event) => {
    if (event?.sessionId) {
      state.tunnels.delete(event.sessionId);
      renderHosts();
      renderTunnelStatus();
    }
    if (event?.message) writeNotice(event.message);
  });

  on(events.webAppClosed, (event) => {
    for (const [webAppID, runtime] of state.webAppRuntimes) {
      if (runtime.runtimeId === event?.runtimeId) state.webAppRuntimes.delete(webAppID);
    }
    renderHosts();
    renderSelection();
    if (event?.message && /failed|error|closed:/i.test(event.message)) writeNotice(event.message, 'error');
  });
  on(events.webAppStatus, (event) => {
    if (event?.webAppId && event?.runtimeId) {
      const current = state.webAppRuntimes.get(event.webAppId) ?? {};
      state.webAppRuntimes.set(event.webAppId, { ...current, ...event });
      renderHosts();
      renderSelection();
    }
    if (event?.message) writeNotice(event.message, event.error ? 'error' : undefined);
  });

  on(events.databaseExported, (event) => {
    if (event?.message) writeNotice(`${event.message} ${event.path ?? ''}`.trim());
  });
  on(events.databaseImported, async (event) => {
    clearSessions();
    closeFileTransfers();
    state.selectedId = null;
    await refreshHosts();
    await loadTunnels();
    writeNotice(event?.message ?? 'Database imported.');
  });
  on(events.hostsFileImported, async (event) => {
    await refreshHosts();
    const imported = event?.imported ?? 0;
    const skipped = event?.skipped ?? 0;
    writeNotice(`Imported ${imported} host${imported === 1 ? '' : 's'} from hosts file. Skipped ${skipped}.`);
    restoreFocus();
  });
  on(events.hostsFilePreview, showHostsFilePreview);
  on(events.openSettings, openSettings);
  on(events.openSSHKeys, openSSHKeys);
  on(events.openAbout, showAbout);
  on(events.updateCheck, (event) => {
    if (event?.error) {
      showUpdateError(event.error);
      return;
    }
    if (event?.info) showUpdate(event.info, Boolean(event.manual));
  });
}
