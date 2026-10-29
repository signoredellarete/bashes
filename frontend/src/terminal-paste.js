export function normalizeTerminalPaste(text) {
  return String(text ?? '').replace(/\r\n?/g, '\n');
}

export function pasteIntoTerminal(terminal, text) {
  const normalized = normalizeTerminalPaste(text);
  if (!normalized) return false;
  terminal.paste(normalized);
  return true;
}
