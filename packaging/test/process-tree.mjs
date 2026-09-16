// Test-only ownership: normal signals still target the launcher, not its group.
import { spawn, spawnSync } from 'node:child_process';

export function spawnTestProcess(command, args, options = {}) {
  return spawn(command, args, { ...options, detached: process.platform !== 'win32' });
}

export function stopTestProcess(child) {
  if (child.pid) {
    if (process.platform === 'win32') {
      if (child.exitCode === null && child.signalCode === null) {
        spawnSync('taskkill', ['/pid', String(child.pid), '/T', '/F'], { stdio: 'ignore', timeout: 5000, windowsHide: true });
      }
    } else {
      try { process.kill(-child.pid, 'SIGKILL'); } catch (error) {
        if (error.code !== 'ESRCH') throw error;
      }
    }
  }
  // Descendants must not keep the test runner's pipe handles open after failure.
  child.stdin?.destroy();
  child.stdout?.destroy();
  child.stderr?.destroy();
}
