export function sessionsForResource(sessions, resourceId) {
  return [...sessions.values()].filter((session) => session.resourceId === resourceId);
}

export function realSessionsForResource(sessions, resourceId) {
  return sessionsForResource(sessions, resourceId).filter((session) => !session.closed && !session.pending);
}

export function pendingSessionForResource(sessions, resourceId) {
  return sessionsForResource(sessions, resourceId).find((session) => !session.closed && session.pending);
}

export function preferredSessionForResource(sessions, lastSessionByResource, resourceId) {
  const lastSessionID = lastSessionByResource.get(resourceId);
  const lastSession = lastSessionID ? sessions.get(lastSessionID) : null;
  if (lastSession && !lastSession.closed && !lastSession.pending) return lastSession;

  const realSessions = realSessionsForResource(sessions, resourceId);
  return realSessions.at(-1) ?? pendingSessionForResource(sessions, resourceId);
}

export function orderSessions(sessions, orderedSessionIds) {
  const ordered = new Map();
  for (const id of orderedSessionIds) {
    const session = sessions.get(id);
    if (session) ordered.set(id, session);
  }
  for (const [id, session] of sessions) {
    if (!ordered.has(id)) ordered.set(id, session);
  }
  return ordered;
}

export function rememberFocus(history, sessionId) {
  return [...history.filter((id) => id !== sessionId), sessionId];
}

export function lastFocusedSessionId(history, sessions) {
  for (let index = history.length - 1; index >= 0; index -= 1) {
    if (sessions.has(history[index])) return history[index];
  }
  return sessions.keys().next().value ?? null;
}

export function closedSessionShortcut(event) {
  if (event?.type !== 'keydown' || event.repeat || event.isComposing) return '';
  if (!event.ctrlKey || event.metaKey || event.altKey) return '';
  switch (String(event.key).toLowerCase()) {
    case 'd':
      return 'close';
    case 'r':
      return 'reconnect';
    default:
      return '';
  }
}
