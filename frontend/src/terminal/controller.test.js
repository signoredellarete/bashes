import { describe, expect, it, vi } from 'vitest';

globalThis.self = globalThis;
const { createTerminalController } = await import('./controller.js');

function testEnvironment() {
  const panes = [];
  const documentObject = {
    createElement: () => ({
      className: '',
      dataset: {},
      hidden: false,
      listeners: {},
      addEventListener(name, listener) { this.listeners[name] = listener; },
      remove() { panes.splice(panes.indexOf(this), 1); },
    }),
  };
  const stack = {
    append: (pane) => panes.push(pane),
    querySelectorAll: () => panes,
  };
  const windowObject = {
    requestAnimationFrame: (callback) => { callback(); return 1; },
    cancelAnimationFrame: vi.fn(),
    setTimeout: (callback) => { callback(); return 1; },
  };
  return { documentObject, panes, stack, windowObject };
}

class FakeTerminal {
  constructor(options) {
    this.options = options;
    this.cols = 100;
    this.rows = 30;
    this.modes = {};
  }

  loadAddon(addon) { this.addon = addon; }
  open(element) { this.element = element; }
  onData(callback) { this.sendData = callback; }
  onSelectionChange(callback) { this.selectionChanged = callback; }
  attachCustomKeyEventHandler(callback) { this.keyHandler = callback; }
  getSelection() { return 'selected'; }
  focus() { this.focused = true; }
  dispose() { this.disposed = true; }
}

class FakeFitAddon {
  fit = vi.fn();
}

describe('terminal controller', () => {
  it('owns terminal creation, focus, fit, resize and disposal', async () => {
    const environment = testEnvironment();
    const sessions = new Map();
    const writeSession = vi.fn(async () => {});
    const resizeSession = vi.fn(async () => {});
    const controller = createTerminalController({
      ...environment,
      sessions: () => sessions,
      activeSessionId: () => 'session-1',
      settings: () => ({
        terminalFontFamily: 'JetBrains Mono',
        terminalFontSize: 16,
        terminalScrollback: 100000,
        terminalCopyOnSelect: true,
      }),
      writeSession,
      resizeSession,
      readClipboard: async () => '',
      writeClipboard: async () => {},
      TerminalClass: FakeTerminal,
      FitAddonClass: FakeFitAddon,
    });

    const view = controller.createTerminal('session-1');
    sessions.set('session-1', { id: 'session-1', ...view, closed: false, pending: false });
    view.terminal.sendData('pwd\r');
    await Promise.resolve();
    await Promise.resolve();
    controller.focusActive();
    controller.scheduleFit();

    expect(writeSession).toHaveBeenCalledWith('session-1', 'pwd\r');
    expect(view.terminal.focused).toBe(true);
    expect(view.fitAddon.fit).toHaveBeenCalledOnce();
    expect(resizeSession).toHaveBeenCalledWith('session-1', 100, 30);

    controller.dispose(sessions.get('session-1'));
    expect(view.terminal.disposed).toBe(true);
    expect(environment.panes).toEqual([]);
  });
});
