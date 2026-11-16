export function createTooltipController({ element, delay = 500, windowRef = window }) {
  let timer = 0;

  function hide() {
    if (timer) windowRef.clearTimeout(timer);
    timer = 0;
    if (element) element.hidden = true;
  }

  function render(anchor) {
    if (!anchor?.isConnected || !element) return;
    const text = anchor.dataset.tooltip;
    if (!text) return;
    const anchorRect = anchor.getBoundingClientRect();
    element.textContent = text;
    element.hidden = false;
    const position = tooltipPosition(
      anchorRect,
      element.getBoundingClientRect(),
      { width: windowRef.innerWidth, height: windowRef.innerHeight },
      anchor.dataset.tooltipPlacement || 'right',
    );
    element.style.left = `${position.left}px`;
    element.style.top = `${position.top}px`;
  }

  function show(anchor, wait = delay) {
    hide();
    if (wait <= 0) {
      render(anchor);
      return;
    }
    timer = windowRef.setTimeout(() => {
      timer = 0;
      render(anchor);
    }, wait);
  }

  return { hide, show };
}

export function tooltipPosition(anchor, tooltip, viewport, placement = 'right') {
  const proposedLeft = placement === 'bottom'
    ? anchor.left + anchor.width / 2 - tooltip.width / 2
    : anchor.right + 10;
  const proposedTop = placement === 'bottom'
    ? anchor.bottom + 8
    : anchor.top + anchor.height / 2 - tooltip.height / 2;
  return {
    left: Math.round(clamp(proposedLeft, 8, Math.max(8, viewport.width - tooltip.width - 8))),
    top: Math.round(clamp(proposedTop, 8, Math.max(8, viewport.height - tooltip.height - 8))),
  };
}

function clamp(value, minimum, maximum) {
  return Math.min(maximum, Math.max(minimum, value));
}
