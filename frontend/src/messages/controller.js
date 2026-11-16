export function createMessageCenter({
  statusElement,
  notify = () => {},
  showLog = () => {},
  now = () => new Date(),
  limit = 50,
}) {
  let entries = [];

  function write(message, level = '') {
    const text = String(message ?? '');
    if (statusElement) {
      statusElement.textContent = text;
      statusElement.title = text;
    }
    const resolvedLevel = level || noticeLevel(text);
    entries.unshift({
      time: now().toLocaleTimeString(),
      level: resolvedLevel,
      message: text,
    });
    entries = entries.slice(0, limit);
    notify(text, resolvedLevel);
  }

  function openLog() {
    const details = entries.length > 0
      ? entries.map((entry) => [`${entry.time} ${entry.level.toUpperCase()}`, entry.message])
      : [['Log', 'No messages yet.']];
    showLog({
      kicker: 'Tools',
      title: 'Message Log',
      message: 'Latest Bashes messages.',
      details,
    });
  }

  function snapshot() {
    return entries.map((entry) => ({ ...entry }));
  }

  return { openLog, snapshot, write };
}

export function noticeLevel(message) {
  const text = String(message ?? '');
  if (/error|could not|failed|unable|invalid|mismatch|required|denied|refused|timeout|unreachable/i.test(text)) {
    return 'error';
  }
  if (/updated|added|imported|installed|generated|connected|started|stopped|active|saved/i.test(text)) {
    return 'success';
  }
  return 'info';
}
