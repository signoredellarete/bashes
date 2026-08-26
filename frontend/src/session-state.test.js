import { describe, expect, it } from 'vitest';
import { closedSessionShortcut, lastFocusedSessionId, orderSessions, preferredSessionForResource, rememberFocus } from './session-state.js';

describe('session state', () => {
  it('prefers the last live session for a resource', () => {
    const sessions = new Map([
      ['one', { id: 'one', resourceId: 'host', closed: false }],
      ['two', { id: 'two', resourceId: 'host', closed: false }],
    ]);
    expect(preferredSessionForResource(sessions, new Map([['host', 'one']]), 'host')?.id).toBe('one');
  });

  it('orders tabs from their rendered order and keeps omitted sessions', () => {
    const sessions = new Map([['one', { id: 'one' }], ['two', { id: 'two' }], ['three', { id: 'three' }]]);
    expect([...orderSessions(sessions, ['three', 'one']).keys()]).toEqual(['three', 'one', 'two']);
  });

  it('tracks focus history without duplicates', () => {
    const history = rememberFocus(['one', 'two'], 'one');
    expect(history).toEqual(['two', 'one']);
    expect(lastFocusedSessionId(history, new Map([['two', {}]]))).toBe('two');
  });

  it('maps closed-session control shortcuts without intercepting normal keys', () => {
    expect(closedSessionShortcut({ type: 'keydown', key: 'd', ctrlKey: true })).toBe('close');
    expect(closedSessionShortcut({ type: 'keydown', key: 'R', ctrlKey: true })).toBe('reconnect');
    expect(closedSessionShortcut({ type: 'keydown', key: 'r', metaKey: true })).toBe('');
    expect(closedSessionShortcut({ type: 'keydown', key: 'd', ctrlKey: true, repeat: true })).toBe('');
  });
});
