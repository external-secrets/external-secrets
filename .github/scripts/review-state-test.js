/**
 * Tests for review-state.js.
 *
 * Imports the real classifier rather than duplicating it, so the ordered
 * evaluation cannot drift away from what the workflow actually runs.
 *
 * Run with: node .github/scripts/review-state-test.js
 */

import assert from 'node:assert/strict';
import test from 'node:test';
import { classify, isBot, gateComment, clearedComment, STATE } from './review-state.js';

function pr(overrides = {}) {
  return {
    number: 1,
    isDraft: false,
    labels: [],
    assignees: [],
    reviews: [],
    threads: [],
    ciState: 'SUCCESS',
    ...overrides,
  };
}

const botThread = (o = {}) => ({ author: 'coderabbitai', isResolved: false, isOutdated: false, ...o });
const humanThread = (o = {}) => ({ author: 'someone', isResolved: false, isOutdated: false, ...o });

test('bot detection covers plain and [bot] suffixed logins', () => {
  assert.equal(isBot('coderabbitai'), true);
  assert.equal(isBot('coderabbitai[bot]'), true);
  assert.equal(isBot('some-new-bot[bot]'), true, 'unknown [bot] accounts still count as bots');
  assert.equal(isBot('Skarlso'), false);
  assert.equal(isBot(null), false);
});

test('a fresh pull request with nothing on it needs review', () => {
  assert.equal(classify(pr()).state, STATE.NEEDS_REVIEW);
});

test('unresolved bot findings gate before a human is involved', () => {
  const r = classify(pr({ threads: [botThread(), botThread()] }));
  assert.equal(r.state, STATE.BOT_FINDINGS_OPEN);
  assert.equal(r.botFindingsOpen, 2);
});

test('resolved bot threads do not gate', () => {
  assert.equal(classify(pr({ threads: [botThread({ isResolved: true })] })).state, STATE.NEEDS_REVIEW);
});

test('outdated bot threads do not gate, because the author already pushed over them', () => {
  assert.equal(classify(pr({ threads: [botThread({ isOutdated: true })] })).state, STATE.NEEDS_REVIEW);
});

test('human threads never gate', () => {
  assert.equal(classify(pr({ threads: [humanThread()] })).state, STATE.NEEDS_REVIEW);
});

test('the override label pushes a pull request past the gate', () => {
  const p = pr({ threads: [botThread()], labels: ['review/bots-overridden'] });
  assert.equal(classify(p).state, STATE.NEEDS_REVIEW);
});

test('the gate sits below in-review, so a reviewer is never interrupted', () => {
  const p = pr({ threads: [botThread()], reviews: [{ author: 'Skarlso', state: 'COMMENTED' }] });
  assert.equal(classify(p).state, STATE.IN_REVIEW);
});

test('an assignee alone counts as a claim', () => {
  assert.equal(classify(pr({ assignees: ['moolen'] })).state, STATE.IN_REVIEW);
});

test('a bot review does not make a pull request look engaged', () => {
  const p = pr({ reviews: [{ author: 'coderabbitai', state: 'COMMENTED' }] });
  assert.equal(classify(p).state, STATE.NEEDS_REVIEW, 'bot chatter is not human engagement');
});

test('a bot approval never satisfies the threshold', () => {
  const p = pr({ reviews: [{ author: 'coderabbitai', state: 'APPROVED' }] });
  assert.equal(classify(p).state, STATE.NEEDS_REVIEW);
});

test('one human approval merges a small change', () => {
  const p = pr({ reviews: [{ author: 'Skarlso', state: 'APPROVED' }], labels: ['size/s'] });
  const r = classify(p);
  assert.equal(r.state, STATE.READY_TO_MERGE);
  assert.equal(r.required, 1);
});

test('a large change needs a second approval', () => {
  const p = pr({ reviews: [{ author: 'Skarlso', state: 'APPROVED' }], labels: ['size/l'] });
  const r = classify(p);
  assert.equal(r.state, STATE.NEEDS_2ND_APPROVAL);
  assert.equal(r.required, 2);
});

test('two human approvals clear a large change', () => {
  const p = pr({
    labels: ['size/xl'],
    reviews: [
      { author: 'Skarlso', state: 'APPROVED' },
      { author: 'gusfcarvalho', state: 'APPROVED' },
    ],
  });
  assert.equal(classify(p).state, STATE.READY_TO_MERGE);
});

test('one human plus one bot approval is still one approval', () => {
  const p = pr({
    labels: ['size/l'],
    reviews: [
      { author: 'Skarlso', state: 'APPROVED' },
      { author: 'coderabbitai', state: 'APPROVED' },
    ],
  });
  assert.equal(classify(p).state, STATE.NEEDS_2ND_APPROVAL);
});

test('draft outranks everything, including failing CI and open findings', () => {
  const p = pr({
    isDraft: true,
    ciState: 'FAILURE',
    threads: [botThread()],
    reviews: [{ author: 'Skarlso', state: 'CHANGES_REQUESTED' }],
  });
  assert.equal(classify(p).state, STATE.DRAFT);
});

test('changes requested outranks failing CI', () => {
  const p = pr({ ciState: 'FAILURE', reviews: [{ author: 'Skarlso', state: 'CHANGES_REQUESTED' }] });
  assert.equal(classify(p).state, STATE.CHANGES_REQUESTED);
});

test('failing CI outranks an approval', () => {
  const p = pr({ ciState: 'FAILURE', reviews: [{ author: 'Skarlso', state: 'APPROVED' }] });
  assert.equal(classify(p).state, STATE.CI_RED);
});

test('a bot changes-requested review does not park the pull request with the author', () => {
  const p = pr({ reviews: [{ author: 'github-advanced-security', state: 'CHANGES_REQUESTED' }] });
  assert.notEqual(classify(p).state, STATE.CHANGES_REQUESTED);
});

test('pending CI is not treated as failing', () => {
  assert.equal(classify(pr({ ciState: 'PENDING' })).state, STATE.NEEDS_REVIEW);
});

test('absent CI is not treated as failing', () => {
  assert.equal(classify(pr({ ciState: null })).state, STATE.NEEDS_REVIEW);
});

test('the guidance comment carries a marker and pluralises', () => {
  assert.match(gateComment(1), /<!-- eso-review-routing -->/);
  assert.match(gateComment(1), /\*\*1 open item\*\*/);
  assert.match(gateComment(3), /\*\*3 open items\*\*/);
  assert.match(clearedComment(STATE.NEEDS_REVIEW), /<!-- eso-review-routing -->/);
  assert.match(clearedComment(STATE.NEEDS_REVIEW), /human review queue/);
});
