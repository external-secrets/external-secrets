/**
 * Tests for pr-template-conformance.js.
 *
 * Reads the real .github/pull_request_template.md rather than a fixture
 * copy, so the tests cannot drift from what the workflow actually checks
 * against.
 *
 * Run with: node .github/scripts/pr-template-conformance-test.js
 */

import assert from 'node:assert/strict';
import test from 'node:test';
import { readFileSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import run, {
  isBot, OVERRIDE_LABEL, CUTOFF_PR_NUMBER, normalise, extractSection,
  extractChecklistItems, missingChecklistItems, checkAiDisclosure,
  checkConformance, closeMessage,
} from './pr-template-conformance.js';

const SCRIPTS_DIR = path.dirname(fileURLToPath(import.meta.url));
const REAL_TEMPLATE = readFileSync(path.join(SCRIPTS_DIR, '..', 'pull_request_template.md'), 'utf8');

function fakeCore() {
  const infos = [];
  const failures = [];
  return {
    infos,
    failures,
    info: (m) => infos.push(m),
    warning: () => {},
    setFailed: (m) => failures.push(m),
  };
}

// Captures what the workflow would have written, without touching GitHub.
function fakeGithub() {
  const comments = [];
  const closed = [];
  return {
    comments,
    closed,
    rest: {
      issues: {
        createComment: async (args) => { comments.push(args); },
      },
      pulls: {
        update: async (args) => { closed.push(args); },
      },
    },
  };
}

function pr(overrides = {}) {
  return {
    number: CUTOFF_PR_NUMBER + 1,
    user: { login: 'contributor' },
    labels: [],
    body: REAL_TEMPLATE,
    ...overrides,
  };
}

const context = (payload) => ({ repo: { owner: 'external-secrets', repo: 'external-secrets' }, payload });

// A filled-in Checklist section, matching every real item, for tests that
// only care about the AI Assistance disclosure.
const FILLED_CHECKLIST = extractSection(REAL_TEMPLATE, 'Checklist')
  .replace(/\[ \]/g, '[x]');

function withSections({ checklist = FILLED_CHECKLIST, aiDisclosure } = {}) {
  const base = REAL_TEMPLATE
    .replace(/## Checklist[\s\S]*$/, `## Checklist\n${checklist}\n`);
  if (aiDisclosure === undefined) return base;
  return base.replace(
    /## AI Assistance disclosure[\s\S]*?(?=\n## )/,
    `## AI Assistance disclosure\n\n${aiDisclosure}\n\n`,
  );
}

// ---------------------------------------------------------------------------
// normalise
// ---------------------------------------------------------------------------

test('normalise strips markup, case and punctuation, collapses whitespace', () => {
  assert.equal(normalise('**AI Assistance disclosure!**'), 'ai assistance disclosure');
  assert.equal(normalise('  Related   Issue  '), 'related issue');
});

// ---------------------------------------------------------------------------
// extractSection
// ---------------------------------------------------------------------------

test('extractSection returns content between a heading and the next one', () => {
  const text = '## A\n\nfirst\n\n## B\n\nsecond\n\n## C\n\nthird';
  assert.equal(extractSection(text, 'B').trim(), 'second');
});

test('extractSection returns everything to end of document for the last heading', () => {
  const text = '## A\n\nfirst\n\n## B\n\nsecond';
  assert.equal(extractSection(text, 'B').trim(), 'second');
});

test('extractSection is empty when the heading does not exist', () => {
  const text = '## A\n\nfirst';
  assert.equal(extractSection(text, 'Missing'), '');
});

test('extractSection ignores a heading-like line inside a fenced code block', () => {
  const text = '## Format\n\n```\n## Checklist\nnot real\n```\n\n## Checklist\n\nreal content';
  assert.equal(extractSection(text, 'Checklist').trim(), 'real content');
});

test('extractSection strips a fence closed with more backticks than it opened with', () => {
  const text = '## Format\n\n```\n# not a real heading\n````\n\n## Checklist\n\nreal content';
  assert.equal(extractSection(text, 'Checklist').trim(), 'real content');
});

// Regression: every regex here anchored `$` without the `m` flag, relying
// on `.`/`.*` stopping at a lone `\n`. JavaScript's `.` does not match
// `\r`, so a line ending in `\r` (a CRLF body, what the GitHub web editor
// produces) matched nothing, and extractSection returned an empty section
// for every heading. A fully conformant CRLF body was reported as though
// every section had been deleted.
test('checkConformance treats a CRLF body the same as the equivalent LF body', () => {
  const lfBody = withSections({ aiDisclosure: 'AI assistance used: No' });
  const crlfBody = lfBody.replace(/\n/g, '\r\n');
  const result = checkConformance(REAL_TEMPLATE, crlfBody);
  assert.equal(result.conformant, true);
  assert.deepEqual(result.problems, []);
});

test('a CRLF body missing a checklist item is still caught, not silently passed', () => {
  const items = extractChecklistItems(FILLED_CHECKLIST);
  const gutted = items.slice(1).map((item) => `- [x] ${item}`).join('\n');
  const lfBody = withSections({ checklist: gutted, aiDisclosure: 'AI assistance used: No' });
  const result = checkConformance(REAL_TEMPLATE, lfBody.replace(/\n/g, '\r\n'));
  assert.equal(result.conformant, false);
  assert.match(result.problems[0], /^Checklist:/);
});

// ---------------------------------------------------------------------------
// extractChecklistItems / missingChecklistItems
// ---------------------------------------------------------------------------

test('extractChecklistItems pulls the label text off each checkbox line', () => {
  const text = '- [ ] First item\n- [x] Second item\nNot a checkbox line';
  assert.deepEqual(extractChecklistItems(text), ['First item', 'Second item']);
});

test('extractChecklistItems also recognises numbered list markers', () => {
  const text = '1. [ ] First item\n2. [x] Second item';
  assert.deepEqual(extractChecklistItems(text), ['First item', 'Second item']);
});

test('an emoji-only template item is silently unenforceable, not a false pass', () => {
  // Regression: the old heading matcher let an empty normalisation match
  // everything, which is a bypass. Here the guard is the other shape: an
  // item that normalises to nothing is dropped from the missing-list
  // rather than ever being reportable, which is a safe degrade, not a
  // bypass, since it can never cause a body to wrongly look conformant.
  const missing = missingChecklistItems(['🚀', 'Do the real thing'], []);
  assert.deepEqual(missing, ['Do the real thing']);
});

test('missingChecklistItems finds nothing when every template item has a body match', () => {
  const missing = missingChecklistItems(['Do the thing'], ['Do the thing']);
  assert.deepEqual(missing, []);
});

test('missingChecklistItems tolerates markdown link syntax around an item', () => {
  const missing = missingChecklistItems(
    ['I have read the [contribution guidelines](https://example.com/process)'],
    ['I have read the [contribution guidelines](https://example.com/process)'],
  );
  assert.deepEqual(missing, []);
});

test('missingChecklistItems reports an item with no match at all', () => {
  const missing = missingChecklistItems(['Do the thing', 'Do another thing'], ['Do the thing']);
  assert.deepEqual(missing, ['Do another thing']);
});

test('the real template checklist has all 7 known items', () => {
  const items = extractChecklistItems(extractSection(REAL_TEMPLATE, 'Checklist'));
  assert.equal(items.length, 7);
});

// ---------------------------------------------------------------------------
// checkAiDisclosure
// ---------------------------------------------------------------------------

test('the unedited template placeholder is not an answer', () => {
  const problems = checkAiDisclosure(extractSection(REAL_TEMPLATE, 'AI Assistance disclosure'));
  assert.equal(problems.length, 1);
  assert.match(problems[0], /must be answered Yes or No/);
});

test('an empty AI Assistance disclosure section is not an answer', () => {
  const problems = checkAiDisclosure('');
  assert.equal(problems.length, 1);
});

test('"No" needs no further detail', () => {
  const problems = checkAiDisclosure('AI assistance used: No\n');
  assert.deepEqual(problems, []);
});

test('"Nope" is not recognised as an answer (word-boundary match only)', () => {
  const problems = checkAiDisclosure('AI assistance used: Nope\n');
  assert.equal(problems.length, 1);
});

// Regression: the placeholder guard used to be a single literal-string
// check (`answer !== 'yes no'`), which only caught the exact unedited
// text. Emphasising or striking one option, or reordering the two,
// normalises to different text that still contains both a yes-token and a
// no-token, and slipped through as an answer.
test('emphasising one option instead of deleting it is still ambiguous', () => {
  const problems = checkAiDisclosure('AI assistance used: **Yes** / No\n');
  assert.equal(problems.length, 1);
  assert.match(problems[0], /must be answered Yes or No/);
});

test('striking one option instead of deleting it is still ambiguous', () => {
  const problems = checkAiDisclosure('AI assistance used: Yes / ~~No~~\n');
  assert.equal(problems.length, 1);
});

test('reordering the placeholder is still ambiguous, not a considered "No"', () => {
  const problems = checkAiDisclosure('AI assistance used: No / Yes\n');
  assert.equal(problems.length, 1);
  assert.match(problems[0], /must be answered Yes or No/);
});

test('"Yes / No (delete one)" is reported as ambiguous, not as 4 missing fields', () => {
  const problems = checkAiDisclosure('AI assistance used: Yes / No (delete one)\n');
  assert.equal(problems.length, 1);
  assert.match(problems[0], /must be answered Yes or No/);
});

test('"None", "N/A" and "Not applicable" are recognised as No', () => {
  for (const value of ['None', 'N/A', 'Not applicable']) {
    const problems = checkAiDisclosure(`AI assistance used: ${value}\n`);
    assert.deepEqual(problems, [], `expected "${value}" to need no further detail`);
  }
});

test('"Yes" on its own is missing all four detail fields', () => {
  const problems = checkAiDisclosure('AI assistance used: Yes\n');
  assert.equal(problems.length, 4);
  assert.match(problems[0], /Tool\(s\) used/);
});

test('"Yes" with every field answered on the same line has no problems', () => {
  const body = [
    'AI assistance used: Yes',
    '',
    'Tool(s) used: Claude Code',
    '',
    'Purpose of assistance: implementation',
    '',
    'Parts of the contribution affected: the whole diff',
    '',
    'Human validation performed: reviewed and tested',
  ].join('\n');
  assert.deepEqual(checkAiDisclosure(body), []);
});

test('"Yes" with fields answered on the following line also has no problems', () => {
  const body = [
    'AI assistance used: Yes',
    '',
    'Tool(s) used:',
    'Claude Code',
    '',
    'Purpose of assistance:',
    'implementation',
    '',
    'Parts of the contribution affected:',
    'the whole diff',
    '',
    'Human validation performed:',
    'reviewed and tested',
  ].join('\n');
  assert.deepEqual(checkAiDisclosure(body), []);
});

test('"Yes" with one field left blank reports only that field', () => {
  const body = [
    'AI assistance used: Yes',
    '',
    'Tool(s) used: Claude Code',
    '',
    'Purpose of assistance:',
    '',
    'Parts of the contribution affected: the whole diff',
    '',
    'Human validation performed: reviewed and tested',
  ].join('\n');
  const problems = checkAiDisclosure(body);
  assert.equal(problems.length, 1);
  assert.match(problems[0], /Purpose of assistance/);
});

test('a blank field followed by the next label, not an answer, is still blank', () => {
  const body = [
    'AI assistance used: Yes',
    '',
    'Tool(s) used:',
    '',
    'Purpose of assistance: implementation',
    '',
    'Parts of the contribution affected: the whole diff',
    '',
    'Human validation performed: reviewed and tested',
  ].join('\n');
  const problems = checkAiDisclosure(body);
  assert.equal(problems.length, 1);
  assert.match(problems[0], /Tool\(s\) used/);
});

// ---------------------------------------------------------------------------
// checkConformance
// ---------------------------------------------------------------------------

test('a fully filled-in body is conformant', () => {
  const body = withSections({
    aiDisclosure: 'AI assistance used: No',
  });
  const result = checkConformance(REAL_TEMPLATE, body);
  assert.equal(result.conformant, true);
  assert.deepEqual(result.problems, []);
});

test('an unedited template body is not conformant (checklist ok, AI disclosure not answered)', () => {
  const result = checkConformance(REAL_TEMPLATE, REAL_TEMPLATE);
  assert.equal(result.conformant, false);
  assert.equal(result.problems.length, 1);
  assert.match(result.problems[0], /AI Assistance disclosure/);
});

test('a deleted checklist item and an unanswered disclosure are both reported', () => {
  const items = extractChecklistItems(FILLED_CHECKLIST);
  const gutted = items.slice(1).map((item) => `- [x] ${item}`).join('\n'); // drop the first item
  const body = withSections({ checklist: gutted, aiDisclosure: 'AI assistance used: Yes / No' });
  const result = checkConformance(REAL_TEMPLATE, body);
  assert.equal(result.conformant, false);
  assert.equal(result.problems.length, 2);
  assert.match(result.problems[0], /^Checklist:/);
  assert.match(result.problems[1], /^AI Assistance disclosure:/);
});

test('problem statement, related issue, proposed changes and format are never checked', () => {
  const body = withSections({ aiDisclosure: 'AI assistance used: No' })
    .replace(/## Problem Statement[\s\S]*?(?=\n## )/, '')
    .replace(/## Related Issue[\s\S]*?(?=\n## )/, '')
    .replace(/## Proposed Changes[\s\S]*?(?=\n## )/, '')
    .replace(/## Format[\s\S]*?(?=\n## )/, '');
  const result = checkConformance(REAL_TEMPLATE, body);
  assert.equal(result.conformant, true);
});

test('checkConformance treats a null pull request body as missing everything, not a crash', () => {
  const result = checkConformance(REAL_TEMPLATE, null);
  assert.equal(result.conformant, false);
  assert.ok(result.problems.length > 0);
});

// ---------------------------------------------------------------------------
// closeMessage
// ---------------------------------------------------------------------------

test('closeMessage names every problem and the override label', () => {
  const body = closeMessage(['Checklist: missing "Do the thing"']);
  assert.match(body, /Do the thing/);
  assert.match(body, new RegExp(OVERRIDE_LABEL));
});

// ---------------------------------------------------------------------------
// run(): orchestration. readTemplate is injected so these never touch disk.
// ---------------------------------------------------------------------------

test('run() takes no action on a conformant pull request', async () => {
  const github = fakeGithub();
  const body = withSections({ aiDisclosure: 'AI assistance used: No' });
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ body }) }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'none');
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() comments and closes a pull request with problems', async () => {
  const github = fakeGithub();
  const number = CUTOFF_PR_NUMBER + 42;
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ number, body: REAL_TEMPLATE }) }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'closed');
  assert.equal(result.problems.length, 1);
  assert.equal(github.comments.length, 1);
  assert.equal(github.comments[0].issue_number, number);
  assert.match(github.comments[0].body, /AI Assistance disclosure/);
  assert.equal(github.closed.length, 1);
  assert.deepEqual(github.closed[0], {
    owner: 'external-secrets', repo: 'external-secrets', pull_number: number, state: 'closed',
  });
});

test('run() skips a pull request at or below the cutoff number, even with problems', async () => {
  const github = fakeGithub();
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ number: CUTOFF_PR_NUMBER, body: '' }) }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'skip-before-cutoff');
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() skips a bot author without writing anything', async () => {
  const github = fakeGithub();
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ user: { login: 'dependabot[bot]' }, body: '' }) }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'skip-bot');
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() skips a pull request carrying the override label', async () => {
  const github = fakeGithub();
  const result = await run({
    core: fakeCore(),
    github,
    context: context({
      pull_request: pr({ body: '', labels: [{ name: OVERRIDE_LABEL }] }),
    }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'skip-override');
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() fails loudly and takes no action when the template has no checklist items', async () => {
  const github = fakeGithub();
  const core = fakeCore();
  const result = await run({
    core,
    github,
    context: context({ pull_request: pr({ body: '' }) }),
    readTemplate: () => 'no checklist in here at all',
  });
  assert.equal(result.action, 'error-empty-template');
  assert.equal(core.failures.length, 1);
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() fails loudly and takes no action when the template cannot be read', async () => {
  const github = fakeGithub();
  const core = fakeCore();
  const result = await run({
    core,
    github,
    context: context({ pull_request: pr() }),
    readTemplate: () => { throw new Error('ENOENT: no such file'); },
  });
  assert.equal(result.action, 'error-read-template');
  assert.match(core.failures[0], /ENOENT/);
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() reads the real template from disk when readTemplate is not overridden', async () => {
  // Exercises the default parameter, which is what actually runs in CI: every
  // other test injects readTemplate and never touches this path.
  const github = fakeGithub();
  const body = withSections({ aiDisclosure: 'AI assistance used: No' });
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ body }) }),
  });
  assert.equal(result.action, 'none');
});

test('isBot matches known accounts and the generic [bot] suffix', () => {
  assert.equal(isBot('dependabot'), true);
  assert.equal(isBot('some-app[bot]'), true);
  assert.equal(isBot('a-human-contributor'), false);
  assert.equal(isBot(null), false);
});

// ---------------------------------------------------------------------------
// Template drift: the AI disclosure field labels are hardcoded (see the
// comment in pr-template-conformance.js on why), so if a maintainer renames
// one in the template this test fails loudly instead of the check silently
// checking for a label that no longer exists.
// ---------------------------------------------------------------------------

test('the real template still contains every hardcoded AI disclosure field label', () => {
  const section = extractSection(REAL_TEMPLATE, 'AI Assistance disclosure');
  for (const label of ['AI assistance used', 'Tool(s) used', 'Purpose of assistance',
    'Parts of the contribution affected', 'Human validation performed']) {
    assert.match(
      section,
      new RegExp(`^[ \\t]*${label.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}[ \\t]*:`, 'im'),
      `expected to find the label "${label}" in the real template`,
    );
  }
});
