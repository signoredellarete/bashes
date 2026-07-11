import { describe, expect, it } from 'vitest';
import { lastFocusedSessionId, preferredSessionForResource, reorderSessions, rememberFocus } from './session-state.js';

describe('session state', () => {
  it('prefers the last live session for a resource', () => {
    const sessions = new Map([
      ['one', { id: 'one', resourceId: 'host', closed: false }],
      ['two', { id: 'two', resourceId: 'host', closed: false }],
    ]);
    expect(preferredSessionForResource(sessions, new Map([['host', 'one']]), 'host')?.id).toBe('one');
  });

  it('reorders tabs without losing session objects', () => {
    const sessions = new Map([['one', { id: 'one' }], ['two', { id: 'two' }], ['three', { id: 'three' }]]);
    expect([...reorderSessions(sessions, 'one', 'three', true).keys()]).toEqual(['two', 'three', 'one']);
  });

  it('tracks focus history without duplicates', () => {
    const history = rememberFocus(['one', 'two'], 'one');
    expect(history).toEqual(['two', 'one']);
    expect(lastFocusedSessionId(history, new Map([['two', {}]]))).toBe('two');
  });
});
