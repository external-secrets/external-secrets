<!-- This is a tutorial. -->
<!-- An introduction to teach we intend prevent burnouts in our team -->
# Contributor Burnout Mitigation Guide

## Overview

This document provides a framework for identifying, preventing, and addressing contributor burnout in the External Secrets Operator (ESO) project. Based on lessons learned from past experiences and successful community outreach efforts, this guide aims to help maintain a sustainable project without the need for release pauses or drastic measures.
It is everyone's responsibility to _identify_ burnout and inform the team at a community meeting. "See it, say it, sorted."

## Understanding Burnout

### What is Contributor Burnout?

Contributor burnout occurs when the demands of maintaining an open source project exceed the available resources and energy of the maintainer team. In ESO's context, this manifests as:

- Disproportionate contributor-to-user ratio: Millions of users supported by a handful of maintainers
- Unsustainable workload: Everything from code review to security responses handled by the same small group
- High responsibility with limited resources: Critical infrastructure component with enterprise-grade expectations

### Why ESO is Particularly Vulnerable

ESO is particularly affected because:

1. Critical Infrastructure Role: Often the first component deployed in Kubernetes clusters
2. Security-Critical Nature: Due to it's nature the project demands a high focus and high attention during reviews
3. Wide Enterprise Adoption: Used by a lot of high-stakes organizations

## Early Detection Signals

### Individual Maintainer Signals

Monitor for these warning signs in team members:

#### Behavioral Changes

- Delayed responses to issues, PRs, or community messages
- Decreased participation in community meetings or discussions
- Shorter, more terse communication style
- Avoiding complex issues or deferring decisions consistently
- Working only on "fun" tasks while avoiding maintenance work
- Change in behavior and borderline violating the code of conduct during responses and user interactions

#### Workload Indicators

- One person handling multiple critical areas (releases, security, CI, support)
- Working outside normal hours consistently to keep up
- Expressing frustration about repetitive or mundane tasks

#### Quality Signals

- Rushing through reviews to clear backlog
- Postponing or skipping testing for expedience
- Technical debt accumulation due to time pressure

### Project-Level Signals

#### Community Health Metrics

There are a couple of things to keep an eye on for the overall health of the project and issue cadences.

- Issues keep being reopened
- Review times consistently increasing
- Release cadence becoming irregular or delayed
- Community meeting attendance declining among maintainers

#### External Pressure Indicators

Also keep in mind that external pressure can increase. There are time where the project sees a sudden spike in usage and times of lul as well.
We need to keep monitoring influx items and pay attention to when the pressure is being put on.

## Prevention Strategies

None of these things will guaranteed solutions, however, they might help.

### Workload Distribution

#### Create Ownership Areas

- Certain areas could be covered by the same maintainer (e.g., specific providers, testing, documentation)
- Keep release and support roles on a rotation so people don't think they are in a rut
- Document tribal knowledge to make it accessible to others
- Take over other contributor's work by extending your own ownership area when something goes wrong.

#### Automate Repetitive Tasks

CI/CD pipelines can help a lot in taking away some of the menial tasks while working on the project.
Immediate bot responses for triage issues could be configured using copilot, or other means like claude code github action.
These responses would use the repository as a context and could give immediate valuable info to the submitter such as:
- Duplicate issues
- Possible solutions looking at the documentation
- Link to existing documentation based on context

These need to be fine-tuned but could potentially alleviate some of the tress and pressure for the maintainers.

### Community Building

It is important that we nurture an understanding and caring community. People who use ESO will have to understand that demands will lead no-where.
The maintainers are offering their time and efforts to keep the project sustained. Only _requests_ and _questions_ will be answered and met with similar responses.

_Respect is earned, not given_. The code of conduct is there for a reason and it will be enforced. People who _demand_ that maintainers do something and people who _expect_ that
maintainers support their every need will be met with a brick wall. Please understand that we are doing this as a hobby and as something out of the goodness of our hearth
and because we believe in open source software. That does _NOT_ mean that you as a user are entitled to demands.

## Community Outreach Framework

Add more templates here if required for Issues ( a pinned issue on external-secrets GitHub page ), LinkedIn, blog posts on the [website](https://external-secrets.io/latest/eso-blogs/).

### Reddit Post Template

~~~markdown
# 🔄 ESO Community Update: Growing Our Maintainer Team

Hey r/kubernetes community!

We're reaching out with an update on External Secrets Operator (ESO) and an opportunity for the community to get involved.

## Current State
ESO continues to grow in adoption - we're now deployed in [specific stats] environments and serve as critical infrastructure for organizations ranging from [examples]. This growth is amazing, but it also means we need to scale our maintainer team to match.

## What We Need
We're looking for contributors who can help with:
- Code review and development (Go experience helpful but not required)
- Provider maintenance (AWS, Azure, GCP, HashiCorp Vault, etc.)
- Documentation and user guides
- Issue triage and community support
- Testing and quality assurance

## What We Offer
- Onboarding with experienced maintainers
- Flexible commitment levels - contribute what works for your schedule
- Real impact on critical Kubernetes infrastructure
- Learning opportunities in security, secrets management, and operator development
- Recognition in a high-visibility CNCF project

## How to Get Involved
1. Fill out our [contributor interest form](https://github.com/external-secrets/external-secrets/blob/636ce0578dda4a623a681066def8998a68b051a6/CONTRIBUTOR_LADDER.md)
2. Join our [next community meeting](https://zoom-lfx.platform.linuxfoundation.org/meetings/externalsecretsoperator?view=month)
3. Check out our [contributor guide](https://external-secrets.io/latest/contributing/devguide/)
4. Start with a [good first issue](https://github.com/orgs/external-secrets/projects/2/views/9)

## Questions?
Drop them below or reach out on [Slack/Discord/GitHub Discussions].

Thanks for being part of this community! 🚀

---
*Cross-posted to relevant communities - thanks for your patience if you see this multiple times*
~~~

## Conclusion

This document sums up various procedures and things that we can do and we can start on. The important part is publication,
visibility and outreach. There are many channel on which ESO can communicate but the most important ones are:
- Slack ( [external-secrets](https://kubernetes.slack.com/archives/C017BF84G2Y), [external-secrets-dev](https://kubernetes.slack.com/archives/C047LA9MUPJ) channels )
- Reddit [Kubernetes Subreddit](https://www.reddit.com/r/kubernetes/) ( this was particulalry helpful in the past )
- HackerNews pos
- LinkedIn
- CNCF help channels and issue requests
- Pinned Issue on GitHub page

Whatever we do the most important part is visibility _BEFORE_ we get to this point. Before all of this, the most important part is
monitoring the maintainers health and general well being. Prevention instead of escalation.

## Our reaction when things do not go as planned

Contributors will come and go. It is perfectly normal (and even welcomed!) in an open source project.
When events occur and response do not go as planned, the maintainers team will take decisions and expose them in a community meeting.

Here is our DNA: Contributor's healths come first. We will never compromise humans for software.

The team will try (best effort) to:
- minimize impact on community
- be transparent over any potential impact

Maintainers stepping back from the project is perfectly _fine_, the project slowing down is _fine_. this shouldn't be seen as a negative. People need to take care of themselves first before they can take care of the project.
