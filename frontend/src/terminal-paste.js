export function normalizeTerminalPaste(text) {
  return String(text ?? '').replace(/\r\n?/g, '\n');
}

function dispatchTerminalPasteEvent(terminal, text, clipboard = globalThis) {
  const target = terminal?.textarea ?? terminal?.element;
  if (!target?.dispatchEvent || !clipboard?.ClipboardEvent || !clipboard?.DataTransfer) return false;

  try {
    const clipboardData = new clipboard.DataTransfer();
    clipboardData.setData('text/plain', text);
    const pasteEvent = new clipboard.ClipboardEvent('paste', {
      bubbles: true,
      cancelable: true,
      clipboardData,
    });
    if (pasteEvent.clipboardData?.getData('text/plain') !== text) return false;
    target.dispatchEvent(pasteEvent);
    return true;
  } catch {
    return false;
  }
}

export function pasteIntoTerminal(terminal, text, clipboard = globalThis) {
  const normalized = normalizeTerminalPaste(text);
  if (!normalized) return false;
  terminal.focus?.();
  if (!dispatchTerminalPasteEvent(terminal, normalized, clipboard)) {
    terminal.paste(normalized);
  }
  return true;
}
