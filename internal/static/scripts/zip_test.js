import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { chmodSync, existsSync, lstatSync, mkdirSync, mkdtempSync, readFileSync, readdirSync, rmSync, utimesSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const scripts = dirname(fileURLToPath(import.meta.url));
const staticRoot = dirname(scripts);
const root = resolve(scripts, '../../../../../..');

test('embedded archive matches the current distribution', t => {
  if (!existsSync(join(staticRoot, 'dist'))) {
    t.skip('build the frontend to compare dist/ with dist.zip');
    return;
  }
  const dir = mkdtempSync(join(root, 'tmp/statsviz-archive-'));
  try {
    const copyTree = (source, target) => {
      mkdirSync(target, { recursive: true });
      for (const name of readdirSync(source)) {
        const input = join(source, name);
        const output = join(target, name);
        if (lstatSync(input).isDirectory()) copyTree(input, output);
        else writeFileSync(output, readFileSync(input));
      }
    };
    mkdirSync(join(dir, 'scripts'));
    writeFileSync(join(dir, 'scripts/zip.mjs'), readFileSync(join(scripts, 'zip.mjs')));
    copyTree(join(staticRoot, 'dist'), join(dir, 'dist'));
    execFileSync('node', [join(dir, 'scripts/zip.mjs')]);
    assert.deepEqual(readFileSync(join(dir, 'dist.zip')), readFileSync(join(staticRoot, 'dist.zip')));
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});

test('ZIP bytes do not depend on source metadata or timezone', () => {
  const dir = mkdtempSync(join(root, 'tmp/statsviz-zip-'));
  try {
    mkdirSync(join(dir, 'scripts'));
    for (const name of readdirSync(scripts)) {
      writeFileSync(join(dir, 'scripts', name), readFileSync(join(scripts, name)));
    }
    mkdirSync(join(dir, 'dist/assets'), { recursive: true });
    writeFileSync(join(dir, 'dist/index.html'), '<html>fixture</html>');
    writeFileSync(join(dir, 'dist/assets/a.js'), 'export const a=1;');
    execFileSync('sh', [join(dir, 'scripts/zip.sh')], { env: { ...process.env, TZ: 'UTC' } });
    const first = readFileSync(join(dir, 'dist.zip'));
    for (const name of ['dist/index.html', 'dist/assets/a.js']) {
      utimesSync(join(dir, name), new Date('2001-01-01'), new Date('2001-01-01'));
      chmodSync(join(dir, name), 0o600);
    }
    execFileSync('sh', [join(dir, 'scripts/zip.sh')], { env: { ...process.env, TZ: 'Pacific/Honolulu' } });
    assert.deepEqual(readFileSync(join(dir, 'dist.zip')), first);
    assert.equal(first.readUInt32LE(0), 0x04034b50);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
