import { spawnTestProcess, stopTestProcess } from './process-tree.mjs';
// Builds and installs the actual CLI. Never sends a request to the restaurant.
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync, readdirSync, readFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { createInterface } from 'node:readline';
import { test } from 'node:test';
import { json, npm, root, run, writeJSON } from '../../scripts/release/common.mjs';

test('real CLI: six builds, private offline package install, empty configuration, help/version, MCP discovery and EOF', {timeout:300_000},async()=>{
 const temp=mkdtempSync(path.join(os.tmpdir(),'sushiro-real-cli-'));
 let success=false;
 try {
  const dist=path.join(temp,'dist'),out=path.join(temp,'release'),config=path.join(temp,'config.json');
  const version='0.0.0-integration.1';
  writeJSON(config,{scope:'@sushiro-private-test',name:'client',license:'UNLICENSED'});
  run(process.execPath,[path.join(root,'scripts/release/build.mjs'),'--version',version,'--dist',dist]);
  run(process.execPath,[path.join(root,'scripts/release/prepare.mjs'),'--version',version,'--config',config,'--dist',dist,'--out',out], {env:{...process.env,npm_config_cache:path.join(temp,'pack-cache')}});
  const release=json(path.join(out,'release.json'));assert.equal(release.packages.length,7);
  assert.equal(release.publishReady,false);
  for(const pkg of release.packages)for(const file of ['LICENSE','CREDITS.md','PRIVACY.md','DISCLAIMER.md','SECURITY.md','THIRD_PARTY_NOTICES.md','THIRD_PARTY_LICENSES.txt'])assert.ok(pkg.files.includes(`legal/${file}`),`missing ${file} in ${pkg.name}`);
  const install=path.join(temp,'install');mkdirSync(install);
  const dependencies=Object.fromEntries(release.packages.map(p=>[p.name,`file:${path.join(out,p.tarball)}`]));
  const main=release.name,mainTarball=dependencies[main];delete dependencies[main];
  writeJSON(path.join(install,'package.json'),{name:'real-cli-test',private:true,dependencies:{[main]:mainTarball},optionalDependencies:dependencies});
  const user=path.join(temp,'user.npmrc'),global=path.join(temp,'global.npmrc');writeFileSync(user,'');writeFileSync(global,'');
  npm(['install','--offline','--ignore-scripts','--no-audit','--no-fund','--cache',path.join(temp,'cache'),'--userconfig',user,'--globalconfig',global],{cwd:install});
  // Exercise npm's installed command shim, not a direct binary/launcher shortcut.
  const shim=path.join(install,'node_modules','.bin',process.platform==='win32'?'sushiro-cli.cmd':'sushiro-cli');
  const launcher=process.platform==='win32'?`"${shim}"`:shim;
  const credentials=path.join(temp,'credentials');mkdirSync(credentials,{mode:0o700});
  const publicDirectory=path.join(temp,'public');mkdirSync(publicDirectory,{mode:0o700});
  const env=Object.fromEntries(Object.entries(process.env).filter(([key])=>!key.startsWith('SUSHIRO_')));
  env.SUSHIRO_CONFIG_DIR=credentials;
  env.SUSHIRO_PUBLIC_CONFIG_DIR=publicDirectory;
  const execute=args=>spawnSync(launcher,args,{encoding:'utf8',env,timeout:10_000,shell:process.platform==='win32'});
  for(const args of [['help'],['--help']]){
   const help=execute(args);assert.equal(help.status,0,help.stderr);assert.equal(help.stderr,'');assert.ok(help.stdout.includes('stores'));
  }
  const packageReadme=readFileSync(path.join(install,'node_modules',main,'README.md'),'utf8');
  assert.match(packageReadme,/No personal login or credential import is required to install/);
  const result=execute(['version','--json']);
  assert.equal(result.status,0,result.stderr);assert.equal(result.stderr,'');assert.equal(JSON.parse(result.stdout).data.version,version);
  if(process.platform!=='win32'){
   const status=execute(['public','status','--json']);
   assert.equal(status.status,0,status.stderr);
   const data=JSON.parse(status.stdout).data;
   assert.equal(data.configuration_source,'builtin_default');assert.equal(data.persisted,false);
   assert.deepEqual(readdirSync(publicDirectory),[],'built-in defaults must remain in memory');
   const invalidFile=path.join(publicDirectory,'default.json');
   writeFileSync(invalidFile,'{invalid JSON',{mode:0o600});
   const invalid=execute(['public','status','--json']);
   assert.equal(invalid.status,1);assert.equal(JSON.parse(invalid.stdout).error.code,'public_config_invalid');
   rmSync(invalidFile);
  }
  await protocol(launcher,env,false);
  if(process.platform!=='win32')await protocol(launcher,env,true);
  assert.deepEqual(readdirSync(credentials),[],'startup must not create personal credential files');
  assert.deepEqual(readdirSync(publicDirectory),[],'startup must not persist public defaults');
  console.log(`Actual CLI/npm/MCP runtime passed on ${process.platform}-${process.arch}; all six targets compiled. No live Sushiro requests.`);
  success=true;
 }finally{if(success&&process.env.SUSHIRO_KEEP_TEST_ARTIFACTS!=='1')rmSync(temp,{recursive:true,force:true});else console.error(`Artifacts retained: ${temp}`);}
});
function protocol(launcher,env,signal){
 return new Promise((resolve,reject)=>{
  const p=spawnTestProcess(launcher,['mcp'],{env,stdio:['pipe','pipe','pipe'],shell:process.platform==='win32'});let stderr='',stage=0;
  let settled=false;
  const finish=e=>{if(settled)return;settled=true;clearTimeout(timer);lines.close();stopTestProcess(p);if(e)reject(e);else resolve();};
  const timer=setTimeout(()=>finish(new Error('MCP timeout')),10_000);
  const fail=e=>finish(e);
  p.on('error',fail);p.stdin.on('error',fail);p.stderr.on('data',d=>stderr+=d);
  const send=x=>p.stdin.write(JSON.stringify(x)+'\n');
  const lines=createInterface({input:p.stdout});
  lines.on('line',line=>{if(settled)return;try{
   const m=JSON.parse(line);
   if(m.id===1){assert.equal(m.result.serverInfo.name,'sushiro-cli');stage=1;send({jsonrpc:'2.0',method:'notifications/initialized'});send({jsonrpc:'2.0',id:2,method:'tools/list',params:{}});}
   else if(m.id===2){assert.deepEqual(m.result.tools.map(t=>t.name).sort(),['cancel','reservations','reserve','slots','stores','ticket_status']);stage=2;if(signal)p.kill('SIGTERM');else p.stdin.end();}
  }catch(e){fail(e);}});
  p.on('close',(code)=>{try{assert.equal(stage,2);assert.equal(stderr,'');assert.equal(code,0);finish();}catch(e){finish(e);}});
  send({jsonrpc:'2.0',id:1,method:'initialize',params:{protocolVersion:'2025-06-18',capabilities:{},clientInfo:{name:'integration-test',version:'1'}}});
 });
}
