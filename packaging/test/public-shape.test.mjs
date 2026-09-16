import { test } from 'node:test';
import assert from 'node:assert/strict';
import { publicShape } from './public-shape.mjs';

test('public success requires meaningful data, correct store and complete slots', () => {
  const store = { id: 3006, name: 'synthetic store', address: 'synthetic address', wait: 0 };
  const slot = { storeId: '3006', date: '20260916', start: '180000', end: '183000', availability: 'available' };
  for (const label of ['stores', 'store-3006', 'slots-3006-2-T']) {
    assert.equal(publicShape(label, []), false);
    assert.equal(publicShape(label, null), false);
    assert.equal(publicShape(label, [{}]), false);
  }
  assert.equal(publicShape('stores', [store]), true);
  assert.equal(publicShape('stores', [store, store, store]), false);
  assert.equal(publicShape('stores', [{ ...store, id: '3006' }]), false);
  assert.equal(publicShape('stores', [{ ...store, name: '' }]), false);
  assert.equal(publicShape('store-3006', [store]), true);
  assert.equal(publicShape('store-3006', [{ ...store, id: 1 }]), false);
  assert.equal(publicShape('slots-3006-2-T', [slot]), true);
  for (const key of Object.keys(slot)) {
    assert.equal(publicShape('slots-3006-2-T', [{ ...slot, [key]: '' }]), false);
    assert.equal(publicShape('slots-3006-2-T', [{ ...slot, [key]: 1 }]), false);
  }
  assert.equal(publicShape('slots-3006-2-T', [{ ...slot, storeId: '1' }]), false);
  assert.equal(publicShape('slots-3006-2-T', [{ ...slot, date: '20260230' }]), false);
  assert.equal(publicShape('slots-3006-2-T', [{ ...slot, start: '250000' }]), false);
  assert.equal(publicShape('slots-3006-2-T', [{ ...slot, end: '186000' }]), false);
});
