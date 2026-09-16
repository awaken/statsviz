import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
import { mkdirSync, mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import test from 'node:test';

const scripts = dirname(fileURLToPath(import.meta.url));
const root = resolve(scripts, '../../../../../..');

test('release uses the lockfile offline and reproduces its archive', () => {
  const dir = mkdtempSync(join(root, 'tmp/statsviz-release-'));
  try {
    mkdirSync(join(dir, 'scripts'));
    for (const name of ['release.sh', 'zip.sh', 'zip.mjs']) {
      writeFileSync(join(dir, 'scripts', name), readFileSync(join(scripts, name)));
    }
    const pkg = { name: 'release-fixture', version: '1.0.0', scripts: { build: 'node build.mjs' } };
    writeFileSync(join(dir, 'package.json'), JSON.stringify(pkg));
    const lock = JSON.stringify({ name: pkg.name, version: pkg.version, lockfileVersion: 3, packages: { '': pkg } });
    writeFileSync(join(dir, 'package-lock.json'), lock);
    writeFileSync(join(dir, 'build.mjs'), "import {mkdirSync,writeFileSync} from 'node:fs'; mkdirSync('dist'); writeFileSync('dist/index.html','fixture');");
    const env = { ...process.env, npm_config_cache: join(dir, 'cache'), STATSVIZ_RELEASE_IMAGE: '' };
    const run = () => execFileSync('sh', [join(dir, 'scripts/release.sh')], { cwd: dir, env, stdio: 'pipe' });
    assert.throws(run, /immutable image digest/);
    env.STATSVIZ_RELEASE_IMAGE = `sha256:${'a'.repeat(64)}`;
    run();
    const first = readFileSync(join(dir, 'dist.zip'));
    run();
    assert.deepEqual(readFileSync(join(dir, 'dist.zip')), first);
    assert.equal(readFileSync(join(dir, 'package-lock.json'), 'utf8'), lock);
    pkg.dependencies = { 'missing-offline-fixture': '1.0.0' };
    writeFileSync(join(dir, 'package.json'), JSON.stringify(pkg));
    assert.throws(run);
    assert.deepEqual(readFileSync(join(dir, 'dist.zip')), first);
  } finally {
    rmSync(dir, { recursive: true, force: true });
  }
});
