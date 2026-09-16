import { test } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, readFileSync, rmSync, statSync, writeFileSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { archive } from '../../scripts/release/archive.mjs';
import { run } from '../../scripts/release/common.mjs';

test('binary archive is deterministic and readable by native tar with licenses and executable permissions', () => {
  const entries = [{name:'sushiro-cli',data:Buffer.from('TEST BINARY ONLY'),mode:0o755},{name:'legal/LICENSE',data:Buffer.from('TEST LICENSE ONLY')}];
  const data = archive(entries);
  assert.deepEqual(data,archive([...entries].reverse()));
  assert.throws(()=>archive([{name:'../escape',data:Buffer.alloc(0)}]),/invalid archive/);
  const temp=mkdtempSync(path.join(os.tmpdir(),'sushiro-release-tar-'));
  try {
    const file=path.join(temp,'test.tar.gz');writeFileSync(file,data);
    assert.deepEqual(run('tar',['-tzf',file]).split(/\r?\n/).sort(),['legal/LICENSE','sushiro-cli']);
    run('tar',['-xzf',file,'-C',temp]);
    assert.equal(readFileSync(path.join(temp,'sushiro-cli'),'utf8'),'TEST BINARY ONLY');
    assert.equal(readFileSync(path.join(temp,'legal/LICENSE'),'utf8'),'TEST LICENSE ONLY');
    if(process.platform!=='win32')assert.equal(statSync(path.join(temp,'sushiro-cli')).mode&0o777,0o755);
  } finally {rmSync(temp,{recursive:true,force:true});}
});
