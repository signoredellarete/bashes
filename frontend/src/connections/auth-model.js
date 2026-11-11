import { SAVED_PASSWORD_MASK } from '../password-field.js';

export const CUSTOM_KEY_PATH_HELP = [
  'Select the private key file, not the .pub file.',
  'Linux/macOS: ~/.ssh/id_ed25519',
  'Windows: C:\\Users\\YourUser\\.ssh\\id_ed25519',
].join('\n\n');

export function keySource(key) {
  return key?.source || 'bashes';
}

export function keyChoiceValue(key) {
  if (keySource(key) !== 'bashes') return `path:${key.privateKey || ''}`;
  return `bashes:${key.name || ''}`;
}

export function keyOptionLabel(key) {
  if (keySource(key) !== 'bashes') return `${key.name} - ${key.privateKey}`;
  return key.name;
}

export function authKeyGroups(keys) {
  const group = (label, source, predicate = () => true) => ({
    label,
    options: (keys ?? [])
      .filter((key) => keySource(key) === source && predicate(key))
      .map((key) => ({
        value: keyChoiceValue(key),
        label: keyOptionLabel(key),
        title: key.privateKey || key.name,
      })),
  });
  const groups = [
    group('Bashes keys', 'bashes'),
    group('System keys (~/.ssh)', 'system'),
  ];
  const distributions = [...new Set((keys ?? [])
    .filter((key) => keySource(key) === 'wsl')
    .map((key) => key.distribution || 'WSL'))];
  for (const distribution of distributions) {
    groups.push(group(`WSL keys (${distribution})`, 'wsl', (key) => (key.distribution || 'WSL') === distribution));
  }
  groups.push(group('Custom keys directory', 'custom'));
  return groups.filter((entry) => entry.options.length > 0);
}

export function resolveKeyChoice(keys, value) {
  if (!value) return null;
  const key = (keys ?? []).find((item) => keyChoiceValue(item) === value);
  if (key) return key;
  if (value.startsWith('path:')) {
    const privateKey = value.slice('path:'.length);
    return {
      source: 'path',
      name: privateKey.split(/[\\/]/).pop(),
      privateKey,
      publicKey: `${privateKey}.pub`,
    };
  }
  if (value.startsWith('bashes:')) {
    return { source: 'bashes', name: value.slice('bashes:'.length) };
  }
  return null;
}

export function authValues(resource, keys, hasSavedPassword = false) {
  const values = {
    authMethod: 'agent',
    keyChoice: '',
    password: '',
    hadSavedPassword: false,
    savePassword: false,
    privateKeyPath: '',
    privateKeyPassphrase: '',
    trustHostKey: false,
  };
  const auth = resource?.auth;
  if (!auth) return values;

  if (auth.method === 'password') {
    values.authMethod = 'password';
    values.hadSavedPassword = Boolean(hasSavedPassword);
    values.savePassword = Boolean(hasSavedPassword);
    values.password = hasSavedPassword ? SAVED_PASSWORD_MASK : '';
  } else if (auth.method === 'key' && auth.keyName) {
    const choice = `bashes:${auth.keyName}`;
    if ((keys ?? []).some((key) => keyChoiceValue(key) === choice)) {
      values.authMethod = 'key';
      values.keyChoice = choice;
    }
  } else if (auth.method === 'path' && auth.privateKeyPath) {
    const choice = `path:${auth.privateKeyPath}`;
    if ((keys ?? []).some((key) => keyChoiceValue(key) === choice)) {
      values.authMethod = 'key';
      values.keyChoice = choice;
    } else {
      values.authMethod = 'path';
      values.privateKeyPath = auth.privateKeyPath;
    }
  }
  return values;
}

export function authInput(values, keys) {
  if (values.authMethod === 'password') {
    const savePassword = Boolean(values.savePassword);
    return {
      password: values.hadSavedPassword && values.password === SAVED_PASSWORD_MASK ? '' : values.password,
      managePassword: savePassword || Boolean(values.hadSavedPassword),
      savePassword,
      keyName: '',
      privateKeyPath: '',
      privateKeyPassphrase: '',
    };
  }
  if (values.authMethod === 'path') {
    return {
      keyName: '',
      privateKeyPath: values.privateKeyPath.trim(),
      privateKeyPassphrase: values.privateKeyPassphrase,
    };
  }
  if (values.authMethod === 'agent') {
    return { keyName: '', privateKeyPath: '', privateKeyPassphrase: '' };
  }

  const key = resolveKeyChoice(keys, values.keyChoice);
  if (key && keySource(key) !== 'bashes') {
    return {
      keyName: '',
      privateKeyPath: key.privateKey,
      privateKeyPassphrase: values.privateKeyPassphrase,
    };
  }
  return {
    keyName: key?.name || '',
    privateKeyPath: '',
    privateKeyPassphrase: values.privateKeyPassphrase,
  };
}
