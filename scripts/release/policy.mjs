import { existsSync } from 'node:fs';
import path from 'node:path';
import { root } from './common.mjs';

export const repository = 'https://github.com/lmxx1234567/sushiro-cli';
export const requiredLegalFiles = ['LICENSE', 'THIRD_PARTY_NOTICES.md', 'THIRD_PARTY_LICENSES.txt', 'CREDITS.md', 'PRIVACY.md', 'DISCLAIMER.md', 'SECURITY.md'];
export function packagedMarkdown(text, filenames, commit = 'main') {
  // Keep local legal links local. Public source references point to the exact
  // release commit; internal or missing documents must be removed at source.
  return text.replace(/\]\(([^\s)]+)\)/g, (link, target) => {
    if (/^https?:\/\//.test(target) || target.startsWith('#') || target.startsWith('mailto:')) return link;
    const file = target.split('#')[0];
    if (filenames.includes(file)) return link;
    const publicFile = file === 'docs/open-source/provenance.md' ? 'docs/provenance.md' : file;
    if (['docs/provenance.md', 'docs/configuration.md', 'docs/limitations.md'].includes(publicFile)) {
      const fragment = target.includes('#') ? '#' + target.split('#').slice(1).join('#') : '';
      return `](${repository}/blob/${commit}/${publicFile}${fragment})`;
    }
    throw new Error(`packaged documentation has an unresolved relative link: ${target}`);
  });
}
export function legalFilesFor(config, configPath, publishReady) {
  if (publishReady) for (const file of requiredLegalFiles) {
    if (!existsSync(path.join(root, file))) throw new Error(`publish preparation requires final ${file}`);
  }
  return [...new Set([
    // These two files were mandatory even before the open-source release stage.
    path.join(root, 'THIRD_PARTY_NOTICES.md'), path.join(root, 'THIRD_PARTY_LICENSES.txt'),
    ...['LICENSE', 'CREDITS.md', 'PRIVACY.md', 'DISCLAIMER.md', 'SECURITY.md', 'NOTICE', 'NOTICE.md'].map(file => path.join(root, file)).filter(existsSync),
    ...(config.legalFiles || []).map(file => path.resolve(path.dirname(configPath), file)),
  ])];
}

// Public GitHub source must precede preparation of publishable npm archives.
// Never use local GitHub credentials or an authenticated/private-repo fallback.
export async function verifyPublicSource(commit) {
  if (!/^[0-9a-f]{40}$/.test(commit || '')) throw new Error('publish preparation requires a full source commit');
  const api = 'https://api.github.com/repos/lmxx1234567/sushiro-cli';
  const request = async suffix => {
    const response = await fetch(api + suffix, { redirect: 'error', headers: { Accept: 'application/vnd.github+json' }, signal: AbortSignal.timeout(20_000) });
    if (!response.ok) throw new Error(`GitHub public source verification failed: HTTP ${response.status}`);
    return response.json();
  };
  const project = await request('');
  if (project.private !== false || project.html_url !== repository) throw new Error('GitHub repository must already be public');
  const source = await request(`/commits/${commit}`);
  if (source.sha !== commit) throw new Error('GitHub source commit does not match build.json');
  return { repository, commit, url: `${repository}/commit/${commit}`, verified_at: new Date().toISOString() };
}
