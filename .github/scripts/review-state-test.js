/**
 * Tests for review-state.js.
 *
 * Imports the real functions rather than duplicating them, so the ordered
 * evaluation cannot drift away from what the workflow actually runs.
 *
 * Run with: node .github/scripts/review-state-test.js
 */

import assert from 'node:assert/strict';
import test from 'node:test';
import run, {
  classify, isBot, effectiveVerdicts, gateComment, clearedComment, STATE, ALL_STATES,
  fetchPullRequest, syncLabels, syncComment, deconflictSweep,
} from './review-state.js';

// run() checks that every state label exists before touching anything.
const ALL_STATES_FOR_TEST = ALL_STATES;

function pr(overrides = {}) {
  return {
    number: 1,
    isDraft: false,
    reviewDecision: null,
    author: 'contributor',
    labels: [],
    assignees: [],
    reviews: [],
    threads: [],
    ciState: 'SUCCESS',
    ...overrides,
  };
}

const review = (author, state, submittedAt) => ({ author, state, submittedAt });

// Captures what the workflow reported, so tests can assert on it.
function fakeCore() {
  const warnings = [];
  const errors = [];
  return {
    warnings,
    errors,
    info: () => {},
    warning: (m) => warnings.push(m),
    error: (m) => errors.push(m),
    setFailed: (m) => errors.push(m),
  };
}
const botThread = (o = {}) => ({ author: 'coderabbitai', isResolved: false, isOutdated: false, ...o });
const humanThread = (o = {}) => ({ author: 'someone', isResolved: false, isOutdated: false, ...o });

// ---------------------------------------------------------------------------
// Effective verdicts. This is the fix for latestReviews losing a verdict as
// soon as its author comments again, so it gets the most attention.
// ---------------------------------------------------------------------------

test('a comment after requesting changes does not erase the request', () => {
  const v = effectiveVerdicts([
    review('Skarlso', 'CHANGES_REQUESTED', '2026-07-09T10:00:00Z'),
    review('Skarlso', 'COMMENTED', '2026-07-09T11:00:00Z'),
  ]);
  assert.equal(v.get('Skarlso'), 'CHANGES_REQUESTED');
});

test('a comment after approving does not erase the approval', () => {
  const v = effectiveVerdicts([
    review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z'),
    review('Skarlso', 'COMMENTED', '2026-07-09T11:00:00Z'),
  ]);
  assert.equal(v.get('Skarlso'), 'APPROVED');
});

test('a later verdict does replace an earlier one', () => {
  const v = effectiveVerdicts([
    review('Skarlso', 'CHANGES_REQUESTED', '2026-07-09T10:00:00Z'),
    review('Skarlso', 'APPROVED', '2026-07-10T10:00:00Z'),
  ]);
  assert.equal(v.get('Skarlso'), 'APPROVED');
});

test('a dismissed approval stops counting', () => {
  const v = effectiveVerdicts([
    review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z'),
    review('Skarlso', 'DISMISSED', '2026-07-10T10:00:00Z'),
  ]);
  assert.equal(v.get('Skarlso'), 'DISMISSED');
});

test('verdicts are ordered by submission time, not array order', () => {
  const v = effectiveVerdicts([
    review('Skarlso', 'APPROVED', '2026-07-10T10:00:00Z'),
    review('Skarlso', 'CHANGES_REQUESTED', '2026-07-09T10:00:00Z'),
  ]);
  assert.equal(v.get('Skarlso'), 'APPROVED', 'the later timestamp wins');
});

test('bots and the pull request author are excluded from verdicts', () => {
  const v = effectiveVerdicts([
    review('coderabbitai', 'APPROVED', '2026-07-09T10:00:00Z'),
    review('contributor', 'APPROVED', '2026-07-09T11:00:00Z'),
    review('Skarlso', 'APPROVED', '2026-07-09T12:00:00Z'),
  ], { excludeAuthor: 'contributor' });
  assert.deepEqual([...v.keys()], ['Skarlso']);
});

// ---------------------------------------------------------------------------
// The real PR that exposed the bug: external-secrets#6613. reviewDecision is
// CHANGES_REQUESTED, but the reviewer commented after requesting changes.
// ---------------------------------------------------------------------------

test('regression, external-secrets#6613 routes to changes-requested', () => {
  const p = pr({
    number: 6613,
    reviewDecision: 'CHANGES_REQUESTED',
    author: 'someone-else',
    reviews: [
      review('evrardj-roche', 'CHANGES_REQUESTED', '2026-07-09T08:00:00Z'),
      review('evrardj-roche', 'COMMENTED', '2026-07-09T09:00:00Z'),
      review('Skarlso', 'COMMENTED', '2026-07-09T10:00:00Z'),
      review('coderabbitai', 'COMMENTED', '2026-07-09T11:00:00Z'),
    ],
    threads: [botThread(), botThread()],
  });
  assert.equal(classify(p).state, STATE.CHANGES_REQUESTED);
});

test('reviewDecision alone is enough, even without a matching review row', () => {
  const p = pr({ reviewDecision: 'CHANGES_REQUESTED', reviews: [] });
  assert.equal(classify(p).state, STATE.CHANGES_REQUESTED);
});

test('per-author verdicts alone are enough when reviewDecision is null', () => {
  const p = pr({
    reviewDecision: null,
    reviews: [review('Skarlso', 'CHANGES_REQUESTED', '2026-07-09T10:00:00Z')],
  });
  assert.equal(classify(p).state, STATE.CHANGES_REQUESTED);
});

// ---------------------------------------------------------------------------
// Bots
// ---------------------------------------------------------------------------

test('bot detection covers plain and [bot] suffixed logins', () => {
  assert.equal(isBot('coderabbitai'), true);
  assert.equal(isBot('coderabbitai[bot]'), true);
  assert.equal(isBot('some-new-bot[bot]'), true, 'unknown [bot] accounts still count as bots');
  assert.equal(isBot('Skarlso'), false);
  assert.equal(isBot(null), false);
});

test('a bot approval never satisfies the threshold', () => {
  const p = pr({ reviews: [review('coderabbitai', 'APPROVED', '2026-07-09T10:00:00Z')] });
  const r = classify(p);
  assert.equal(r.approvals, 0);
  assert.notEqual(r.state, STATE.READY_TO_MERGE);
});

test('a bot changes-requested review does not park the pull request with the author', () => {
  const p = pr({ reviews: [review('github-advanced-security', 'CHANGES_REQUESTED', '2026-07-09T10:00:00Z')] });
  assert.notEqual(classify(p).state, STATE.CHANGES_REQUESTED);
});

// ---------------------------------------------------------------------------
// The bot gate
// ---------------------------------------------------------------------------

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
  const p = pr({ threads: [botThread()], reviews: [review('Skarlso', 'COMMENTED', '2026-07-09T10:00:00Z')] });
  assert.equal(classify(p).state, STATE.IN_REVIEW);
});

// ---------------------------------------------------------------------------
// Engagement, including the author exclusion
// ---------------------------------------------------------------------------

test('an assignee alone counts as a claim', () => {
  assert.equal(classify(pr({ assignees: ['moolen'] })).state, STATE.IN_REVIEW);
});

test('the author commenting on their own diff does not count as engagement', () => {
  const p = pr({
    author: 'contributor',
    threads: [botThread()],
    reviews: [review('contributor', 'COMMENTED', '2026-07-09T10:00:00Z')],
  });
  assert.equal(classify(p).state, STATE.BOT_FINDINGS_OPEN, 'the author cannot clear their own gate');
});

test('someone else commenting does count as engagement', () => {
  const p = pr({
    author: 'contributor',
    threads: [botThread()],
    reviews: [review('Skarlso', 'COMMENTED', '2026-07-09T10:00:00Z')],
  });
  assert.equal(classify(p).state, STATE.IN_REVIEW);
});

// ---------------------------------------------------------------------------
// Approval thresholds. Each asserts `required` explicitly, so removing a size
// label from LARGE_LABELS fails a test rather than passing silently.
// ---------------------------------------------------------------------------

test('one human approval merges a small change', () => {
  const p = pr({ reviews: [review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z')], labels: ['size/s'] });
  const r = classify(p);
  assert.equal(r.required, 1);
  assert.equal(r.state, STATE.READY_TO_MERGE);
});

test('size/l needs a second approval', () => {
  const p = pr({ reviews: [review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z')], labels: ['size/l'] });
  const r = classify(p);
  assert.equal(r.required, 2);
  assert.equal(r.state, STATE.NEEDS_2ND_APPROVAL);
});

test('size/xl needs a second approval too', () => {
  const p = pr({ reviews: [review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z')], labels: ['size/xl'] });
  const r = classify(p);
  assert.equal(r.required, 2, 'size/xl must be in LARGE_LABELS');
  assert.equal(r.state, STATE.NEEDS_2ND_APPROVAL);
});

test('two human approvals clear a large change', () => {
  const p = pr({
    labels: ['size/xl'],
    reviews: [
      review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z'),
      review('gusfcarvalho', 'APPROVED', '2026-07-09T11:00:00Z'),
    ],
  });
  const r = classify(p);
  assert.equal(r.required, 2);
  assert.equal(r.approvals, 2);
  assert.equal(r.state, STATE.READY_TO_MERGE);
});

test('the same person approving twice is still one approval', () => {
  const p = pr({
    labels: ['size/l'],
    reviews: [
      review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z'),
      review('Skarlso', 'APPROVED', '2026-07-10T10:00:00Z'),
    ],
  });
  assert.equal(classify(p).approvals, 1);
});

// ---------------------------------------------------------------------------
// Precedence and CI
// ---------------------------------------------------------------------------

test('draft outranks everything, including failing CI and open findings', () => {
  const p = pr({
    isDraft: true,
    ciState: 'FAILURE',
    threads: [botThread()],
    reviewDecision: 'CHANGES_REQUESTED',
  });
  assert.equal(classify(p).state, STATE.DRAFT);
});

test('changes requested outranks failing CI', () => {
  const p = pr({ ciState: 'FAILURE', reviewDecision: 'CHANGES_REQUESTED' });
  assert.equal(classify(p).state, STATE.CHANGES_REQUESTED);
});

test('failing CI outranks an approval', () => {
  const p = pr({ ciState: 'FAILURE', reviews: [review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z')] });
  assert.equal(classify(p).state, STATE.CI_RED);
});

test('a crashed check reports ERROR and still counts as red', () => {
  const p = pr({ ciState: 'ERROR', reviews: [review('Skarlso', 'APPROVED', '2026-07-09T10:00:00Z')] });
  assert.equal(classify(p).state, STATE.CI_RED, 'ERROR must not fall through to ready-to-merge');
});

test('pending, expected and absent CI are not treated as failing', () => {
  for (const ciState of ['PENDING', 'EXPECTED', null]) {
    assert.equal(classify(pr({ ciState })).state, STATE.NEEDS_REVIEW, `ciState=${ciState}`);
  }
});

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

test('the guidance comment carries a marker and pluralises', () => {
  assert.match(gateComment(1), /<!-- eso-review-routing -->/);
  assert.match(gateComment(1), /\*\*1 open item\*\*/);
  assert.match(gateComment(3), /\*\*3 open items\*\*/);
  assert.match(clearedComment(STATE.NEEDS_REVIEW), /<!-- eso-review-routing -->/);
  assert.match(clearedComment(STATE.NEEDS_REVIEW), /human review queue/);
});

// ---------------------------------------------------------------------------
// Data plumbing and write paths, against a fake Octokit.
// ---------------------------------------------------------------------------

function fakeGithub({ graphql, comments = [], removeLabelError, liveLabels } = {}) {
  const calls = {
    addLabels: [], removeLabel: [], createComment: [], updateComment: [], deleteComment: [],
    labelReads: 0,
  };
  const listComments = () => {};
  listComments.__kind = 'comments';

  // `liveLabels` as an array of arrays models another run changing labels
  // mid-flight: each read returns the next entry. As a flat array it seeds a
  // mutable set, so a removal is actually reflected in the next read the way
  // the real API would.
  const scripted = Array.isArray(liveLabels) && Array.isArray(liveLabels[0]);
  const state = new Set(scripted ? [] : (liveLabels || []));

  const github = {
    graphql: async () => graphql,
    paginate: async (fn) => (fn.__kind === 'comments' ? comments : []),
    rest: {
      issues: {
        listComments,
        listLabelsOnIssue: async () => {
          const answer = scripted
            ? (liveLabels[Math.min(calls.labelReads, liveLabels.length - 1)] || [])
            : [...state];
          calls.labelReads += 1;
          return { data: answer.map((name) => ({ name })) };
        },
        addLabels: async (p) => { calls.addLabels.push(p); p.labels.forEach((l) => state.add(l)); },
        removeLabel: async (p) => {
          calls.removeLabel.push(p);
          if (removeLabelError) throw removeLabelError;
          state.delete(p.name);
        },
        createComment: async (p) => {
          calls.createComment.push(p);
          // Derived from the shared comment list, not a per-client counter, so
          // two clients writing to one pull request get distinct ids the way
          // the real API guarantees.
          const created = { id: 1001 + comments.length, body: p.body, user: { type: 'Bot' } };
          comments.push(created);
          return { data: created };
        },
        updateComment: async (p) => { calls.updateComment.push(p); },
        deleteComment: async (p) => { calls.deleteComment.push(p); },
      },
    },
  };
  return { github, calls };
}

const graphqlPr = (overrides = {}) => ({
  repository: {
    pullRequest: {
      number: 42,
      isDraft: false,
      reviewDecision: 'REVIEW_REQUIRED',
      author: { login: 'contributor' },
      labels: { nodes: [{ name: 'size/l' }] },
      assignees: { nodes: [{ login: 'moolen' }] },
      reviews: {
        totalCount: 1,
        pageInfo: { hasNextPage: false },
        nodes: [{ author: { login: 'Skarlso' }, state: 'APPROVED', submittedAt: '2026-07-09T10:00:00Z' }],
      },
      reviewThreads: {
        totalCount: 1,
        pageInfo: { hasPreviousPage: false },
        nodes: [{ isResolved: false, isOutdated: false, comments: { nodes: [{ author: { login: 'coderabbitai' } }] } }],
      },
      commits: { nodes: [{ commit: { statusCheckRollup: { state: 'SUCCESS' } } }] },
      ...overrides,
    },
  },
});

test('fetchPullRequest maps a full response onto the classifier shape', async () => {
  const { github } = fakeGithub({ graphql: graphqlPr() });
  const p = await fetchPullRequest(github, 'o', 'r', 42);
  assert.deepEqual(p, {
    truncated: null,
    number: 42,
    isDraft: false,
    reviewDecision: 'REVIEW_REQUIRED',
    author: 'contributor',
    labels: ['size/l'],
    assignees: ['moolen'],
    reviews: [{ author: 'Skarlso', state: 'APPROVED', submittedAt: '2026-07-09T10:00:00Z' }],
    threads: [{ author: 'coderabbitai', isResolved: false, isOutdated: false }],
    ciState: 'SUCCESS',
  });
  assert.equal(classify(p).state, STATE.NEEDS_2ND_APPROVAL);
});

test('fetchPullRequest returns null when the pull request is missing', async () => {
  const { github } = fakeGithub({ graphql: { repository: { pullRequest: null } } });
  assert.equal(await fetchPullRequest(github, 'o', 'r', 999), null);
});

test('fetchPullRequest survives a thread with no comments', async () => {
  const g = graphqlPr();
  g.repository.pullRequest.reviewThreads.nodes = [
    { isResolved: false, isOutdated: false, comments: { nodes: [] } },
  ];
  const { github } = fakeGithub({ graphql: g });
  const p = await fetchPullRequest(github, 'o', 'r', 42);
  assert.equal(p.threads[0].author, null);
  assert.equal(classify(p).botFindingsOpen, 0, 'an authorless thread cannot gate');
});

test('fetchPullRequest survives deleted users and a missing author', async () => {
  const g = graphqlPr();
  g.repository.pullRequest.author = null;
  g.repository.pullRequest.reviews.nodes = [{ author: null, state: 'APPROVED', submittedAt: 'x' }];
  const { github } = fakeGithub({ graphql: g });
  const p = await fetchPullRequest(github, 'o', 'r', 42);
  assert.equal(p.author, null);
  assert.equal(p.reviews[0].author, null);
  assert.equal(classify(p).approvals, 0, 'an authorless review cannot be attributed');
});

test('fetchPullRequest treats a missing check rollup as no CI rather than failure', async () => {
  const g = graphqlPr();
  g.repository.pullRequest.commits.nodes = [{ commit: {} }];
  const { github } = fakeGithub({ graphql: g });
  const p = await fetchPullRequest(github, 'o', 'r', 42);
  assert.equal(p.ciState, null);
  assert.notEqual(classify(p).state, STATE.CI_RED);
});

test('fetchPullRequest reports both connections when both are truncated', async () => {
  const core = fakeCore();
  const g = graphqlPr();
  g.repository.pullRequest.reviews.pageInfo.hasNextPage = true;
  g.repository.pullRequest.reviewThreads.pageInfo.hasPreviousPage = true;
  const { github } = fakeGithub({ graphql: g });
  const p = await fetchPullRequest(github, 'o', 'r', 42, core);
  assert.deepEqual(p.truncated, ['reviews', 'reviewThreads']);
  assert.equal(core.warnings.length, 1);
  assert.match(core.warnings[0], /reviews and reviewThreads/);
});

test('syncLabels adds the desired label and strips only stale review states', async () => {
  const { github, calls } = fakeGithub({ liveLabels: ['review/needs-review', 'size/l'] });
  const res = await syncLabels(github, 'o', 'r', 1, ['review/needs-review', 'size/l'], STATE.IN_REVIEW);
  assert.deepEqual(calls.removeLabel.map((c) => c.name), ['review/needs-review']);
  assert.deepEqual(calls.addLabels[0].labels, [STATE.IN_REVIEW]);
  assert.equal(res.applied, true);
});

test('syncLabels is a no-op when the label is already correct', async () => {
  const { github, calls } = fakeGithub({ liveLabels: [STATE.IN_REVIEW, 'size/s'] });
  await syncLabels(github, 'o', 'r', 1, [STATE.IN_REVIEW, 'size/s'], STATE.IN_REVIEW);
  assert.equal(calls.removeLabel.length + calls.addLabels.length, 0);
});

test('syncLabels never touches the override label', async () => {
  const { github, calls } = fakeGithub({ liveLabels: ['review/bots-overridden'] });
  await syncLabels(github, 'o', 'r', 1, ['review/bots-overridden'], STATE.NEEDS_REVIEW);
  assert.equal(calls.removeLabel.length, 0, 'the override is a human decision, not derived state');
});

test('syncLabels tolerates a 404 but rethrows a permissions failure', async () => {
  const live = ['review/needs-review'];
  const gone = Object.assign(new Error('Label does not exist'), { status: 404 });
  await syncLabels(fakeGithub({ removeLabelError: gone, liveLabels: live }).github, 'o', 'r', 1, live, STATE.IN_REVIEW);

  const denied = Object.assign(new Error('Resource not accessible by integration'), { status: 403 });
  await assert.rejects(
    () => syncLabels(fakeGithub({ removeLabelError: denied, liveLabels: live }).github, 'o', 'r', 1, live, STATE.IN_REVIEW),
    /not accessible/,
  );
});

test('syncLabels writes nothing in a dry run but still reports the change', async () => {
  const { github, calls } = fakeGithub({ liveLabels: ['review/needs-review', 'size/l'] });
  const res = await syncLabels(
    github, 'o', 'r', 1, ['review/needs-review'], STATE.IN_REVIEW, { dryRun: true },
  );
  assert.equal(calls.removeLabel.length + calls.addLabels.length, 0, 'dry run must not write');
  assert.equal(res.applied, false);
  assert.equal(res.added, STATE.IN_REVIEW);
  assert.deepEqual(res.removed, ['review/needs-review']);
});

test('a dry run reports from a live read, not the classification snapshot', async () => {
  // The snapshot says needs-review, but the pull request has moved on since.
  const { github, calls } = fakeGithub({ liveLabels: ['review/draft'] });
  const res = await syncLabels(
    github, 'o', 'r', 1, ['review/needs-review'], STATE.IN_REVIEW, { dryRun: true },
  );
  assert.deepEqual(res.removed, ['review/draft'], 'reports what is actually on the pull request');
  assert.equal(calls.labelReads, 1, 'it reads');
  assert.equal(calls.removeLabel.length + calls.addLabels.length, 0, 'but never writes');
});

test('syncComment stays silent on a clean pull request', async () => {
  const { github, calls } = fakeGithub({ comments: [] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.NEEDS_REVIEW, botFindingsOpen: 0 });
  assert.equal(action, 'none');
  assert.equal(calls.createComment.length, 0, 'never introduce itself unprompted');
});

test('syncComment posts once when the gate first closes', async () => {
  const { github, calls } = fakeGithub({ comments: [] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 3 });
  assert.equal(action, 'created');
  assert.match(calls.createComment[0].body, /3 open items/);
});

test('syncComment edits its own comment rather than posting a second', async () => {
  const { github, calls } = fakeGithub({ comments: [{ id: 7, body: gateComment(3), user: { type: 'Bot' } }] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1 });
  assert.equal(action, 'updated');
  assert.equal(calls.createComment.length, 0);
  assert.equal(calls.updateComment[0].comment_id, 7);
});

test('syncComment rewrites to the cleared text once the gate opens', async () => {
  const { github, calls } = fakeGithub({ comments: [{ id: 7, body: gateComment(2), user: { type: 'Bot' } }] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.NEEDS_REVIEW, botFindingsOpen: 0 });
  assert.equal(action, 'updated');
  assert.match(calls.updateComment[0].body, /human review queue/);
});

test('syncComment does not rewrite an identical body', async () => {
  const { github, calls } = fakeGithub({ comments: [{ id: 7, body: gateComment(2), user: { type: 'Bot' } }] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 2 });
  assert.equal(action, 'unchanged');
  assert.equal(calls.updateComment.length, 0, 'no needless notification');
});

test('syncComment ignores comments from other authors', async () => {
  const { github, calls } = fakeGithub({ comments: [{ id: 1, body: 'unrelated', user: { type: 'User' } }, { id: 2, body: null, user: { type: 'Bot' } }] });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1 });
  assert.equal(action, 'created');
  assert.equal(calls.updateComment.length, 0);
});

test('syncComment writes nothing in a dry run', async () => {
  const clean = fakeGithub({ comments: [] });
  assert.equal(
    await syncComment(clean.github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1 }, { dryRun: true }),
    'would-create',
  );
  assert.equal(clean.calls.createComment.length, 0, 'dry run must not write');

  const existing = fakeGithub({ comments: [{ id: 7, body: gateComment(9), user: { type: 'Bot' } }] });
  assert.equal(
    await syncComment(existing.github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1 }, { dryRun: true }),
    'would-update',
  );
  assert.equal(existing.calls.updateComment.length, 0, 'dry run must not write');
});

// ---------------------------------------------------------------------------
// Fixes from review of the first version.
// ---------------------------------------------------------------------------

test('truncated data is flagged so the caller can refuse to write', async () => {
  const core = fakeCore();
  const g = graphqlPr();
  g.repository.pullRequest.reviewThreads.pageInfo.hasPreviousPage = true;
  const { github } = fakeGithub({ graphql: g });
  const p = await fetchPullRequest(github, 'o', 'r', 42, core);
  assert.deepEqual(p.truncated, ['reviewThreads']);
  assert.match(core.warnings[0], /skipping rather than/);
});

test('complete data carries no truncation flag', async () => {
  const { github } = fakeGithub({ graphql: graphqlPr() });
  const p = await fetchPullRequest(github, 'o', 'r', 42);
  assert.equal(p.truncated, null);
});

test('syncLabels writes against a fresh read, not the stale snapshot', async () => {
  // Another run relabelled this pull request after we classified it. The stale
  // snapshot says needs-review; the live state says in-review.
  const { github, calls } = fakeGithub({ liveLabels: ['review/in-review', 'size/l'] });
  await syncLabels(github, 'o', 'r', 1, ['review/needs-review'], STATE.CI_RED);
  assert.deepEqual(
    calls.removeLabel.map((c) => c.name),
    ['review/in-review'],
    'removes what is actually there, not what we last saw',
  );
});

test('a human cannot capture the guidance comment slot with the marker', async () => {
  const { github, calls } = fakeGithub({
    comments: [{ id: 99, body: `spoof ${gateComment(1)}`, user: { type: 'User' } }],
  });
  const action = await syncComment(github, 'o', 'r', 1, { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1 });
  assert.equal(action, 'created', 'the workflow posts its own rather than editing a human comment');
  assert.equal(calls.updateComment.length, 0);
});

test('the gate comment tells contributors to resolve, not merely reply', () => {
  const body = gateComment(2);
  assert.match(body, /Resolve conversation/);
  assert.match(body, /a reply on its own/i);
  assert.match(body, /review\/bots-overridden/);
  assert.doesNotMatch(body, /leave it open/i, 'the old text promised an open reply was enough');
});

test('run() refuses to write a label when the data is truncated', async () => {
  const calls = { addLabels: [], removeLabel: [], createComment: [], updateComment: [] };
  const listComments = () => {};
  listComments.__kind = 'comments';
  const listLabelsForRepo = () => {};
  listLabelsForRepo.__kind = 'repoLabels';

  const g = graphqlPr();
  g.repository.pullRequest.reviewThreads.pageInfo.hasPreviousPage = true;

  const github = {
    graphql: async () => g,
    paginate: async (fn) => (fn.__kind === 'repoLabels'
      ? [...ALL_STATES_FOR_TEST].map((name) => ({ name }))
      : []),
    rest: {
      issues: {
        listComments,
        listLabelsForRepo,
        listLabelsOnIssue: async () => ({ data: [] }),
        addLabels: async (p) => { calls.addLabels.push(p); },
        removeLabel: async (p) => { calls.removeLabel.push(p); },
        createComment: async (p) => { calls.createComment.push(p); },
        updateComment: async (p) => { calls.updateComment.push(p); },
      },
    },
  };
  const core = fakeCore();

  const results = await run({
    core, github, context: { repo: { owner: 'o', repo: 'r' } }, numbers: [42],
  });

  assert.deepEqual(results, [], 'a truncated pull request produces no result');
  assert.equal(calls.addLabels.length, 0, 'no label written');
  assert.equal(calls.removeLabel.length, 0, 'no label removed');
  assert.equal(calls.createComment.length, 0, 'no comment written');
  assert.match(core.warnings.join(' '), /truncated data/);
});

test('run() does write when the data is complete', async () => {
  const calls = { addLabels: [] };
  const listComments = () => {};
  listComments.__kind = 'comments';
  const listLabelsForRepo = () => {};
  listLabelsForRepo.__kind = 'repoLabels';
  const github = {
    graphql: async () => graphqlPr(),
    paginate: async (fn) => (fn.__kind === 'repoLabels'
      ? [...ALL_STATES_FOR_TEST].map((name) => ({ name }))
      : []),
    rest: {
      issues: {
        listComments,
        listLabelsForRepo,
        listLabelsOnIssue: async () => ({ data: [] }),
        addLabels: async (p) => { calls.addLabels.push(p); },
        removeLabel: async () => {},
        createComment: async () => {},
        updateComment: async () => {},
      },
    },
  };
  const core = fakeCore();
  const results = await run({
    core, github, context: { repo: { owner: 'o', repo: 'r' } }, numbers: [42],
  });
  assert.equal(results.length, 1, 'the control case still produces a result');
  assert.deepEqual(calls.addLabels[0].labels, [STATE.NEEDS_2ND_APPROVAL]);
});

// A shared mutable label store driven by two concurrent syncLabels calls. The
// earlier canned-sequence tests could not catch the zero-label bug, because
// only one side of the race actually ran.
function sharedRepo(initial) {
  const state = new Set(initial);
  const github = {
    rest: {
      issues: {
        listLabelsOnIssue: async () => ({ data: [...state].map((name) => ({ name })) }),
        addLabels: async (p) => { p.labels.forEach((l) => state.add(l)); },
        removeLabel: async (p) => {
          if (!state.has(p.name)) {
            throw Object.assign(new Error('not found'), { status: 404 });
          }
          state.delete(p.name);
        },
      },
    },
  };
  return { github, state, reviewLabels: () => [...state].filter((l) => l.startsWith('review/')) };
}

test('two concurrent runs never strip a pull request of every review label', async () => {
  const repo = sharedRepo(['review/needs-review', 'size/l']);
  await Promise.all([
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.CI_RED),
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.IN_REVIEW),
  ]);
  const surviving = repo.reviewLabels();
  assert.notEqual(
    surviving.length, 0,
    'zero labels would silently drop the pull request out of the queue',
  );
  assert.ok(repo.state.has('size/l'), 'unrelated labels are untouched');
});

test('a two-label race converges on the next run', async () => {
  const repo = sharedRepo(['review/needs-review', 'size/l']);
  await Promise.all([
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.CI_RED),
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.IN_REVIEW),
  ]);
  await syncLabels(repo.github, 'o', 'r', 1, repo.reviewLabels(), STATE.CI_RED);
  assert.deepEqual(repo.reviewLabels(), [STATE.CI_RED]);
});

test('the verification pass restores a label a concurrent run deleted', async () => {
  const repo = sharedRepo([STATE.IN_REVIEW]);
  // Between our add and our verification read, something removes our label.
  const realRead = repo.github.rest.issues.listLabelsOnIssue;
  let reads = 0;
  repo.github.rest.issues.listLabelsOnIssue = async () => {
    reads += 1;
    if (reads === 2) repo.state.delete(STATE.CI_RED);
    return realRead();
  };
  const res = await syncLabels(repo.github, 'o', 'r', 1, [STATE.IN_REVIEW], STATE.CI_RED);
  assert.equal(res.restored, true, 'the deletion is noticed and undone');
  assert.ok(repo.state.has(STATE.CI_RED));
});

test('a sequential run still lands exactly one label', async () => {
  const repo = sharedRepo(['review/needs-review', 'review/draft', 'size/l']);
  await syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.READY_TO_MERGE);
  assert.deepEqual(repo.reviewLabels(), [STATE.READY_TO_MERGE]);
  assert.ok(repo.state.has('size/l'));
});

test('two concurrent runs do not leave two guidance comments', async () => {
  // Both runs see no marker comment, so both create one. The later creator
  // must clean up after itself rather than orphaning a duplicate.
  const shared = [];
  const make = () => fakeGithub({ comments: shared });
  const a = make();
  const b = make();
  const gate = { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 2 };
  const [ra, rb] = await Promise.all([
    syncComment(a.github, 'o', 'r', 1, gate),
    syncComment(b.github, 'o', 'r', 1, gate),
  ]);

  const markers = shared.filter((c) => c.body.includes('<!-- eso-review-routing -->'));
  const deleted = [...a.calls.deleteComment, ...b.calls.deleteComment].map((c) => c.comment_id);
  const surviving = markers.filter((c) => !deleted.includes(c.id));
  assert.equal(surviving.length, 1, 'exactly one guidance comment survives');
  assert.ok(
    [ra, rb].includes('created-then-deduped'),
    'the loser reports that it cleaned up',
  );
});

test('a single run creating a comment does not delete it again', async () => {
  const { github, calls } = fakeGithub({ comments: [] });
  const action = await syncComment(github, 'o', 'r', 1, {
    state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen: 1,
  });
  assert.equal(action, 'created');
  assert.equal(calls.deleteComment.length, 0, 'nothing to dedupe');
});

// ---------------------------------------------------------------------------
// End-of-sweep deconfliction. Only the sweep cleans up, and only by
// withdrawing its own state, which is what makes zero unreachable.
// ---------------------------------------------------------------------------

test('a sweep withdraws its own state when an event left one too', async () => {
  const repo = sharedRepo([STATE.CI_RED, STATE.IN_REVIEW, 'size/l']);
  const core = fakeCore();
  const out = await deconflictSweep(
    repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], core,
  );
  assert.deepEqual(repo.reviewLabels(), [STATE.CI_RED], "the event's state stands");
  assert.deepEqual(out, [{ number: 1, label: STATE.IN_REVIEW, action: 'yielded' }]);
  assert.ok(repo.state.has('size/l'), 'unrelated labels untouched');
});

test('a sweep leaves a single state alone', async () => {
  const repo = sharedRepo([STATE.IN_REVIEW]);
  const out = await deconflictSweep(
    repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], fakeCore(),
  );
  assert.deepEqual(repo.reviewLabels(), [STATE.IN_REVIEW]);
  assert.deepEqual(out, [], 'nothing to deconflict');
});

test('a sweep does nothing when its own state is already gone', async () => {
  // An event ran after the sweep's write and replaced its label outright.
  const repo = sharedRepo([STATE.CI_RED, STATE.READY_TO_MERGE]);
  const out = await deconflictSweep(
    repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], fakeCore(),
  );
  assert.equal(repo.reviewLabels().length, 2, 'not ours to resolve, next run converges');
  assert.deepEqual(out, []);
});

test('a sweep never leaves a pull request with no state at all', async () => {
  const repo = sharedRepo([STATE.CI_RED, STATE.IN_REVIEW]);
  // The other state vanishes between our read and our removal.
  const realRemove = repo.github.rest.issues.removeLabel;
  repo.github.rest.issues.removeLabel = async (p) => {
    repo.state.delete(STATE.CI_RED);
    return realRemove(p);
  };
  const core = fakeCore();
  const out = await deconflictSweep(
    repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], core,
  );
  assert.deepEqual(repo.reviewLabels(), [STATE.IN_REVIEW], 'put back rather than left empty');
  assert.equal(out[0].action, 'restored');
  assert.match(core.warnings.join(' '), /put it back/);
});

test('a sweep will not withdraw the maintainer override', async () => {
  const repo = sharedRepo([STATE.IN_REVIEW, 'review/bots-overridden']);
  const out = await deconflictSweep(
    repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], fakeCore(),
  );
  assert.ok(repo.state.has('review/bots-overridden'), 'the override is not derived state');
  assert.deepEqual(out, [], 'the override does not count as a competing state');
});

test('the whole race resolves to exactly one state, the event side', async () => {
  const repo = sharedRepo(['review/needs-review', 'size/l']);
  // An event run and a sweep run overlap on the same pull request.
  await Promise.all([
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.CI_RED),
    syncLabels(repo.github, 'o', 'r', 1, ['review/needs-review'], STATE.IN_REVIEW),
  ]);
  assert.equal(repo.reviewLabels().length, 2, 'the race does leave two');
  // Then the sweep's end-of-cycle pass runs.
  await deconflictSweep(repo.github, 'o', 'r', [{ number: 1, label: STATE.IN_REVIEW }], fakeCore());
  assert.deepEqual(repo.reviewLabels(), [STATE.CI_RED], 'one state, the event wins');
});

test('an event-driven run never deconflicts, which is what keeps zero unreachable', async () => {
  // A competing state appears only AFTER this run has written its own, which is
  // the real shape of the race: syncLabels cannot have cleaned it up already.
  // If an event run deconflicted, it would withdraw its own label here and,
  // with the other side doing the same, the pull request would end with none.
  const build = () => {
    const state = new Set();
    const calls = { removeLabel: [] };
    let reads = 0;
    const listComments = () => {};
    listComments.__kind = 'comments';
    const listLabelsForRepo = () => {};
    listLabelsForRepo.__kind = 'repoLabels';
    return {
      state,
      calls,
      github: {
        graphql: async () => graphqlPr(),
        paginate: async (fn) => (fn.__kind === 'repoLabels'
          ? ALL_STATES.map((name) => ({ name })) : []),
        rest: {
          issues: {
            listComments,
            listLabelsForRepo,
            listLabelsOnIssue: async () => {
              reads += 1;
              // Reads 1 and 2 belong to syncLabels; from read 3 on, an
              // overlapping run has landed its own state.
              if (reads >= 3) state.add(STATE.CI_RED);
              return { data: [...state].map((name) => ({ name })) };
            },
            addLabels: async (p) => { p.labels.forEach((l) => state.add(l)); },
            removeLabel: async (p) => { calls.removeLabel.push(p.name); state.delete(p.name); },
            createComment: async () => ({ data: { id: 1 } }),
            updateComment: async () => {},
            deleteComment: async () => {},
          },
        },
      },
    };
  };

  const asEvent = build();
  await run({
    core: fakeCore(),
    github: asEvent.github,
    context: { repo: { owner: 'o', repo: 'r' } },
    numbers: [42],
    sweep: false,
  });
  assert.ok(
    asEvent.state.has(STATE.NEEDS_2ND_APPROVAL),
    'an event run must never withdraw the label it just applied',
  );
  assert.ok(
    !asEvent.calls.removeLabel.includes(STATE.NEEDS_2ND_APPROVAL),
    'and must not even try',
  );

  const asSweep = build();
  await run({
    core: fakeCore(),
    github: asSweep.github,
    context: { repo: { owner: 'o', repo: 'r' } },
    numbers: [42],
    sweep: true,
  });
  assert.ok(
    asSweep.calls.removeLabel.includes(STATE.NEEDS_2ND_APPROVAL),
    'a sweep yields the state it applied when an overlapping run left one',
  );
  assert.deepEqual(
    [...asSweep.state].filter((l) => ALL_STATES.includes(l)),
    [STATE.CI_RED],
    'exactly one state survives, the event side',
  );
});
