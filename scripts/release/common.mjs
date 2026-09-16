import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import path from 'node:path';

export const root = fileURLToPath(new URL('../../', import.meta.url));
export const auditedGo = 'go1.26.5';
export const targets = ['darwin-amd64', 'darwin-arm64', 'linux-amd64', 'linux-arm64', 'windows-amd64', 'windows-arm64'];
export function platform(target) {
  const [goos, goarch] = target.split('-');
  return { goos, goarch, os: goos === 'windows' ? 'win32' : goos, cpu: goarch === 'amd64' ? 'x64' : goarch, binary: goos === 'windows' ? 'sushiro-cli.exe' : 'sushiro-cli' };
}
export function options(allowed) {
  const result = {};
  for (let i = 2; i < process.argv.length; i++) {
    const key = process.argv[i];
    if (!allowed.includes(key)) throw new Error(`unknown option: ${key}`);
    if (key === '--publish-ready' || key === '--execute') result[key] = true;
    else {
      const value = process.argv[++i];
      if (!value || value.startsWith('--')) throw new Error(`missing value: ${key}`);
      result[key] = value;
    }
  }
  return result;
}
export function run(command, args, opts = {}) {
  const result = spawnSync(command, args, { encoding: 'utf8', cwd: root, ...opts });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} failed (${result.status}): ${result.stderr || result.stdout}`);
  return result.stdout.trim();
}
export function npm(args, opts = {}) {
  // NPM_CLI is an optional local npm-cli.js path, useful with bundled Node runtimes.
  const candidates = [process.env.NPM_CLI,
    path.join(path.dirname(process.execPath), 'node_modules/npm/bin/npm-cli.js'),
    path.resolve(path.dirname(process.execPath), '../lib/node_modules/npm/bin/npm-cli.js')];
  const cli = candidates.find(file => file && existsSync(file));
  if (cli) return run(process.execPath, [cli, ...args], opts);
  if (process.platform === 'win32') throw new Error('Set NPM_CLI to the installed npm-cli.js path');
  return run('npm', args, opts);
}
export function json(file) { return JSON.parse(readFileSync(file, 'utf8')); }
export function writeJSON(file, data) { writeFileSync(file, JSON.stringify(data, null, 2) + '\n'); }
export function sha256(file) { return createHash('sha256').update(readFileSync(file)).digest('hex'); }
export function version(value) {
  const number = '(?:0|[1-9][0-9]*)';
  const identifier = `(?:${number}|[0-9]*[a-zA-Z-][0-9a-zA-Z-]*)`;
  if (typeof value !== 'string' || value.trim() !== value || !new RegExp(`^${number}\\.${number}\\.${number}(?:-${identifier}(?:\\.${identifier})*)?$`).test(value)) {
    throw new Error('provide an exact SemVer --version (build metadata is not supported)');
  }
  return value;
}
export function artifact(dist, target) { return path.join(dist, target, platform(target).binary); }
