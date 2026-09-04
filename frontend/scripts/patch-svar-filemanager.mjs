import { readFile, writeFile } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';

const layoutURL = new URL('../node_modules/@svar-ui/svelte-filemanager/src/components/Layout.svelte', import.meta.url);
const layoutPath = fileURLToPath(layoutURL);
const marker = '\t\tswitch (action) {\n\t\t\tcase "download":';
const replacement = [
  '\t\tswitch (action) {',
  '\t\t\tcase "open":',
  '\t\t\t\tapi.exec("open-file", { id: context.id });',
  '\t\t\t\tbreak;',
  '\t\t\tcase "download":',
].join('\n');

const source = await readFile(layoutPath, 'utf8');
if (source.includes(replacement)) process.exit(0);
if (!source.includes(marker)) {
  throw new Error('The installed SVAR File Manager layout is incompatible with the Bashes open-file patch.');
}
await writeFile(layoutPath, source.replace(marker, replacement));
