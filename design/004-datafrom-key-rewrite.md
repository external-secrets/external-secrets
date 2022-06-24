```yaml
---
title: dataFrom key rewrite
version: v1alpha1
authors: 
creation-date: 2022-05-25
status: draft
---
```

# Datafrom Key Rewrite

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->


## Summary
DataFrom key rewrite aims to give users the possibility to rewrite secret keys before manipulating into a template. This feature is specially useful with the addition of `dataFrom.find` introduced in v0.5.0, as a way to allow users to rewrite secret keys in batches, without knowing the full content of the key.

## Motivation
It is hard to administer key creation without having weird names that convolutes with the path. It is natural to have a feature to allow secret Keys manipulation both from the lifecycle of the secret management and as part of the templating feature.

### Goals
- CRD Design for dataFrom key rewrite

### Non-Goals
Do not implement secret values rewrite - this should be done with templating - nor handle initial secret value conversion (see [#920](https://github.com/external-secrets/external-secrets/issues/920)). 

### Terminology
- Secret Key: the kubernetes secret key, aka the `string` in `map[string][]byte` in the codebase used to render Kubernetes Secrets.

## Proposal

Add a rewrite step after applying either `GetSecretMap` or `GetAllSecrets`, but before applying any conversion logic. This allows us to apply individual mappings to different `dataFrom` groups so they can be composed well together in combination with `data` (see example below).

### User Stories
1. As an user I want to be able to remove paths from my Secret Keys
2. As an user I want to be able to customize secret keys to have more meaningful value to the tenants consuming it.
3. As an user I want external-secrets to tell me if there is any rewrite conflict and not to overwrite secrets information.

### API
Proposed CRD changes:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: sample
  namespace: default
spec:
  refreshInterval: 1m
  target:
   name: foobar
  secretStoreRef:
    name: secretstore-sample
    kind: SecretStore
  dataFrom:
    - extract:
        key: /foo/bar
      rewrite:
      -  origin: my-known-key
         target: new_key_name
      -  origin: my-unknown-(.*)
         target: unknown-$1
    - find:
        conversionStrategy: Unicode
        path: "my/path/"
        name:
          regexp: "my-pattern"
      rewrite:
      -  origin: my/path/(.*)
         target: $1
      -  origin: creds/(db|system)/(.*)
         target: $2-$1
    - find:
        conversionStrategy: Default
        path: "my/path/"
        tags:
          env: dev
      rewrite:
      -  origin: my/path/(.*)
         target: dev-$1

```

### Behavior
After applying `GetAllSecrets` or `GetSecretMap`, a rewrite logic is applied by using regular expressions capture groups. This will convert the Secret Keys according to a given standard, possibly even up to a well known key for the `GetSecretMap`. After the key rewrite is done, the `ConversionStrategy` should be applied, and then the new collection of Secret Keys (`map[string][]byte`) can be rendered to template.

### Drawbacks

It would not be trivial to replace existing characters for new characters with this strategy. This could be implemented with a `replace` method afterwards.

### Acceptance Criteria
+ If multiple keys have the same name an error should happen to the user referencing the original Secret Keys.
+ A rewrite respects the regular expression limits
+ A rewrite is compatible with `dataFrom.extract` and `dataFrom.find`
+ It should be possible to remove paths with the `rewrite` operation.

## Alternatives
Add provider-specific ConversionStrategies, which would be painfully hard to maintain.