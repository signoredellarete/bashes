export function remainingSeconds(totalBytes, transferredBytes, bytesPerSecond) {
  const total = Number(totalBytes || 0);
  const transferred = Number(transferredBytes || 0);
  const speed = Number(bytesPerSecond || 0);
  if (total <= transferred || speed <= 0) return 0;
  return Math.ceil((total - transferred) / speed);
}

export function formatDuration(seconds) {
  const value = Math.max(0, Math.ceil(Number(seconds || 0)));
  if (value < 60) return `${value}s`;
  const minutes = Math.floor(value / 60);
  const remainder = value % 60;
  if (minutes < 60) return remainder ? `${minutes}m ${remainder}s` : `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  return remainingMinutes ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
}
