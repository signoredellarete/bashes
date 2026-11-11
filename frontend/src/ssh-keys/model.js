export function keySource(key) {
  return key?.source || 'bashes';
}

export function keySourceLabel(key) {
  const source = keySource(key);
  if (source === 'bashes') return 'Managed by Bashes';
  if (source === 'system') return 'System';
  if (source === 'wsl') return key.distribution ? `WSL: ${key.distribution}` : 'WSL';
  if (source === 'custom') return 'Custom directory';
  return 'External';
}

export function keyChoiceValue(key) {
  if (keySource(key) !== 'bashes') return `path:${key.privateKey || ''}`;
  return `bashes:${key.name || ''}`;
}

export function keyChoiceLabel(key) {
  if (!key) return '';
  return keySource(key) === 'bashes' ? key.name : key.privateKey;
}

export function keyChoice(keys, value) {
  if (!value) return null;
  const key = (keys ?? []).find((item) => keyChoiceValue(item) === value);
  if (key) return key;
  if (value.startsWith('path:')) {
    const privateKey = value.slice('path:'.length);
    return { source: 'path', name: privateKey.split(/[\\/]/).pop(), privateKey, publicKey: `${privateKey}.pub` };
  }
  if (value.startsWith('bashes:')) return { source: 'bashes', name: value.slice('bashes:'.length) };
  return null;
}

export function installInputForKey(key) {
  if (keySource(key) !== 'bashes') {
    return {
      keyName: '',
      privateKeyPath: key.privateKey,
      publicKeyPath: key.publicKey || key.privateKey,
    };
  }
  return { keyName: key.name };
}

export function keyGroups(keys, query = '') {
  const normalizedQuery = String(query).trim().toLocaleLowerCase();
  const groups = new Map();
  for (const key of keys ?? []) {
    if (normalizedQuery && ![key.name, key.privateKey, keySourceLabel(key)]
      .some((value) => String(value ?? '').toLocaleLowerCase().includes(normalizedQuery))) continue;
    const source = keySource(key);
    const id = source === 'wsl' ? `wsl:${key.distribution || 'WSL'}` : source;
    if (!groups.has(id)) groups.set(id, { id, label: keySourceLabel(key), keys: [] });
    groups.get(id).keys.push({ ...key, value: keyChoiceValue(key) });
  }
  const order = ['bashes', 'system', 'wsl', 'custom'];
  return [...groups.values()].sort((left, right) => {
    const leftOrder = order.indexOf(left.id.split(':')[0]);
    const rightOrder = order.indexOf(right.id.split(':')[0]);
    return leftOrder - rightOrder || left.label.localeCompare(right.label);
  });
}

export function sshKeysPanelData({ keys, sources, settings, selectedValue, installTarget, installEnabled } = {}) {
  const values = new Set((keys ?? []).map(keyChoiceValue));
  const selection = values.has(selectedValue) ? selectedValue : (keys?.[0] ? keyChoiceValue(keys[0]) : '');
  return {
    keys: keys ?? [],
    sources: sources ?? [],
    customDirectory: settings?.customDirectory ?? '',
    selectedValue: selection,
    installTarget: installEnabled && installTarget ? {
      id: installTarget.id,
      name: installTarget.hostname,
      label: `${installTarget.user}@${installTarget.ip || installTarget.hostname}:${installTarget.port}`,
    } : null,
  };
}
