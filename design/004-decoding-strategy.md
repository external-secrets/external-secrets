```yaml
---
title: ExternalSecrets Decoding Strategy
version: v1alpha1
authors: @gusfcarvalho
creation-date: 2022-04-27
status: draft
---
```

# External Secrets Decoding Strategy

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->


## Summary
ExternalSecrets Decoding Strategy aims to add a possibility for users to decode their secrets from the provider in a more convenient way. Currently, any encoded secret must be previously loaded by using templates, which can be quite cumbersome for new users to ESO.

## Motivation
A Decoding Strategy can decrease the load needed for new users to join the project, and can benefit a better interface when loading secrets.

### Goals
The goals of this feature is to allow decoding to happen after a Secret has been fetched from the provider, without the need to specifically dictate a template to it. This feature should also have an "Auto" mode where several common decoding strategies are adopted.

### Non-Goals
This feature will not cover provider-specific encoding methods.

## Proposal

```
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: my-external-secret
  namespace: default
spec:
  refreshInterval: 1m
  target:
   name: secret-to-be-created
  secretStoreRef:
    name: secretstore-sample
    kind: SecretStore
  data:
    - secretKey: my-secret-key
      remoteRef:
        key: /foo/bar
      #Decode strategy can be one of "Auto", "Base64", "JSON", or "Raw". "Auto" behavior will try to apply "Base64" -> "JSON" -> "Raw"
      dedodeStrategy: "Auto"
```

### User Stories
How would users use this feature, what are their needs?

### API
Please describe the API (CRD or other) and show some examples.

### Behavior
How should the new CRD or feature behave? Are there edge cases?

### Drawbacks
If we implement this feature, what are drawbacks and disadvantages of this approach?

### Acceptance Criteria
What does it take to make this feature producation ready? Please take the time to think about:
* how would you rollout this feature and rollback if it causes harm?
* Test Roadmap: what kinds of tests do we want to ensure a good user experience?
* observability: Do users need to get insights into the inner workings of that feature?
* monitoring: How can users tell whether the feature is working as expected or not?
              can we provide dashboards, metrics, reasonable SLIs/SLOs
              or example alerts for this feature?
* troubleshooting: How would users want to troubleshoot this particular feature?
                   Think about different failure modes of this feature.

## Alternatives
What alternatives do we have and what are their pros and cons?


