const SCROLLBAR_VISUAL_WIDTH = 8;
const CARD_SCROLLBAR_GAP = 8;

export function sidebarListInset(offsetWidth, clientWidth) {
  const layoutWidth = Math.max(0, Number(offsetWidth) - Number(clientWidth));
  return CARD_SCROLLBAR_GAP + Math.max(0, SCROLLBAR_VISUAL_WIDTH - layoutWidth);
}
