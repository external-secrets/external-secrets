```yaml
---
title: Adding Cluster External Secrets
version: v1alpha1
authors: Daniel "ADustyOldMuffin" Hix
creation-date: 2020-09-01
status: draft
---
```

# Adding Cluster External Secrets

## Table of Contents

<!-- toc -->
- [Adding Cluster External Secrets](#adding-cluster-external-secrets)
  - [Table of Contents](#table-of-contents)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
    - [API](#api)
    - [Behavior](#behavior)
    - [Drawbacks](#drawbacks)
    - [Acceptance Criteria](#acceptance-criteria)
  - [Alternatives](#alternatives)
<!-- /toc -->


## Summary
This would provide a way to template an `ExternalSecret` and based on a namespace selector in the CRD it would generate `ExternalSecret`s in matching namespaces.

## Motivation
It's a pain point to have to create a `Secret`/`ExternalSecret` in every namespace where it's needed, and this would provide a way to to do this easily. Another motivation is the possible creation of a Kubernetes secret provider which would provide an `ExternalSecret` from a secret already located in cluster. This in combination with that provider could provide a way to sync a secret across namespaces with a single provider call.

### Goals
To provide a way to deploy multiple `ExternalSecret`s with a single CRD.

### Non-Goals
Lower provider calls based on the created `ExternalSecrets` or manage their sync.

## Proposal

### User Stories
As an ESO User I would like to create the same `Secret`/`ExternalSecret` in multiple namespaces.

### API
``` yaml
apiVersion: external-secrets.io/v1alpha1
kind: ClusterExternalSecret
metadata:
  name: testing-secret
spec:
  # The selector used to select the namespaces to deploy to
  namespaceSelector:
    matchLabels:
      foo: bar
  # This spec is the exact same spec as an ExternalSecret, and is used as a template
  externalSecretSpec:
    refreshInterval: "15s"
    secretStoreRef:
      name: some-provider
      kind: ClusterSecretStore
    target:
      name: my-cool-new-secret
      creationPolicy: Owner
    data:
    - secretKey: my-cool-new-secret-key
      remoteRef:
        key: test
        property: foo
```

### Behavior
When created the controller will find namespaces via a label selector, and matching namespaces will then have an `ExternalSecret` deployed to them.

Edge cases are, 

1. namespaces being labeled after creation - currently handled via a re-queue interval, but the interval is a tad high and the changes won't take place right away. --
This has been handled by adding a refreshInterval to the spec which can be defined and controls when the controller is requeued.

1. Template being changed after deployment - Handled via the `createOrUpdate` function which should reconcile the `ExternalSecrets`

### Drawbacks
This will incur a high load on providers as it will create N number of `ExternalSecret`s which will also poll the provider separately. This can be fixed in two ways,

- In this CRD by adding the ability to reference an existing secret to replicate instead of creating `ExternalSecret`s, but this is not the "spirit" of the CRD and is less of a `ClusterExternalSecret` and more of a `ClusterSecret`. This is not the preferred way.

- The creation of a new Kubernetes Provider which will allow for the targeting of secrets in Kubernetes for `ExternalSecret`s

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

For this to be production ready it will need to be tested to ensure the expected behavior occurs, specifically around edge cases like

- Adding labels after creation of resource
- Changing of created `ExternalSecret`s
- Cleanup of `ExternalSecret`s when resources is deleted
- Deletion of owned resource
- Removal of label from namespace after `ExternalSecret` is created

Everything else is on the `ExternalSecret` and not the `ClusterExternalSecret` and troubleshooting would be the same.

## Alternatives

Adding a namespace selector to the regular `ExternalSecret`, but this would cause issues since it's not cluster scoped, and can't use "owned by" which would cause issues for cleanup.


