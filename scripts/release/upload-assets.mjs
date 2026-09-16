import { createHash } from 'node:crypto';
import { readdirSync } from 'node:fs';
import path from 'node:path';
import { json, options, run, sha256 } from './common.mjs';
import { assetPlan } from './release-model.mjs';

const args = options(['--dir']);
if (!args['--dir'] || process.env.GITHUB_ACTIONS !== 'true' || process.env.GITHUB_EVENT_NAME !== 'release') throw new Error('requires GitHub release workflow and --dir');
const dir = path.resolve(args['--dir']), identity = json(path.join(dir, 'event.json'));
const repo = 'lmxx1234567/sushiro-cli';
const remote = JSON.parse(run('gh', ['api', `repos/${repo}/releases/${identity.releaseId}`]));
if (remote.tag_name !== identity.tag || remote.draft) throw new Error('published GitHub release identity mismatch');
const files = readdirSync(dir, { withFileTypes: true }).flatMap(entry => entry.isDirectory()
  ? entry.name === 'tarballs' ? readdirSync(path.join(dir, entry.name)).map(file => path.join(dir, entry.name, file)) : []
  : [path.join(dir, entry.name)]);
const missing = await assetPlan(files.map(file => ({ name: path.basename(file), file, sha256: sha256(file) })), remote.assets, async existing => {
  const response = await fetch(existing.browser_download_url, { signal: AbortSignal.timeout(60_000) });
  if (!response.ok) throw new Error(`cannot verify existing asset ${existing.name}`);
  return createHash('sha256').update(Buffer.from(await response.arrayBuffer())).digest('hex');
});
if (missing.length) run('gh', ['release', 'upload', identity.tag, ...missing.map(file => file.file), '--repo', repo]);
console.log(`Release assets verified; uploaded ${missing.length}, reused ${files.length - missing.length}.`);
