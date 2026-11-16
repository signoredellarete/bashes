const DEFAULT_LIMIT = 262144;

export function createTerminalOutputBuffer({ sessions, limit = DEFAULT_LIMIT, decodeBase64 = base64ToBytes }) {
  const pending = new Map();

  function decode(event) {
    if (event?.bytes) return decodeBase64(event.bytes);
    return event?.data ?? '';
  }

  function write(sessionID, data) {
    if (!sessionID || !data || data.length === 0) return;
    const session = sessions().get(sessionID);
    if (session?.terminal) {
      session.terminal.write(data);
      return;
    }
    const current = pending.get(sessionID);
    const chunks = Array.isArray(current) ? current : current ? [current] : [];
    chunks.push(data);
    pending.set(sessionID, trimChunks(chunks, limit));
  }

  function flush(sessionID) {
    const chunks = pending.get(sessionID);
    if (!chunks) return;
    pending.delete(sessionID);
    const terminal = sessions().get(sessionID)?.terminal;
    if (!terminal) return;
    for (const chunk of Array.isArray(chunks) ? chunks : [chunks]) terminal.write(chunk);
  }

  function discard(sessionID) {
    pending.delete(sessionID);
  }

  function clear() {
    pending.clear();
  }

  function size(sessionID) {
    return (pending.get(sessionID) ?? []).reduce?.((total, chunk) => total + chunk.length, 0) ?? 0;
  }

  return { clear, decode, discard, flush, size, write };
}

export function base64ToBytes(value) {
  const binary = globalThis.atob(String(value));
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) bytes[index] = binary.charCodeAt(index);
  return bytes;
}

export function trimChunks(chunks, limit = DEFAULT_LIMIT) {
  let total = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
  while (chunks.length > 1 && total > limit) {
    total -= chunks.shift().length;
  }
  if (total > limit) chunks[0] = chunks[0].slice(chunks[0].length - limit);
  return chunks;
}
