import { appendFileSync, mkdirSync } from 'node:fs';
import path from 'node:path';
import { json, options, run, writeJSON } from './common.mjs';
import { releaseIdentity } from './release-model.mjs';

const args = options(['--out']);
if (!args['--out'] || !process.env.GITHUB_EVENT_PATH || process.env.GITHUB_EVENT_NAME !== 'release') throw new Error('requires a GitHub release event and --out');
const event = json(process.env.GITHUB_EVENT_PATH);
const commit = run('git', ['rev-parse', 'HEAD']);
if (process.env.GITHUB_SHA !== commit) throw new Error('checkout must match the original release event commit');
const identity = releaseIdentity(event, commit);
if (run('git', ['rev-parse', `refs/tags/${identity.tag}^{commit}`]) !== commit) throw new Error('checked-out commit is not the exact release tag');
mkdirSync(path.dirname(path.resolve(args['--out'])), { recursive: true });
writeJSON(args['--out'], identity);
if (process.env.GITHUB_OUTPUT) appendFileSync(process.env.GITHUB_OUTPUT, `version=${identity.version}\nnpm_tag=${identity.npmTag}\ncommit=${identity.commit}\n`);
console.log(JSON.stringify(identity));
