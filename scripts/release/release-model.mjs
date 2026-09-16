import { platform, targets, version } from './common.mjs';

export function releaseIdentity(event, commit) {
  if (event.action !== 'published' || event.release?.draft || event.repository?.full_name !== 'lmxx1234567/sushiro-cli' || event.repository.private !== false) throw new Error('requires a published release in the public lmxx1234567/sushiro-cli repository');
  const tag = event.release.tag_name;
  if (typeof tag !== 'string') throw new Error('release tag is required');
  const exactVersion = version(tag.replace(/^v/, ''));
  const prerelease = exactVersion.includes('-');
  if (event.release.prerelease !== prerelease) throw new Error('release prerelease flag must match SemVer');
  if (!/^[0-9a-f]{40}$/.test(commit)) throw new Error('release source commit must be full SHA');
  if (!Number.isSafeInteger(event.release.id) || event.release.id <= 0) throw new Error('release id must be a positive integer');
  return { tag, version: exactVersion, npmTag: prerelease ? 'next' : 'latest', commit, releaseId: event.release.id };
}

export function validatePackageSet(release, identity) {
  version(identity.version);
  if (identity.npmTag !== (identity.version.includes('-') ? 'next' : 'latest')) throw new Error('npm dist-tag must match prerelease status');
  if (!release.publishReady || release.version !== identity.version || release.build?.version !== identity.version || release.build?.dirty || release.build?.commit !== identity.commit || release.publicSource?.commit !== identity.commit) throw new Error('release tag, source, build and package versions must match');
  const names = targets.map(target => { const { os, cpu } = platform(target); return `${release.name}-${os}-${cpu}`; }).concat(release.name);
  if (release.packages?.length !== 7 || release.packages.some((pkg, index) => pkg.name !== names[index] || pkg.version !== identity.version || !pkg.integrity?.startsWith('sha512-'))) throw new Error('expected six exact-version platform packages followed by the main package');
  return release.packages;
}

export function publicationPlan(packages, metadata, npmTag) {
  // Inspect every package before any publish, so a known conflict causes zero writes.
  return packages.map(pkg => {
    const remote = metadata[pkg.name];
    if (!remote) throw new Error(`bootstrap required: create ${pkg.name} and configure its trusted publisher first`);
    const existing = remote.versions?.[pkg.version];
    if (existing) {
      if (existing.dist?.integrity !== pkg.integrity) throw new Error(`integrity conflict: ${pkg.name}@${pkg.version}`);
      return { ...pkg, action: 'verified-existing' };
    }
    const latest = remote['dist-tags']?.latest;
    if (npmTag === 'latest' && /^\d+\.\d+\.\d+$/.test(latest || '')) {
      const a = latest.split('.').map(Number), b = pkg.version.split('.').map(Number);
      const differing = a.findIndex((value, index) => value !== b[index]);
      if (differing !== -1 && a[differing] > b[differing]) throw new Error(`refusing to move latest backward for ${pkg.name}`);
    }
    return { ...pkg, action: 'publish' };
  });
}

export async function assetPlan(files, existing, remoteHash) {
  const missing = [];
  for (const file of files) {
    const prior = existing.find(asset => asset.name === file.name);
    if (!prior) { missing.push(file); continue; }
    if (await remoteHash(prior) !== file.sha256) throw new Error(`existing release asset differs: ${file.name}; never overwrite it`);
  }
  return missing;
}
