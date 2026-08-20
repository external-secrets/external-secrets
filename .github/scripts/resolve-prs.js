/**
 * Works out which pull requests a Review State run should evaluate.
 *
 * Split out of the workflow so the fork handling below can be tested: getting
 * it wrong fails silently, as an empty list rather than an error.
 */

// Events that name their pull request directly in the payload.
const DIRECT_EVENTS = new Set([
  'pull_request_target',
  'pull_request_review',
  'pull_request_review_comment',
  'pull_request',
]);

// Events that identify a pull request only by the commit they ran against.
const SHA_EVENTS = new Set(['check_suite', 'check_run', 'status']);

async function listOpenPullRequests(github, owner, repo) {
  return github.paginate(github.rest.pulls.list, {
    owner, repo, state: 'open', per_page: 100,
  });
}

function headShaFor(event, payload) {
  if (event === 'check_suite') return (payload.check_suite || {}).head_sha;
  if (event === 'check_run') return ((payload.check_run || {}).check_suite || {}).head_sha;
  if (event === 'status') return payload.sha;
  return undefined;
}

/**
 * Maps a head commit back to its open pull requests by matching against the
 * open pull request list.
 *
 * The obvious call, listPullRequestsAssociatedWithCommit, returns an empty
 * array for pull requests from forks, which is most of our traffic, and gives
 * no error to distinguish that from "no such pull request". Matching head SHAs
 * has no fork special case.
 */
export async function pullRequestsForSha(github, owner, repo, sha) {
  const open = await listOpenPullRequests(github, owner, repo);
  return open.filter((p) => p.head && p.head.sha === sha).map((p) => p.number);
}

/**
 * Maps a workflow_run back to its pull request by head repository and branch.
 *
 * Not by head SHA: for review events GITHUB_SHA is the merge commit rather
 * than the pull request head, so the SHA would not match. A head branch is
 * unique per repository, so the pair identifies exactly one open pull request.
 */
export async function pullRequestsForBranch(github, owner, repo, headRepo, headRef) {
  const open = await listOpenPullRequests(github, owner, repo);
  return open
    .filter((p) => p.head
      && p.head.ref === headRef
      && p.head.repo
      && p.head.repo.full_name === headRepo)
    .map((p) => p.number);
}

export default async function resolvePullRequests({ core, github, context }) {
  const owner = context.repo.owner;
  const repo = context.repo.repo;
  const event = context.eventName;
  const payload = context.payload || {};

  if (DIRECT_EVENTS.has(event)) {
    const number = payload.pull_request && payload.pull_request.number;
    if (!number) {
      core.warning(`${event} carried no pull request number`);
      return [];
    }
    return [number];
  }

  if (event === 'workflow_run') {
    const wr = payload.workflow_run || {};
    const headRepo = (wr.head_repository || {}).full_name;
    if (!wr.head_branch || !headRepo) {
      core.warning('workflow_run carried no head branch or head repository');
      return [];
    }
    const numbers = await pullRequestsForBranch(github, owner, repo, headRepo, wr.head_branch);
    if (numbers.length === 0) {
      core.info(`workflow_run: ${headRepo}:${wr.head_branch} matches no open pull request`);
    }
    return numbers;
  }

  if (SHA_EVENTS.has(event)) {
    const sha = headShaFor(event, payload);
    if (!sha) {
      core.warning(`${event} carried no head SHA`);
      return [];
    }
    const numbers = await pullRequestsForSha(github, owner, repo, sha);
    if (numbers.length === 0) {
      core.info(`${event}: head SHA ${sha} matches no open pull request`);
    }
    return numbers;
  }

  const requested = ((payload.inputs && payload.inputs.pr) || '').trim();
  if (requested) {
    // Reject anything that is not a plain pull request number rather than
    // querying NaN and reporting a confusing not-found.
    if (!/^\d+$/.test(requested)) {
      throw new Error(`Invalid pr input: ${JSON.stringify(requested)}. Expected a number.`);
    }
    return [Number(requested)];
  }

  const open = await listOpenPullRequests(github, owner, repo);
  return open.map((p) => p.number);
}
