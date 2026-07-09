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
  const result = await appAPI('CreateFileTransferItem')({
    sessionId: sessionID,
    parentId,
    name: file.name,
    type: file.type || (file.file ? 'file' : 'folder'),
    data: '',
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

export async function startCopyJob(sessionID, ids, targetId, move = false) {
  return await appAPI('StartFileTransferCopyJob')({ sessionId: sessionID, ids, targetId, move });
}

export async function startUploadJob(sessionID, paths, targetId, move = false) {
  return await appAPI('StartFileTransferUploadJob')({ sessionId: sessionID, paths, targetId, move });
}

export async function listJobs(sessionID) {
  return await appAPI('ListFileTransferJobs')(sessionID);
}

export async function cancelJob(jobId) {
  return await appAPI('CancelFileTransferJob')({ jobId });
}

export async function resolveDroppedFilePaths(files) {
  const list = [...files];
  const fromRuntime = await resolveFilePathsWithRuntime(list);
  if (fromRuntime.length) return fromRuntime;

  const fromFileObjects = list.map((file) => file.path || file.webkitRelativePath).filter(Boolean);
  if (fromFileObjects.length) return fromFileObjects;

  throw new Error('Native file paths are not available for this drop. Use the local panel to transfer files.');
}

function parseEntry(entry) {
  if (!entry) return entry;
  return {
    ...entry,
    date: entry.date ? new Date(entry.date) : undefined,
  };
}

async function resolveFilePathsWithRuntime(files) {
  const runtime = globalThis.runtime;
  if (!runtime?.CanResolveFilePaths || !runtime?.ResolveFilePaths) return [];
  const canResolve = await runtime.CanResolveFilePaths();
  if (!canResolve) return [];
  const paths = await runtime.ResolveFilePaths(files);
  return Array.isArray(paths) ? paths.filter(Boolean) : [];
}
