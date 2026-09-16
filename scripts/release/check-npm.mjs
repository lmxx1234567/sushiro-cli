// Read-only registry metadata and current npm identity. No login or publish.
import { spawnSync } from 'node:child_process';
import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { options, platform, targets, writeJSON } from './common.mjs';

const args = options(['--out']);
if (!args['--out']) throw new Error('--out is required');
const registry = 'https://registry.npmjs.org/';
const bases = ['sushiro-cli', '@lmxx1234567/sushiro-cli'];
const names = bases.flatMap(name => [name, ...targets.map(target => {
  const { os, cpu } = platform(target);
  return `${name}-${os}-${cpu}`;
})]);
const packages = await Promise.all(names.map(async name => {
  try {
    // Native fetch has no npm credentials. These are public metadata requests.
    const response = await fetch(registry + encodeURIComponent(name), { redirect: 'error', signal: AbortSignal.timeout(20_000) });
    const result = { name, http_status: response.status, state: response.status === 404 ? 'not_found' : response.ok ? 'exists' : 'unknown' };
    if (response.ok) result.latest = (await response.json())['dist-tags']?.latest;
    return result;
  } catch { return { name, state: 'unknown', error: 'metadata_request_failed' }; }
}));
const cache = mkdtempSync(path.join(os.tmpdir(), 'sushiro-npm-identity-'));
let identity;
try {
  const result = spawnSync(process.platform === 'win32' ? 'npm.cmd' : 'npm',
    ['whoami', '--json', '--registry=' + registry, '--loglevel=silent', '--logs-max=0', '--cache=' + cache],
    { encoding: 'utf8', timeout: 25_000, shell: process.platform === 'win32' });
  let output;
  try { output = JSON.parse(result.stdout); } catch { /* only report bounded status */ }
  if (result.status === 0 && typeof output === 'string' && /^[a-z0-9][a-z0-9._-]*$/i.test(output)) {
    identity = { state: 'authenticated', username: output, matches_candidate_user_scope: output === 'lmxx1234567' };
  } else {
    const code = output?.error?.code;
    identity = { state: code === 'ENEEDAUTH' ? 'not_authenticated' : 'unverified', ...(typeof code === 'string' && /^[A-Z0-9_]+$/.test(code) ? { code } : {}) };
  }
} finally { rmSync(cache, { recursive: true, force: true }); }
const report = { checked_at: new Date().toISOString(), registry, packages, identity, publication_performed: false,
  limitation: 'A public 404 does not reserve a name or prove publish permission. GitHub ownership does not prove npm scope ownership.' };
writeJSON(path.resolve(args['--out']), report);
console.log(JSON.stringify(report, null, 2));
