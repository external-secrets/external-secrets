```yaml
---
title: Deprecaation of OLM Builds
authors: @gusfcarvalho
creation-date: 2024-12-04
status: approved
---
```
This Proposal was approved on community meeting of 4th december 2024 (meeting notes: https://hackmd.io/GSGEpTVdRZCP6LDxV3FHJA?both)

# Deprecaation of OLM Builds

## Introduction

As part of our Release process, we currently build & maintain several docker images, helm releases, bundle manifests for users, and OLM builds via a community effort based on olm helm operator.

However, OLM helm operator itself would require a better support and constant maintenance, and its process when building OLM builds is already not automated anymore.
## Summary
Stpo building OLM Releases

## Motivation
Make maintenance lives easier for a project that is struggling to get maintainers together :)

### Goals
Remove OLM builds as part of our build assets

## Proposal
Archive repository & communicate on next release within the release notes.

### API
None

### Behavior
None

### Drawbacks
Users might complain - but then they can fork the archived repository to build their own OLM builds locally.

## Alternatives
Find community members to handle the maintanence aspect of it. Have a new dedicated OLM repository in/out of the org. Make this be maintained by other parties than external-secrets maintainers.

Do not use the current olm helm operator anymore as anyways this is not really supported.

