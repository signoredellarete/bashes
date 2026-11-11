import Sortable from 'sortablejs';

export function createSidebarSorting({
  onReorder,
  onTagAssignment,
  onHideTooltip,
  onClearDragOutline,
  onClearAllDragOutlines,
  onSuppressTagFilter,
} = {}) {
  let hostSortable = null;
  let tagCatalogSortable = null;
  let tagRemoveSortable = null;
  let resourceTagSortables = [];
  let activeTagDropTarget = null;

  function initialize({ container, canReorder }) {
    destroy();
    initializeTagSorting();
    if (canReorder) initializeHostSorting(container);
  }

  function initializeHostSorting(container) {
    hostSortable = Sortable.create(container, {
      animation: 170,
      easing: 'cubic-bezier(0.2, 0.8, 0.2, 1)',
      draggable: '.host-block',
      filter: '.resource-tag, .resource-tag-drop-target',
      preventOnFilter: false,
      forceFallback: true,
      fallbackOnBody: true,
      fallbackTolerance: 4,
      ghostClass: 'host-drag-ghost',
      chosenClass: 'host-drag-chosen',
      fallbackClass: 'host-drag-mirror',
      onUnchoose() {
        onClearDragOutline?.('hosts');
      },
      onStart() {
        onHideTooltip?.();
        document.documentElement.classList.add('sorting-hosts');
      },
      onEnd() {
        onClearDragOutline?.('hosts');
        document.documentElement.classList.remove('sorting-hosts');
        const order = [...container.querySelectorAll(':scope > .host-block')]
          .map((block) => block.dataset.topLevelId);
        onReorder?.(order);
      },
    });
  }

  function initializeTagSorting() {
    const groupName = 'bashes-resource-tags';
    const catalog = document.querySelector('#tag-filter-chips');
    const removeZone = document.querySelector('#tag-remove-zone');
    if (!catalog || !removeZone) return;

    if (catalog.querySelector('.tag-filter-chip')) {
      tagCatalogSortable = Sortable.create(catalog, {
        animation: 150,
        draggable: '.tag-filter-chip',
        sort: false,
        group: { name: groupName, pull: 'clone', put: false, revertClone: true },
        forceFallback: true,
        fallbackOnBody: true,
        fallbackTolerance: 4,
        ghostClass: 'tag-drag-ghost',
        chosenClass: 'tag-drag-chosen',
        fallbackClass: 'tag-drag-mirror',
        onStart() {
          beginTagSorting('catalog');
        },
        onMove(event) {
          setActiveTagDropTarget(event.to.closest?.('.resource-tag-drop-target'));
          return true;
        },
        onEnd() {
          onSuppressTagFilter?.();
          finishTagSorting();
        },
      });
    }

    document.querySelectorAll('.resource-tag-drop-target').forEach((target) => {
      resourceTagSortables.push(Sortable.create(target, {
        sort: false,
        group: {
          name: groupName,
          pull: false,
          put: (_to, from) => from.el?.id === 'tag-filter-chips',
        },
        onAdd(event) {
          const resourceID = event.to.dataset.resourceId;
          const tagName = event.item.dataset.tagName;
          event.item.remove();
          finishTagSorting();
          onTagAssignment?.(resourceID, tagName, true);
        },
      }));
    });

    document.querySelectorAll('.resource-tags').forEach((container) => {
      if (!container.querySelector('.resource-tag:not(.overflow)')) return;
      resourceTagSortables.push(Sortable.create(container, {
        animation: 150,
        draggable: '.resource-tag:not(.overflow)',
        sort: false,
        group: { name: groupName, pull: 'clone', put: false, revertClone: true },
        forceFallback: true,
        fallbackOnBody: true,
        fallbackTolerance: 4,
        ghostClass: 'tag-drag-ghost',
        chosenClass: 'tag-drag-chosen',
        fallbackClass: 'tag-drag-mirror',
        onStart() {
          beginTagSorting('assigned');
        },
        onMove(event) {
          setActiveTagDropTarget(event.to.closest?.('.tag-remove-zone'));
          return true;
        },
        onEnd() {
          finishTagSorting();
        },
      }));
    });

    tagRemoveSortable = Sortable.create(removeZone, {
      sort: false,
      group: {
        name: groupName,
        pull: false,
        put: (_to, from) => from.el?.classList.contains('resource-tags'),
      },
      onAdd(event) {
        const resourceID = event.from.dataset.resourceId;
        const tagName = event.item.dataset.tagName;
        event.item.remove();
        finishTagSorting();
        onTagAssignment?.(resourceID, tagName, false);
      },
    });
  }

  function beginTagSorting(kind) {
    onHideTooltip?.();
    onClearAllDragOutlines?.();
    document.documentElement.classList.add('sorting-tags', `sorting-${kind}-tag`);
  }

  function finishTagSorting() {
    setActiveTagDropTarget(null);
    document.documentElement.classList.remove('sorting-tags', 'sorting-catalog-tag', 'sorting-assigned-tag');
  }

  function setActiveTagDropTarget(target) {
    if (activeTagDropTarget === target) return;
    activeTagDropTarget?.classList.remove('active');
    activeTagDropTarget = target;
    activeTagDropTarget?.classList.add('active');
  }

  function destroy() {
    hostSortable?.destroy();
    hostSortable = null;
    tagCatalogSortable?.destroy();
    tagCatalogSortable = null;
    tagRemoveSortable?.destroy();
    tagRemoveSortable = null;
    resourceTagSortables.forEach((sortable) => sortable.destroy());
    resourceTagSortables = [];
    finishTagSorting();
    document.documentElement.classList.remove('sorting-hosts');
  }

  return {
    initialize,
    destroy,
    beginTagSorting,
    finishTagSorting,
    hostSortingActive: () => Boolean(hostSortable),
  };
}
