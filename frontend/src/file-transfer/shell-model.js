export const FILE_TRANSFER_MARGIN = 18;
export const FILE_TRANSFER_MIN_WIDTH = 720;
export const FILE_TRANSFER_MIN_HEIGHT = 460;

export function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max);
}

export function fileTransferModalSize(width, height, viewportWidth, viewportHeight) {
  const maxWidth = Math.max(0, viewportWidth - FILE_TRANSFER_MARGIN * 2);
  const maxHeight = Math.max(0, viewportHeight - FILE_TRANSFER_MARGIN * 2);
  const minWidth = Math.min(FILE_TRANSFER_MIN_WIDTH, maxWidth);
  const minHeight = Math.min(FILE_TRANSFER_MIN_HEIGHT, maxHeight);

  return {
    width: clamp(width, minWidth, maxWidth),
    height: clamp(height, minHeight, maxHeight),
  };
}

export function fileTransferModalPosition(left, top, width, height, viewportWidth, viewportHeight) {
  return {
    left: clamp(left, FILE_TRANSFER_MARGIN, Math.max(FILE_TRANSFER_MARGIN, viewportWidth - width - FILE_TRANSFER_MARGIN)),
    top: clamp(top, FILE_TRANSFER_MARGIN, Math.max(FILE_TRANSFER_MARGIN, viewportHeight - height - FILE_TRANSFER_MARGIN)),
  };
}
