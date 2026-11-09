import { readdirSync, readFileSync } from 'node:fs';
import { dirname, extname, join, relative } from 'node:path';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';

const sourceRoot = join(dirname(fileURLToPath(import.meta.url)), '..');
const bindingAdapters = new Set([
  join('api', 'backend.js'),
  join('file-transfer', 'api.js'),
]);

describe('frontend architecture', () => {
  it('keeps generated Go binding access inside the backend adapter', () => {
    const violations = sourceFiles(sourceRoot)
      .filter((path) => !bindingAdapters.has(relative(sourceRoot, path)))
      .filter((path) => readFileSync(path, 'utf8').includes('globalThis.go'))
      .map((path) => relative(sourceRoot, path));

    expect(violations).toEqual([]);
  });
});

function sourceFiles(directory) {
  const result = [];
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    if (entry.name === 'node_modules' || entry.name === 'dist') continue;
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      result.push(...sourceFiles(path));
      continue;
    }
    if (['.js', '.svelte'].includes(extname(entry.name)) && !entry.name.endsWith('.test.js')) {
      result.push(path);
    }
  }
  return result;
}
