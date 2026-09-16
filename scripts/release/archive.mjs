// Minimal deterministic POSIX ustar for the bounded release file allowlist.
import { gzipSync } from 'node:zlib';

export function archive(entries) {
  const chunks = [];
  for (const { name, data, mode = 0o644 } of [...entries].sort((a, b) => a.name < b.name ? -1 : a.name > b.name ? 1 : 0)) {
    if (!/^[a-zA-Z0-9._/-]+$/.test(name) || name.startsWith('/') || name.split('/').includes('..') || Buffer.byteLength(name) > 100) throw new Error('invalid archive entry name');
    if (!Buffer.isBuffer(data) || data.length > 0o77777777777) throw new Error('invalid archive content');
    const header = Buffer.alloc(512);
    const octal = (value, offset, width) => header.write(value.toString(8).padStart(width - 1, '0') + '\0', offset, width, 'ascii');
    header.write(name, 0, 100, 'ascii');
    octal(mode, 100, 8); octal(0, 108, 8); octal(0, 116, 8);
    octal(data.length, 124, 12); octal(0, 136, 12);
    header.fill(32, 148, 156); header.write('0', 156); header.write('ustar\0', 257); header.write('00', 263);
    const sum = header.reduce((a, b) => a + b, 0);
    header.write(sum.toString(8).padStart(6, '0') + '\0 ', 148, 8, 'ascii');
    chunks.push(header, data, Buffer.alloc((512 - data.length % 512) % 512));
  }
  chunks.push(Buffer.alloc(1024));
  return gzipSync(Buffer.concat(chunks), { level: 9 });
}
