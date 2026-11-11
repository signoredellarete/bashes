import Sortable from 'sortablejs';

export function createTerminalTabSorting({ onReorder, onClearDragOutline, onFocus } = {}) {
  let sortable = null;

  function initialize(container, enabled) {
    destroy();
    if (!enabled) return;
    sortable = Sortable.create(container, {
      animation: 170,
      easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)',
      direction: 'horizontal',
      draggable: '.session-tab-wrap',
      handle: '.session-tab',
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      ghostClass: 'tab-drag-ghost',
      chosenClass: 'tab-drag-chosen',
      fallbackClass: 'tab-drag-mirror',
      onUnchoose() {
        onClearDragOutline?.();
      },
      onStart() {
        document.documentElement.classList.add('sorting-tabs');
      },
      onEnd() {
        onClearDragOutline?.();
        document.documentElement.classList.remove('sorting-tabs');
        const order = [...container.querySelectorAll(':scope > .session-tab-wrap')]
          .map((tab) => tab.dataset.sessionId);
        onReorder?.(order);
        onFocus?.();
      },
    });
  }

  function destroy() {
    sortable?.destroy();
    sortable = null;
    document.documentElement.classList.remove('sorting-tabs');
  }

  return {
    initialize,
    destroy,
    isActive: () => Boolean(sortable),
  };
}
