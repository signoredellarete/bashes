#!/usr/bin/env node

import fs from 'node:fs';
import path from 'node:path';
import process from 'node:process';

const root = path.resolve(import.meta.dirname, '..');
const files = {
  package: path.join(root, 'frontend/package.json'),
  lock: path.join(root, 'frontend/package-lock.json'),
  wails: path.join(root, 'wails.json'),
};

function readJson(file) {
  return JSON.parse(fs.readFileSync(file, 'utf8'));
}

function writeJson(file, value) {
  fs.writeFileSync(file, `${JSON.stringify(value, null, 2)}\n`);
}

function versions() {
  const packageJson = readJson(files.package);
  const lock = readJson(files.lock);
  const wails = readJson(files.wails);
  return {
    packageJson,
    lock,
    wails,
    values: [packageJson.version, lock.version, lock.packages?.['']?.version, wails.info?.productVersion],
  };
}

function validVersion(value) {
  return /^\d+\.\d+\.\d+$/.test(value);
}

const [command = 'check', value] = process.argv.slice(2);
const current = versions();

if (command === 'check') {
  const distinct = new Set(current.values);
  if (distinct.size !== 1 || !validVersion(current.values[0])) {
    console.error(`Version mismatch: ${current.values.join(', ')}`);
    process.exit(1);
  }
  console.log(current.values[0]);
} else if (command === 'set') {
  if (!validVersion(value)) {
    console.error('Usage: node scripts/version.mjs set <major.minor.patch>');
    process.exit(1);
  }
  current.packageJson.version = value;
  current.lock.version = value;
  current.lock.packages[''].version = value;
  current.wails.info.productVersion = value;
  writeJson(files.package, current.packageJson);
  writeJson(files.lock, current.lock);
  writeJson(files.wails, current.wails);
  console.log(value);
} else {
  console.error('Usage: node scripts/version.mjs [check|set <major.minor.patch>]');
  process.exit(1);
}
