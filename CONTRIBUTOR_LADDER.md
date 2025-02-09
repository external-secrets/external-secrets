# Contributor Ladder Template

* [Contributor Ladder](#contributor-ladder)
    * [Community Participant](#community-participant)
    * [Contributor](#contributor)
    * [Organization Member](#organization-member)
    * [Maintainer](#maintainer)
* [Inactivity](#inactivity)
* [Involuntary Removal](#involuntary-removal-or-demotion)
* [Stepping Down/Emeritus Process](#stepping-downemeritus-process)
* [Contact](#contact)


## Contributor Ladder

Hello! We are excited that you want to learn more about our project contributor ladder! This contributor ladder outlines the different contributor roles within the project, along with the responsibilities and privileges that come with them. Community members generally start at the first levels of the "ladder" and advance up it as their involvement in the project grows.  Our project members are happy to help you advance along the contributor ladder.

Each of the contributor roles below is organized into lists of three types of things. "Responsibilities" are things that a contributor is expected to do. "Requirements" are qualifications a person needs to meet to be in that role, and "Privileges" are things contributors on that level are entitled to.

ESO contribution ladder is _not_ strict. We realise that people have lives outside of this project so there are very minimal actual _numbered_ requirements on things like prs/month or issues reviewed/month.

We do expect people who would like to be involved in the project to be there and somewhat active members, but there are no hard requirements. Each member is evaluated individually and will be expected to be part of the team effort to better ESO in the future.

Some amount of inactivity will be considered for demotion, but it's not a hard limit. We will always discuss people and their circumstances in their lives.

We are very open and supportive community, and we would like to keep it that way. Everyone is welcome, be their contribution however small, even if it's a single character fix.

However, please don't mistake this openness for a lack of authoritative responses of demands are made to us or the CNCF CoC is ignored. We will take swift action to protect our members and we have zero tolerance against people who disregard our maintainers time and efforts.

### Community Participant
Description: A Community Participant engages with the project and its community, contributing their time, thoughts, etc. Community participants are usually users who have stopped being anonymous and started being active in project discussions.

* Responsibilities:
  * Must follow the [CNCF CoC](https://github.com/cncf/foundation/blob/main/code-of-conduct.md)
* How users can get involved with the community:
  * Participating in community discussions
  * Helping other users
  * Submitting bug reports
  * Commenting on issues
  * Trying out new releases
  * Attending community events

### Provider Maintainer
Description: ESO's providers are created and maintained by community members. [Stability Support](docs/introduction/stability-support.md) documentation describes these people and organizations who maintain their own provider implementation.
A provider maintainer's main focus is the provider they are maintaining, therefore they usually don't get involved in
overarching design decisions that involve the core of ESO.

* Responsibilities:
  * Must follow the [CNCF CoC](https://github.com/cncf/foundation/blob/main/code-of-conduct.md)
  * Required to keep their provider up-to-date with latest changes and modifications to the provider's API. Core ESO
    maintainers are not responsible to provider major overhaul or complex updates to the provider if versions are updated.
  * Respond to bug reports and issues if it's determined that a bug in a provider is due to the provider's own implementation and not the fault of the core ESO software.
* Privileges:
  * We usually respond fast to updates on code that is done on the provider's code and localized on that section
  * Website mention of supported providers and their companies

### Contributor
Description: A Contributor contributes directly to the project and adds value to it. Contributions need not be code. People at the Contributor level may be new contributors, or they may only contribute occasionally.
Contributors are usually there to contribute and maintain the core of ESO and not provider code. Unless overarching changes occur that need certain updates in all providers.

* Responsibilities include:
    * Follow the CNCF CoC
    * Follow the project contributing guide
* Requirements (one or several of the below):
    * Report and sometimes resolve issues
    * Occasionally submit PRs
    * Contribute to the documentation
    * Show up at meetings, takes notes
    * Answer questions from other community members
    * Submit feedback on issues and PRs
    * Test releases and patches and submit reviews
    * Run or helps run events
* Privileges:
    * Invitations to contributor events
    * Eligible to become an Organization Member

### Organization Member
Description: An Organization Member is an established contributor who regularly participates in the project. Organization Members have privileges in both project repositories and elections, and as such are expected to act in the interests of the whole project.

An Organization Member must meet the responsibilities and has the requirements of a Contributor, plus:

* Responsibilities include:
    * Continues to contribute regularly, as demonstrated by having good cadence each year. We don't have strictly defined metrics, because we recognize that contributors have lives too. If a contributor has left the project for a long time but returns and wants to contribute again, we will run through updates and have them catch up on latest changes. But after that, they will be welcomed to keep contributing to the project.
* Requirements:
    * Must have successful contributions to the project, including at least one of the following:
        * accepted PRs
        * Reviewed a number of PRs,
        * Resolved and closed Issues,
        * Become responsible for a key project management area,
        * Or some equivalent combination or contribution
    * Must have been contributing for at least a couple of months
    * Must be actively contributing to at least one project area
    * Must have two sponsors who are also Organization Members, at least one of whom does not work for the same employer
    * Must have 2FA set up on GitHub
    

* Privileges:
    * May be responsible for running tests, releasing, and accounting for the test infrastructure of various providers
    * May be assigned Issues and Reviews
    * May give commands to CI/CD automation
    * Will be included in design decisions
    * Can be added to [maintainers](https://github.com/orgs/external-secrets/teams/maintainers) team
    * Can recommend other contributors to become Org Members

The process for a Contributor to become an Organization Member is as follows:

1. Starts to spend some time on the project and tries to understand overarching architecture
2. Gives good feedback
3. Understands the project focus and starts partaking on community meetings
4. Once they spend more time on the project and has good vibes with the rest of the maintainers an offer is extended to be part of the project organization
5. If accepted, after a while, [maintainer ship](#maintainer) can be achieved

### Maintainer
Description: Maintainers are very established contributors who are responsible for the entire project. As such, they have the ability to approve PRs against any area of the project, and are expected to participate in making decisions about the strategy and priorities of the project.

A Maintainer must meet the responsibilities and requirements of a Reviewer, plus:

* Responsibilities include:
    * Reviewing at least [TODO: Number] PRs per year, especially PRs that involve multiple parts of the project
    * Mentoring new Reviewers
    * Writing refactoring PRs
    * Participating in CNCF maintainer activities
    * Determining strategy and policy for the project
    * Participating in, and leading, community meetings
* Requirements
    * Experience as a Reviewer for at least a couple of months
    * Demonstrates a broad knowledge of the project across multiple areas
    * Is able to exercise judgment for the good of the project, independent of their employer, friends, or team
    * Mentors other contributors
* Additional privileges:
    * Approve PRs to any area of the project
    * Represent the project in public as a Maintainer
    * Communicate with the CNCF on behalf of the project
    * Have a vote in Maintainer decision-making meetings

Process of becoming a maintainer:
1.
2.
3.

## Inactivity
It is important for contributors to be and stay active to set an example and show commitment to the project. We realize that people have lives too, but after a certain
period of time of absolute inactivity and if the rest of the team votes thus, people can be removed from the organization and demoted from being a project member.

* Inactivity is measured by:
    * Periods of no contributions for longer than 12 months
    * Periods of no communication for longer than 6 months
* Consequences of being inactive include:
    * Being asked to move to Emeritus status; if they still don't respond then
    * Involuntary removal or demotion

## Involuntary Removal or Demotion

Involuntary removal/demotion of a contributor happens when responsibilities and requirements aren't being met. This may include repeated patterns of inactivity, extended period of inactivity, a period of failing to meet the requirements of your role, and/or a violation of the Code of Conduct. This process is important because it protects the community and its deliverables while also opens up opportunities for new contributors to step in.

Involuntary removal or demotion is handled through a vote by a majority of the current Maintainers.

## Stepping Down/Emeritus Process
If and when contributors' commitment levels change, contributors can consider stepping down (moving down the contributor ladder) vs moving to emeritus status (completely stepping away from the project).

Contact the Maintainers about changing to Emeritus status, or reducing your contributor level.

## Contact
* For inquiries, please reach out to:
  * [Kubernetes Slack channel](https://kubernetes.slack.com/messages/external-secrets)
