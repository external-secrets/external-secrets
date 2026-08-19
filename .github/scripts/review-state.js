/**
 * Review state router.
 *
 * Derives one review/* label per open pull request from facts GitHub already
 * knows, so "which PR should I review, and has anyone looked at it" is a query
 * rather than a read of every comment thread. See design/015-review-routing.md.
 */

// Accounts whose review activity never counts as human coverage. They gate
// (see BOT_FINDINGS_OPEN) but can never approve a change through.
export const BOTS = new Set([
  'coderabbitai',
  'coderabbitai[bot]',
  'github-advanced-security',
  'github-advanced-security[bot]',
  'copilot-pull-request-reviewer',
  'copilot-pull-request-reviewer[bot]',
  'dependabot',
  'dependabot[bot]',
  'eso-service-account-app',
  'eso-service-account-app[bot]',
]);

export const STATE = {
  DRAFT: 'review/draft',
  CHANGES_REQUESTED: 'review/changes-requested',
  CI_RED: 'review/ci-red',
  BOT_FINDINGS_OPEN: 'review/bot-findings-open',
  NEEDS_REVIEW: 'review/needs-review',
  IN_REVIEW: 'review/in-review',
  NEEDS_2ND_APPROVAL: 'review/needs-2nd-approval',
  READY_TO_MERGE: 'review/ready-to-merge',
};

export const ALL_STATES = Object.values(STATE);

// Applied by a maintainer to push a pull request past the bot gate: false
// positives, a newcomer stuck on nitpicks, or a bot outage.
export const OVERRIDE_LABEL = 'review/bots-overridden';

// Changes this size need two human approvals, per docs/contributing/process.md.
const LARGE_LABELS = new Set(['size/l', 'size/xl']);

const COMMENT_MARKER = '<!-- eso-review-routing -->';

export function isBot(login) {
  return BOTS.has(login) || (typeof login === 'string' && login.endsWith('[bot]'));
}

/**
 * Decide the review state. Ordered evaluation, first match wins: conditions
 * overlap constantly, so the order below is the specification, not a detail.
 */
export function classify(pr) {
  const labels = new Set(pr.labels || []);
  const humanReviews = (pr.reviews || []).filter((r) => !isBot(r.author));
  const approvals = humanReviews.filter((r) => r.state === 'APPROVED').length;
  const required = [...LARGE_LABELS].some((l) => labels.has(l)) ? 2 : 1;

  // Only bot threads that are unresolved AND still current gate the pull
  // request. An outdated thread means the author already pushed over the
  // flagged line, so counting it would trap someone who did respond.
  const botFindingsOpen = (pr.threads || []).filter(
    (t) => isBot(t.author) && !t.isResolved && !t.isOutdated,
  ).length;

  const changesRequested = humanReviews.some((r) => r.state === 'CHANGES_REQUESTED');
  const humanEngaged = humanReviews.length > 0 || (pr.assignees || []).length > 0;

  if (pr.isDraft) return { state: STATE.DRAFT, botFindingsOpen, approvals, required };
  if (changesRequested) return { state: STATE.CHANGES_REQUESTED, botFindingsOpen, approvals, required };
  if (pr.ciState === 'FAILURE') return { state: STATE.CI_RED, botFindingsOpen, approvals, required };
  if (approvals >= required) return { state: STATE.READY_TO_MERGE, botFindingsOpen, approvals, required };
  if (approvals >= 1) return { state: STATE.NEEDS_2ND_APPROVAL, botFindingsOpen, approvals, required };

  // The gate sits below IN_REVIEW on purpose. Once a human is engaged their
  // judgement supersedes the bot, and a card must not be pulled out from
  // under a reviewer mid-read by a fresh nitpick.
  if (humanEngaged) return { state: STATE.IN_REVIEW, botFindingsOpen, approvals, required };
  if (botFindingsOpen > 0 && !labels.has(OVERRIDE_LABEL)) {
    return { state: STATE.BOT_FINDINGS_OPEN, botFindingsOpen, approvals, required };
  }
  return { state: STATE.NEEDS_REVIEW, botFindingsOpen, approvals, required };
}

export function gateComment(count) {
  const items = count === 1 ? '**1 open item**' : `**${count} open items**`;
  return [
    COMMENT_MARKER,
    `Automated review has flagged ${items} on this pull request.`,
    '',
    'Review here runs in two stages: automated review first, then a maintainer.',
    'This pull request moves into the human review queue once the automated',
    'comments are addressed, either by pushing a fix or by replying in the thread.',
    '',
    'You can resolve a thread yourself with **Resolve conversation**. If you think a',
    'finding is wrong, say so in the thread and leave it open; a maintainer will',
    'judge it. You are not expected to change code you disagree with.',
    '',
    `Status: \`${STATE.BOT_FINDINGS_OPEN}\``,
  ].join('\n');
}

export function clearedComment(state) {
  return [
    COMMENT_MARKER,
    'Automated review is clear. This pull request is now in the human review queue.',
    '',
    `Status: \`${state}\``,
  ].join('\n');
}

const PR_QUERY = `
query($owner:String!, $repo:String!, $number:Int!) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$number) {
      number isDraft
      labels(first:50){nodes{name}}
      assignees(first:10){nodes{login}}
      latestReviews(first:50){nodes{author{login} state}}
      reviewThreads(first:100){nodes{
        isResolved isOutdated
        comments(first:1){nodes{author{login}}}
      }}
      commits(last:1){nodes{commit{statusCheckRollup{state}}}}
    }
  }
}`;

export async function fetchPullRequest(github, owner, repo, number) {
  const data = await github.graphql(PR_QUERY, { owner, repo, number });
  const pr = data.repository.pullRequest;
  if (!pr) return null;
  return {
    number: pr.number,
    isDraft: pr.isDraft,
    labels: pr.labels.nodes.map((l) => l.name),
    assignees: pr.assignees.nodes.map((a) => a.login),
    reviews: pr.latestReviews.nodes.map((r) => ({
      author: r.author ? r.author.login : null,
      state: r.state,
    })),
    threads: pr.reviewThreads.nodes.map((t) => ({
      author: t.comments.nodes[0] ? t.comments.nodes[0].author?.login ?? null : null,
      isResolved: t.isResolved,
      isOutdated: t.isOutdated,
    })),
    ciState: pr.commits.nodes[0]?.commit?.statusCheckRollup?.state ?? null,
  };
}

async function syncLabels(github, owner, repo, number, current, desired) {
  const stale = current.filter((l) => ALL_STATES.includes(l) && l !== desired);
  for (const name of stale) {
    await github.rest.issues.removeLabel({ owner, repo, issue_number: number, name })
      .catch(() => {});
  }
  if (!current.includes(desired)) {
    await github.rest.issues.addLabels({
      owner, repo, issue_number: number, labels: [desired],
    });
  }
  return { added: current.includes(desired) ? null : desired, removed: stale };
}

async function syncComment(github, owner, repo, number, result) {
  const comments = await github.paginate(github.rest.issues.listComments, {
    owner, repo, issue_number: number, per_page: 100,
  });
  const existing = comments.find((c) => c.body && c.body.includes(COMMENT_MARKER));
  const gated = result.state === STATE.BOT_FINDINGS_OPEN;

  // Never open the conversation unprompted: the comment appears only when
  // there is something to act on, which is also why it cannot post before
  // the async bot review has produced findings.
  if (!gated && !existing) return 'none';

  const body = gated ? gateComment(result.botFindingsOpen) : clearedComment(result.state);
  if (existing) {
    if (existing.body.trim() === body.trim()) return 'unchanged';
    await github.rest.issues.updateComment({ owner, repo, comment_id: existing.id, body });
    return 'updated';
  }
  await github.rest.issues.createComment({ owner, repo, issue_number: number, body });
  return 'created';
}

/**
 * Entry point. Evaluates every pull request number in `numbers` and reconciles
 * its label and guidance comment to the derived state.
 */
export default async function run({ core, github, context, numbers }) {
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  // Fail loudly rather than silently mislabelling: a missing label would make
  // the board look calm while pull requests pile up unrouted.
  const repoLabels = await github.paginate(github.rest.issues.listLabelsForRepo, {
    owner, repo, per_page: 100,
  });
  const known = new Set(repoLabels.map((l) => l.name));
  const missing = ALL_STATES.filter((l) => !known.has(l));
  if (missing.length > 0) {
    core.setFailed(
      `Missing required labels in ${owner}/${repo}: ${missing.join(', ')}. ` +
      'Create them before enabling this workflow.',
    );
    return;
  }

  for (const number of numbers) {
    const pr = await fetchPullRequest(github, owner, repo, number);
    if (!pr) {
      core.info(`PR #${number}: not found, skipping`);
      continue;
    }
    const result = classify(pr);
    const labelChange = await syncLabels(github, owner, repo, number, pr.labels, result.state);
    const commentAction = await syncComment(github, owner, repo, number, result);
    core.info(
      `PR #${number}: ${result.state} ` +
      `(approvals ${result.approvals}/${result.required}, ` +
      `bot findings ${result.botFindingsOpen}, ci ${pr.ciState ?? 'none'}, ` +
      `draft ${pr.isDraft}) ` +
      `labels[+${labelChange.added ?? '-'} -${labelChange.removed.join(',') || '-'}] ` +
      `comment[${commentAction}]`,
    );
  }
}
