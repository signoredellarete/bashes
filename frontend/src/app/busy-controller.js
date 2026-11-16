export function createBusyController({ state, panels, renderSelection, writeNotice }) {
  async function run(task) {
    if (state.busy) {
      writeNotice('Operation in progress, please wait.', 'warning');
      return null;
    }
    state.busy = true;
    renderSelection();
    setPanelsBusy(true);
    try {
      return await task();
    } catch (error) {
      writeNotice(`Error: ${error?.message ?? error}`, 'error');
      return null;
    } finally {
      state.busy = false;
      renderSelection();
      setPanelsBusy(false);
    }
  }

  async function runConnection(resourceID, task) {
    if (state.connectingResourceIds.has(resourceID)) {
      writeNotice('A connection attempt is already in progress for this node.', 'warning');
      return null;
    }
    state.connectingResourceIds.add(resourceID);
    renderSelection();
    try {
      return await task();
    } finally {
      state.connectingResourceIds.delete(resourceID);
      renderSelection();
    }
  }

  function setPanelsBusy(busy) {
    for (const panel of panels) panel.setBusy?.(busy);
  }

  return { run, runConnection };
}
