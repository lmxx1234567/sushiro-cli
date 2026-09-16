// Publish a scoped GitHub Packages entry from verified release assets.
// Platform dependencies remain on npmjs.org; binaries are never rebuilt here.
import { readFileSync, writeFileSync, mkdtempSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { createHash } from 'node:crypto';
import { execFileSync } from 'node:child_process';
import { validatePackageSet } from './release-model.mjs';

const dir = path.resolve(process.argv[2] || 'packaging/out/assets');
const execute = process.argv.includes('--execute');
const registry = 'https://npm.pkg.github.com';
const name = '@lmxx1234567/sushiro-cli';
const repository = 'https://github.com/lmxx1234567/sushiro-cli';
const release = JSON.parse(readFileSync(path.join(dir, 'release.json')));
const event = JSON.parse(readFileSync(path.join(dir, 'event.json')));
const packages = validatePackageSet(release, event);
if (release.name !== 'sushiro-cli' || release.publicSource.repository !== repository) throw Error('Unexpected source');
const main = packages.at(-1);
if (!/^tarballs\/[a-zA-Z0-9._-]+\.tgz$/.test(main.tarball)) throw Error('Unsafe tarball path');
const nested = path.join(dir, main.tarball);
const source = existsSync(nested) ? nested : path.join(dir, path.basename(main.tarball));
const bytes = readFileSync(source);
if (createHash('sha256').update(bytes).digest('hex') !== main.sha256 || 'sha512-'+createHash('sha512').update(bytes).digest('base64') !== main.integrity) throw Error('Main artifact integrity mismatch');
const work = mkdtempSync(path.join(tmpdir(), 'sushiro-github-package-'));
const entries = execFileSync('tar', ['-tzf', source], {encoding:'utf8'}).trim().split(/\r?\n/);
if (entries.some(p => !p.startsWith('package/') || p.split('/').includes('..'))) throw Error('Unsafe archive entries');
execFileSync('tar', ['-xzf', source, '-C', work]);
const packageDir = path.join(work, 'package');
const manifestPath = path.join(packageDir, 'package.json');
const manifest = JSON.parse(readFileSync(manifestPath));
if (manifest.name !== release.name || manifest.version !== event.version || manifest.repository?.url !== 'git+'+repository+'.git') throw Error('Main manifest identity mismatch');
const expected = Object.fromEntries(packages.slice(0,-1).map(p => [p.name,p.version]));
if (JSON.stringify(manifest.optionalDependencies) !== JSON.stringify(expected)) throw Error('Platform dependencies differ');
// Confirm dependencies are downloadable before exposing a new installation entry.
for (const pkg of packages.slice(0,-1)) {
  const r = await fetch('https://registry.npmjs.org/'+encodeURIComponent(pkg.name)+'/'+pkg.version);
  if (!r.ok || (await r.json()).dist?.integrity !== pkg.integrity) throw Error('npm platform not ready: '+pkg.name);
}
manifest.name = name;
manifest.publishConfig = {registry};
writeFileSync(manifestPath, JSON.stringify(manifest,null,2)+'\n');
writeFileSync(path.join(packageDir,'README.md'), `# ${name}\n\nGitHub Packages entry for [sushiro-cli](${repository}), version ${event.version}.\n\nFor the simplest installation, use the public npm registry:\n\n\`\`\`sh\nnpm install -g sushiro-cli\n\`\`\`\n\nGitHub Packages installation requires GitHub authentication and \`@lmxx1234567:registry=https://npm.pkg.github.com\` in your npm configuration. Keep the default registry set to https://registry.npmjs.org for the exact-version platform dependencies.\n\n\`\`\`sh\nnpm install -g ${name}@${event.version}\nsushiro-cli help\n\`\`\`\n\nSupports macOS/Linux x64 and ARM64. Windows binaries are available from GitHub Releases. This entry reuses the original launcher and legal files; platform binaries are installed from npmjs.org. See the repository for privacy, limitations, credits and MCP setup.\n`);
const packed = JSON.parse(execFileSync('npm',['pack',packageDir,'--pack-destination',work,'--ignore-scripts','--json'],{encoding:'utf8'}))[0];
const tarball = path.join(work,packed.filename);
console.log(JSON.stringify({name,version:event.version,integrity:packed.integrity,tarball,execute}));
if (execute) {
  function metadata(spec) {
    try { const raw=execFileSync('npm',['view',spec,'--registry',registry,'--json'],{encoding:'utf8',stdio:['ignore','pipe','pipe']}); return raw.trim()?JSON.parse(raw):null; }
    catch(e) { const text=e.stdout?.toString()||''; let error; try {error=JSON.parse(text).error;} catch {} if(error?.code==='E404')return null; throw Error('GitHub registry lookup failed (authentication or registry error)'); }
  }
  const existing = metadata(name+'@'+event.version);
  if (existing) {
    if (existing.dist?.integrity !== packed.integrity) throw Error('Existing GitHub version differs; never overwrite');
    console.log('Verified existing GitHub version');
  } else {
    const current = metadata(name);
    if (event.npmTag==='latest' && current?.version) {
      const a=current.version.split('.').map(Number), b=event.version.split('.').map(Number);
      const i=a.findIndex((v,i)=>v!==b[i]);
      if(i>=0 && a[i]>b[i])throw Error('Refusing to move latest backward');
    }
    execFileSync('npm',['publish',tarball,'--registry',registry,'--tag',event.npmTag,'--ignore-scripts'],{stdio:'inherit'});
  }
  let remote;
  for(let i=0;i<12;i++) {
    remote=metadata(name+'@'+event.version);
    if(remote?.dist?.integrity===packed.integrity)break;
    await new Promise(resolve=>setTimeout(resolve,5000));
  }
  if(remote?.dist?.integrity!==packed.integrity)throw Error('Published integrity not yet verified');
  const prefix=path.join(work,'smoke');
  execFileSync('npm',['install','--prefix',prefix,name+'@'+event.version,'--registry','https://registry.npmjs.org','--ignore-scripts','--no-audit','--no-fund'],{stdio:'inherit'});
  const result=execFileSync(path.join(prefix,'node_modules','.bin','sushiro-cli'),['version','--json'],{encoding:'utf8'});
  if(JSON.parse(result).data.version!==event.version)throw Error('Installed CLI version differs');
  console.log('GitHub Packages registry installation and CLI version verified');
}
