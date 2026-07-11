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

export function reorderSessions(sessions, draggedSessionId, targetSessionId, after) {
  const order = [...sessions.keys()];
  const from = order.indexOf(draggedSessionId);
  if (from < 0 || !order.includes(targetSessionId)) return sessions;
  order.splice(from, 1);
  let insertAt = order.indexOf(targetSessionId);
  if (after) insertAt += 1;
  order.splice(insertAt, 0, draggedSessionId);
  return new Map(order.map((id) => [id, sessions.get(id)]).filter(([, session]) => Boolean(session)));
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
