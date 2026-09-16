// Explicit, read-only live acceptance. Not part of the offline test suite.
// Only stores/detail/slots are called; no personal or write tools are invoked.
import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { createInterface } from 'node:readline';
import { existsSync, readdirSync } from 'node:fs';
import path from 'node:path';
import { json, options, writeJSON } from '../../scripts/release/common.mjs';
import { publicShape } from './public-shape.mjs';

const args = options(['--install', '--config-dir', '--public-config-dir', '--mode', '--out']);
for (const key of ['--install', '--config-dir', '--public-config-dir', '--out']) assert.ok(args[key], `${key} is required`);
assert.ok(path.isAbsolute(args['--public-config-dir']), 'public configuration directory must be absolute');
const mode = args['--mode'] || 'configured';
assert.ok(['configured', 'default'].includes(mode), '--mode must be configured or default');
const publicFilesBefore = readdirSync(args['--public-config-dir']).sort();
if (mode === 'default') assert.deepEqual(publicFilesBefore, [], 'default acceptance requires an empty public directory');
else assert.ok(publicFilesBefore.includes('default.json'), 'configured acceptance requires the default public profile');
const install = path.resolve(args['--install']);
const configDir = path.resolve(args['--config-dir']);
assert.deepEqual(readdirSync(configDir), [], 'personal configuration directory must start empty');
const shim = path.join(install, 'node_modules/.bin', process.platform === 'win32' ? 'sushiro-cli.cmd' : 'sushiro-cli');
assert.ok(existsSync(shim), 'use an existing local npm installation');
const command = process.platform === 'win32' ? `"${shim}"` : shim;
const shell = process.platform === 'win32';
const env = Object.fromEntries(Object.entries(process.env).filter(([key]) => !key.startsWith('SUSHIRO_')));
env.SUSHIRO_CONFIG_DIR = configDir;
env.SUSHIRO_PUBLIC_CONFIG_DIR = args['--public-config-dir'];
const cases = [
  { label: 'stores', cli: ['stores', '--near', '39.97,116.43', '--limit', '2'], tool: 'stores', params: { near: '39.97,116.43', limit: 2 } },
  { label: 'store-3006', cli: ['stores', '--store', '3006'], tool: 'stores', params: { store_id: '3006' } },
  { label: 'slots-3006-2-T', cli: ['slots', '--store', '3006', '--adult', '2', '--child', '0', '--table', 'T'], tool: 'slots', params: { store_id: '3006', adult: 2, child: 0, table_type: 'T' } },
];
function summary(value, label) {
  return {
    ok: value?.ok === true, source: value?.source, state: value?.state,
    count: Array.isArray(value?.data) ? value.data.length : null,
    business_shape_valid: publicShape(label, value?.data),
    error_code: value?.error?.code, http_status: value?.error?.http_status,
  };
}
const report = { date: new Date().toISOString(), platform: `${process.platform}-${process.arch}`, mode, configuration: 'empty personal directory; explicit separate public directory; no inherited SUSHIRO settings', queries: cases.map(item => ({ name: item.tool, arguments: item.params })), cli: {}, mcp: {} };
const version = spawnSync(command, ['version', '--json'], { env, shell, encoding: 'utf8', timeout: 10_000 });
assert.equal(version.status, 0);
report.version = JSON.parse(version.stdout).data.version;
const release = path.resolve(install, '../release/release.json');
if (existsSync(release)) report.build = json(release).build;
for (const item of cases) {
  const result = spawnSync(command, [...item.cli, '--json'], { env, shell, encoding: 'utf8', timeout: 25_000 });
  let envelope;
  try { envelope = JSON.parse(result.stdout); } catch { /* record failure without logging raw output */ }
  report.cli[item.label] = { ...summary(envelope, item.label), exit: result.status, stderr_empty: result.stderr === '' };
}

const child = spawn(command, ['mcp'], { env, shell, stdio: ['pipe', 'pipe', 'pipe'] });
let nextID = 0, stderrBytes = 0;
const pending = new Map();
const closed = new Promise(resolve => child.on('close', (code, signal) => {
  for (const item of pending.values()) { clearTimeout(item.timer); item.reject(new Error('MCP closed before response')); }
  pending.clear(); resolve({ code, signal });
}));
child.stderr.on('data', data => { stderrBytes += data.length; });
child.on('error', () => { for (const item of pending.values()) item.reject(new Error('MCP launch failed')); });
createInterface({ input: child.stdout }).on('line', line => {
  try {
    const message = JSON.parse(line);
    const item = pending.get(message.id);
    if (item) {
      clearTimeout(item.timer); pending.delete(message.id);
      message.error ? item.reject(new Error('MCP protocol error')) : item.resolve(message.result);
    }
  } catch { child.kill('SIGTERM'); }
});
function request(method, params) {
  return new Promise((resolve, reject) => {
    const id = ++nextID;
    const timer = setTimeout(() => { pending.delete(id); reject(new Error('MCP response timeout')); }, 25_000);
    pending.set(id, { resolve, reject, timer });
    child.stdin.write(JSON.stringify({ jsonrpc: '2.0', id, method, params }) + '\n');
  });
}
try {
  const initialized = await request('initialize', { protocolVersion: '2025-06-18', capabilities: {}, clientInfo: { name: 'npm-public-acceptance', version: '1' } });
  report.mcp.server = initialized.serverInfo;
  child.stdin.write(JSON.stringify({ jsonrpc: '2.0', method: 'notifications/initialized' }) + '\n');
  report.mcp.tools = (await request('tools/list', {})).tools.map(tool => tool.name).sort();
  for (const item of cases) {
    const result = await request('tools/call', { name: item.tool, arguments: item.params });
    report.mcp[item.label] = { ...summary(result.structuredContent, item.label), is_error: result.isError === true };
  }
} catch {
  report.mcp.protocol_error = true;
} finally {
  child.stdin.end();
  const timer = setTimeout(() => child.kill('SIGKILL'), 5_000);
  report.mcp.exit = await closed;
  clearTimeout(timer);
  report.mcp.stderr_empty = stderrBytes === 0;
}
report.personal_directory_empty = readdirSync(configDir).length === 0;
report.public_directory_unchanged = JSON.stringify(readdirSync(args['--public-config-dir']).sort()) === JSON.stringify(publicFilesBefore);
report.passed = cases.every(item => {
  const cli = report.cli[item.label], mcp = report.mcp[item.label];
  return cli.ok && cli.source === 'live' && cli.business_shape_valid && cli.exit === 0 && cli.stderr_empty &&
    mcp?.ok && mcp.source === 'live' && mcp.business_shape_valid && !mcp.is_error;
}) && report.personal_directory_empty && report.public_directory_unchanged && report.mcp.exit.code === 0 && report.mcp.stderr_empty &&
  report.mcp.server?.version === report.version && report.mcp.server?.name === 'sushiro-cli' &&
  JSON.stringify(report.mcp.tools) === JSON.stringify(['cancel', 'reservations', 'reserve', 'slots', 'stores', 'ticket_status']);
writeJSON(path.resolve(args['--out']), report);
console.log(JSON.stringify(report, null, 2));
process.exitCode = report.passed ? 0 : 1;
