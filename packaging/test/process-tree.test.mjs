import assert from 'node:assert/strict';
import { once } from 'node:events';
import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { setTimeout as delay } from 'node:timers/promises';
import { test } from 'node:test';
import { spawnTestProcess, stopTestProcess } from './process-tree.mjs';

for (const orphan of [false, true]) test(`failure cleanup stops inherited-pipe descendants (launcher already exited: ${orphan})`, {
  timeout: 10_000, skip: orphan && process.platform === 'win32',
}, async () => {
  const dir = mkdtempSync(path.join(os.tmpdir(), 'sushiro-process-cleanup-'));
  const heartbeat = path.join(dir, 'heartbeat');
  const grandchild = `const fs=require('node:fs');setInterval(()=>fs.appendFileSync(process.argv[1],'x'),20);console.log('READY');`;
  const parent = `require('node:child_process').spawn(process.execPath,['-e',${JSON.stringify(grandchild)},process.argv[1]],{stdio:'inherit'});`;
  const child = spawnTestProcess(process.execPath, ['-e', parent, heartbeat], { stdio: ['pipe', 'pipe', 'pipe'] });
  const close = once(child, 'close');
  const emergency = setTimeout(() => stopTestProcess(child), 5000);
  try {
    await once(child.stdout, 'data');
    await delay(100);
    assert.ok(readFileSync(heartbeat).length > 0);
    if (orphan) {
      const exited = once(child, 'exit');
      child.kill('SIGKILL');
      await exited;
      // The Go-like descendant outlives its launcher and still holds the pipes.
    }
    stopTestProcess(child);
    await close;
    await delay(100);
    const stopped = readFileSync(heartbeat).length;
    await delay(150);
    assert.equal(readFileSync(heartbeat).length, stopped, 'descendant must stop running');
  } finally {
    clearTimeout(emergency);
    stopTestProcess(child);
    rmSync(dir, { recursive: true, force: true });
  }
});
