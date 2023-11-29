```yaml
---
title: PushSecret generator integration
version: v1alpha1
authors: Moritz Johner
creation-date: 2023-08-25
status: draft
---
```

# PushSecret Generator integration

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

This design document describes how `PushSecret` can leverage generators to generate short-lived credentials without the need of an intermediary `Secret` resource.

## Motivation

Currently, the process of using secure passwords and short-lived credentials within the External Secrets Operator involves multiple steps.
Users need to create an `ExternalSecret` resource to generate a value which is stored in a `Secret` resource. This Secret resource is then pushed to a provider using a `PushSecret` resource. However, this intermediary step adds unnecessary complexity and inconvenience to the workflow.

## Proposal

To simplify the workflow and enhance user experience, the proposal is to integrate generators directly into the `PushSecret` resource. This will allow users to specify a generator using a `generatorRef` within the PushSecret manifest. When the PushSecret reconciliation process occurs, a value will be generated using the specified generator. This generated value will be securely pushed to the provider and stored there.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
spec:
  selector:
    generatorRef:
      apiVersion: generators.external-secrets.io/v1alpha1
      kind: Password
      name: "my-password"
```

## Consequences

* **Simplified Workflow**: This change will simplify the process of generating and pushing secrets by eliminating the need for an intermediary Secret resource.

* **Enhanced Security**: As secrets are generated and pushed directly, there will be a reduction in potential vulnerabilities that may arise from the management of intermediary resources.

* **Increased Flexibility**: Integrating generators into PushSecrets allows for more customization and flexibility in generating secrets according to specific requirements.

* **Potential Learning Curve**: Users who are accustomed to the previous workflow may need to adapt to the new approach, which could require some learning and adjustment.

* **API and Documentation Update**: The API changes need to be well-documented to ensure users understand how to utilize the new feature effectively.


## Acceptance Criteria

* tests: controller tests for this new field should be sufficient
* the API changes need to be documented
    * API/CRD spec inline documentation
    * PushSecret API documentation
    * Guides section for `PushSecret` + `generator` functionality
