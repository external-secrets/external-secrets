---
title: Complete Docs refactor proposal
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-03-10
status: draft
---

# Docs refactor proposal

## Summary

This proposal is creating a combined "master plan" to improve the state of External Secrets' documentation.
It is a combination of my previous docs proposals into a coherent plan, with actionable steps.

## Motivation

I have noticed over time that a large amount of questions about external secrets come from:

* Set up questions: people do not understand how to use feature X
* Supportability questions: people do not see if X is supported (or do not understand feature matrix), and expect a bit more clarity.
* Unclear call to actions: "How am I supposed to continue from here?" "Where do I write this?" "How to write this?"
* Unclear past decisions: "I am expecting to do this, why doesn't work like that?" "Because we decided so"

This is all related to documentation in a broad sense, regardless of the media (website, slack).

Next to this, the undocumented decisions and unclear structure prevent us to use tools to automate some content authoring.

### Goals

This proposal aims to provide a clear path about what and how to change our documentation.
This will make the core team more efficient and improve the quality of our project at each contribution.

By the end of this proposal:

- we will have a clear followable website that can differentiate between docs such as adopters, projects ( reloader, eso )
- people will be able to find clear information on current state of the project and its maintained versions, including about the provider's features
- a website that does not require tons of maintenance in the long run
- people find the information they are looking for without relying on LLMs, with a persona based website (contribution guide, security best practices, manager guide, ... ) 
- Marketing material to help us promote us to new maturity level from the CNCF perspective

### Non-Goals

This is mostly focused on ESO and its website, and does not cover the documentation of other sub-projects, like the CLI and reloader.

## Proposal

### Step 1 (Optional): New multi-project website

Create a simple, single-page landing site for external-secrets.io inspired by [kured.dev](https://kured.dev/) or cilium's.
The landing page will provide a clear value proposition, key features, and direct users to the documentation.

The documentation from each sub-project will be accessible as the top level docs.
Releases needs to be listed directly as a top level docs.
Community links common to all projects will also be a top level document.

#### Actual implementation

The current kubernetes ecosystem is focused on hugo + docsy.
This website will make use of hugo+docsy to "feel" similar to other CNCF projects.

It is worth noting that docsy is really made for 1 project and its versions.
In our case, we want a coherent documentation across projects (CLI/reloader/ESO), so it means we will need to customize docsy to our needs.

Amongst the notable changes, we will need to adapt the build process, the JS and the templates to support our search cases.
This was discussed on our slack channel: We want to be able to have a coherent search in the _current_ version of the sub-project's doc + _stable_ version of the other subproject's docs.
No online search service can be used, and the local search has to be used.
Due to our large history of versions, we need to efficiently build a local cache for search.

You can find a demo on https://github.com/evrardj-roche/external-secrets-website

#### Acceptance criteria

The criteria for success of step 1 is to actually use this new website as front page of https://external-secrets.io.
This will improve our attractiveness and introduce our "multi-project" documentation architecture, raising visibility to reloader or our CLI.

#### Side-effects

We will need to adapt our release management process to this new website building pipeline.
This has to be done before moving to the next step.

### Step 2: Agree on a documentation architecture in a community meeting

The step 1 will make our website more visible but it will also highlight that our content architecture is difficult to browse or unnatural to follow.
People currently know how to add a provider, but our overall architecture is hard to follow. The documentation in each provider starts to become unreadable, and the other sections abandonned.
This results in a poorly designed and hard to follow documentation.

The personas are not clear and therefore the content is hard to follow. If you felt "Am I supposed to read this?" while browsing ESO docs, this is what this step is intending to fix.

A long term effort to improve the documentation needs to be started on solid foundations. This step is defining the ground of the architecture of the documentation for the future.

#### Actual implementation

As there is no content architect available in our community, I am proposing to use an existing framework instead, allowing us to "stand on the shoulder of giants" to establish a systematic approach to our documentation.
Diataxis (https://www.diataxis.fr) is what I propose.

Diataxis brings a systematic approach to technical documentation authoring.
It allows humans (or bots!) to write in a systematic and standardized fashion.
This improves the documentation IN THE LONG term.

The _structure_ of the documentation does NOT need to change on day 1.
The _way we write_ our documentation needs adapting starting at that step 2.

#### Acceptance criteria

- We have a community meeting decision about the _personas_ that will be used in the architecture of our documentation (not the structure, but the readers!).
- We have a CI tool that will help us guarantee that the documentation is written in a correct fashion by ensuring new documentation is always split by persona.
- Have a series of LLM skills that can help our developers write the documentation according to our ways

#### Side-effects

At this point, each project's documentation will start to spread into multiple documents:

- Introduction
- API
- Guides (contributor focused, operator focused)
- Providers
- Examples/How-tos (deployer focused?)
- Reference (security focused)
- Tutorials (deployer focused?)
- Explanations

This is fine and will eventually be cleaned up in the long term.

### Step 3 (Optional): Accelerate the adaptation of our documentation based on these new foundations

JP mentioned in https://github.com/external-secrets/external-secrets/pull/5822 that rewriting ALL the content in one go is a massive effort.
In this step, we can decide to accelerate or not the effort to adapt our current documentation.

#### Actual implementation

One or multiple persons can become volunteers for docs rewrite.

#### Acceptance criteria

- All our existing pages have been rewritten for their audience;
- The structure of the project's documentation is simplified to a minimum set of high level items (For example: Providers/Guides/How-tos/Reference/Tutorials).

#### Side-effects

We cannot know in advance the structure of the final documentation, it will only show up by doing the work.
We cannot use that for the next steps.

### Step 4: Clarify governance around features stability and their documentation

We had conversations in our #external-secrets-dev slack channel, asking "When is X going out of beta" or "Is X supported?".
The support matrix is not clearly well understood as there is no governance about it.

This step aims to have a documented and autoritative answer over what is alpha/beta/stable level for a feature/provider.

#### Actual implementation

JP proposes a PR clarifying the lifecycle of the features.
We discuss this in the PR and we raise awareness in a community meeting.

We can hold a meeting to clarify the lifecycle of the features.

#### Acceptance criteria

- We have documented and merged this policy change.

#### Side-effects

Some implementations might be promoted/demoted in terms of maturity. This might have a morale impact.

### Step 5: Automatically generate lifecycle events in our documentation, from our codebase.

Based on the stage 4 policy, we could now generate parts of documentation directly in each provider!

The idea is the following: Each provider's code contain the data necessary to generate the documentation of their feature support/maturity status.
We already have parts necessary for that (see below).

#### Actual implementation

You can find a PoC that was abandonned here: https://github.com/evrardj-roche/external-secrets/commits/feature-flags/ .

The idea here is the following:

- We move all providers to register into a central registry (containing features, flags, ...)
- The central registry can be imported from the docs perspective (I used a separate go client for that, which allows loading modules) to generate a markdown file with a support matrix.
- Each call to generate the documentation would improve the latest unreleased documentation with an updated information based on policy.

#### Acceptance criteria

- No docs for feature support in manually generated;
- We have updated CI to prevent manual edition of the feature matrix.

#### Side-effects

This part also deprecates a previous design proposal: https://github.com/external-secrets/external-secrets/blob/81078c9ab6a7cf2ddbd0fe5856188a120e09e87a/design/014-feature-flag-consolidation.md

### User Stories

- As a user, I want to know to configure X, docs are not clear so far
- As an advocate, I want to know what ESO does (to share it with my org), without having to rely on an LLM to search the information I need.
- As a new contributor, I want to know how to contribute in a "story" fashion: How to triage effectively, how to setup my environment... This is distributed or outdated in our website/contributing docs.

### API

No API change.

### Behavior

No user behaviour change.

### Drawbacks

None.

### Acceptance Criteria

Please read each step.

## Alternatives

Do not do anything and suffer in our docs.
