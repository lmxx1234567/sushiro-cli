import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { artifact, auditedGo, options, platform, root, run, sha256, targets, version, writeJSON } from './common.mjs';

const args = options(['--version', '--dist']);
const releaseVersion = version(args['--version']);
const goVersion = run('go', ['version']);
if (!goVersion.startsWith(`go version ${auditedGo} `)) throw new Error(`license bundle audited for ${auditedGo}; use that toolchain or refresh the license audit`);
const dist = path.resolve(args['--dist'] || path.join(root, 'packaging/out/dist'));
const commit = run('git', ['rev-parse', 'HEAD']);
const dirty = Boolean(run('git', ['status', '--porcelain']));
const binaries = {};
for (const target of targets) {
  const { goos, goarch } = platform(target);
  const output = artifact(dist, target);
  mkdirSync(path.dirname(output), { recursive: true });
  run('go', ['build', '-trimpath', '-buildvcs=false', '-ldflags', `-s -w -X main.version=${releaseVersion}`, '-o', output, './cmd/sushiro-cli'], {
    env: { ...process.env, CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch },
  });
  binaries[target] = sha256(output);
  console.log(`built ${target}`);
}
if (run('git', ['rev-parse', 'HEAD']) !== commit) throw new Error('source commit changed during build; rebuild from a stable checkout');
const dirtyAfterBuild = Boolean(run('git', ['status', '--porcelain']));
writeJSON(path.join(dist, 'build.json'), { version: releaseVersion, entry: './cmd/sushiro-cli', commit, dirty: dirty || dirtyAfterBuild, go: goVersion, binaries });
