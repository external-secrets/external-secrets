/**
 * Tests for resolve-prs.js.
 *
 * The fork cases matter most. listPullRequestsAssociatedWithCommit returns an
 * empty array for pull requests from forks, with no error to distinguish that
 * from "no such pull request", so this module matches against the open pull
 * request list instead. These tests pin that.
 *
 * Run with: node .github/scripts/resolve-prs-test.js
 */

import assert from 'node:assert/strict';
import test from 'node:test';
import resolvePullRequests, { pullRequestsForSha, pullRequestsForBranch, SWEEP_TARGET } from './resolve-prs.js';

// resolvePullRequests returns { numbers, targets }; most tests care about numbers.
const numbersOf = async (args) => (await resolvePullRequests(args)).numbers;
const targetsOf = async (args) => (await resolvePullRequests(args)).targets;

// Two forks and one same-repo branch, mirroring the real mix on this repository.
const OPEN = [
  { number: 6856, head: { sha: 'aaa111', ref: 'feat/routing', repo: { full_name: 'alekc/fork-external-secrets' } } },
  { number: 6855, head: { sha: 'bbb222', ref: 'make-license.check', repo: { full_name: 'dronenb/external-secrets' } } },
  { number: 6842, head: { sha: 'ccc333', ref: 'chore/bump', repo: { full_name: 'external-secrets/external-secrets' } } },
];

function harness({ open = OPEN } = {}) {
  const warnings = [];
  const infos = [];
  const core = { warning: (m) => warnings.push(m), info: (m) => infos.push(m) };
  const github = {
    paginate: async () => open,
    rest: { pulls: { list: () => {} } },
  };
  return { core, github, warnings, infos };
}

const ctx = (eventName, payload = {}) => ({
  eventName,
  repo: { owner: 'external-secrets', repo: 'external-secrets' },
  payload,
});

for (const event of ['pull_request_target', 'pull_request_review', 'pull_request_review_comment']) {
  test(`${event} resolves to the pull request in its payload`, async () => {
    const { core, github } = harness();
    const numbers = await numbersOf({
      core, github, context: ctx(event, { pull_request: { number: 6856 } }),
    });
    assert.deepEqual(numbers, [6856]);
  });
}

test('a pull request event without a number warns instead of throwing', async () => {
  const { core, github, warnings } = harness();
  assert.deepEqual(await numbersOf({ core, github, context: ctx('pull_request_target', {}) }), []);
  assert.equal(warnings.length, 1);
});

// --- workflow_run, the relay from Review Signal -----------------------------

test('workflow_run resolves a FORK pull request by head repo and branch', async () => {
  const { core, github } = harness();
  const numbers = await numbersOf({
    core,
    github,
    context: ctx('workflow_run', {
      workflow_run: {
        head_branch: 'feat/routing',
        head_repository: { full_name: 'alekc/fork-external-secrets' },
        pull_requests: [],
      },
    }),
  });
  assert.deepEqual(numbers, [6856], 'fork pull requests must still be found');
});

test('workflow_run does not confuse the same branch name on a different fork', async () => {
  const open = [
    { number: 1, head: { sha: 'x', ref: 'shared', repo: { full_name: 'alice/fork' } } },
    { number: 2, head: { sha: 'y', ref: 'shared', repo: { full_name: 'bob/fork' } } },
  ];
  const { core, github } = harness({ open });
  const numbers = await numbersOf({
    core,
    github,
    context: ctx('workflow_run', {
      workflow_run: { head_branch: 'shared', head_repository: { full_name: 'bob/fork' } },
    }),
  });
  assert.deepEqual(numbers, [2]);
});

test('workflow_run without a branch or head repository warns', async () => {
  const { core, github, warnings } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('workflow_run', { workflow_run: {} }) }),
    [],
  );
  assert.equal(warnings.length, 1);
});

test('workflow_run for a branch with no open pull request resolves to nothing', async () => {
  const { core, github, infos } = harness();
  const numbers = await numbersOf({
    core,
    github,
    context: ctx('workflow_run', {
      workflow_run: { head_branch: 'gone', head_repository: { full_name: 'alekc/fork-external-secrets' } },
    }),
  });
  assert.deepEqual(numbers, []);
  assert.match(infos.join(' '), /matches no open pull request/);
});

// --- check_suite and friends, resolved by head SHA -------------------------

test('check_suite resolves a fork pull request by head SHA', async () => {
  const { core, github } = harness();
  const numbers = await numbersOf({
    core, github, context: ctx('check_suite', { check_suite: { head_sha: 'bbb222', pull_requests: [] } }),
  });
  assert.deepEqual(numbers, [6855]);
});

test('check_run and status also resolve by SHA', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({
      core, github, context: ctx('check_run', { check_run: { check_suite: { head_sha: 'ccc333' } } }),
    }),
    [6842],
  );
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('status', { sha: 'aaa111' }) }),
    [6856],
  );
});

test('a SHA event with no SHA warns and resolves to nothing', async () => {
  const { core, github, warnings } = harness();
  assert.deepEqual(await numbersOf({ core, github, context: ctx('check_suite', {}) }), []);
  assert.equal(warnings.length, 1);
});

test('an unknown SHA resolves to nothing rather than everything', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('check_suite', { check_suite: { head_sha: 'nope' } }) }),
    [],
  );
});

test('the SHA and branch helpers are exported and usable directly', async () => {
  const { github } = harness();
  assert.deepEqual(await pullRequestsForSha(github, 'o', 'r', 'aaa111'), [6856]);
  assert.deepEqual(
    await pullRequestsForBranch(github, 'o', 'r', 'dronenb/external-secrets', 'make-license.check'),
    [6855],
  );
});

// --- sweeps and manual dispatch --------------------------------------------

test('a scheduled run sweeps every open pull request', async () => {
  const { core, github } = harness();
  assert.deepEqual(await numbersOf({ core, github, context: ctx('schedule') }), [6856, 6855, 6842]);
});

test('workflow_dispatch with a number evaluates just that pull request', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('workflow_dispatch', { inputs: { pr: '6613' } }) }),
    [6613],
  );
});

test('workflow_dispatch tolerates surrounding whitespace', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('workflow_dispatch', { inputs: { pr: '  42  ' } }) }),
    [42],
  );
});

test('workflow_dispatch with a blank input sweeps everything', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: ctx('workflow_dispatch', { inputs: { pr: '   ' } }) }),
    [6856, 6855, 6842],
  );
});

test('workflow_dispatch rejects a non-numeric input rather than querying NaN', async () => {
  const { core, github } = harness();
  await assert.rejects(
    () => resolvePullRequests({ core, github, context: ctx('workflow_dispatch', { inputs: { pr: 'main' } }) }),
    /Invalid pr input/,
  );
});

test('an absent payload does not blow up', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({ core, github, context: { eventName: 'schedule', repo: { owner: 'o', repo: 'r' } } }),
    [6856, 6855, 6842],
  );
});

// --- targets drive the job matrix and the concurrency group ----------------

test('an event resolves to per-pull-request targets, one job each', async () => {
  const { core, github } = harness();
  const targets = await targetsOf({
    core, github, context: ctx('pull_request_target', { pull_request: { number: 6857 } }),
  });
  assert.deepEqual(targets, ['6857'], 'the group becomes review-state-6857');
});

test('a check_suite matching two pull requests yields two targets', async () => {
  const open = [
    { number: 10, head: { sha: 'dup', ref: 'a', repo: { full_name: 'x/y' } } },
    { number: 11, head: { sha: 'dup', ref: 'b', repo: { full_name: 'x/z' } } },
  ];
  const { core, github } = harness({ open });
  const targets = await targetsOf({
    core, github, context: ctx('check_suite', { check_suite: { head_sha: 'dup' } }),
  });
  assert.deepEqual(targets, ['10', '11'], 'each gets its own group, not a shared one');
});

test('a sweep collapses to one sentinel target rather than one job per PR', async () => {
  const { core, github } = harness();
  const out = await resolvePullRequests({ core, github, context: ctx('schedule') });
  assert.deepEqual(out.targets, [SWEEP_TARGET]);
  assert.equal(out.numbers.length, 3, 'but it still carries every pull request to evaluate');
});

test('the sweep sentinel cannot be mistaken for a pull request number', () => {
  assert.ok(!/^\d+$/.test(SWEEP_TARGET), 'it becomes part of a concurrency group name');
});

test('nothing to do yields no targets, so the matrix job is skipped', async () => {
  const { core, github } = harness({ open: [] });
  const out = await resolvePullRequests({ core, github, context: ctx('schedule') });
  assert.deepEqual(out.targets, []);
  assert.deepEqual(out.numbers, []);
});

test('check_run prefers its own head_sha over the nested check_suite', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({
      core,
      github,
      context: ctx('check_run', {
        check_run: { head_sha: 'aaa111', check_suite: { head_sha: 'bbb222' } },
      }),
    }),
    [6856],
    'the direct head_sha wins',
  );
});

test('check_run still falls back to the nested check_suite head_sha', async () => {
  const { core, github } = harness();
  assert.deepEqual(
    await numbersOf({
      core, github, context: ctx('check_run', { check_run: { check_suite: { head_sha: 'ccc333' } } }),
    }),
    [6842],
  );
});
