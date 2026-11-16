export function createDragOutlineController({ delay = 300, documentRef = document, windowRef = window }) {
  const timers = { hosts: 0, tabs: 0 };
  const targets = { hosts: null, tabs: null };

  function schedule(kind, target) {
    clear(kind);
    targets[kind] = target;
    target?.classList.add('drag-outline-source');
    timers[kind] = windowRef.setTimeout(() => {
      timers[kind] = 0;
      documentRef.documentElement.classList.add(`${kind}-drag-outline-ready`);
    }, delay);
  }

  function clear(kind) {
    windowRef.clearTimeout(timers[kind]);
    timers[kind] = 0;
    targets[kind]?.classList.remove('drag-outline-source');
    targets[kind] = null;
    documentRef.documentElement.classList.remove(`${kind}-drag-outline-ready`);
  }

  function clearAll() {
    clear('hosts');
    clear('tabs');
  }

  return { schedule, clear, clearAll };
}
