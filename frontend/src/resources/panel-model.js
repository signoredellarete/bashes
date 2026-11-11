export function resourcePanelData({ mode = 'host', parent = null, resource = null, catalog = [] } = {}) {
  const editing = mode === 'edit';
  const subsystem = mode === 'subsystem' || (editing && resource?.type !== 'host');
  const parentSummary = subsystem && parent
    ? `Parent: ${parent.hostname} (${parent.user}@${parent.ip || parent.hostname})`
    : '';
  return {
    mode,
    parentId: parent?.id ?? '',
    editId: editing ? resource?.id ?? '' : '',
    subsystem,
    title: editing ? `Edit ${subsystem ? 'subsystem' : 'host'}` : `Add ${subsystem ? 'subsystem' : 'host'}`,
    submitLabel: editing ? 'Save changes' : `Save ${subsystem ? 'subsystem' : 'host'}`,
    parentSummary,
    catalog,
    values: {
      type: subsystem ? resource?.type ?? 'vm' : '',
      hostname: resource?.hostname ?? '',
      ip: resource?.ip ?? '',
      port: resource?.port ?? 22,
      user: resource?.user ?? '',
      tags: resource?.tags ?? [],
    },
  };
}
