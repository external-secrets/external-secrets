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
  'github-advanced-security',
  'copilot-pull-request-reviewer',
  'dependabot',
  'eso-service-account-app',
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

// Review states that express a verdict. COMMENTED and PENDING do not, so they
// must never overwrite one: that is the difference between this and GitHub's
// latestReviews field, which returns the last SUBMISSION per author and so
// silently loses a changes-requested as soon as its author comments again.
const VERDICTS = new Set(['APPROVED', 'CHANGES_REQUESTED', 'DISMISSED']);

// A crashed runner reports ERROR, not FAILURE. Both mean CI is not green.
const CI_RED_STATES = new Set(['FAILURE', 'ERROR']);

const COMMENT_MARKER = '<!-- eso-review-routing -->';

export function isBot(login) {
  return BOTS.has(login) || (typeof login === 'string' && login.endsWith('[bot]'));
}

/**
 * Each reviewer's standing verdict: their most recent APPROVED,
 * CHANGES_REQUESTED or DISMISSED, ignoring anything that expressed no verdict.
 */
export function effectiveVerdicts(reviews, { excludeAuthor } = {}) {
  const byAuthor = new Map();
  const ordered = [...reviews].sort(
    (a, b) => String(a.submittedAt || '').localeCompare(String(b.submittedAt || '')),
  );
  for (const r of ordered) {
    const login = r.author;
    if (!login || isBot(login) || login === excludeAuthor) continue;
    if (!VERDICTS.has(r.state)) continue;
    byAuthor.set(login, r.state);
  }
  return byAuthor;
}

/**
 * Decide the review state. Ordered evaluation, first match wins: conditions
 * overlap constantly, so the order below is the specification, not a detail.
 */
export function classify(pr) {
  const labels = new Set(pr.labels || []);
  const verdicts = effectiveVerdicts(pr.reviews || [], { excludeAuthor: pr.author });
  const approvals = [...verdicts.values()].filter((v) => v === 'APPROVED').length;
  const required = [...LARGE_LABELS].some((l) => labels.has(l)) ? 2 : 1;

  // Trust GitHub's own reviewDecision when it says changes are requested, and
  // fall back to the per-author verdicts so the signal survives a repo where
  // reviewDecision is null (no review policy configured).
  const changesRequested = pr.reviewDecision === 'CHANGES_REQUESTED'
    || [...verdicts.values()].includes('CHANGES_REQUESTED');

  // Only bot threads that are unresolved AND still current gate the pull
  // request. An outdated thread means the author already pushed over the
  // flagged line, so counting it would trap someone who did respond.
  const botFindingsOpen = (pr.threads || []).filter(
    (t) => isBot(t.author) && !t.isResolved && !t.isOutdated,
  ).length;

  // The author commenting on their own diff is not someone else reviewing it.
  const others = (pr.reviews || []).filter(
    (r) => r.author && !isBot(r.author) && r.author !== pr.author,
  );
  const humanEngaged = others.length > 0 || (pr.assignees || []).length > 0;

  const verdict = (state) => ({ state, botFindingsOpen, approvals, required });

  if (pr.isDraft) return verdict(STATE.DRAFT);
  if (changesRequested) return verdict(STATE.CHANGES_REQUESTED);
  if (CI_RED_STATES.has(pr.ciState)) return verdict(STATE.CI_RED);
  if (approvals >= required) return verdict(STATE.READY_TO_MERGE);
  if (approvals >= 1) return verdict(STATE.NEEDS_2ND_APPROVAL);

  // The gate sits below IN_REVIEW on purpose. Once a human is engaged their
  // judgement supersedes the bot, and a card must not be pulled out from
  // under a reviewer mid-read by a fresh nitpick.
  if (humanEngaged) return verdict(STATE.IN_REVIEW);
  if (botFindingsOpen > 0 && !labels.has(OVERRIDE_LABEL)) {
    return verdict(STATE.BOT_FINDINGS_OPEN);
  }
  return verdict(STATE.NEEDS_REVIEW);
}

export function gateComment(count) {
  const items = count === 1 ? '**1 open item**' : `**${count} open items**`;
  return [
    COMMENT_MARKER,
    `Automated review has flagged ${items} on this pull request.`,
    '',
    'Review here runs in two stages: automated review first, then a maintainer.',
    'This pull request moves into the human review queue once every automated',
    'comment above is resolved.',
    '',
    'Push a fix, or reply in the thread if you disagree with a finding, then mark it',
    '**Resolve conversation**. Resolving is what moves this along: a reply on its own',
    'leaves the thread open and the pull request here.',
    '',
    'You are not expected to change code you disagree with. If a finding is wrong and',
    'you would rather a maintainer decided, say so and ask for the',
    '`review/bots-overridden` label, which skips this stage entirely.',
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

// reviewThreads takes `last` so truncation drops the OLDEST threads. Taking
// `first` would drop the newest, which is exactly what a freshness-sensitive
// gate needs to see.
const PR_QUERY = `
query($owner:String!, $repo:String!, $number:Int!) {
  repository(owner:$owner, name:$repo) {
    pullRequest(number:$number) {
      number isDraft reviewDecision
      author{login}
      labels(first:100){nodes{name}}
      assignees(first:10){nodes{login}}
      reviews(first:100){
        totalCount
        pageInfo{hasNextPage}
        nodes{author{login} state submittedAt}
      }
      reviewThreads(last:100){
        totalCount
        pageInfo{hasPreviousPage}
        nodes{
          isResolved isOutdated
          comments(first:1){nodes{author{login}}}
        }
      }
      commits(last:1){nodes{commit{statusCheckRollup{state}}}}
    }
  }
}`;

export async function fetchPullRequest(github, owner, repo, number, core) {
  const data = await github.graphql(PR_QUERY, { owner, repo, number });
  const pr = data.repository && data.repository.pullRequest;
  if (!pr) return null;

  // Truncated data would derive a state from an incomplete picture, so the
  // caller must skip the pull request rather than write a label it cannot
  // stand behind.
  const truncated = [];
  if (pr.reviews.pageInfo.hasNextPage) truncated.push('reviews');
  if (pr.reviewThreads.pageInfo.hasPreviousPage) truncated.push('reviewThreads');
  if (truncated.length > 0 && core) {
    core.warning(
      `PR #${number}: more than 100 ${truncated.join(' and ')}, skipping rather than `
      + 'deriving a state from incomplete data',
    );
  }

  return {
    truncated: truncated.length > 0 ? truncated : null,
    number: pr.number,
    isDraft: pr.isDraft,
    reviewDecision: pr.reviewDecision,
    author: pr.author ? pr.author.login : null,
    labels: pr.labels.nodes.map((l) => l.name),
    assignees: pr.assignees.nodes.map((a) => a.login),
    reviews: pr.reviews.nodes.map((r) => ({
      author: r.author ? r.author.login : null,
      state: r.state,
      submittedAt: r.submittedAt,
    })),
    threads: pr.reviewThreads.nodes.map((t) => ({
      author: t.comments.nodes[0] ? (t.comments.nodes[0].author || {}).login ?? null : null,
      isResolved: t.isResolved,
      isOutdated: t.isOutdated,
    })),
    ciState: (((pr.commits.nodes[0] || {}).commit || {}).statusCheckRollup || {}).state ?? null,
  };
}

export async function syncLabels(github, owner, repo, number, current, desired, { dryRun } = {}) {
  const readLive = async () => {
    const fresh = await github.rest.issues.listLabelsOnIssue({
      owner, repo, issue_number: number, per_page: 100,
    });
    return fresh.data.map((l) => l.name);
  };

  // A dry run reads live as well: reporting from the classification-time
  // snapshot would describe a pull request that may already have moved on,
  // which is exactly what the live path re-reads to avoid.
  if (dryRun) {
    const live = await readLive();
    return {
      added: live.includes(desired) ? null : desired,
      removed: live.filter((l) => ALL_STATES.includes(l) && l !== desired),
      restored: false,
      applied: false,
    };
  }

  const removeStale = async (labels) => {
    const stale = labels.filter((l) => ALL_STATES.includes(l) && l !== desired);
    for (const name of stale) {
      try {
        await github.rest.issues.removeLabel({ owner, repo, issue_number: number, name });
      } catch (error) {
        // A label already gone is fine. Anything else, a permissions problem or
        // a rate limit, must surface: swallowing it leaves stale labels behind
        // while the run still reports success.
        if (error.status !== 404) throw error;
      }
    }
    return stale;
  };

  // Read live rather than trusting the classification-time snapshot: another
  // run may have relabelled this pull request since we classified it.
  const live = await readLive();
  const removed = await removeStale(live);
  const missing = live.includes(desired) ? null : desired;
  if (missing) {
    await github.rest.issues.addLabels({
      owner, repo, issue_number: number, labels: [desired],
    });
  }

  // The write is not atomic and cannot be: a sweep and an event-driven run for
  // the same pull request sit in different concurrency groups by design.
  //
  // This pass therefore only ever ADDS. An earlier version also removed labels
  // it did not expect, which converged two racing runs to ZERO labels: each
  // stripped the other's state and neither re-added its own, so the pull
  // request silently left the queue. Two labels for one cycle is visible and
  // self-corrects on the next run; none is invisible.
  const after = await readLive();
  const restored = !after.includes(desired);
  if (restored) {
    await github.rest.issues.addLabels({
      owner, repo, issue_number: number, labels: [desired],
    });
  }

  return {
    added: missing || (restored ? desired : null),
    removed,
    restored,
    applied: true,
  };
}

export async function syncComment(github, owner, repo, number, result, { dryRun } = {}) {
  // Require the marker AND a bot author. The marker is public, so a human
  // could otherwise post one and capture the slot, leaving this workflow
  // trying to edit a comment it does not own and the guidance never shown.
  const findMarkers = async () => {
    const comments = await github.paginate(github.rest.issues.listComments, {
      owner, repo, issue_number: number, per_page: 100,
    });
    return comments.filter(
      (c) => c.body
        && c.body.includes(COMMENT_MARKER)
        && c.user
        && c.user.type === 'Bot',
    );
  };

  const markers = await findMarkers();
  const existing = markers[0];
  const gated = result.state === STATE.BOT_FINDINGS_OPEN;

  // Never open the conversation unprompted: the comment appears only when
  // there is something to act on, which is also why it cannot post before
  // the async bot review has produced findings.
  if (!gated && !existing) return 'none';

  const body = gated ? gateComment(result.botFindingsOpen) : clearedComment(result.state);
  if (existing) {
    if (existing.body.trim() === body.trim()) return 'unchanged';
    if (dryRun) return 'would-update';
    await github.rest.issues.updateComment({ owner, repo, comment_id: existing.id, body });
    return 'updated';
  }
  if (dryRun) return 'would-create';

  const created = await github.rest.issues.createComment({
    owner, repo, issue_number: number, body,
  });

  // Creating is the one write here that cannot be made idempotent by reading
  // first: a concurrent run may create its own between our read and our write.
  // Keep the earliest and drop ours, so a pull request never accumulates two
  // guidance comments that nothing would ever clean up.
  const after = await findMarkers();
  if (after.length > 1) {
    const earliest = after.reduce((a, b) => (a.id < b.id ? a : b));
    const mine = created && created.data && created.data.id;
    if (mine && mine !== earliest.id) {
      await github.rest.issues.deleteComment({ owner, repo, comment_id: mine });
      return 'created-then-deduped';
    }
  }
  return 'created';
}

/**
 * End-of-sweep deconfliction.
 *
 * A sweep and an event-driven run for one pull request sit in different
 * concurrency groups by design, so both can write and leave two states. Only
 * the sweep cleans that up, and only ever by withdrawing the label it applied
 * itself. That asymmetry is the point: when both sides cleaned up, each
 * removed the other's state and neither re-added its own, leaving the pull
 * request with none at all.
 *
 * The event's state wins, which is the right precedence: it was raised by an
 * actual change, while a sweep is a backstop.
 */
export async function deconflictSweep(github, owner, repo, applied, core) {
  const outcomes = [];
  for (const { number, label } of applied) {
    const read = async () => {
      const r = await github.rest.issues.listLabelsOnIssue({
        owner, repo, issue_number: number, per_page: 100,
      });
      return r.data.map((l) => l.name).filter((n) => ALL_STATES.includes(n));
    };

    const live = await read();
    // Nothing to yield unless an overlapping run left its own state next to ours.
    if (live.length < 2 || !live.includes(label)) continue;

    try {
      await github.rest.issues.removeLabel({ owner, repo, issue_number: number, name: label });
    } catch (error) {
      if (error.status !== 404) throw error;
    }

    // A sweep must never be the reason a pull request carries no state at all.
    const after = await read();
    if (after.length === 0) {
      await github.rest.issues.addLabels({
        owner, repo, issue_number: number, labels: [label],
      });
      core.warning(`PR #${number}: withdrew ${label} but nothing remained, put it back`);
      outcomes.push({ number, label, action: 'restored' });
    } else {
      core.info(`PR #${number}: withdrew sweep state ${label}, event state ${after.join(',')} stands`);
      outcomes.push({ number, label, action: 'yielded' });
    }
  }
  return outcomes;
}

/**
 * Evaluates every pull request in `numbers` and reconciles its label and
 * guidance comment. With `dryRun` it only reports what it would change.
 */
export default async function run({
  core, github, context, numbers, dryRun = false, sweep = false,
}) {
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
    const message = `Missing required labels in ${owner}/${repo}: ${missing.join(', ')}. `
      + 'Create them before enabling this workflow.';
    if (dryRun) core.warning(message);
    else { core.setFailed(message); return []; }
  }

  if (dryRun) core.info('DRY RUN: no labels or comments will be written');

  // One bad pull request must not abandon the rest of a sweep, but it must
  // still turn the run red rather than passing quietly.
  const results = [];
  const failures = [];
  const skipped = [];
  const applied = [];
  for (const number of numbers) {
    try {
      const pr = await fetchPullRequest(github, owner, repo, number, core);
      if (!pr) {
        core.info(`PR #${number}: not found, skipping`);
        continue;
      }
      if (pr.truncated) {
        skipped.push(number);
        continue;
      }
      const result = classify(pr);
      const labelChange = await syncLabels(
        github, owner, repo, number, pr.labels, result.state, { dryRun },
      );
      const commentAction = await syncComment(github, owner, repo, number, result, { dryRun });
      results.push({ number, ...result, labelChange, commentAction, current: pr.labels });
      if (labelChange.added) applied.push({ number, label: labelChange.added });
      core.info(
        `PR #${number}: ${result.state} `
        + `(approvals ${result.approvals}/${result.required}, `
        + `bot findings ${result.botFindingsOpen}, ci ${pr.ciState ?? 'none'}, `
        + `decision ${pr.reviewDecision ?? 'none'}, draft ${pr.isDraft}) `
        + `labels[+${labelChange.added ?? '-'} -${labelChange.removed.join(',') || '-'}] `
        + (labelChange.restored ? 'RACED(label was deleted under us, restored) ' : '')
        + `comment[${commentAction}]`,
      );
    } catch (error) {
      failures.push(number);
      core.error(`PR #${number}: ${error.message}`);
    }
  }
  // Deferred to the end of the cycle on purpose: by now the event-driven runs
  // that overlapped these pull requests have almost certainly finished, so this
  // sees settled state instead of racing what it is trying to repair.
  if (sweep && !dryRun && applied.length > 0) {
    const outcomes = await deconflictSweep(github, owner, repo, applied, core);
    if (outcomes.length > 0) {
      core.info(`deconflicted ${outcomes.length} pull request(s) after overlapping writes`);
    }
  }

  if (skipped.length > 0) {
    core.warning(`Skipped ${skipped.length} pull request(s) with truncated data: ${skipped.join(', ')}`);
  }
  if (failures.length > 0) {
    core.setFailed(`Failed to evaluate ${failures.length} pull request(s): ${failures.join(', ')}`);
  }
  return results;
}
