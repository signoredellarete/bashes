export function createFileTransferWorkspaceController({
  shell,
  createElement,
  mountWorkspace,
  confirmBackground,
  confirmCancellation,
  resolveConflict,
  notify = () => {},
  restoreFocus = () => {},
  onIndicatorChange = () => {},
}) {
  const workspaces = new Map();
  const backgroundResourceIds = new Set();
  let activeResourceId = null;

  async function open(resource) {
    const root = shell.workspaceRoot();
    if (!root) return false;

    shell.open(`${resource.user}@${resource.ip || resource.hostname}:${resource.port}`);
    let workspace = workspaces.get(resource.id);
    if (!workspace) {
      const element = createElement(resource);
      workspace = { active: false, element, resource, unmount: () => {} };
      element.addEventListener('bashes-file-transfer-active', (event) => {
        workspace.active = Boolean(event.detail?.active);
        renderIndicator();
      });
      root.append(element);
      workspaces.set(resource.id, workspace);
      try {
        workspace.unmount = await mountWorkspace(element, resource, {
          confirmCancellation,
          resolveConflict,
        });
      } catch (error) {
        destroy(resource.id);
        hide();
        throw error;
      }
    }

    show(resource.id);
    return true;
  }

  async function requestClose() {
    const workspace = workspaces.get(activeResourceId);
    if (workspace?.active) {
      if (!await confirmBackground()) return false;
      background();
      notify('File transfer continues in background.');
      return true;
    }
    if (workspace) destroy(activeResourceId);
    hide();
    return true;
  }

  function background() {
    if (activeResourceId) backgroundResourceIds.add(activeResourceId);
    hide();
    renderIndicator();
  }

  function hide() {
    shell.hide();
    restoreFocus();
  }

  function show(resourceId) {
    for (const [id, workspace] of workspaces) {
      workspace.element.hidden = id !== resourceId;
    }
    activeResourceId = resourceId;
    backgroundResourceIds.delete(resourceId);
    renderIndicator();
  }

  function destroy(resourceId) {
    if (!resourceId) return;
    const workspace = workspaces.get(resourceId);
    if (!workspace) return;
    workspace.unmount?.();
    workspace.element.remove();
    workspaces.delete(resourceId);
    backgroundResourceIds.delete(resourceId);
    if (activeResourceId === resourceId) activeResourceId = null;
    renderIndicator();
  }

  function closeAll() {
    for (const resourceId of [...workspaces.keys()]) destroy(resourceId);
    hide();
  }

  function renderIndicator() {
    const backgrounded = backgroundResourceIds.size > 0;
    const active = [...backgroundResourceIds].some(
      (resourceId) => workspaces.get(resourceId)?.active,
    );
    onIndicatorChange({ active, backgrounded });
  }

  function snapshot() {
    return {
      activeResourceId,
      backgroundResourceIds: [...backgroundResourceIds],
      workspaceResourceIds: [...workspaces.keys()],
    };
  }

  return { background, closeAll, destroy, hide, open, requestClose, show, snapshot };
}
