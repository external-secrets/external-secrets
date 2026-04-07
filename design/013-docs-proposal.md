---
title: Complete Docs refactor proposal
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-03-10
status: draft
---

# Docs refactor proposal

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->


## Summary
This proposal is recycling _all docs proposals_ into a coherent proposal to improve the state of the docs for External Secrets

## Motivation

I have noticed over time that a large amount of questions about external secrets come from:

* Set up questions: people do not understand how to use feature X
* Supportability questions: people do not see if X is supported (or do not understand feature matrix), and expect a bit more clarity.
* Unclear call to actions: "How am I supposed to continue from here?" "Where do I write this?" "How to write this?"
* Unclear past decisions: "I am expecting to do this, why doesn't work like that?" "Because we decided so"

This is all related to documentation in a broad sense, regardless of the media (website, slack).

Next to this, the undocumented decisions and unclear documentation architecture prevent us to use tools (like LLMs) to automate some content writing.

### Goals

This proposal aims to provide a clear path about the list of changes to introduce to improve our documentation state.

### Non-Goals

This does not go as far as refactoring our whole code base, while it impacts _ALL CODE WHICH COULD HAVE GENERATED DOCUMENTATION_.

## Proposal

### Stage 1: New multi-project website

Create a simple, single-page landing site for external-secrets.io inspired by [kured.dev](https://kured.dev/) or cilium's.
The landing page will provide a clear value proposition, key features, and direct users to the documentation.
The existing documentation from each will be accessible as the top level docs.
Releases needs to be listed directly as a top level docs.

If possible, use kubernetes ecosystem tools, like hugo+docsy.
No server side solution is to be added.

This will improve our attractiveness and introduce the start of a new "multi-project" documentation architecture.
It will introduce the first steps of documentation release management too.

Example iteration: https://github.com/evrardj-roche/external-secrets-website

### Stage 2: Introduce clear documentation architecture: Adapted diataxis

Now that a website is created, we need to make sure the information is browsable, searchable, but also practical for the users.
This is the biggest pain point to solve.

However, as https://github.com/external-secrets/external-secrets/pull/5822 has listed, this effort is massive.

Without this stage, we will not be able to decide _where_ and _how_ to write content.
Diataxis is _one solution_ that we can easily adapt to fit our needs.

ONE SOLUTION IS BETTER THAN NONE.

In this case, I want to adapt diataxis by relying on its 4 types of documentation as top level doc of each project.
However, many people like the ESO's Provider documentation, so I suggest to have a 5th top level documentation: the how-to of each provider.

### Stage 3: Add the ability to adapt docs using an LLM (optional)

While https://github.com/external-secrets/external-secrets/pull/5822 intended to do it manually, this proposal
is adapting its opinion to rely on tools like LLM to generate content.

With a good architecture described in our docs website, we can rely on _skills_ or similar tools to help us (re)write our documetation.

This is totally optional.

### Stage 4: Introduce governance around features stability and their documentation

We had conversations in our #external-secrets-dev slack channel, asking "When is X going out of beta" or "Is X supported?".
The support matrix is not clearly well understood as there is no governance about it.

This stage intends to clarify the governance around feature introduction, stability, and general lifecycle in our `policy.md`

### Stage 5: Automatically generate parts of the documentation from our codebase.

Based on the stage 4 policy, we can now generate parts of documentation directly in each provider.

You can find a PoC that was abandonned here: https://github.com/evrardj-roche/external-secrets/commits/feature-flags/ .
This part also deprecates a previous design proposal: https://github.com/external-secrets/external-secrets/blob/81078c9ab6a7cf2ddbd0fe5856188a120e09e87a/design/014-feature-flag-consolidation.md

The idea here is the following:

- We move all providers to register into a central registry (containing features, flags, ...)
- The central registry can be imported from the docs perspective to generate a markdown file with a support matrix.
- The whole code needs to clean up to adapt to providers patterns we decided for "v2" and future plugin extraction.

### User Stories

- As a user, I want to know to configure X, docs are not clear so far
- As a new maintainer, I want to know how to triage X, I don't know where to find accurate information on the website.
- As a developer, I want to know whether I can update the code and the support matrix without impacting other providers.

### API

No API change.

### Behavior

No user behaviour change. Many changes of code are however introduced at step 1 and 5.

### Drawbacks

None.

### Acceptance Criteria

You tell me :)

## Alternatives

Do not do anything and suffer in our docs.

