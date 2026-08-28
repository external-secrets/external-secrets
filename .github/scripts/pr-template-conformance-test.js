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
  isBot, OVERRIDE_LABEL, extractHeadings, normalise, missingHeadings,
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
    number: 1,
    user: { login: 'contributor' },
    labels: [],
    body: REAL_TEMPLATE,
    ...overrides,
  };
}

const context = (payload) => ({ repo: { owner: 'external-secrets', repo: 'external-secrets' }, payload });

// ---------------------------------------------------------------------------
// extractHeadings / normalise / missingHeadings: the pure matching logic.
// ---------------------------------------------------------------------------

test('extractHeadings pulls heading text at any level, ignoring body content', () => {
  const text = '## Problem Statement\n\nSome body text.\n\n### Related Issue\n\nFixes #1\n';
  assert.deepEqual(extractHeadings(text), ['Problem Statement', 'Related Issue']);
});

test('extractHeadings ignores lines that are not headings', () => {
  assert.deepEqual(extractHeadings('Not a heading\n#no-space-after-hash\nA `#tag` mid-sentence.'), []);
});

test('normalise strips markup, case and punctuation, collapses whitespace', () => {
  assert.equal(normalise('**AI Assistance disclosure!**'), 'ai assistance disclosure');
  assert.equal(normalise('  Related   Issue  '), 'related issue');
});

test('missingHeadings finds nothing when every template heading has a body match', () => {
  const missing = missingHeadings(['Problem Statement', 'Checklist'], ['## Problem Statement', '## Checklist']);
  assert.deepEqual(missing, []);
});

test('missingHeadings allows a reworded heading via substring match', () => {
  const missing = missingHeadings(['Related Issue'], ['Related Issue / Ticket']);
  assert.deepEqual(missing, []);
});

test('missingHeadings reports a heading with no match at all', () => {
  const missing = missingHeadings(['Problem Statement', 'Checklist'], ['Problem Statement']);
  assert.deepEqual(missing, ['Checklist']);
});

test('an emoji-only or non-Latin heading cannot satisfy every template field', () => {
  // Regression: normalise('🎉') and normalise('问题描述') both collapse to '',
  // and an unfiltered '' is a substring of everything, which used to clear
  // the whole template with a single heading.
  const missing = missingHeadings(['Problem Statement', 'Checklist'], ['🎉', '问题描述']);
  assert.deepEqual(missing, ['Problem Statement', 'Checklist']);
});

test('extractHeadings ignores a heading-like line inside a fenced code block', () => {
  const text = [
    '## Format',
    '',
    '```',
    '# not a real heading',
    'feat(scope): add new feature',
    '```',
    '',
    '## Checklist',
  ].join('\n');
  assert.deepEqual(extractHeadings(text), ['Format', 'Checklist']);
});

test('extractHeadings strips a fence closed with more backticks than it opened with', () => {
  const text = [
    '## Format',
    '',
    '```',
    '# not a real heading',
    '````',
    '',
    '## Checklist',
  ].join('\n');
  assert.deepEqual(extractHeadings(text), ['Format', 'Checklist']);
});

test('missingHeadings does not let "Reformat" satisfy "Format"', () => {
  const missing = missingHeadings(['Format'], ['Reformat']);
  assert.deepEqual(missing, ['Format']);
});

// ---------------------------------------------------------------------------
// checkConformance, against the real template.
// ---------------------------------------------------------------------------

test('a pull request body that is the untouched template is conformant', () => {
  const result = checkConformance(REAL_TEMPLATE, REAL_TEMPLATE);
  assert.equal(result.conformant, true);
  assert.deepEqual(result.missing, []);
});

test('an empty pull request body is missing every section', () => {
  const result = checkConformance(REAL_TEMPLATE, '');
  assert.equal(result.conformant, false);
  assert.equal(result.missing.length, extractHeadings(REAL_TEMPLATE).length);
});

test('a null pull request body is missing every section, not a crash', () => {
  const result = checkConformance(REAL_TEMPLATE, null);
  assert.equal(result.conformant, false);
});

test('a body with one section deleted names only that section', () => {
  const withoutChecklist = REAL_TEMPLATE.replace(/## Checklist[\s\S]*$/, '');
  const result = checkConformance(REAL_TEMPLATE, withoutChecklist);
  assert.equal(result.conformant, false);
  assert.deepEqual(result.missing, ['Checklist']);
});

// ---------------------------------------------------------------------------
// closeMessage
// ---------------------------------------------------------------------------

test('closeMessage names every missing section and the override label', () => {
  const body = closeMessage(['Problem Statement', 'Checklist']);
  assert.match(body, /Problem Statement/);
  assert.match(body, /Checklist/);
  assert.match(body, new RegExp(OVERRIDE_LABEL));
});

// ---------------------------------------------------------------------------
// run(): orchestration. readTemplate is injected so these never touch disk.
// ---------------------------------------------------------------------------

test('run() takes no action on a conformant pull request', async () => {
  const github = fakeGithub();
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr() }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'none');
  assert.equal(github.comments.length, 0);
  assert.equal(github.closed.length, 0);
});

test('run() comments and closes a pull request missing a section', async () => {
  const github = fakeGithub();
  const body = REAL_TEMPLATE.replace(/## Checklist[\s\S]*$/, '');
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ number: 42, body }) }),
    readTemplate: () => REAL_TEMPLATE,
  });
  assert.equal(result.action, 'closed');
  assert.deepEqual(result.missing, ['Checklist']);
  assert.equal(github.comments.length, 1);
  assert.equal(github.comments[0].issue_number, 42);
  assert.match(github.comments[0].body, /Checklist/);
  assert.equal(github.closed.length, 1);
  assert.deepEqual(github.closed[0], {
    owner: 'external-secrets', repo: 'external-secrets', pull_number: 42, state: 'closed',
  });
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

test('run() fails loudly and takes no action when the template has no headings', async () => {
  const github = fakeGithub();
  const core = fakeCore();
  const result = await run({
    core,
    github,
    context: context({ pull_request: pr({ body: '' }) }),
    readTemplate: () => 'no headings in here at all',
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
  const result = await run({
    core: fakeCore(),
    github,
    context: context({ pull_request: pr({ body: REAL_TEMPLATE }) }),
  });
  assert.equal(result.action, 'none');
});

test('isBot matches known accounts and the generic [bot] suffix', () => {
  assert.equal(isBot('dependabot'), true);
  assert.equal(isBot('some-app[bot]'), true);
  assert.equal(isBot('a-human-contributor'), false);
  assert.equal(isBot(null), false);
});
