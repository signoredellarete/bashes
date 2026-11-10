const SCROLLBAR_VISUAL_WIDTH = 8;
const CARD_SCROLLBAR_GAP = 8;

export function sidebarListInset(offsetWidth, clientWidth) {
  const layoutWidth = Math.max(0, Number(offsetWidth) - Number(clientWidth));
  return CARD_SCROLLBAR_GAP + Math.max(0, SCROLLBAR_VISUAL_WIDTH - layoutWidth);
}

export function revealSidebarResource(container, resourceId, schedule = requestAnimationFrame) {
  if (!container || !resourceId) return;
  schedule(() => {
    const row = [...container.querySelectorAll('.host-row')]
      .find((element) => element.dataset.id === resourceId);
    row?.scrollIntoView({ behavior: 'smooth', block: 'nearest', inline: 'nearest' });
  });
}
