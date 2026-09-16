import assert from 'node:assert/strict';
import { test } from 'node:test';
import { assetPlan, publicationPlan, releaseIdentity, validatePackageSet } from '../../scripts/release/release-model.mjs';
import { platform, targets } from '../../scripts/release/common.mjs';
const commit = 'a'.repeat(40);
const event = { action: 'published', repository: { full_name: 'lmxx1234567/sushiro-cli', private: false }, release: { id: 1, tag_name: 'v1.2.3', draft: false, prerelease: false } };
test('published release binds tag, semver, source, prerelease and all seven packages', () => {
  const identity = releaseIdentity(event, commit);
  assert.equal(identity.version, '1.2.3'); assert.equal(identity.npmTag, 'latest');
  assert.equal(releaseIdentity({ ...event, release: { ...event.release, tag_name: 'v1.2.3-rc.1', prerelease: true } }, commit).npmTag, 'next');
  assert.throws(() => releaseIdentity({ ...event, release: { ...event.release, tag_name: 'v1.2.3\n' } }, commit), /SemVer/);
  for (const changed of [{ ...event, action: 'created' }, { ...event, release: { ...event.release, draft: true } }, { ...event, release: { ...event.release, tag_name: 'v1.2.3-rc.1' } }, { ...event, release: { ...event.release, tag_name: 'v1.2.3\nmalicious' } }]) assert.throws(() => releaseIdentity(changed, commit));
  const packages = targets.map(target => { const {os,cpu}=platform(target); return { name:`sushiro-cli-${os}-${cpu}`,version:'1.2.3',integrity:'sha512-test' }; }).concat({name:'sushiro-cli',version:'1.2.3',integrity:'sha512-test'});
  const release = { name: 'sushiro-cli', version: '1.2.3', publishReady: true, build: { version: '1.2.3', commit, dirty: false }, publicSource: { commit }, packages };
  assert.equal(validatePackageSet(release, identity).at(-1).name, 'sushiro-cli');
  assert.throws(() => validatePackageSet({...release,packages:[...packages].reverse()},identity), /six exact/);
  assert.throws(() => validatePackageSet({...release,build:{...release.build,commit:'b'.repeat(40)}},identity), /must match/);
});
test('rerun preflight verifies existing integrity; bootstrap and conflicts never become writes', () => {
  const packages = [{name:'platform',version:'1.2.3',integrity:'sha512-one'},{name:'main',version:'1.2.3',integrity:'sha512-two'}];
  const metadata = {platform:{versions:{'1.2.3':{dist:{integrity:'sha512-one'}}}},main:{versions:{}}};
  assert.deepEqual(publicationPlan(packages,metadata,'latest').map(p=>p.action),['verified-existing','publish']);
  assert.throws(()=>publicationPlan(packages,{...metadata,main:null},'latest'),/bootstrap required/);
  assert.throws(()=>publicationPlan(packages,{...metadata,platform:{versions:{'1.2.3':{dist:{integrity:'sha512-wrong'}}}}},'latest'),/integrity conflict/);
  assert.throws(()=>publicationPlan(packages,{...metadata,main:{versions:{},'dist-tags':{latest:'2.0.0'}}},'latest'),/backward/);
  assert.equal(publicationPlan(packages,{...metadata,main:{versions:{},'dist-tags':{latest:'2.0.0'}}},'next')[1].action,'publish');
});
test('existing GitHub assets are reused only on exact hashes and never overwritten', async () => {
  const files=[{name:'one.tgz',sha256:'aaa'},{name:'two.tgz',sha256:'bbb'}];
  assert.deepEqual(await assetPlan(files,[{name:'one.tgz'}],async()=> 'aaa'),[files[1]]);
  await assert.rejects(assetPlan(files,[{name:'one.tgz'}],async()=> 'wrong'),/never overwrite/);
});
