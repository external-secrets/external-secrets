```yaml
---
title: AWS S3 Provider
version: v1alpha1
authors: Hilton Samuel
creation-date: 2026-04-27
status: draft
---
```

# AWS S3 Provider

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary

Add S3 as a third service type in the existing AWS provider, allowing users to sync S3 objects into Kubernetes Secrets or ConfigMaps. This enables teams that store configuration (endpoints, timeouts, feature flags) or semi-sensitive data in encrypted S3 buckets to use ExternalSecrets as a unified sync mechanism.

## Motivation

Many organizations store non-secret configuration in S3 buckets encrypted with SSE-KMS. Today, syncing these into Kubernetes requires custom operators. The ExternalSecrets ecosystem already supports ConfigMap targets via `target.manifest`, but lacks an S3 source. Adding S3 completes the pipeline: **S3 object → ExternalSecret → ConfigMap/Secret**.

Common use cases:
- Application config files (JSON/YAML) stored in S3, synced as ConfigMaps
- Shared configuration across multiple tenants/namespaces
- Centralized config management with S3 versioning and audit trails
- Encrypted config that needs KMS-based access control via IAM roles

### Goals

- Support S3 as a source in the existing AWS provider (`service: S3`)
- Implement `GetSecret`, `GetSecretMap`, and `GetAllSecrets`
- Support S3 object versioning via `version` in remote refs
- Reuse existing AWS auth (IRSA, static creds, STS assume-role)
- Work with both Secret and ConfigMap targets (via `target.manifest`)

### Non-Goals

- S3 event-driven sync (push-based reconciliation) - out of scope, polling-based like other providers
- Multi-part uploads or streaming large objects
- S3 bucket creation or lifecycle management

## Proposal

Extend the AWS provider with a third service type `S3`. The implementation follows the same pattern as SecretsManager and ParameterStore - a new sub-package under `providers/v1/aws/s3/` that implements `SecretsClient`.

### User Stories

**Story 1: Sync a JSON config file as a ConfigMap**

As a platform engineer, I store application config as JSON files in an S3 bucket. I want to sync `s3://my-bucket/tenant-a/app-config` into a ConfigMap so my pods can mount it.

**Story 2: Sync individual keys from a JSON object**

As a developer, I have a JSON config in S3 with multiple keys. I want to extract specific keys (`database.host`, `cache.ttl`) into a ConfigMap using `dataFrom` with `property`.

**Story 3: Sync all configs under a prefix**

As a platform engineer, I want to sync all S3 objects under `configs/production/` into a single ConfigMap where each object key becomes a data key.

### API

Add `S3` to the `AWSServiceType` enum:

```go
// +kubebuilder:validation:Enum=SecretsManager;ParameterStore;S3
type AWSServiceType string

const (
    AWSServiceSecretsManager AWSServiceType = "SecretsManager"
    AWSServiceParameterStore AWSServiceType = "ParameterStore"
    AWSServiceS3             AWSServiceType = "S3"
)
```

Add S3-specific config to `AWSProvider`:

```go
type AWSProvider struct {
    // ... existing fields ...

    // S3 defines how the provider behaves when interacting with AWS S3.
    // +optional
    S3 *AWSS3 `json:"s3,omitempty"`
}

// AWSS3 defines S3-specific provider configuration.
type AWSS3 struct {
    // Bucket is the S3 bucket name.
    // +kubebuilder:validation:Required
    Bucket string `json:"bucket"`
}
```

**SecretStore example:**

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: s3-config-store
spec:
  provider:
    aws:
      service: S3
      region: us-east-1
      s3:
        bucket: my-config-bucket
      auth:
        jwt:
          serviceAccountRef:
            name: my-sa
```

**ExternalSecret - single object as ConfigMap:**

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-config
spec:
  refreshInterval: 5m
  secretStoreRef:
    name: s3-config-store
  target:
    name: app-config
    manifest:
      apiVersion: v1
      kind: ConfigMap
  data:
    - secretKey: config.json
      remoteRef:
        key: tenant-a/app-config.json
```

**ExternalSecret - extract JSON keys:**

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: db-config
spec:
  refreshInterval: 5m
  secretStoreRef:
    name: s3-config-store
  target:
    name: db-config
    manifest:
      apiVersion: v1
      kind: ConfigMap
  data:
    - secretKey: db_host
      remoteRef:
        key: tenant-a/app-config.json
        property: database.host
    - secretKey: cache_ttl
      remoteRef:
        key: tenant-a/app-config.json
        property: cache.ttl
```

**ExternalSecret - all objects under a prefix:**

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: all-configs
spec:
  refreshInterval: 5m
  secretStoreRef:
    name: s3-config-store
  target:
    name: all-configs
    manifest:
      apiVersion: v1
      kind: ConfigMap
  dataFrom:
    - find:
        path: configs/production/
        name:
          regexp: ".*\\.json$"
```

### Behavior

**`GetSecret(ref)`**
- `ref.Key` → S3 object key
- `ref.Version` → S3 version ID (optional; latest if omitted)
- `ref.Property` → JSON path extraction (dot-notation, e.g. `database.host`)
- Returns raw object bytes, or extracted value if `property` is set

**`GetSecretMap(ref)`**
- Fetches the S3 object, parses as JSON
- Returns `map[string][]byte` of top-level keys
- Error if object is not valid JSON

**`GetAllSecrets(find)`**
- `find.Path` → S3 prefix (e.g. `configs/production/`)
- `find.Name.RegExp` → filter object keys by regex
- Lists objects under prefix, fetches each, returns map keyed by object name (last path segment)

**`ValidateStore`**
- Require `s3.bucket` when `service: S3`
- Validate bucket name format

**Error handling:**
- `NoSuchKey` / `NoSuchBucket` → return not-found error (triggers ExternalSecret status condition)
- `AccessDenied` → return auth error
- Objects > 1MB → return size limit error (configurable, default 1MB)

**Edge cases:**
- Binary objects: returned as raw bytes; user is responsible for base64 handling via templates
- Empty objects: return empty byte slice, no error
- S3 delete markers: treated as not-found

### Drawbacks

1. **S3 is not a secret store** - this stretches the definition of "external secrets." However, the project already supports ConfigMap targets and non-sensitive data via `target.manifest`. S3 with SSE-KMS provides encryption at rest and IAM-based access control, which is comparable to ParameterStore for non-secret config.

2. **Polling-based only** - unlike a custom operator that can use S3 event notifications, ESO uses polling (`refreshInterval`). For config that changes rarely, this is acceptable. For real-time sync, users would need a separate mechanism.

3. **Object size** - S3 objects can be arbitrarily large. A size limit (default 1MB) prevents memory issues but may surprise users with large config files.

4. **N+1 queries for `GetAllSecrets`** - listing objects then fetching each individually. Acceptable for small config sets but not ideal for hundreds of objects.

### Acceptance Criteria

**Rollout:**
- Feature is available when `service: S3` is selected - no additional feature flag needed (it's a new service type, not a new behavior)
- Rollback: remove the S3 service type; existing SecretStores with `service: S3` will fail validation

**Tests:**
- Unit tests for S3 client (`GetSecret`, `GetSecretMap`, `GetAllSecrets`, `ValidateStore`)
- Unit tests for JSON property extraction
- Unit tests for error handling (not found, access denied, size limit)
- E2e tests: S3 object → Secret, S3 object → ConfigMap, S3 prefix → ConfigMap
- E2e tests: versioned object retrieval

**Observability:**
- Standard ESO metrics apply (`externalsecret_sync_calls_total`, `externalsecret_sync_calls_error`)
- S3 API call errors surface in ExternalSecret status conditions

**Monitoring:**
- ExternalSecret status conditions show sync state
- `refreshInterval` controls polling frequency

**Troubleshooting:**
- IAM permission errors: clear error message with required S3 actions (`s3:GetObject`, `s3:ListBucket`, `s3:GetObjectVersion`)
- Bucket/key not found: specific error messages
- Size limit exceeded: error with object size and limit

## Alternatives

### 1. Standalone S3 provider (separate from AWS)

Create `providers/v1/s3/` as an independent provider. Pros: clean separation, S3 is conceptually different from secret managers. Cons: duplicates AWS auth code, inconsistent with the existing pattern where AWS services are grouped.

**Decision:** Extend the existing AWS provider. Auth reuse is significant, and the pattern of multiple services under one provider is established.

### 2. Custom operator (current approach)

Build a purpose-built operator that watches S3 events and syncs to ConfigMaps. Pros: real-time sync, full control over sharing/tenant logic. Cons: not reusable by the community, maintenance burden, doesn't leverage ESO's reconciliation, templating, and policy features.

**Decision:** The custom operator remains useful for advanced use cases (event-driven sync, tenant sharing). The ESO provider covers the common case and benefits the broader community.

### 3. Use ParameterStore instead of S3

Store config values in SSM ParameterStore, which ESO already supports. Pros: no new code needed. Cons: 4KB/8KB parameter size limits, no native JSON file storage, different access patterns, higher cost at scale compared to S3.
