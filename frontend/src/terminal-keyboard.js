export function repeatedKeyData(event, applicationCursor = false) {
  if (event.ctrlKey || event.metaKey || event.altKey) return '';

  if (event.key.length === 1) return event.key;

  switch (event.key) {
    case 'ArrowUp':
      return applicationCursor ? '\x1bOA' : '\x1b[A';
    case 'ArrowDown':
      return applicationCursor ? '\x1bOB' : '\x1b[B';
    case 'ArrowRight':
      return applicationCursor ? '\x1bOC' : '\x1b[C';
    case 'ArrowLeft':
      return applicationCursor ? '\x1bOD' : '\x1b[D';
    case 'Backspace':
      return '\x7f';
    case 'Delete':
      return '\x1b[3~';
    case 'Home':
      return applicationCursor ? '\x1bOH' : '\x1b[H';
    case 'End':
      return applicationCursor ? '\x1bOF' : '\x1b[F';
    case 'PageUp':
      return '\x1b[5~';
    case 'PageDown':
      return '\x1b[6~';
    case 'Tab':
      return event.shiftKey ? '\x1b[Z' : '\t';
    case 'Enter':
      return '\r';
    case 'Escape':
      return '\x1b';
    default:
      return '';
  }
}
