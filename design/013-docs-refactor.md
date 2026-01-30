```yaml
---
title: docs website restructuring
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2025-12-17
status: draft
---
```

# Docs website restructuring

## Summary

Restructure our docs website (outside our "marketing" landing page, not part of this proposal) into different sections, mapping to diataxis to reduce the mental complexity of writing docs.

It is currently hard to find information, but it is even harder to know where to write or HOW to write it.

### Goals

By the end of the implementation of this proposal, we will:

* Be able to find information on docs website according to a standard
* Know how to write documentation in the future
* Have a "persona oriented" documentation site, which allows each person to find their information more easily

### Non-Goals

This proposal does not cover the writing of the public "marketing landing page" (see other proposal).

## Proposal

Adopt [Diátaxis](https://diataxis.fr) as a framework to GUIDE us through the writing of our docs in the future.

The core idea of Diátaxis is that there are fundamentally
four identifiable kinds of documentation, that respond to four different needs.
The four kinds are: tutorials, how-to guides, reference and explanation.
Each has a different purpose, and needs to be written in a different way.

Diátaxis is NOT A PLAN telling us how to complete our docs in a certain architecture.
It's a GUIDE to helps us write our docs and clarify the right direction for our docs.

This framework is suited for technical project and is in use by the following technical projects:

* [Cloudflare docs](https://developers.cloudflare.com/agents/)
* [Gatsby docs](https://www.gatsbyjs.com/docs/)
* Canonical docs, for example [cloud-init doc](https://cloudinit.readthedocs.io/en/latest/index.html)
* [Divio docs](https://docs.divio.com/)
* [Django docs](https://docs.djangoproject.com/en/3.0/contents/)

There are other examples, but I did not delve deeper.

To note, there are other frameworks, like the [good docs project](https://gitlab.com/tgdp/templates/-/tree/main?ref_type=heads) which base themselves upon diataxis.

I also don't know many other frameworks (the alternatives I know become quite heavy quite fast).
I relied on Diátaxis in the past to structure my thinking/writing with moderate to good success.

Some of the downsides are also explained in multiple blog posts, for example this one: https://www.hillelwayne.com/post/problems-with-the-4doc-model/ .
I do agree with some of the downsides.

My experience is that our docs website will EVENTUALLY be restructured with a main page (the landing page)
pointing to _at least 4_ top Diátaxis categories, then extra based on practical needs of the docs, to keep a certain readability without delving into sub-sub-subfolder trend.

| Section     | Landing page content                           |
|-------------|------------------------------------------------|
| How-TO      | A page explaining all the how-tos, per persona |
| Reference   | List of all reference information              |
| Tutorials   | A page explaining each persona's tutorial      |
| Explanation | List of all user journeys                      |

Only the future will tell the amount of sections we will have. We should see Diátaxis as a guide, not a complete solution. If it does not FIT for certain content, that's fine! We will figure it by then.

### How we will work with our docs - how we will approach the rewriting of our docs

Diátaxis come with a very simple workflow:

1. Consider what you see in the documentation, in front of you right now (which might be literally nothing, if you haven’t started yet).
2. Ask: is there any way in which it could be improved?
3. Decide on one thing you could do to it right now, however small, that would improve it.
4. Do that thing.

I think this will work fine for ESO.

### User Stories

Here is a beginning of the list of the problems we need to solve, regardless of order:

* A contributor has trouble finding a page with all the content they need to contribute meaningfully.
  They therefore lose time in how to contribute.
  This prevents community ramp-up.
* A contributor does not like to do back-and-forth with docs to deliver their feature.
  They want to be told how to do things for the project.
  Removing interpretation by having a framework makes it easier.
* A system administrator wants to quickly start ESO, and they do not see the value of ESO immediately.
  We should guide them to use ESO and see its value asap. This will get them on board easily.
* A security researcher wants to get rewarded for their work to secure ESO.
  They do not find easily the information on how to act if the ESO team is not very active.
  (Need to find the security, governance, escalation policies in a structured way)
* A security operational wants to know what's included in ESO, what are ESO security policies, or if their system administrator has properly secured their deployment.

All of this is ALREADY in our documentation, but not easily searchable.
Diátaxis will tell us HOW we can think about this doc and how to rewrite it.

### API

None

### Behavior

If possible as much as possible close to the diataxis standard.

### Drawbacks

A lot of work has to be done over time.

### Acceptance Criteria

Docs is not a ONE SHOT delivery. We should never consider our work on docs as "done".
Yet, we can split the work in two phases:

* Initial implementation
* Continued maintenance

The initial implementation should be enough to consider this proposal as "done".

To avoid interpretation, I propose that initial implementation contains the following:
* All 5 user stories are solved
* Every single page of our docs is rewritten to the format of either a how-to, a reference, a tutorial, or an explanation.
* The separated pages do not result in more complexity for docs, but instead rely on only those 4 big sections (and extra sections on a need basis)

### Docs migration guide

See docs/MIGRATION_GUIDE.md

## Alternatives

* Leave the docs as it is (but we agreed to not to!)
* "Just wing it".
  If we do it without a framework, we will eventually have another spaghetti of docs over time, when documentation is written with less rigor

