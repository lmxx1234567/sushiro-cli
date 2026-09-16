import assert from 'node:assert/strict';
import { test } from 'node:test';
import { packagedMarkdown, repository, verifyPublicSource } from '../../scripts/release/policy.mjs';

test('packaged legal links resolve locally or to public source, never internal missing docs', () => {
  assert.equal(packagedMarkdown('[MIT](LICENSE)', ['LICENSE']), '[MIT](LICENSE)');
  assert.match(packagedMarkdown('[source](docs/open-source/provenance.md)', [], 'a'.repeat(40)), /github.com\/lmxx1234567\/sushiro-cli\/blob\/a{40}\//);
  assert.match(packagedMarkdown('[source](docs/open-source/provenance.md)', [], 'a'.repeat(40)), /\/docs\/provenance\.md\)/);
  assert.throws(() => packagedMarkdown('[audit](docs/open-source/privacy-audit.md)', []), /unresolved relative/);
});

test('publish preparation requires a publicly accessible exact GitHub source commit', async () => {
  const original = globalThis.fetch;
  const commit = 'a'.repeat(40);
  try {
    await assert.rejects(verifyPublicSource('main'), /full source commit/);
    let calls = [];
    globalThis.fetch = async (url, opts) => {
      calls.push(url);
      assert.equal(opts.redirect, 'error');
      assert.equal(opts.headers.Authorization, undefined);
      return { ok: true, json: async () => url.endsWith(commit) ? { sha: commit } : { private: false, html_url: repository } };
    };
    const result = await verifyPublicSource(commit);
    assert.equal(result.commit, commit); assert.equal(result.repository, repository); assert.equal(calls.length, 2);
    globalThis.fetch = async () => ({ ok: false, status: 404 });
    await assert.rejects(verifyPublicSource(commit), /HTTP 404/);
    globalThis.fetch = async () => ({ ok: true, json: async () => ({ private: true, html_url: repository }) });
    await assert.rejects(verifyPublicSource(commit), /already be public/);
    globalThis.fetch = async url => ({ ok: true, json: async () => url.endsWith(commit) ? { sha: 'b'.repeat(40) } : { private: false, html_url: repository } });
    await assert.rejects(verifyPublicSource(commit), /does not match/);
  } finally { globalThis.fetch = original; }
});
