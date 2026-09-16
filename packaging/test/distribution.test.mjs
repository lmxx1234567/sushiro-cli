import assert from 'node:assert/strict';
import { spawn, spawnSync } from 'node:child_process';
import { chmodSync, existsSync, mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { test } from 'node:test';
import { artifact, json, npm, platform, root, run, sha256, targets, writeJSON } from '../../scripts/release/common.mjs';

test('six packages: offline pack/install, streams, arguments, exit status, signals and errors', { timeout: 300_000 }, async () => {
  const temp = mkdtempSync(path.join(os.tmpdir(), 'sushiro npm-test-'));
  let success = false;
  try {
    const dist = path.join(temp, 'dist');
    for (const target of targets) {
      const { goos, goarch } = platform(target);
      const output = artifact(dist, target);
      mkdirSync(path.dirname(output), { recursive: true });
      run('go', ['build', '-trimpath', '-o', output, path.join(root, 'packaging/test/fixture.go')], {
        env: { ...process.env, GO111MODULE: 'off', GOWORK: 'off', CGO_ENABLED: '0', GOOS: goos, GOARCH: goarch },
      });
    }
    console.log('Fixture cross-builds completed: ' + targets.join(', '));
    const config = path.join(temp, 'config.json');
    writeJSON(config, { scope: '@sushiro-local-test', name: 'client', license: 'UNLICENSED' });
    const out = path.join(temp, 'release');
    const releaseVersion = '0.0.0-test.1';
    const args = [path.join(root, 'scripts/release/prepare.mjs'), '--config', config, '--dist', dist, '--out', out, '--version', releaseVersion];
    run(process.execPath, args);
    assert.throws(() => run(process.execPath, args), /output already exists/);
    assert.throws(() => run(process.execPath, [...args, '--version', '01.0.0']), /SemVer/);
    assert.throws(() => run(process.execPath, [...args, '--publish-ready']), /publish preparation requires/);
    const legal = path.join(temp, 'TEST-NOTICE');
    writeFileSync(legal, 'TEST FIXTURE ONLY. Not a release license.\n');
    writeJSON(config, { scope: '@sushiro-local-test', name: 'client', license: 'MIT', legalFiles: ['TEST-NOTICE'] });
    assert.throws(() => run(process.execPath, [...args, '--publish-ready']), /matching build.json/);
    const buildPath = path.join(dist, 'build.json');
    const fixtureBuild = { version: releaseVersion, entry: 'packaging/test/fixture.go', dirty: false, binaries: Object.fromEntries(targets.map(target => [target, sha256(artifact(dist, target))])) };
    writeJSON(buildPath, fixtureBuild);
    assert.throws(() => run(process.execPath, [...args, '--publish-ready']), /matching build.json/);
    writeJSON(buildPath, { ...fixtureBuild, binaries: { ...fixtureBuild.binaries, 'darwin-amd64': 'corrupt' } });
    assert.throws(() => run(process.execPath, args), /build manifest mismatch/);
    rmSync(buildPath);
    const release = json(path.join(out, 'release.json'));
    assert.equal(release.packages.length, 7);
    const main = json(path.join(out, 'packages/main/package.json'));
    assert.equal(main.private, true);
    assert.equal(Object.keys(main.optionalDependencies).length, 6);
    assert.deepEqual(new Set(Object.values(main.optionalDependencies)), new Set([main.version]));
    for (const pkg of release.packages) {
      assert.equal(pkg.version, main.version);
      const files = new Set(pkg.files);
      for (const file of ['README.md', 'legal/THIRD_PARTY_NOTICES.md', 'legal/THIRD_PARTY_LICENSES.txt', pkg.name === main.name ? 'bin/sushiro-cli.cjs' : pkg.name.includes('-win32-') ? 'bin/sushiro-cli.exe' : 'bin/sushiro-cli', 'package.json']) assert.ok(files.delete(file), `missing ${file}`);
      for (const file of files) assert.ok(['LICENSE','CREDITS.md','PRIVACY.md','DISCLAIMER.md','SECURITY.md','NOTICE','NOTICE.md'].some(name => file === `legal/${name}`), `unexpected packed file ${file}`);
    }
    const install = path.join(temp, 'install');
    mkdirSync(install);
    const local = Object.fromEntries(release.packages.map(pkg => [pkg.name, `file:${path.join(out, pkg.tarball)}`]));
    const mainTarball = local[main.name];
    delete local[main.name];
    writeJSON(path.join(install, 'package.json'), { name: 'isolated-install-test', version: '1.0.0', private: true, dependencies: { [main.name]: mainTarball }, optionalDependencies: local });
    const userConfig = path.join(temp, 'empty-user.npmrc');
    const globalConfig = path.join(temp, 'empty-global.npmrc');
    writeFileSync(userConfig, ''); writeFileSync(globalConfig, '');
    npm(['install', '--offline', '--ignore-scripts', '--no-audit', '--no-fund', '--cache', path.join(temp, 'cache'), '--userconfig', userConfig, '--globalconfig', globalConfig], { cwd: install });
    const nativeName = `${main.name}-${process.platform}-${process.arch}`;
    const nativeDir = path.join(install, 'node_modules', nativeName);
    assert.ok(existsSync(nativeDir));
    for (const name of Object.keys(local)) assert.equal(existsSync(path.join(install, 'node_modules', name)), name === nativeName);
    const launcher = path.join(install, 'node_modules', main.name, 'bin/sushiro-cli.cjs');
    const execute = (argv, input = '') => spawnSync(process.execPath, [launcher, ...argv], { encoding: 'utf8', input, timeout: 10_000 });
    const echo = execute(['echo', 'space value', '寿司郎', '--flag=a&b', ''], 'stdin\n第二行\n');
    assert.equal(echo.status, 0);
    assert.deepEqual(JSON.parse(echo.stdout), { fixture: true, args: ['space value', '寿司郎', '--flag=a&b', ''], stdin: 'stdin\n第二行\n' });
    assert.equal(echo.stderr, 'FIXTURE stderr\n');
    const protocol = '{"jsonrpc":"2.0","id":1,"method":"initialize"}\n';
    const mcp = execute(['mcp'], protocol);
    assert.equal(mcp.stdout, protocol); assert.equal(mcp.stderr, ''); assert.equal(mcp.status, 0);
    for (const code of [0, 7, 130, 255]) assert.equal(execute(['exit', String(code)]).status, code);
    const shim = path.join(install, 'node_modules/.bin', process.platform === 'win32' ? 'sushiro-cli.cmd' : 'sushiro-cli');
    const viaShim = spawnSync(process.platform === 'win32' ? `"${shim}"` : shim, ['exit', '7'], { encoding: 'utf8', timeout: 10_000, shell: process.platform === 'win32' });
    assert.equal(viaShim.status, 7, viaShim.stderr);
    if (process.platform !== 'win32') {
      for (const signal of ['SIGTERM', 'SIGINT', 'SIGHUP']) {
        const result = await signalAfterReady(launcher, 'signal', signal);
        assert.equal(result.code, 42); assert.equal(result.stderr, 'FIXTURE received signal\n');
      }
      const result = await signalAfterReady(launcher, 'self-signal', 'SIGTERM');
      assert.equal(result.signal, 'SIGTERM');
      const binary = path.join(nativeDir, 'bin/sushiro-cli');
      chmodSync(binary, 0o644);
      const denied = execute(['mcp']);
      assert.equal(denied.status, 1); assert.equal(denied.stdout, ''); assert.match(denied.stderr, /cannot start/);
      assert.equal(denied.stderr.includes(install), false);
      chmodSync(binary, 0o755);
    }
    const nativeManifest = path.join(nativeDir, 'package.json');
    const native = json(nativeManifest);
    writeJSON(nativeManifest, { ...native, version: '0.0.0-wrong' });
    const mismatch = execute(['mcp']);
    assert.equal(mismatch.status, 1); assert.equal(mismatch.stdout, ''); assert.match(mismatch.stderr, /version mismatch/);
    assert.equal(mismatch.stderr.includes(install), false);
    rmSync(nativeDir, { recursive: true });
    const missing = execute(['mcp']);
    assert.equal(missing.status, 1); assert.equal(missing.stdout, ''); assert.match(missing.stderr, /optional dependencies enabled/);
    assert.equal(missing.stderr.includes(install), false);
    // Explicitly synthetic release metadata: tests asset assembly, never publication.
    const synthetic = { ...release, publishReady: true, publicSource: { repository: 'https://github.com/lmxx1234567/sushiro-cli', commit: 'a'.repeat(40), verified_at: 'test-only' }, build: { ...fixtureBuild, commit: 'a'.repeat(40) } };
    const syntheticPath = path.join(out, 'test-release.json'), eventPath = path.join(temp, 'event.json');
    writeJSON(syntheticPath, synthetic);
    writeJSON(eventPath, { version: releaseVersion, npmTag: 'next', commit: 'a'.repeat(40), tag: `v${releaseVersion}`, releaseId: 1 });
    const assets = path.join(temp, 'assets');
    run(process.execPath, [path.join(root, 'scripts/release/assets.mjs'), '--release', syntheticPath, '--event', eventPath, '--out', assets]);
    const tar = path.join(assets, `sushiro-cli-${releaseVersion}-${process.platform === 'win32' ? 'windows' : process.platform}-${process.arch === 'x64' ? 'amd64' : process.arch}.tar.gz`);
    const names = run('tar', ['-tzf', tar]).split('\n');
    assert.ok(names.includes(process.platform === 'win32' ? 'sushiro-cli.exe' : 'sushiro-cli'));
    assert.ok(names.includes('legal/THIRD_PARTY_NOTICES.md'));
    assert.ok(names.includes('legal/THIRD_PARTY_LICENSES.txt'));
    const assetsAgain = path.join(temp, 'assets-again');
    synthetic.publicSource.verified_at = 'different-test-time'; writeJSON(syntheticPath, synthetic);
    run(process.execPath, [path.join(root, 'scripts/release/assets.mjs'), '--release', syntheticPath, '--event', eventPath, '--out', assetsAgain]);
    assert.equal(sha256(path.join(assets, 'SHA256SUMS')), sha256(path.join(assetsAgain, 'SHA256SUMS')));
    const bootstrap = JSON.parse(run(process.execPath, [path.join(root, 'scripts/release/bootstrap-plan.mjs'), '--dir', assets]));
    assert.equal(bootstrap.execute, false); assert.equal(bootstrap.plan.length, 7);
    assert.equal(bootstrap.plan.at(-1).name, main.name); assert.equal(bootstrap.npm_tag, 'next');
    // The package name may be unscoped; the command name stays sushiro-cli.
    writeJSON(config, { scope: '', name: 'sushiro-cli-local-test', license: 'UNLICENSED' });
    const unscopedOut = path.join(temp, 'unscoped');
    run(process.execPath, [...args, '--out', unscopedOut]);
    const unscoped = json(path.join(unscopedOut, 'packages/main/package.json'));
    assert.equal(unscoped.name, 'sushiro-cli-local-test');
    assert.deepEqual(unscoped.bin, { 'sushiro-cli': 'bin/sushiro-cli.cjs' });
    assert.equal(unscoped.optionalDependencies['sushiro-cli-local-test-darwin-arm64'], releaseVersion);
    console.log(`Runtime checks passed on ${process.platform}-${process.arch}; other targets only cross-built.`);
    success = true;
  } finally {
    if (success) rmSync(temp, { recursive: true, force: true });
    else console.error(`Test artifacts retained for inspection: ${temp}`);
  }
});

function signalAfterReady(launcher, mode, signal) {
  return new Promise((resolve, reject) => {
    const child = spawn(process.execPath, [launcher, mode], { stdio: ['pipe', 'pipe', 'pipe'] });
    let stderr = '', stdout = '', sent = false;
    const timeout = setTimeout(() => { child.kill('SIGKILL'); reject(new Error('signal test timed out')); }, 10_000);
    child.on('error', error => { clearTimeout(timeout); reject(error); });
    child.stderr.on('data', data => { stderr += data; });
    child.stdout.on('data', data => {
      stdout += data;
      if (!sent && stdout.includes('READY\n')) { sent = true; child.kill(signal); }
    });
    child.on('close', (code, signal) => { clearTimeout(timeout); resolve({ code, signal, stderr }); });
  });
}
