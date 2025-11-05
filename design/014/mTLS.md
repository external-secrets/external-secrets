# mTLS & Service Discovery for Out-of-Process Providers

## Problem Description

ESO Core needs to establish a secure and authenticated communication channel with out-of-process providers. It has to be encrypted in transit because we transmit secrets over the network. It must to be authenticated, otherwise a malicious actor could call GetSecret() to retrieve secrets and take advantage over the out-of-process provider's service account to read secrets or other resources within the cluster.

We have to take the following into consideration:

1. How do we establish a secure connection with a provider?
2. How do we discover providers within a Kubernetes cluster?
3. Do we support providers outside of a Kubernetes cluster? How?
4. How do we handle certificate rotation?

## Context

Out-of-process providers run in separate pods and communicate with ESO Core via gRPC. We need a certificate management system that:

- Provides mutual authentication between ESO Core and providers
- Works with in-cluster provider deployments
- Minimizes configuration burden on users and provider developers
- Supports automatic certificate rotation
- Maintains security by default

We do want to integrate with `cert-manager` eventually, but this is out of scope for the proposal.

## Key Decisions

### 1. mTLS Distribution Model: Push vs. Pull

**Decision: Push Model**

ESO Core generates and distributes certificates to provider namespaces.

#### Architecture

```
User labels Service → ESO discovers Service → ESO generates certificates → 
ESO creates Secret in provider namespace → Provider mounts Secret → 
Provider starts with mTLS → ESO connects with mTLS
```

#### Push Model

**How it works:**
- ESO acts as Certificate Authority
- ESO generates server certificates for the provider
- ESO generates client certificates for ESO core to authenticate with a provider
- ESO creates secrets in provider namespaces which container the CA and server certificates
- Providers mount secrets and start gRPC servers
- ESO connects using client certificates

**Pros:**
- Simple provider implementation (mount secret, start server)
- No circular dependencies (providers don't authenticate before receiving certs)
- Centralized certificate management and rotation
- Providers restart independently
- ESO controls security policy
- Uses standard Kubernetes secret mounting

**Cons:**
- Requires cross-namespace secret write permissions
- Secrets stored at rest in kube-apiserver/etcd
- Providers must implement hot certificate reload

**Why chosen:** Simplicity for provider developers, no chicken-and-egg authentication problem, leverages Kubernetes primitives.

#### Pull Model

**How it works:**
- Providers generate Certificate Signing Requests (CSRs) at startup
- Providers request certificates from ESO
- ESO signs and returns certificates

**Pros:**
- Certificates contain actual runtime addresses (DNS SANs)
- Providers control their own keys
- Short-lived certificates possible
- Dynamic addressing

**Cons:**
- Complex provider implementation
- Circular dependency: how does provider authenticate to ESO before it has certs?
- Provider startup depends on ESO availability
- Requires ESO to expose certificate signing API
- Additional attack surface
- Non-standard pattern in Kubernetes

**Why rejected:** Circular dependency problem, excessive complexity, tight coupling between ESO and providers.

---

### 2. mTLS Discovery Method: How ESO Finds Providers

**Decision: Label-Based Discovery**

Services are labeled with `external-secrets.io/provider: "true"`.

#### Label-Based Discovery

**Configuration:**
```yaml
apiVersion: v1
kind: Service
metadata:
  name: provider-aws
  namespace: external-secrets-system
  labels:
    external-secrets.io/provider: "true"
spec:
  ports:
    - port: 8080
```

**Pros:**
- Explicit contract (no magic)
- Kubernetes-native pattern
- Dynamic discovery (no restarts needed)
- Clear intent (label means "manage my certs")
- Works with any deployment method
- Supports multiple providers per namespace

**Cons:**
- Users must remember to label services
- No validation of label correctness

**Why chosen:** Explicit contract over implicit behavior, Kubernetes-native, clear intent.

#### Alternative 1: Parse Address from Provider Resource

**How it works:**
Extract namespace from `Provider.spec.address` (e.g., `provider-aws.external-secrets-system.svc:8080`).

**Pros:**
- Zero configuration
- Fully automatic

**Cons:**
- Only works for in-cluster services
- Too "magical" - no explicit contract
- Fails on non-standard addresses
- Doesn't handle multiple Providers pointing to same pod

**Why rejected:** Too implicit, preference for explicit contracts.

#### Alternative 2: Static Controller Configuration

**How it works:**
Configure providers via controller flags or ConfigMap.

**Pros:**
- Simple, centralized
- Explicit configuration

**Cons:**
- Static (requires controller restart)
- Manual maintenance
- Doesn't scale with dynamic deployments
- Not Kubernetes-native

**Why rejected:** Not dynamic, requires manual maintenance.

---

### 3. Scope: In-Cluster vs. External Providers

**Decision: In-Cluster Only**

Automatic certificate management only supports providers running inside the Kubernetes cluster.

We will support out-of-cluster providers, but we don't manage mTLS credentials for them.

**Rationale:**
- ESO focuses on in-cluster architecture
- External providers are edge cases
- External providers add significant complexity
- Users can implement their own CA infrastructure for external providers (cert-manager, etc.) and need to take care of distributing the CSRs/Certificates anyway

**Out of scope:**
- Providers running outside the cluster
- Providers in different clusters
- Providers with custom CA requirements

---

### 4. Hot Certificate Reload

**Decision: Required**

Providers must reload certificates without restarting when secrets are updated.

**Implementation requirement:**
```go
// Watch certificate files for changes
// Use tls.Config.GetCertificate callback for dynamic loading
// Reload certificates in-memory when files change
// Use fsnotify or similar to detect file changes
```

**Rationale:**
- Avoids connection disruption during rotation
- Enables seamless certificate updates
- Standard practice for production systems

**Metrics requirement:**
- `provider_certificate_hot_reload_total`
- `provider_certificate_hot_reload_failures_total`

## Certificate Structure

### Secret Distribution

**In ESO Namespace:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: eso-provider-ca-internal
  namespace: external-secrets-system
data:
  ca.crt: <CA certificate>
  ca.key: <CA private key>  # ONLY in ESO namespace
```

**In Provider Namespaces:**
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: external-secrets-provider-tls
  namespace: <provider-namespace>
data:
  ca.crt: <CA certificate>
  tls.crt: <Server certificate with DNS SANs>
  tls.key: <Server private key>
  # note: no client certs/keys!
```

**Security:** 
1. CA private key is never distributed to provider namespaces.
2. client certificates/keys are never distributed to provider namespaces.

### Certificate Validity Periods

| Certificate Type   | Validity | Rotation Lookahead |
| ------------------ | -------- | ------------------ |
| CA Certificate     | 1 year   | 60 days            |
| Server Certificate | 90 days  | 35 days            |
| Client Certificate | 90 days  | 35 days            |

**Reconciliation Interval:** 10 minutes

### DNS Subject Alternative Names (SANs)

E.g. for service `provider-aws` in namespace `provider-system`:

```
DNS SANs in server certificate:
  - provider-aws
  - provider-aws.provider-system
  - provider-aws.provider-system.svc
  - provider-aws.provider-system.svc.cluster.local
```

Covers all Kubernetes DNS resolution patterns. The `cluster.local` must be configurable, as some clusters have custom cluster domains.

## Certificate Lifecycle

### Controller: Rotation Triggers

1. Service labeled for first time (initial generation)
2. Certificate expires within 35 days (rotation)
3. Service DNS changes (regeneration with new SANs)
4. Secret deleted (recreation)
5. CA certificate rotated (regenerate all)
6. Leaf certificate doesn't match CA (regeneration)

### Rotation Strategy

- CA certificate preserved when valid
- ESO verifies leaf certificates were signed by current CA
- If CA changes or is deleted, regenerate all leaf certificates
- Providers hot-reload from mounted volume (no restart)

## Security Model

### Trust Boundaries

1. ESO acts as Certificate Authority for provider communication only. The CA certificate for out-of-process Providers must be distinct from any other certificates used by ESO (e.g. conversion/validating webhook)
2. Providers present server certificates to ESO (ESO is the client)
3. ESO presents client certificates to providers
4. Mutual authentication is enforced (mTLS required)
5. Providers retrieve secrets from external APIs using their own RBAC
6. ESO does not transmit secrets over gRPC - providers fetch and return them

### Why ESO Can Trust Providers

ESO can trust any provider. ESO can trust anyone who is able to create a service object with the appropriate labels.

- Providers do not have access to ESO's secrets
- Providers only return data they fetch from external APIs
- Provider RBAC is separate (AWS IAM roles, Kubernetes service accounts, etc.)
- ESO signing **server** certificates for providers doesn't grant additional privileges
- mTLS ensures only authenticated providers can communicate

**However:** We must not distribute client certificates anyhwere, as this will allow anyone with access to a client certificate + key to fetch secrets from a provider.

## RBAC Requirements

ESO controller requires:

```yaml
# Watch services cluster-wide
- apiGroups: [""]
  resources: ["services"]
  verbs: ["get", "list", "watch"]

# Manage secrets in any namespace
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["create", "update", "patch", "get", "list", "watch"]
```

**Security consideration:** Cross-namespace secret write is privileged. ESO only writes to the fixed secret name `external-secrets-provider-tls` and only in namespaces with labeled services.

## User Experience

### For Provider Users (Zero Config)

1. Deploy provider with labeled service:
```yaml
labels:
  external-secrets.io/provider: "true"
```

2. Create Provider resource:
```yaml
apiVersion: external-secrets.io/v1
kind: Provider
metadata:
  name: my-aws
spec:
  address: provider-aws.provider-system.svc:8080
```

**That's it.** Certificates are automatic.

### For Provider Developers

1. Label your service with `external-secrets.io/provider: "true"`

2. Mount secret in pod:
```yaml
volumeMounts:
  - name: certs
    mountPath: /etc/provider/certs
volumes:
  - name: certs
    secret:
      secretName: external-secrets-provider-tls
```

3. Configure gRPC server to use certs from `/etc/provider/certs/`

4. Implement hot certificate reload (required)

5. Expose Prometheus metrics

## Monitoring

### Required Metrics

```
# Certificate expiration (seconds until expiry)
eso_provider_certificate_expiry_seconds{namespace, service}

# CA certificate expiration
eso_ca_certificate_expiry_seconds

# Certificate rotations
eso_provider_certificate_rotation_total{namespace, service, reason}

# Rotation failures
eso_provider_certificate_rotation_failures_total{namespace, service}

# Hot reload events (in provider pods)
eso_provider_certificate_hot_reload_total{namespace, service}

# Hot reload failures (in provider pods)
eso_provider_certificate_hot_reload_failures_total{namespace, service}
```

### Recommended Alerts

- Certificate expiring in <3 days
- Certificate rotation failure
- Hot reload failure
- CA certificate expiring in <30 days

## Open Questions

### 1. gRPC connection handling during rotation

**Options:**
- A: Automatically re-establish connections when certificates rotate
- B: Let existing connections complete, new connections use new certs
- C: Force reconnect on certificate rotation

**Impact:** Affects connection stability during rotation.

### 2. Graceful certificate rollover

Should ESO briefly accept both old and new certificates during rotation?

**Pros:**
- Smoother rotation experience
- Reduces connection errors

**Cons:**
- More complex implementation
