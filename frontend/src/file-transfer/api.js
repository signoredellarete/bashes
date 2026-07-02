function wailsAPI() {
  return globalThis.go?.main?.App ?? globalThis.go?.desktop?.App;
}

function appAPI(name) {
  const api = wailsAPI();
  const fn = api?.[name];
  if (!fn) throw new Error(`File transfer API ${name} is not available`);
  return fn.bind(api);
}

export async function startFileTransfer(input) {
  return await appAPI('StartFileTransfer')(input);
}

export async function closeFileTransfer(sessionID) {
  return await appAPI('CloseFileTransfer')(sessionID);
}

export async function listFiles(sessionID, id) {
  const data = await appAPI('ListFileTransferFiles')({ sessionId: sessionID, id });
  return (data ?? []).map(parseEntry);
}

export async function createItem(sessionID, parentId, file) {
  const data = file.file ? await fileToBase64(file.file) : '';
  const result = await appAPI('CreateFileTransferItem')({
    sessionId: sessionID,
    parentId,
    name: file.name,
    type: file.type || (file.file ? 'file' : 'folder'),
    data,
  });
  return parseEntry(result);
}

export async function renameItem(sessionID, id, name) {
  const result = await appAPI('RenameFileTransferItem')({ sessionId: sessionID, id, name });
  return parseEntry(result);
}

export async function deleteItems(sessionID, ids) {
  return await appAPI('DeleteFileTransferItems')({ sessionId: sessionID, ids });
}

export async function copyItems(sessionID, ids, targetId, move = false) {
  return await appAPI('CopyFileTransferItems')({ sessionId: sessionID, ids, targetId, move });
}

function parseEntry(entry) {
  if (!entry) return entry;
  return {
    ...entry,
    date: entry.date ? new Date(entry.date) : undefined,
  };
}

async function fileToBase64(file) {
  const buffer = await file.arrayBuffer();
  const bytes = new Uint8Array(buffer);
  let binary = '';
  const chunkSize = 0x8000;
  for (let offset = 0; offset < bytes.length; offset += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(offset, offset + chunkSize));
  }
  return btoa(binary);
}
