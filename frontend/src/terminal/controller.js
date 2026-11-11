import { FitAddon } from '@xterm/addon-fit';
import { Terminal } from '@xterm/xterm';
import { createOrderedWriter } from '../ordered-writer.js';
import { closedSessionShortcut } from '../session-state.js';
import { repeatedKeyData } from '../terminal-keyboard.js';
import { pasteIntoTerminal } from '../terminal-paste.js';
import { terminalOptions } from './options.js';

export function createTerminalController({
  stack,
  sessions,
  activeSessionId,
  settings,
  writeSession,
  resizeSession,
  readClipboard,
  writeClipboard,
  openLink,
  closeSession,
  reconnectSession,
  inputError,
  windowObject = window,
  documentObject = document,
  TerminalClass = Terminal,
  FitAddonClass = FitAddon,
}) {
  let fitFrame = 0;

  function createPendingPane(sessionId, message) {
    const pane = documentObject.createElement('section');
    pane.className = 'terminal-pane pending-pane';
    pane.dataset.sessionId = sessionId;
    pane.textContent = message;
    stack.append(pane);
    return pane;
  }

  function createTerminal(sessionId, kind = 'ssh', disableStdin = false) {
    const pane = documentObject.createElement('section');
    pane.className = 'terminal-pane';
    pane.dataset.sessionId = sessionId;
    stack.append(pane);

    let terminal;
    try {
      terminal = new TerminalClass(terminalOptions(settings(), {
        disableStdin,
        onLink: (event, url) => openLink?.(event, url),
      }));
      const fitAddon = new FitAddonClass();
      terminal.loadAddon(fitAddon);
      terminal.open(pane);
      const writeInput = disableStdin ? null : createOrderedWriter((data) => writeSession(sessionId, data));
      installKeyHandler(terminal, sessionId, writeInput);

      if (!disableStdin) {
        terminal.onData((data) => {
          if (sessions().get(sessionId)?.closed) return;
          writeInput(data).catch((error) => inputError?.(kind, error));
        });
        terminal.onSelectionChange(() => {
          if (!settings().terminalCopyOnSelect) return;
          const selected = terminal.getSelection();
          if (selected) writeClipboard(selected).catch(() => {});
        });
        pane.addEventListener('contextmenu', (event) => {
          event.preventDefault();
          event.stopPropagation();
          readClipboard()
            .then((text) => {
              if (!sessions().get(sessionId)?.closed) pasteIntoTerminal(terminal, text);
            })
            .catch(() => {});
        });
      }

      return { terminal, fitAddon, element: pane };
    } catch (error) {
      terminal?.dispose();
      pane.remove();
      throw error;
    }
  }

  function installKeyHandler(terminal, sessionId, writeInput) {
    terminal.attachCustomKeyEventHandler((event) => {
      const session = sessions().get(sessionId);
      if (session?.closed && event.type === 'keydown') {
        const shortcut = closedSessionShortcut(event);
        event.preventDefault();
        if (shortcut === 'close') {
          windowObject.setTimeout(() => closeSession?.(sessionId), 0);
        } else if (shortcut === 'reconnect') {
          windowObject.setTimeout(() => reconnectSession?.(sessionId), 0);
        }
        return false;
      }

      if (!writeInput || event.type !== 'keydown' || !event.repeat || event.isComposing) return true;
      const data = repeatedKeyData(event, Boolean(terminal.modes?.applicationCursorKeysMode));
      if (!data) return true;

      event.preventDefault();
      writeInput(data).catch((error) => inputError?.(session?.kind, error));
      return false;
    });
  }

  function dispose(session) {
    session?.terminal?.dispose();
    session?.element?.remove();
  }

  function showActivePanes() {
    const activeId = activeSessionId();
    stack.querySelectorAll('.terminal-pane').forEach((pane) => {
      pane.hidden = pane.dataset.sessionId !== activeId;
    });
  }

  function applySettings(next) {
    for (const session of sessions().values()) {
      if (!session.terminal) continue;
      session.terminal.options.fontSize = next.terminalFontSize;
      session.terminal.options.fontFamily = next.terminalFontFamily;
      session.terminal.options.scrollback = next.terminalScrollback;
    }
  }

  function focusActive() {
    const session = sessions().get(activeSessionId());
    if (!session?.terminal) return;
    windowObject.requestAnimationFrame(() => session.terminal.focus());
  }

  function scheduleFit() {
    if (fitFrame) windowObject.cancelAnimationFrame(fitFrame);
    fitFrame = windowObject.requestAnimationFrame(() => {
      fitFrame = 0;
      const session = sessions().get(activeSessionId());
      if (!session?.fitAddon || !session.terminal) return;
      session.fitAddon.fit();
      if (!session.pending && !session.closed) {
        resizeSession(session.id, session.terminal.cols, session.terminal.rows).catch(() => {});
      }
    });
  }

  return {
    applySettings,
    createPendingPane,
    createTerminal,
    dispose,
    focusActive,
    scheduleFit,
    showActivePanes,
  };
}
