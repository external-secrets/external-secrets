```yaml
---
title: PushSecret metadata
version: v1alpha1
authors: Moritz Johner
creation-date: 2023-08-25
status: draft
---
```

# PushSecret Metadata

[#2600](https://github.com/external-secrets/external-secrets/pull/2600) introduced a new feature that allows users to pass arbitrary `metadata` to the provider.

The data is arbitrary json/yaml and can be anything.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ...
  data:
    - match:
        secretKey: key1
        remoteRef:
          remoteKey: test1
      metadata:
        annotations:
          key1: value1
        labels:
          key1: value1

```

Here is a overview of current implementations of PushSecret metadata:

```yaml
# AWS Parameter Store
# more to come in https://github.com/external-secrets/external-secrets/pull/3581
parameterStoreType: "..."
parameterStoreKeyID: "..."
```

```yaml
# GCP Secrets Manager
labels: {}
annotations: {}
```

```yaml
# AWS Secrets Manager
secretPushFormat: "..."
```

## Problem Description

We will never be able to make disruptive changes, we can only append to the existing structure.

**Why is that a problem?**

It limits our ability to fix mistakes that have been merged and released. Having an `apiVersion` field would allow us decode the metadata differently and apply the appropriate logic in a code branch. 

This would simplify fixing simple mis-nomers or doing large-scale refactorings in the future. 

ESO is a community based project and relies on contributions from different backgrounds and experience levels. As a result, the approach and perspective to a solution highly depends
on the contributor and the reviewer. We will eventually have to align the structure or naming of metadata across providers once we see patterns emerge.

## Proposed Solution

I would propose to wrap the unstructured metadata in a Kubernetes *alike* resource containing an `apiVersion`, `kind` and `spec`. 

#### 1. Kubernetes Provider Example

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ...
  data:
    - match:
        secretKey: key1
        remoteRef:
          remoteKey: test1
      metadata:
        apiVersion: kubernetes.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          sourceMergePolicy: Merge
          targetMergePolicy: Merge
          labels:
            color: red
          annotations:
            yes: please
```

#### 2. AWS Secrets Manager Example

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ...
  data:
    - match:
        secretKey: key1
        remoteRef:
          remoteKey: test1
      metadata:
        apiVersion: secretsmanager.aws.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          secretFormat: binary # string
```

#### 3. AWS Parameter Store Example

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ...
  data:
    - match:
        secretKey: key1
        remoteRef:
          remoteKey: test1
      metadata:
        apiVersion: parameterstore.aws.external-secrets.io/v1alpha1
        kind: PushSecretMetadata
        spec:
          tier: "Advanced"
          type: "StringList"
          keyID: "arn:..."
          policies: 
            - type: "ExpirationNotification"
              version: "1.0"
              attributes: 
                before: "15"
                unit: "Days"
```

**PROS**
- familiar structure for Kubernetes users, other projectes use that pattern already
- we may be able to re-use existing tooling, e.g. for validating the structure and generating documentation

**CONS**
- may confuse users if they encounter a nested custom resource
- a little bit of boilerplate to chew through


### What would we do with the existing implementations?

We should keep them as a backward compatible measure for the `v1alpha1` stage and remove them with the `v1beta1` release. We can remove them from the documentation right away and only document the "new" scheme. The old scheme is still accessible through the version switch in the docs. This allows us to slowly direct users to the new scheme.

With a PushSecret `v1beta1` we can consider removing those APIs.


## Alternatives

The minimum would be to have a `version` field which provides a hint for decoding the structure in `spec`. That is technically enough to meet the requirements outlined above.


```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-example
spec:
  # ...
  data:
    - match:
        secretKey: key1
        remoteRef:
          remoteKey: test1
      metadata:
        version: kubernetes/v1alpha1
        spec:
          sourceMergePolicy: Merge
          targetMergePolicy: Merge
          labels:
            color: red
          annotations:
            yes: please
```

**PROS**
- more concise, less boilerplate

**CONS**
- no ability to directly re-use existing tooling
