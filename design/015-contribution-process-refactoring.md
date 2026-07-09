# Proposal: `ready-for-review` label workflow

## Problem

The PR list is noisy. Maintainers cannot tell at a glance which PRs are actually
ready for review versus draft, WIP, or still being iterated on after feedback.
This wastes reviewer time and makes it hard to prioritize the queue. The rise
of AI-assisted editing has also sharply increased commit volume, often producing
many small follow-up commits for individual inline comments, which further
amplifies the noise.

## Goal

Introduce a `ready-for-review` label that is only present on a PR when the
author has explicitly signalled it is ready and no state has changed since
that signal. Maintainers can then filter the PR list to `label:ready-for-review`
and trust that every result deserves attention.

## Mechanics

### 1. Author signals readiness

Author comments `/ready-for-review` on its own line in a PR comment. A single
workflow job (`add-label`) listens for PR comments, checks for an exact-line
match (not just a substring, so passing mentions of the command elsewhere in
a comment don't trigger it), verifies the commenter is the PR author or a
maintainer, and either applies the `ready-for-review` label or comments back
explaining why the command was rejected.

### 2. Label removal triggers

The label is removed automatically whenever the PR's state changes in a way
that invalidates the "ready" signal. In every case, the only action taken is
removing the `ready-for-review` label; the automation never changes the PR's
own draft/open state:

- A new commit is pushed to the PR branch (`synchronize` event).
- A maintainer submits a review with `REQUEST_CHANGES` or a `COMMENT` review
  containing unresolved feedback (treat `CHANGES_REQUESTED` as the reliable
  signal; treat plain comment reviews as optional/configurable).
- Optionally: the PR goes stale past a configurable number of days.

Each of these is a separate trigger condition in the same workflow, all
converging on "remove label if present." Running removal in the same
workflow as the gate check (below) is what lets removal finish before the
gate re-evaluates, rather than racing it.

### 3. Gate check

A job in the same workflow (`check-label`) runs on every relevant
`pull_request_target` event (push, open, reopen, label change) and verifies
the PR carries `ready-for-review`:

- Draft PRs are skipped entirely. GitHub's own draft state already signals
  "not ready," so the gate doesn't pile a second signal on top of it.
- If the label is present: the job passes.
- If the label is absent: the job fails via `core.setFailed`. This does not
  post a PR comment; the explanation (what to run, and a link to docs) shows
  up as the failure message on the check itself and in the workflow run log,
  which is enough of a pointer without adding another comment to the thread.

Because `check-label` declares `needs: [remove-label]` and both run within
the same `pull_request_target` event, `check-label` always executes after
`remove-label` completes on a push, and it re-fetches the PR's labels from
the API rather than trusting the event payload. This removes the race that
existed when the two lived in separate workflow files: previously a push
could trigger the gate and the label removal in parallel, with no guarantee
the gate would see the post-removal state.

Adding the label (via `add-label`) or removing it (via `remove-label` after a
change-requested review) each fire their own `labeled`/`unlabeled` event,
which re-triggers this same workflow and re-runs `check-label` on the
current commit, so the check reflects reality without waiting for a new
push.

This failing check is what actually reduces noise for maintainers, since it
can be made a required check that keeps the PR out of default review views
until the author opts in, without blocking merge mechanics for drafts.

## Implementation sketch

### Combined workflow (`.github/workflows/ready-for-review.yml`)

```yaml
name: PR readiness

on:
  issue_comment:
    types: [created]
  pull_request_review:
    types: [submitted]
  pull_request_target:
    types: [synchronize, opened, reopened, labeled, unlabeled]

permissions:
  pull-requests: write
  issues: write

jobs:
  add-label:
    if: >
      github.event_name == 'issue_comment' &&
      github.event.issue.pull_request &&
      contains(github.event.comment.body, '/ready-for-review')
    runs-on: ubuntu-latest
    steps:
      - name: Handle /ready-for-review command
        uses: actions/github-script@v7
        with:
          script: |
            const body = context.payload.comment.body || '';
            const isCommand = /^\s*\/ready-for-review\s*$/m.test(body);
            if (!isCommand) {
              core.info('Comment does not contain the /ready-for-review command on its own line, skipping.');
              return;
            }

            const commenter = context.payload.comment.user.login;
            const pr = await github.rest.pulls.get({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.issue.number,
            });
            const isAuthor = commenter === pr.data.user.login;
            const perms = await github.rest.repos.getCollaboratorPermissionLevel({
              owner: context.repo.owner,
              repo: context.repo.repo,
              username: commenter,
            });
            const isMaintainer = ['admin', 'write'].includes(perms.data.permission);

            if (!isAuthor && !isMaintainer) {
              await github.rest.issues.createComment({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: context.issue.number,
                body: `@${commenter} only the PR author or a maintainer can run \`/ready-for-review\`.`,
              });
              core.setFailed(`${commenter} is not authorized to run /ready-for-review`);
              return;
            }

            await github.rest.issues.addLabels({
              owner: context.repo.owner,
              repo: context.repo.repo,
              issue_number: context.issue.number,
              labels: ['ready-for-review'],
            });

  remove-label:
    if: >
      (github.event_name == 'pull_request_target' &&
        github.event.action == 'synchronize') ||
      (github.event_name == 'pull_request_review' &&
        github.event.review.state == 'changes_requested')
    runs-on: ubuntu-latest
    steps:
      - name: Remove label
        uses: actions/github-script@v7
        with:
          script: |
            try {
              await github.rest.issues.removeLabel({
                owner: context.repo.owner,
                repo: context.repo.repo,
                issue_number: context.issue.number,
                name: 'ready-for-review',
              });
            } catch (e) {
              if (e.status !== 404) throw e; // label already absent, fine
            }

  check-label:
    name: Verify ready-for-review label
    needs: [remove-label]
    if: >
      always() &&
      github.event_name == 'pull_request_target' &&
      github.event.pull_request.draft == false
    runs-on: ubuntu-latest
    steps:
      - name: Verify ready-for-review label
        uses: actions/github-script@v7
        with:
          script: |
            // Re-fetch fresh rather than trusting the event payload, since
            // the remove-label job may have just changed the label set.
            const { data: pr } = await github.rest.pulls.get({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: context.payload.pull_request.number,
            });
            const labels = pr.labels.map(l => l.name);
            if (!labels.includes('ready-for-review')) {
              core.setFailed(
                'This PR is not marked ready for review. Comment `/ready-for-review` once it is ready. See LINK_TO_DOCS for the contributor guide.'
              );
            }
```

Notes on the sketch:

- `pull_request_target` is used for the whole workflow, not just label
  mutation, so `add-label`, `remove-label`, and `check-label` all run with
  full base-repo permissions and share one event context, even for PRs from
  forks. None of the jobs check out PR code, so this remains safe.
- The `remove-label` job only ever calls `removeLabel`. It does not touch
  the PR's draft/ready state, mergeability, or any other field.
- `check-label` re-fetches labels via `pulls.get` instead of using
  `context.payload.pull_request.labels`, so it reflects `remove-label`'s
  outcome rather than the stale state from when the event fired.
- `check-label` is skipped outright when `pull_request.draft == true`.
- `check-label` never comments; its failure message is visible on the check
  itself and in the run log. `add-label` does comment, but only once, on a
  rejected command, not repeatedly across pushes, so no deduplication logic
  is needed there.
- Consider making `check-label` a required status check in branch protection
  so it visibly blocks merge until addressed, or leave it non-blocking if
  the intent is purely list hygiene rather than merge gating.
- `check-label` is given an explicit `name:` so its check run reads as
  "Verify ready-for-review label" rather than the raw job id. This is what
  makes the required-check string in branch protection predictable and
  readable; see the Rollout plan for the exact context string.

## Edge cases to decide before implementation

| Case | Options |
|---|---|
| Maintainer wants to force-mark a PR ready on author's behalf | Allow maintainers to run `/ready-for-review` too (sketch above already allows this) |
| Author pushes a trivial fix (typo, rebase) after being marked ready | Accept as a cost of simplicity, or add a `/ready-for-review` alias like a `--skip` flag for trivial pushes (adds complexity, probably not worth it initially) |
| Bots/automated pushes (e.g. Dependabot rebase) | Consider excluding pushes from known bot actors from label removal |
| Comment reviews without changes-requested state | Decide whether to treat these as removal triggers; recommend starting with only `CHANGES_REQUESTED` to avoid over-triggering |
| Stale ready PRs | Optional stale-bot integration to remove label after N days of inactivity |

## Rollout plan

1. Ship the workflow behind a feature flag or on a single pilot repo first.
2. Update `CONTRIBUTING.md` with the `/ready-for-review` instructions and a
   short rationale, and link it from the gate check's failure message.
3. Announce the change in the repo's discussion/announcements channel with a
   grace period before making the gate check required.
4. Monitor false-positive label removals (e.g. bot pushes, non-substantive
   reviews) for the first couple of weeks and tune the removal triggers.
5. Once stable, make the gate check a required status check in branch
   protection rules. For a single, non-reusable workflow the context string
   GitHub reports is `<workflow name> / <job name>`, so this should appear
   as `PR readiness / Verify ready-for-review label`. GitHub only lists
   checks in the branch protection picker once they've actually run against
   the target branch, so trigger the workflow once (e.g. merge a pilot PR
   or push a no-op commit) before trying to select it, and pick it from the
   dropdown rather than typing it from memory since the match must be
   exact.

## Open questions for the maintainer group

- Should the gate check block merge, or only serve as a visible signal?
- Should reviews from any maintainer count, or only from a designated
  CODEOWNERS subset?
- Do we want a stale-label auto-removal policy, and if so what timeout?
- Should inline review comments (`pull_request_review_comment`), separate
  from a formally submitted review, also trigger label removal, or is a
  submitted `CHANGES_REQUESTED` review sufficient on its own?

## Related patterns in other projects

Most large CNCF projects implement this kind of thing through Prow, the
Kubernetes-originated CI/chatops bot, rather than plain GitHub Actions —
heavier infrastructure than what's proposed here, but it validates the
underlying pattern:

- **Kubernetes contributor docs** — a `[WIP]` title prefix or a `/hold`
  comment each drive a `do-not-merge/*` label that blocks merge automation
  until removed. This is the reverse polarity of this proposal (default
  mergeable, explicit block, rather than default blocked, explicit ready),
  but it's the same idea of a bot-managed label as the accurate signal:
  https://www.kubernetes.dev/docs/guide/pull-requests/
- **Prow's `lgtm` plugin** — the closest direct analog to our
  removal-on-push trigger. A reviewer comments `/lgtm`, Prow adds the
  `lgtm` label, and a new commit push automatically strips it again so the
  PR has to be re-reviewed:
  https://docs.prow.k8s.io/docs/components/plugins/lgtm/
- **The original design discussion for that behavior** — kubernetes/test-infra#3795
  walks through adding "strip the label on synchronize" to the `lgtm`
  plugin, including the same false-positive/spam tradeoffs called out in
  the edge cases table above:
  https://github.com/kubernetes/test-infra/issues/3795
- **containerd** (a CNCF graduated project) — runs the Prow `wip` plugin on
  its own repo, so this isn't just a kubernetes/kubernetes-specific
  convention: https://prow.k8s.io/plugin-config
- **A plain GitHub Actions precedent (not CNCF)** — a lighter-weight version
  of "fail the check based on label state," built without Prow, closer in
  spirit to the implementation sketched above:
  https://bgenc.com/2023.02.18.github-actions-do-not-merge-label/
