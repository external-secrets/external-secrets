```yaml
---
title: Review Routing for Maintainers
version: v1
authors: Alexander Chernov
creation-date: 2026-08-19
status: draft
---
```

# Review Routing for Maintainers

## Summary

Review state on `external-secrets/external-secrets` is not recorded anywhere you can
query. It exists only as prose in comment threads, so every maintainer who opens a pull
request pays the cost of reconstructing it: has a human looked at this, did they reach a
verdict, is someone already on it, does it need a second approval.

This proposal derives review state from facts GitHub already knows, writes it to a
single-select field on the Maintainer Queue board (org project #3), and groups a board
view by it. The result is a kanban that nobody has to move cards on. Everything is
computed; the only manual act is claiming a pull request by assigning yourself.

Review runs in two stages. Automated review is the first line of defence: while a bot has
unresolved findings, the pull request sits with its author and never reaches the human
queue. Contributors are told this in a comment on the pull request itself, posted only once
there is actually something to address.

## Motivation

### The problem, measured

Across the 78 open pull requests, formal review state looks like this:

| Review state                  | PRs |
| ----------------------------- | --: |
| Commented only, no verdict    |  47 |
| No review at all              |  15 |
| Changes requested             |  13 |
| Approved                      |   3 |

(Effective current review per reviewer, not cumulative review events. Buckets are
mutually exclusive, with changes-requested taking precedence over approved.)

That first row is the whole problem. Those 47 pull requests have activity but no answer,
so the reconstruction cost is paid again by each person who opens one.

Worse, the activity is mostly not human. Sorted by effective reviews on open pull
requests, the top reviewer is a bot:

| Reviewer                      | Reviews | Kind                    |
| ----------------------------- | ------: | ----------------------- |
| coderabbitai                  |      56 | bot                     |
| evrardj-roche                 |      29 | human, not a maintainer |
| alekc                         |      11 | human                   |
| Skarlso                       |      11 | maintainer              |
| gusfcarvalho                  |       6 | maintainer              |
| github-advanced-security      |       5 | bot                     |
| copilot-pull-request-reviewer |       2 | bot                     |
| moolen                        |       2 | maintainer              |
| knelasevero                   |       1 | maintainer              |

**24 of 78 open pull requests carry reviews from bots only.** In the GitHub UI they look
reviewed. No human has formed a view on any of them. Any signal that does not distinguish
bots from humans is worse than no signal, because it is confidently wrong.

Splitting those 24 by whether the bot still has outstanding findings is what makes them
actionable: **10 have unresolved bot findings** and belong with their author, while **14
are clean** and genuinely need a human. Today both groups look identical.

Meanwhile every native routing mechanism is switched off:

- 67 of 78 pull requests have no requested reviewer.
- Zero carry the `lgtm` label.
- 8 have an assignee, though `docs/contributing/process.md` names the assignee as the
  mechanism that tracks who owns a pull request's lifecycle.
- 60 of 78 have had no maintainer review at all.

### Root cause

Our CODEOWNERS file sits at a path GitHub does not read. Both copies are named
`CODEOWNERS.md`, and GitHub only honours `CODEOWNERS` with no extension at the repository
root, in `.github/`, or in `docs/`. None of those exist:

```
CODEOWNERS               absent
.github/CODEOWNERS       absent
docs/CODEOWNERS          absent
```

So all fifty-odd path-to-team mappings are inert: no automatic review requests, nothing in
anyone's "awaiting your review" filter, no per-team sign-off state in the sidebar. One
file extension explains why 67 of 78 pull requests have no reviewer attached.

The encouraging part is that the hard logic already exists.
`.github/scripts/lgtm-processor.js` parses that file, maps changed files to reviewer
teams, checks team membership, and computes exactly which areas are covered and which are
not. It then writes the answer into a prose comment and discards it, and only when
somebody types `/lgtm`. We have already built the expensive part. It just needs somewhere
durable to put the result.

### Goals

- Make "which pull request should I review next" answerable without reading comment threads.
- Make "has another maintainer already reviewed or claimed this" visible at a glance.
- Separate work that is genuinely ours from work waiting on the contributor.
- Add no step that depends on anyone remembering to do it.

### Non-Goals

- Enforcing review policy. This proposal changes visibility, not merge gating.
- Enabling branch protection or required reviews (see Alternatives).
- Reworking the `Maintainer Queue` priority classification, which is a separate concern.
- Any new service to host or keep alive.

## Proposal

Three layers, each independently useful. If a later layer stalls, the earlier ones keep
working.

### Layer 1: make CODEOWNERS real

Config only, one pull request. **Raised as #6853.** Moving the file to
`.github/CODEOWNERS` restores native review requests and the per-reviewer "awaiting your
review" filter.

A plain rename would not have worked, and this is worth knowing before reviewing that PR.
Precedence is last-match-wins and the project-wide `*` line sat at the bottom of the file,
so the moment GitHub started reading it the maintainers teams would have owned every path
and every area rule would have been dead. The default now comes first.

Auditing the file against the tree and the org team list turned up more:

- The two copies had already drifted. `.github/CODEOWNERS.md` carried the corrected
  `providers/v1/ovh/` path while the root copy still had `pkg/provider/v1/ovh/`, and
  `lgtm-processor.js` read the stale one. #6853 collapses to a single canonical file and
  retargets `lgtm.yml` and `lgtm-processor.js`, rather than keeping two that can drift.
- Five referenced reviewer teams do not exist (dvls, nebius, ngrok, ovh, volcengine).
  GitHub reports unknown owners as errors and ignores those lines, so a rename would have
  surfaced five syntax errors immediately. Those paths now fall through to
  `providers-reviewers`.
- `scripts/`, `build/` and `test/` do not exist in this repository at all.
- `docs-maintainers` and `provider-openbao-reviewers` already exist as teams with matching
  directories and were simply never wired up. Both are now mapped.
- `deploy/` gets no entry after all, because there is no charts reviewer team to point it
  at. Creating one is worth doing separately, given the chart review volume.

### Layer 2: compute review state into labels

One workflow. It evaluates the rules in the Behavior section, applies exactly one
`review/*` label, and maintains the contributor-facing comment described under Contributor
guidance. Labels rather than a board field alone, for two reasons: the state
becomes visible on the pull request itself and in ordinary GitHub search, and it keeps
working if project write access is ever unavailable.

Triggers: `pull_request_review`, `pull_request_review_comment`, `pull_request_target` on
`opened` and `synchronize`, and `check_suite` on `completed`. Thread resolution also fires
`pull_request_review_thread`, which is what makes the bot gate clear promptly. Reset-on-push
has a working precedent in `lgtm-remove-on-update.yml`.

### Layer 3: mirror the label to the board

The same workflow writes a new `Review` single-select field via
`updateProjectV2ItemFieldValue`, and a pull-request-only board view (`is:pr -is:closed`)
groups by it.

A dedicated field rather than reusing `Status`, because the board also carries 180 issues
for which review columns are meaningless, and because the built-in workflows already drive
`Status` (Todo on add, Done on close and merge) and would fight a custom pipeline.

**Prerequisite:** `GITHUB_TOKEN` cannot write org projects. The natural candidate is the
App already behind `LGTM_APP_ID` and `LGTM_PRIVATE_KEY`, but it needs
`organization_projects: write`. Someone with org admin needs to confirm or grant this.
Layers 1 and 2 do not depend on it.

### The board

Eight columns in two lanes. The lane split is the central claim of this proposal: more than
half of what currently sits in one undifferentiated list is not a maintainer's problem at
all.

Counts are the real distribution of today's 78 open pull requests under the rules below.

**Maintainer's queue, waiting on us: 33 PRs**

| Column                 | PRs | Entry condition                                                     |
| ---------------------- | --: | ------------------------------------------------------------------- |
| `📥 Needs Review`      |  17 | No human review yet, and automated review is clear. Front of the queue. |
| `👀 In Review`         |  14 | A human has engaged, or a maintainer self-assigned.                 |
| `🥈 Needs 2nd Approval`|   1 | One human approval on a `size/l` or larger change.                  |
| `🚀 Ready to Merge`    |   1 | Approval threshold met, CI green. Needs a merge, not a review.      |

**Author's court, waiting on the contributor: 45 PRs**

| Column                 | PRs | Entry condition                                                     |
| ---------------------- | --: | ------------------------------------------------------------------- |
| `🤖 Bot Findings Open` |   9 | Automated review has unresolved findings. First line of defence.    |
| `✋ Changes Requested` |  13 | A human asked for changes. Returns to the queue when author pushes. |
| `🔴 CI Red`            |  11 | Checks failing. Not worth a human read until they pass.             |
| `📝 Draft`             |  12 | Author has said it is not ready. We take them at their word.        |

Merging is the exit. The built-in "Pull request merged" workflow sets `Status` to Done,
and auto-archive removes the item after its two week grace window. That path is already
wired and needs no new work.

## Behavior

### Transition logic

Conditions overlap constantly. A draft can have failing CI and a changes-requested review
at the same time. So the rules are an **ordered evaluation with first match winning**, not
a set of independent tests. Order is the specification.

```
# definitions
BOTS      = {coderabbitai, github-advanced-security,
             copilot-pull-request-reviewer, dependabot,
             eso-service-account-app}
reviews   = latestReviews          # current effective review per person
human     = [r for r in reviews if r.author not in BOTS]
approvals = count(r.state == APPROVED for r in human)
required  = 2 if labels & {size/l, size/xl} else 1

# the bot gate: threads opened by a bot, still unresolved, still relevant.
# isOutdated means the author already pushed over the flagged line, so the
# finding may well be moot and must not trap the pull request.
botOpen   = count(t for t in reviewThreads
                  if t.firstComment.author in BOTS
                  and not t.isResolved
                  and not t.isOutdated)
override  = "review/bots-overridden" in labels

# ordered evaluation, first match wins
if isDraft                                 -> 📝 Draft
if any(r.state == CHANGES_REQUESTED)       -> ✋ Changes Requested
if checkRollup == FAILURE                  -> 🔴 CI Red
if approvals >= required                   -> 🚀 Ready to Merge
if approvals >= 1                          -> 🥈 Needs 2nd Approval
if human or assignees                      -> 👀 In Review
if botOpen > 0 and not override            -> 🤖 Bot Findings Open
else                                       -> 📥 Needs Review
```

### Why this order

- **Draft outranks everything.** The author has explicitly said it is not ready, and that
  statement should not be second-guessed by CI or by a stale review.
- **Changes Requested outranks CI Red.** Both put the ball in the author's court, so
  routing is unaffected either way, but "a human asked for changes" tells the author more
  than "checks are failing" and is the more useful label to carry.
- **Ready to Merge outranks Needs 2nd Approval.** Otherwise a single-approval change on a
  small pull request would sit in a column asking for a second reviewer it does not need.
- **The bot gate sits below `👀 In Review`, deliberately.** Once a human has engaged or
  claimed a pull request, human judgment supersedes the bot and the card must not be yanked
  out from under a reviewer mid-read. The gate only diverts pull requests nobody has touched
  yet, which is exactly what "should not move into needing a human review" means.
- **The gate ignores outdated threads.** Of 86 unresolved bot threads, 73 are still current
  and 13 are outdated. An outdated thread means the author already pushed over the flagged
  line, so counting those would trap pull requests whose authors had in fact responded.
- **Needs Review is the fallback,** not a computed state. Anything we cannot classify
  lands at the front of the queue rather than disappearing. Failing loudly beats failing
  silently. Note this means a bot that never runs cannot block anything: no threads, no
  gate, so a CodeRabbit outage degrades to today's behaviour rather than stalling the queue.

### What counts as a review

- **Bots gate, they never satisfy.** Automated review is the first line of defence: while a
  bot has unresolved current findings, the pull request sits in `🤖 Bot Findings Open` and
  never reaches the human queue. A bot approval, equally, never counts toward the approval
  threshold. Both halves matter: bots can hold a pull request back, and they can never wave
  one through.
- **Which bots actually gate.** Of 327 bot-opened threads on open pull requests,
  coderabbitai opened 301 (70 unresolved and current), github-advanced-security 21 (3), and
  copilot-pull-request-reviewer 5 (0). In practice this gate means "CodeRabbit is satisfied",
  which is worth saying out loud rather than pretending it is bot-agnostic.
- **Maintainers can override.** The `review/bots-overridden` label forces a pull request
  past the gate, for false positives, for a newcomer stuck on nitpicks, or for a bot
  outage. Consistent with the existing `/lgtm` dispatcher, this is worth exposing as a
  slash command too.
- **Effective state, not history.** Counting `latestReviews` rather than all reviews means
  an approval that the same person later superseded with a comment stops counting,
  automatically.
- **Two approvals for `size/l` and above,** one otherwise, matching the rule already
  written down in `process.md`.

### Contributor guidance

A gate nobody explains is just an unexplained delay, and 5 of the 9 pull requests this rule
would currently divert are from first-time contributors. So the workflow tells the author
what is happening, on the pull request itself.

**Timing solves itself.** The comment is posted when a pull request first enters
`🤖 Bot Findings Open`, which by definition cannot happen before automated review has
produced findings. There is no need to guess how long CodeRabbit takes, and no risk of
posting "wait for the bot" on a pull request the bot then passes clean. A contributor whose
change is clean never sees a gate message at all.

**One comment, edited, never repeated.** The workflow finds its own previous comment by a
hidden marker and edits it in place rather than posting again, so a pull request that cycles
through the gate three times still has exactly one comment. When the findings clear, the
same comment is rewritten to say so.

Draft text, in the gate state:

> Automated review has flagged **3 open items** on this pull request.
>
> Review here runs in two stages. Automated review comes first, then a maintainer. This pull
> request moves into the human review queue once the automated comments are addressed, either
> by pushing a fix or by replying in the thread.
>
> You can resolve a thread yourself with **Resolve conversation**. If you think a finding is
> wrong, say so in the thread and leave it open; a maintainer will judge it. You are not
> expected to change code you disagree with.
>
> Status: `🤖 Bot Findings Open`

And once it clears:

> Automated review is clear. This pull request is now in the human review queue.
>
> Status: `📥 Needs Review`

Expectations should also be set before a contributor opens a pull request at all, so a short
paragraph describing the two-stage flow belongs in `.github/pull_request_template.md` and in
`docs/contributing/process.md` alongside the existing review rules.

### Worked examples

| Situation                                                                  | Column                  | Why                                                        |
| -------------------------------------------------------------------------- | ----------------------- | ---------------------------------------------------------- |
| New provider PR, `size/l`, coderabbitai left 4 unresolved comments, CI green | `🤖 Bot Findings Open`  | Automated review is the first stage; author addresses these |
| Author resolves all 4 threads                                              | `📥 Needs Review`       | Gate clear, and the guidance comment is rewritten to say so |
| Author pushes a fix but forgets to resolve the threads                     | `📥 Needs Review`       | The threads go outdated, and outdated threads do not gate   |
| Same PR, Skarlso approves                                                  | `🥈 Needs 2nd Approval` | One human approval against a required two                  |
| Same PR, gusfcarvalho also approves                                        | `🚀 Ready to Merge`     | Threshold met, CI still green                              |
| Author pushes a new commit                                                 | recomputed              | See open question Q1 on approval staleness                 |
| `size/xs` docs typo fix, one approval                                      | `🚀 Ready to Merge`     | `required` is 1 below `size/l`                             |
| Draft with failing CI and an old changes-requested review                   | `📝 Draft`              | Draft wins the ordering                                    |
| Maintainer assigns themselves, has not reviewed yet                        | `👀 In Review`          | Assignment is the claim signal, so nobody duplicates work  |
| Maintainer already reviewing when CodeRabbit opens a new thread             | `👀 In Review`          | The gate sits below In Review, so nothing is yanked back    |
| CodeRabbit finding is a false positive, maintainer labels `review/bots-overridden` | `📥 Needs Review` | Override skips the gate without resolving anything falsely  |
| CodeRabbit never ran on the pull request                                    | `📥 Needs Review`       | No threads means no gate; an outage cannot stall the queue  |

### Automated versus manual

Worth being explicit, because this is the part that decides whether the system survives
contact with a busy month.

**Automated:** intake to the board, every column assignment, the transition on every
review and every push, the exit on merge, and archiving afterwards.

**Manual, two things.** Assigning yourself to claim a pull request, which is optional since
engaging with a review moves the card anyway, and exists only so two maintainers do not spend
the same evening on the same pull request. And applying `review/bots-overridden` when a bot
finding does not deserve to hold a change back.

Resolving bot threads is the contributor's, not ours, which is the point of the gate.

This is a deliberate constraint rather than a nice property. The `Maintainer Queue`
classification on this same board was real work by real maintainers, and it stopped in
July because it depended on people remembering. Nothing in this proposal depends on anyone
remembering anything.

## Drawbacks

- **Day one will look worse than today feels.** 26 pull requests land in
  `📥 Needs Review` and `🚀 Ready to Merge` holds one. That is not a regression, it is the
  existing backlog with the reconstruction cost stripped out. Anyone expecting the board
  to look reassuring will be disappointed.
- **Renaming CODEOWNERS produces a notification burst** as pull requests update and teams
  get requested. There is no blocking risk, since the repository has no rulesets and no
  branch protection, but inboxes will notice.
- **The bot list needs maintaining.** A new review bot silently counts as human coverage
  until someone adds it. Keeping the list in the workflow file rather than a secret makes
  the omission visible in review.
- **The gate makes CodeRabbit load-bearing.** It opened 301 of 327 bot threads, so in
  practice the human queue is fed by CodeRabbit's judgment. If it becomes noisy, the queue
  starves, and nobody would notice quickly because a starved queue looks like a calm one.
  Worth watching the size of `🤖 Bot Findings Open` as a signal in its own right.
- **It lands hardest on newcomers.** 5 of the 9 pull requests the gate currently diverts are
  from first-time contributors, and one open pull request carries 19 unresolved bot threads.
  The guidance comment and the override exist for this, but the honest read is that we are
  asking newcomers to clear a bot before a human engages. Open question Q4.
- **Authors can resolve their own threads.** GitHub permits it, which is what makes the gate
  actionable rather than a dead end. It also means an author can resolve without fixing to
  jump the queue. Acceptable, since a human still reviews afterwards and the thread history
  stays visible, but it means the gate is a routing hint and not an enforcement mechanism.
- **Two approvals is aspirational today.** 38 of 78 open pull requests are `size/l` or
  larger, and exactly one of those 38 carries a human approval. Encoding the rule will
  show a persistently populated `🥈 Needs 2nd Approval` column. That is honest, and it is
  also a standing argument for either resourcing the rule or relaxing it.
- **A third field on the board.** `Status`, `Maintainer Queue`, and now `Review`.
  Justified because they answer different questions (pipeline position, priority and
  category, review coverage) but it is more surface to keep coherent.

## Acceptance Criteria

- Layer 1 merged, and a subsequent pull request shows automatic team review requests.
- Layer 2 applies exactly one `review/*` label per open pull request, and the label
  changes within one workflow run of a review being submitted or a commit being pushed.
- Resolving the last open bot thread moves a pull request out of `🤖 Bot Findings Open`
  without any human action, and the guidance comment is rewritten rather than duplicated.
- A pull request with no bot activity at all is never gated.
- Layer 3 has the `Review` field populated for every open pull request on the board, with
  the pull-request-only view grouped by it.
- **Rollback:** each layer is independently revertable. Layer 1 is a file rename. Layer 2
  is deleting a workflow plus the `review/*` labels. Layer 3 is deleting one field and one
  view. No layer leaves state behind that blocks anything.
- **Observability:** the workflow logs the derived state and the inputs it used
  (approval count, required count, CI rollup, draft flag) so a wrong column can be
  diagnosed from the run log alone rather than by re-deriving it by hand.
- **Failure modes to watch:** a new review bot not in `BOTS` inflating coverage; the App
  token losing project write and silently failing the Layer 3 step; label and field drifting
  apart if the two writes are not in the same job; the guidance comment posting more than
  once because the marker lookup failed; and a growing `🤖 Bot Findings Open` column, which
  reads as a quiet queue but means contributors are stuck.

## Open Questions

Four things that should not be decided unilaterally.

**Q1. Should an approval survive a subsequent push?**

GitHub keeps approvals valid across pushes when no branch protection dismisses them, and
we have none. But `lgtm-remove-on-update.yml` already strips the `lgtm` label on
synchronize, so the repository has effectively answered "no" once already. Consistency
argues for invalidating approvals given before the current head commit. Strictness argues
the other way, since it means a rebase costs a re-approval. My position: match the
existing `lgtm` behaviour and invalidate, because a silently stale approval is the failure
mode that actually hurts.

**Q2. Should coverage require the right team, or just a human?**

The strict reading of `OWNERS.md` is that a reviewer approves within their specialty, and
`lgtm-processor.js` already does the team-membership checks needed to enforce it. The cost
is a team-membership API call per reviewer per area, which is noticeably heavier. My
position: ship with "any non-bot human" first, add team awareness as a follow-up once the
pipeline is proven, since the simple version already fixes the 24 bot-only pull requests.

**Q3. Should the board stay private?**

Not strictly part of this proposal, but adjacent. Project #3 is private while project #2
is public. Whatever we decide for review routing, the `🎯 Free for All` and
`⚡ Easy Wins` buckets identify 65 open items of delegatable work that contributors cannot
see, and 42 of those carry neither `good first issue` nor `help wanted`. Given
`burnout-mitigation.md` is explicitly about growing the contributor pool, that seems worth
a decision.

**Q4. Should the bot gate apply to first-time contributors?**

5 of the 9 pull requests the gate currently diverts are from first-time contributors, and the
worst case is a pull request with 19 unresolved bot threads. Exempting newcomers would give
them a human sooner, but it aims the exemption at exactly the pull requests where automated
review earns its keep, and it puts the noisiest changes straight onto a maintainer. My
position: apply it uniformly, rely on the guidance comment to make it navigable, and use
`review/bots-overridden` when someone is clearly stuck. Revisit if the column fills with
newcomers who then go quiet, which is the signal that the gate has become a wall.

## Alternatives

| Alternative                                       | Why not                                                                                                                                                                                                 |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Branch protection with required reviews           | Would hard-block the entire queue overnight: 60 of 78 open pull requests have no maintainer review. Worth revisiting once the queue is healthy, not as the mechanism for getting it there.                |
| Weekly digest to Slack or a tracking issue        | Answers "has this been reviewed" but not "is somebody on it right now", and adds a channel to maintain. The board answers both and already contains all 78 pull requests.                                 |
| Saved GitHub searches only, no board field        | Cheapest option and a genuine fallback if the project token does not materialise, but a list of filtered searches is not a transition path. No sense of a card moving, no claim signal.                    |
| Extend `lgtm` into a fuller command set           | Keeps state in comments, which is where it is unqueryable today, and requires a human to type something. The current `lgtm` label sits at zero uses, which is the evidence against.                       |
| A custom dashboard                                | Something else to host, authenticate, and keep alive. Labels plus a board view reach most of the value with nothing running.                                                                              |

---

Figures come from the 78 open pull requests and 258 board items as of 2026-08-19, read
through the GitHub GraphQL API. The column counts in the board section are produced by
running the ordered evaluation above against that snapshot, so they are what the board
would actually show on the day this lands.
