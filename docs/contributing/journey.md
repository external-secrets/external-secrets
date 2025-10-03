<!-- This is a tutorial. -->
<!-- An introduction to the contribution journey, a teaching experience on contributions to ES -->
<!-- It is not a replacement on how do I contribute (CONTRIBUTING.md) -->
<!-- It is not a place to clarify/explain the why of our processes (policy.md) -->
<!-- It is not a reference of WHAT IS each process (cf separate processes document) -->

<!-- If you want to contribute to this page: -->
<!-- Analyse whether you want to express Can you teach me to x? (Search for the appropriate tutorial) -->
<!-- Analyse whether you want to express HOW DO I follow process x (Search for the appropriate how-to page) -->
<!-- Analyse whether you want to express WHY we have/do process x unrelated to contributions (build another explanation page) -->
<!-- Analyse whether you want to express WHAT IS contributing process x (Search for the appropriate reference page) -->

# Contribute to External Secrets!

Welcome to the External Secrets contributing guide.
This document will guide you where you can help in this project.

It gives the series of steps to contribute, based on what you want to contribute on.

You **do not need to be a code contributor to help the project succeed**.

## A contributor's life

There is work for every volunteer.  Every little bit of help counts.

As of today, most of a contributor's time is spent in the management of slack, github issues and pull requests.

Here is what you can do for us:

- Answering questions and helping others in our slack channel (#external-secrets) or in our github issues
- Triaging issues by adding labels, reproducing bugs, or providing additional information.
- Reviewing pull requests and giving feedback
- Writing documentation or improving existing docs
- Spreading the love on social media, community events, blogs, or newsletters
- Recurrently attending community meetings
- Record evidence of your adoption of External-Secrets!
- Identify, prevent, and address contributor burnout

On top of that, we value code contributions:

- Pull requests/Code reviews (HIGHLY VALUED)
- Documentation contributions
- Helm chart contributions
- CI and infrastructure maintenance contributions
- General operator contributions (bug fixes, refactoring, etc.)
- Provider contributions (setting up accounts, maintaining controllers, etc.)
- Release management

Financial contributions are welcomed and documented in our [financial contributions policy](policy.md#financial-contributions)

## The value of triaging issues

We appreciate help to triage issues in our GitHub repository.
This helps the maintainers' team to focus on the most important/urgent problems are handled first.

If you want to help in triaging issues, you can start by commenting on them with:

- Any additional information you have verified (e.g., steps to reproduce/if you reproduced the bug, environment details, etc.)
- The labels you think fit for the issue (Have a look at our [Labels](labels_reference.md)).

Do not hesitate to ask questions if you need more context about an issue.
We strongly suggest you to not use an AI generation tool at this point, except to format your comments.
If you want to understand why, please read our [LLM Policies](policy.md#llm-policies).

### Helping others on other communication channels

We have other communication channels on top of our GitHub issues, for example our slack channel #external-secrets.
We are extremely thankful of people solving issues and growing community engagement in the multiple conversation channels we have.

NB: If you are a person in our channel, do not EXPECT an answer. Community time is not duty, it's a gift.
If you want technical support, you can purchase a maintenance contract over any company engaged in ESO.

### Helping ESO grow through community events

We sadly do not have a "boilerplate" community event/marketing information or goodies.
This is an open-source project :)

If you are willing to talk about ESO in a community meeting,
please read our [Community events policy](policy.md#handling-community-events)

### Giving feedback in community meetings

ESO community calls can be subscribed to from the CNCF Project Calendar page here [CNCF Calendar](https://www.cncf.io/calendar/).

Our agenda is open. You are free to add anything to a community meeting's agenda.
We are not only open source, we follow an open design/development/community mindset.
The meetings are important timeslots to decide a trajectory we will all take together.

Community meetings are **not** the opportunity to raise your own issue.
Keep that in mind when you add something in the agenda.

Regular attendance will show your interest into the external-secrets's future.

### Officially adopt External-Secrets

Many people are using External-Secrets but do not have the legal capacity to mention it.
If you can mention your usage, why don't you add your company name to our adopters list by filing a PR to `ADOPTERS.md` ?

This way everyone can easily see our community growing (feel-good vibes), but also can contact you.
With a two-side conversation, we are building a long term ecosystem, not just reusable code.

## Code contributions

### Before you work on code

Make sure you follow our CONTRIBUTING.md guide explaining how to contribute.
It also contains a link to our project board, explaining where we are headed and whether your code seems aligned or not.

### Reviewing pull requests and giving feedback

TODO(evrardjp): Please do not hesitate to complete this section!

### Improving our documentation

Code is only as good as its documentation. Having good documentation:

- reduces the requests for help
- improve quality of life for onboarding
- increase community size

You can help us improve our documentation in the `/docs` folder.
Our docs aim to follow the [diataxis](https://diataxis.fr) framework.
We are in the process of migrating our docs to this framework.

Come help us and get started by:

- getting acquainted with the framework
- reading the `Setting up the development environment for documentation contributions` section in CONTRIBUTING.md
- propose your first changes!

### Helm charts

Helm charts are the official way to consume External Secrets.
Making sure our helm charts are practical and up to date is an important duty.
Test our helm charts and propose any change that is willing to reduce our debt and increase our stability.

Do not forget you are not alone. Always think twice before breaking another person's use case.

### CI and infrastructure maintenance contributions

Our software is only as good as how we test it.
The CI should be everyone's focus.

If you see a PR having an issue in CI, you should learn the following reflexes:

- Do not retrigger CI blindly. Do not waste computing power.
- Compare CI's run with your machine. Find if it's reproducible.
- Talk about the issue in #external-secrets-dev channel.
- Solve it before someone else needs to solve it.

### Cloud provider and secrets management solution vendor contributions

We rely on vendor features (cloud provider or secrets management solutions) in ESO.

As External-Secrets code contributors is providing features/free marketing for those vendors,
the maintainers are expecting vendors (or interested community parties) to:

- provide access to vendor's offering in our CI environment
- provide a person of contact to help maintain the vendor part of the external-secret operator

The first one can be done by sponsoring accounts to your solutions to the ESO maintainer team.
The second is done by being active in the community (triaging vendor's issues, solving them).

If you have a relevant product, want to ensure that ESO works well with it and that we can quickly address any issues that arise,
get active on the community. Our quality is your quality.

## Getting more involved

This was just the start. If you do any of the above, you will rapidly get recognized by the community.
You can continue climbing our [Contributor Ladder](contributor_ladder.md)
