#!/usr/bin/env node
// Copyright 2026 Dave Shanley / Quobix / Princess Beef Heavy Industries, LLC
// SPDX-License-Identifier: MIT

import { execFileSync } from 'node:child_process';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { existsSync, mkdirSync, readFileSync, writeFileSync, unlinkSync } from 'node:fs';
import { dirname, resolve, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { runInNewContext } from 'node:vm';

const root = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const cache = join(root, '.cache', 'openlint');
const conformanceRevision = '0714e478a99b271486c93e10a327db4564170f61';
const specRevision = '4e95c96e4a7ecb17320cd11fc2fcd2571e8d0291';
const matcherCorrection = 'range-endpoints-v1';
mkdirSync(cache, { recursive: true });
function command(executable, args, cwd = root) {
  return execFileSync(executable, args, { cwd, encoding: 'utf8', stdio: ['ignore', 'pipe', 'inherit'] }).trim();
}
function checkout(name, revision) {
  const path = join(cache, name);
  if (!existsSync(path)) {
    command('git', ['clone', '--quiet', '--no-checkout', `https://github.com/openlint/${name}.git`, path]);
    command('git', ['checkout', '--quiet', '--detach', revision], path);
  }
  if (command('git', ['rev-parse', 'HEAD'], path) !== revision || command('git', ['status', '--porcelain', '--untracked-files=no'], path)) {
    throw new Error(`${path} must be clean at ${revision}; preserve any local work before replacing the cache`);
  }
  return path;
}
const suite = checkout('conformance', conformanceRevision);
const spec = checkout('spec', specRevision);
const bin = join(cache, 'bin');
mkdirSync(bin, { recursive: true });
const head = command('git', ['rev-parse', 'HEAD']);
const dirty = Boolean(command('git', ['status', '--porcelain']));
const version = `0.0.0-openlint.${head.slice(0, 12)}${dirty ? '.dirty' : ''}`;
command('go', ['build', '-ldflags', `-X main.version=${version} -X main.commit=${head}`, '-o', join(bin, 'vacuum'), '.']);
writeFileSync(join(cache, 'build.json'), JSON.stringify({
  head, dirty, version, specRevision, conformanceRevision, matcherCorrection,
  binarySHA256: createHash('sha256').update(readFileSync(join(bin, 'vacuum'))).digest('hex'),
}, null, 2) + '\n');
let runner = readFileSync(join(suite, 'run.mjs'), 'utf8');
function replace(before, after) {
  if (!runner.includes(before)) throw new Error(`Pinned runner changed: ${before}`);
  runner = runner.replace(before, after);
}
// Keep every upstream fixture unchanged. The pinned matcher compares an entire
// range even when the fixture asserts only start. The draft requires an end too.
// Compare every stated endpoint; do not discard correct ends to satisfy the test.
// Remove this correction when the pinned upstream runner includes the fix.
replace('    } else if (JSON.stringify(v) !== JSON.stringify(act[k])) {', `    } else if (k === "range") {
      if (!act.range || Object.entries(v).some(([endpoint, position]) =>
        JSON.stringify(position) !== JSON.stringify(act.range[endpoint]))) return false;
    } else if (JSON.stringify(v) !== JSON.stringify(act[k])) {`);

// Exercise the corrected runner function itself before running the suite.
const matcherSource = runner.match(/function matches\(exp, act, ctx\) \{[\s\S]*?\n\}/)?.[0];
assert.ok(matcherSource, 'Pinned range matcher function was not found');
const matches = runInNewContext(`${matcherSource}\nmatches`);
const start = { line: 0, character: 0 };
const end = { line: 1, character: 16 };
const actual = { code: 'root-falsy', range: { start, end } };
assert.equal(matches({ range: { start } }, actual, {}), true, 'allow an additional end');
assert.equal(matches({ range: { start, end } }, actual, {}), true, 'match both endpoints');
assert.equal(matches({ range: { start: end } }, actual, {}), false, 'reject a wrong start');
assert.equal(matches({ range: { start } }, { range: { end } }, {}), false, 'reject a missing start');
assert.equal(matches({ range: { start } }, {}, {}), false, 'reject a missing range');
assert.equal(matches({ range: { start, end } }, { range: { start } }, {}), false, 'reject a missing asserted end');
assert.equal(matches({ range: { start, end: start } }, actual, {}), false, 'reject a wrong asserted end');
assert.equal(matches({ code: 'wrong', range: { start } }, actual, {}), false, 'retain other result checks');
console.log(`Local matcher correction: ${matcherCorrection}. Original fixtures unchanged; endpoint regression checks passed.`);

// Adapt the selected tool, invocation flag, worker count and output destination.
replace('const tools = Object.keys(adapters);', 'const tools = ["vacuum"];');
replace('["spectral-report", "-r", ruleset, document, file]', '["spectral-report", "--openlint", "-r", ruleset, document, file]');
replace('Math.max(2, cpus().length)', '4');
runner = runner.replaceAll('results/matrix.', 'results/vacuum-openlint.');
const temporary = join(suite, `.vacuum-runner-${process.pid}.mjs`);
try {
  writeFileSync(temporary, runner);
  execFileSync(process.execPath, [temporary, '--bin', bin, '--spec', join(spec, 'v1/conformance')], {
    cwd: suite, stdio: 'inherit', env: { ...process.env, VACUUM_NO_UPDATE_CHECK: 'true' },
  });
} finally {
  if (existsSync(temporary)) unlinkSync(temporary);
}
const result = JSON.parse(readFileSync(join(suite, 'results/vacuum-openlint.json'), 'utf8'));
if (result.suites.spec.length !== 238 || result.suites.interop.length !== 14) throw new Error('Pinned case count changed');
const failures = Object.entries(result.suites).flatMap(([name, cases]) => cases
  .filter(c => c.results.vacuum?.status !== 'exact')
  .map(c => `${name}/${c.id}: ${c.results.vacuum?.status ?? 'missing'}`));
if (failures.length) throw new Error(`${failures.length}/252 cases are not exact:\n${failures.join('\n')}`);
console.log(`252/252 exact. Spec ${specRevision}; runner ${conformanceRevision}; local matcher correction ${matcherCorrection}.`);
console.log(`Results: ${join(suite, 'results/vacuum-openlint.json')}`);
