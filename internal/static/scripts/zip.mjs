import { execFileSync } from 'node:child_process';
import { chmodSync, lstatSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, renameSync, rmSync, utimesSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

// Stage regular files with stable names, modes and UTC timestamps. zip -X drops
// host-specific extra fields; the release container pins the compressor too.
const root = dirname(dirname(fileURLToPath(import.meta.url)));
const stage = mkdtempSync(join(root, '.statsviz-zip-'));
const epoch = new Date('1980-01-01T00:00:00Z');
const names = [];
try {
  function copy(name) {
    const source = join(root, name);
    const info = lstatSync(source);
    if (info.isDirectory()) {
      for (const child of readdirSync(source).sort()) copy(`${name}/${child}`);
    } else if (info.isFile() && !/[\r\n]/.test(name)) {
      const target = join(stage, name);
      mkdirSync(dirname(target), { recursive: true });
      writeFileSync(target, readFileSync(source));
      chmodSync(target, 0o644);
      utimesSync(target, epoch, epoch);
      names.push(name);
    } else {
      throw new Error(`Unsupported archive entry: ${name}`);
    }
  }
  copy('dist');
  if (names.length === 0) throw new Error('No dist files to package');
  names.sort();
  execFileSync('zip', ['-X', '-q', 'result.zip', '-@'], {
    cwd: stage,
    env: { ...process.env, TZ: 'UTC', LC_ALL: 'C' },
    input: `${names.join('\n')}\n`,
  });
  renameSync(join(stage, 'result.zip'), join(root, 'dist.zip'));
} finally {
  rmSync(stage, { recursive: true, force: true });
}
