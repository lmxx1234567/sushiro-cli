// Package binaries and legal material after prepare --publish-ready.
import { copyFileSync, mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { json, npmTargets, options, platform, sha256, targets, writeJSON } from './common.mjs';
import { validatePackageSet } from './release-model.mjs';
import { archive } from './archive.mjs';

const args = options(['--release', '--event', '--out']);
for (const key of ['--release', '--event', '--out']) if (!args[key]) throw new Error(`${key} is required`);
const release = json(args['--release']), identity = json(args['--event']);
validatePackageSet(release, identity);
const source = path.dirname(path.resolve(args['--release'])), out = path.resolve(args['--out']);
mkdirSync(out); mkdirSync(path.join(out, 'tarballs'));
const files = [];
for (const pkg of release.packages) {
  if (!/^tarballs\/[a-zA-Z0-9._-]+\.tgz$/.test(pkg.tarball) || sha256(path.join(source, pkg.tarball)) !== pkg.sha256) throw new Error('npm artifact hash or path mismatch');
  copyFileSync(path.join(source, pkg.tarball), path.join(out, pkg.tarball)); files.push(pkg.tarball);
}
for (const target of targets) {
  const { os: npmOS, cpu, binary } = platform(target);
  const pkg = path.join(source, 'packages', `${npmOS}-${cpu}`);
  if (sha256(path.join(pkg, 'bin', binary)) !== release.build.binaries[target]) throw new Error(`binary mismatch: ${target}`);
  const entries = [{ name: binary, data: readFileSync(path.join(pkg, 'bin', binary)), mode: 0o755 }, { name: 'README.md', data: Buffer.from(`# sushiro-cli ${identity.version} — ${target}\n\nStandalone binary: extract this archive and run ${target.startsWith('windows-') ? '.\\' : './'}${binary} help.\n\n${npmTargets(release.npmTargets).includes(target) ? 'Alternatively install the main npm package: npm install -g ' + release.name + '.' : 'npm installation is unavailable for this target; use this standalone binary.'}\n\nLicense and privacy documents are in legal/.\n`) },
    ...readdirSync(path.join(pkg, 'legal')).map(file => ({ name: `legal/${file}`, data: readFileSync(path.join(pkg, 'legal', file)) }))];
  const filename = `sushiro-cli-${identity.version}-${target}.tar.gz`;
  writeFileSync(path.join(out, filename), archive(entries)); files.push(filename);
}
// Stable metadata allows reruns to compare release assets byte for byte.
const { verified_at, ...publicSource } = release.publicSource;
writeJSON(path.join(out, 'release.json'), { ...release, publicSource }); files.push('release.json');
writeJSON(path.join(out, 'event.json'), identity); files.push('event.json');
writeFileSync(path.join(out, 'SHA256SUMS'), files.sort().map(file => `${sha256(path.join(out, file))}  ${file}\n`).join(''));
console.log(`Prepared ${files.length} immutable release assets plus SHA256SUMS.`);
