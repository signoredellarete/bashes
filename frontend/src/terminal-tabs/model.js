export function terminalKindLabel(kind = 'ssh') {
  return kind === 'local' ? 'local' : 'ssh';
}

export function terminalTabsData({ sessions = [], activeSessionId = '', resourceTooltips = new Map() } = {}) {
  return {
    activeSessionId,
    tabs: [...sessions].map((session) => ({
      id: session.id,
      resourceId: session.resourceId,
      title: session.title,
      kind: session.kind,
      pending: Boolean(session.pending),
      closed: Boolean(session.closed),
      active: session.id === activeSessionId,
      label: session.kind === 'webapp'
        ? 'webapp'
        : session.closed ? 'closed' : session.pending ? 'new' : terminalKindLabel(session.kind),
      tooltip: resourceTooltips.get(session.resourceId) ?? `${session.title} - ${session.target}`,
    })),
  };
}
