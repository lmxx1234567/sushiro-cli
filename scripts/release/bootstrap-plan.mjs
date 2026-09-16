// Prints reviewed argv only. Never authenticates or publishes.
import { existsSync } from 'node:fs';
import path from 'node:path';
import { json, npmTargets, options, sha256 } from './common.mjs';
import { validatePackageSet } from './release-model.mjs';

const args = options(['--dir']);
if (!args['--dir']) throw new Error('--dir with downloaded release assets is required');
const dir = path.resolve(args['--dir']), release = json(path.join(dir, 'release.json')), identity = json(path.join(dir, 'event.json'));
const plan = validatePackageSet(release, identity).map((pkg, index) => {
  if (!/^tarballs\/[A-Za-z0-9._-]+\.tgz$/.test(pkg.tarball)) throw new Error('invalid bootstrap tarball path');
  const nested = path.join(dir, pkg.tarball), flat = path.join(dir, path.basename(pkg.tarball));
  const file = existsSync(nested) ? nested : flat;
  if (sha256(file) !== pkg.sha256) throw new Error(`bootstrap artifact mismatch: ${pkg.name}`);
  return { order: index + 1, name: pkg.name, version: pkg.version, sha256: pkg.sha256,
    argv: ['npm', 'publish', file, '--registry', 'https://registry.npmjs.org/', '--access', 'public', '--tag', identity.npmTag, '--ignore-scripts'] };
});
console.log(JSON.stringify({ execute: false, npm_targets: npmTargets(release.npmTargets), source_commit: identity.commit, npm_tag: identity.npmTag, prerequisite: 'separate human approval and authenticated npm ownership; publish exact reviewed artifacts only', plan }, null, 2));
