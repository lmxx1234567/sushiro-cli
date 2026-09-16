import { chmodSync, copyFileSync, existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { artifact, auditedGo, json, npm, npmTargets, options, platform, root, sha256, targets, version, writeJSON } from './common.mjs';
import { legalFilesFor, packagedMarkdown, repository, verifyPublicSource } from './policy.mjs';

const args = options(['--config', '--version', '--dist', '--out', '--publish-ready']);
if (!args['--config']) throw new Error('--config is required; copy packaging/npm/config.example.json and choose your scope');
const configPath = path.resolve(args['--config']);
const config = json(configPath);
const releaseVersion = version(args['--version']);
const scope = config.scope ?? '';
if (typeof scope !== 'string' || scope.trim() !== scope || (scope !== '' && !/^@[a-z0-9][a-z0-9._-]*$/.test(scope)) || typeof config.name !== 'string' || config.name.trim() !== config.name || !/^[a-z0-9][a-z0-9._-]*$/.test(config.name)) {
  throw new Error('invalid scope or package name');
}
const selectedTargets = npmTargets(config.npmTargets);
const supportedOS = [...new Set(selectedTargets.map(target => platform(target).os))];
const name = scope ? `${scope}/${config.name}` : config.name;
if (name.length + '-windows-arm64'.length > 214) throw new Error('package name is too long');
const dist = path.resolve(args['--dist'] || path.join(root, 'packaging/out/dist'));
const out = path.resolve(args['--out'] || path.join(root, 'packaging/out/npm'));
const publishReady = Boolean(args['--publish-ready']);
const legalFiles = legalFilesFor(config, configPath, false);
const build = existsSync(path.join(dist, 'build.json')) ? json(path.join(dist, 'build.json')) : null;
if (publishReady) {
  if (scope.includes('replace-with') || config.license !== 'MIT') {
    throw new Error('publish preparation requires a chosen scope, license, and legalFiles');
  }
  if (!build || build.entry !== './cmd/sushiro-cli' || build.version !== releaseVersion || build.dirty) {
    throw new Error('publish preparation requires matching build.json from a clean real CLI build');
  }
  legalFilesFor(config, configPath, true);
  if (!build.go?.startsWith(`go version ${auditedGo} `)) throw new Error(`publish preparation requires the audited ${auditedGo} toolchain`);
}
// Validate all inputs before creating output. Never silently reuse a stale release directory.
for (const target of targets) {
  const digest = sha256(artifact(dist, target));
  if (build && (build.version !== releaseVersion || build.binaries[target] !== digest)) throw new Error(`build manifest mismatch: ${target}`);
}
const legalContent = new Map(legalFiles.map(file => [file, file.endsWith('.md')
  ? packagedMarkdown(readFileSync(file, 'utf8'), legalFiles.map(file => path.basename(file)), publishReady ? build.commit : 'main')
  : readFileSync(file)]));
if (new Set(legalFiles.map(file => path.basename(file))).size !== legalFiles.length) throw new Error('legal file basenames must be unique');
if (existsSync(out)) throw new Error(`output already exists; select a new --out directory: ${out}`);
const publicSource = publishReady ? await verifyPublicSource(build.commit) : null;
mkdirSync(path.join(out, 'tarballs'), { recursive: true });
const base = { version: releaseVersion, description: 'Sushiro China CLI and stdio MCP', license: config.license || 'UNLICENSED', private: !publishReady, publishConfig: { access: 'public' }, repository: { type: 'git', url: `git+${repository}.git` }, homepage: `${repository}#readme`, bugs: { url: `${repository}/issues` } };
const dependencies = {};
const packages = [];
function createPackage(directory, manifest, populate, pack = true) {
  mkdirSync(directory, { recursive: true });
  if (legalFiles.length) mkdirSync(path.join(directory, 'legal'));
  for (const file of legalFiles) writeFileSync(path.join(directory, 'legal', path.basename(file)), legalContent.get(file));
  writeJSON(path.join(directory, 'package.json'), manifest);
  writeFileSync(path.join(directory, 'README.md'), `# ${manifest.name}\n\nVersion ${releaseVersion}. Precompiled Sushiro CLI distribution.\n${publishReady ? '' : '\nPRIVATE LOCAL TEST PACKAGE — not approved for publishing.\n'}\nKeep optional dependencies enabled. No postinstall download is used.\n\nNo personal login or credential import is required to install, run help/version, or initialize MCP and discover tools.\n\nStart with \`sushiro-cli help\`, \`sushiro-cli version --json\`, or \`sushiro-cli mcp\`.\n\nPublic \`stores\` and \`slots\` queries are separate from personal authentication; the upstream may require legitimate query configuration. An empty result and a failed request are different outcomes.\n\nPersonal ticket/reservation queries and reservation/cancellation commands remain available and require their own credentials; writes also require explicit user authorization. Import personal credentials only when using those features. Native login is pending.\n`);
  const policyLinks = legalFiles.map(file => `[${path.basename(file)}](legal/${path.basename(file)})`).join(' · ');
  writeFileSync(path.join(directory, 'README.md'), `\n## License, privacy and attribution\n\n${policyLinks}\n\nThe project license does not replace third-party licenses. Built-in public query configuration is present in the binary; personal credentials are not bundled. npm uninstall does not remove application configuration.\n`, { flag: 'a' });
  if (!publishReady) writeFileSync(path.join(directory, 'README.md'), '\nSource links in this private test package are placeholders; their public availability has not been verified.\n', { flag: 'a' });
  const installHelp = manifest.bin
    ? `Install: \`npm install -g ${name}\`. Then run \`sushiro-cli help\`.\n\n`
    : `Platform dependency: npm installs this package automatically with \`${name}\`. Install the CLI with \`npm install -g ${name}\`; do not install this dependency directly.\n\n`;
  const limitation = supportedOS.includes('win32') ? '' : 'Windows npm installation is currently unavailable. Windows standalone binaries are available from GitHub Releases.\n\n';
  const readme = path.join(directory, 'README.md');
  writeFileSync(readme, `# ${manifest.name}\n\n${installHelp}${limitation}` + readFileSync(readme, 'utf8').replace(/^# [^\n]+\n\n/, ''));
  populate();
  if (!pack) return;
  const [packed] = JSON.parse(npm(['pack', '--json', '--ignore-scripts', '--cache', path.join(out, '.npm-cache'), '--pack-destination', path.join(out, 'tarballs')], { cwd: directory }));
  const tarball = path.join(out, 'tarballs', packed.filename);
  packages.push({ name: manifest.name, version: releaseVersion, tarball: path.relative(out, tarball).split(path.sep).join('/'), sha256: sha256(tarball), integrity: packed.integrity, files: packed.files.map(file => file.path) });
}
for (const target of targets) {
  const { os, cpu, binary } = platform(target);
  const packageName = `${name}-${os}-${cpu}`;
  if (selectedTargets.includes(target)) dependencies[packageName] = releaseVersion;
  const directory = path.join(out, 'packages', `${os}-${cpu}`);
  createPackage(directory, { ...base, name: packageName, os: [os], cpu: [cpu], files: ['bin/', 'legal/'] }, () => {
    mkdirSync(path.join(directory, 'bin'));
    const destination = path.join(directory, 'bin', binary);
    copyFileSync(artifact(dist, target), destination);
    chmodSync(destination, 0o755);
  }, selectedTargets.includes(target));
}
const directory = path.join(out, 'packages', 'main');
createPackage(directory, { ...base, name, os: supportedOS, bin: { 'sushiro-cli': 'bin/sushiro-cli.cjs' }, engines: { node: '>=20' }, files: ['bin/', 'legal/'], optionalDependencies: dependencies }, () => {
  mkdirSync(path.join(directory, 'bin'));
  const destination = path.join(directory, 'bin', 'sushiro-cli.cjs');
  copyFileSync(path.join(root, 'packaging/npm/launcher.cjs'), destination);
  chmodSync(destination, 0o755);
});
writeJSON(path.join(out, 'release.json'), { name, version: releaseVersion, npmTargets: selectedTargets, publishReady, publicSource, build, packages });
writeFileSync(path.join(out, 'SHA256SUMS'), packages.map(pkg => `${pkg.sha256}  ${pkg.tarball}\n`).join(''));
console.log(`Prepared ${packages.length} packages in ${out}. No packages were published.`);
