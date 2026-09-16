#!/usr/bin/env node
'use strict';

const { spawn } = require('node:child_process');
const { constants } = require('node:os');
const path = require('node:path');
const manifest = require('../package.json');

function fail(message) {
  process.stderr.write(`sushiro-cli: ${message}\n`);
  process.exitCode = 1;
}

function main() {
  const target = `${process.platform}-${process.arch}`;
  const packageName = `${manifest.name}-${target}`;
  if (!Object.hasOwn(manifest.optionalDependencies, packageName)) {
    fail(`unsupported platform ${target}`);
    return;
  }
  let binary;
  try {
    const platformManifest = require.resolve(`${packageName}/package.json`);
    const platform = require(platformManifest);
    if (platform.version !== manifest.version) {
      throw Object.assign(new Error('version mismatch'), { code: 'VERSION_MISMATCH' });
    }
    binary = path.join(path.dirname(platformManifest), 'bin', process.platform === 'win32' ? 'sushiro-cli.exe' : 'sushiro-cli');
  } catch (error) {
    const reason = error.code === 'VERSION_MISMATCH' ? 'version mismatch' : 'missing or invalid platform package';
    fail(`cannot load ${packageName}@${manifest.version} (${reason}); reinstall with optional dependencies enabled.`);
    return;
  }

  const child = spawn(binary, process.argv.slice(2), { stdio: 'inherit', windowsHide: false });
  const handlers = new Map();
  for (const signal of ['SIGINT', 'SIGTERM', 'SIGHUP']) {
    const handler = () => { if (child.exitCode === null && child.signalCode === null) child.kill(signal); };
    handlers.set(signal, handler);
    process.on(signal, handler);
  }
  const cleanup = () => {
    for (const [signal, handler] of handlers) process.removeListener(signal, handler);
  };
  child.once('error', error => {
    cleanup();
    const code = ['EACCES', 'ENOENT', 'ENOEXEC'].includes(error.code) ? error.code : 'START_FAILED';
    fail(`cannot start ${packageName} (${code})`);
  });
  child.once('exit', (code, signal) => {
    cleanup();
    if (signal) {
      // Preserve POSIX wait status, not just its shell convention (128 + signal).
      process.exitCode = 128 + (constants.signals[signal] || 1);
      try { process.kill(process.pid, signal); } catch { /* Windows has limited signal support. */ }
    } else {
      process.exitCode = code ?? 1;
    }
  });
}

main();
