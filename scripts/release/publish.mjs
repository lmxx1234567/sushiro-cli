// Actual writes require --execute inside the approved GitHub release workflow.
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { json, npm, npmTargets, options, sha256 } from './common.mjs';
import { publicationPlan, validatePackageSet } from './release-model.mjs';

const args = options(['--release', '--event', '--execute']);
if (!args['--release'] || !args['--event']) throw new Error('--release and --event are required');
const release = json(args['--release']), identity = json(args['--event']);
const packages = validatePackageSet(release, identity);
console.log(`Declared npm targets: ${npmTargets(release.npmTargets).join(", ")}; ${packages.length} packages, main last.`);
const base = path.dirname(path.resolve(args['--release']));
for (const pkg of packages) {
  if (!/^tarballs\/[a-zA-Z0-9._-]+\.tgz$/.test(pkg.tarball)) throw new Error('invalid tarball path');
  pkg.file = path.join(base, pkg.tarball);
  if (sha256(pkg.file) !== pkg.sha256 || 'sha512-' + createHash('sha512').update(readFileSync(pkg.file)).digest('base64') !== pkg.integrity) throw new Error(`local artifact mismatch: ${pkg.name}`);
}
const registry = 'https://registry.npmjs.org/';
const metadata = {};
for (const pkg of packages) {
  const response = await fetch(registry + encodeURIComponent(pkg.name), { redirect: 'error', signal: AbortSignal.timeout(20_000) });
  if (response.status === 404) metadata[pkg.name] = null;
  else if (response.ok) metadata[pkg.name] = await response.json();
  else throw new Error(`registry preflight failed: HTTP ${response.status}`);
}
const plan = publicationPlan(packages, metadata, identity.npmTag);
if (args['--execute']) {
  if (process.env.GITHUB_ACTIONS !== 'true' || process.env.GITHUB_EVENT_NAME !== 'release' || process.env.GITHUB_REPOSITORY !== 'lmxx1234567/sushiro-cli' || !process.env.ACTIONS_ID_TOKEN_REQUEST_URL) throw new Error('execution requires the GitHub release OIDC environment');
  if (process.env.GITHUB_SHA !== identity.commit) throw new Error('publish source must match the original release event commit');
  if (npm(['--version']) !== '11.6.2') throw new Error('publish runner must use pinned npm 11.6.2');
}
for (const pkg of plan) {
  console.log(`${pkg.action}: ${pkg.name}@${pkg.version}`);
  if (!args['--execute'] || pkg.action !== 'publish') continue;
  // A failed or uncertain publish is never retried automatically. Rerun first reconciles integrity.
  npm(['publish', pkg.file, '--registry', registry, '--access', 'public', '--tag', identity.npmTag, '--provenance', '--ignore-scripts']);
  const response = await fetch(registry + encodeURIComponent(pkg.name) + '/' + encodeURIComponent(pkg.version), { redirect: 'error', signal: AbortSignal.timeout(20_000) });
  if (!response.ok || (await response.json()).dist?.integrity !== pkg.integrity) throw new Error(`published artifact verification pending or mismatched: ${pkg.name}; stop and reconcile before rerun`);
}
