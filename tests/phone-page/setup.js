import { execFileSync } from 'node:child_process';
import { rm, mkdtemp } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

export default async function setup() {
  const testDirectory = dirname(fileURLToPath(import.meta.url));
  const root = resolve(testDirectory, '../..');
  const assetsDirectory = await mkdtemp(join(tmpdir(), 'phone-page-assets-'));
  execFileSync(join(root, 'phone-page/phone-page'), ['assets', assetsDirectory], { stdio: 'inherit' });
  process.env.PHONE_PAGE_ASSETS_DIR = assetsDirectory;
  return () => rm(assetsDirectory, { recursive: true, force: true });
}
