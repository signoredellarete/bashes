export function visibleFileEntries(entries, showHiddenFiles = false) {
  const list = Array.isArray(entries) ? entries : [];
  if (showHiddenFiles) return list;
  return list.filter((entry) => !fileEntryName(entry).startsWith('.'));
}

function fileEntryName(entry) {
  const parts = String(entry?.id ?? '').split('/').filter(Boolean);
  return parts.at(-1) ?? '';
}
