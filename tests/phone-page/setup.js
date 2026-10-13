import { $ } from 'bun';
import { mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

export default async function setup() {
  const testDirectory = import.meta.dir;
  const root = resolve(testDirectory, '../..');
  const assetsDirectory = await mkdtemp(join(tmpdir(), 'phone-page-assets-'));
  await $`${join(root, 'phone-page/phone-page')} assets ${assetsDirectory}`;
  process.env.PHONE_PAGE_ASSETS_DIR = assetsDirectory;
  return () => $`rm -rf ${assetsDirectory}`;
}
