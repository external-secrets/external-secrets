# Threat Model for Out-of-Process Providers

## Overview

This document analyzes security threats for the out-of-process provider architecture where providers run in separate pods and communicate with ESO Core via gRPC over mTLS.

## Architecture Summary

**Components:**
- **ESO Core**: Kubernetes controller that reconciles ExternalSecret resources
- **Out-of-Process Providers**: Separate pods that fetch secrets from external APIs (AWS, Azure, etc.)
- **gRPC Communication**: mTLS-secured channel between ESO Core and providers
- **Certificate Authority**: ESO Core manages certificates for provider communication

**Key Properties:**
- Providers run in separate pods with isolated network policies
- ESO Core and providers run with different RBAC permissions
- ESO Core acts as CA and distributes certificates via Kubernetes secrets
- Providers fetch secrets from external APIs using their own credentials
- ESO Core does not have direct access to external secret systems

## Assets

What needs protection:

1. **Secrets in Transit**: Secrets returned by providers to ESO Core over gRPC
2. **TLS Certificates**: CA private key, server certificates, client certificates
3. **Provider Configuration**: Credentials and configuration stored in provider-specific CRs
4. **ESO Core Configuration**: Controller configuration and RBAC permissions
5. **External API Credentials**: AWS IAM roles, Azure service principals, etc. used by providers
6. **Kubernetes Resources**: ExternalSecret, Provider, and provider-specific custom resources

## Trust Boundaries

```
┌─────────────────────────────────────────────────────────────┐
│ Kubernetes Cluster                                          │
│                                                             │
│  ┌─────────────────┐                  ┌──────────────────┐  │
│  │ ESO Core        │  mTLS/gRPC       │ Provider Pods    │  │
│  │ Namespace       │◄────────────────►│ (various NS)     │  │
│  │                 │                  │                  │  │
│  │ - Controller    │                  │ - AWS Provider   │──┼──► AWS API
│  │ - CA Secret     │                  │ - Azure Provider │──┼──► Azure API
│  └─────────────────┘                  └──────────────────┘  │
│         │                                      │            │
│         │                                      │            │
│         ▼                                      ▼            │
│  ┌─────────────────┐                  ┌──────────────────┐  │
│  │ ESO Secrets     │                  │ Provider Secrets │  │
│  │ - CA private key│                  │ - TLS certs      │  │
│  └─────────────────┘                  │ - Config/creds   │  │
│                                       └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Trust Boundaries:**
1. Between ESO Core and Kubernetes API
2. Between ESO Core and provider pods (mTLS)
3. Between provider pods and external secret systems
4. Between namespaces (RBAC and network policies)
5. Between pod and mounted secrets

## Threat Actors

### External Attackers

**Capabilities:**
- Network access to cluster (varies by deployment)
- Ability to observe network traffic
- Social engineering or supply chain attacks

**Goals:**
- Steal secrets
- Compromise workloads
- Establish persistence

### Malicious Pod in Cluster

**Capabilities:**
- Limited RBAC permissions
- Access to pod network
- Access to secrets, mounted to the pod
- Ability to make API requests within assigned permissions

**Goals:**
- Escalate privileges
- Access secrets from other namespaces
- Intercept communication between ESO and providers

### Compromised Provider

**Capabilities:**
- Valid mTLS server certificates
- Access to external secret system credentials
- Ability to communicate with ESO Core

**Goals:**
- Return malicious secrets, potential DOS
- Exfiltrate secret requests
- Pivot to compromise other providers or ESO Core

### Compromised ESO Core

**Capabilities:**
- CA private key access
- Cross-namespace secret write permissions
- Ability to generate arbitrary certificates

**Goals:**
- Access all secrets across all providers
- Modify secrets in target namespaces
- Impersonate providers

## Threat Analysis

### T1: Man-in-the-Middle Attack on gRPC Communication

**Threat:** Attacker intercepts communication between ESO Core and provider to steal secrets in transit.

**Attack Vector:**
- Network position between ESO Core and provider pods
- ARP spoofing or DNS hijacking within cluster
- Compromised network infrastructure

**Mitigation:**
- ✅ mTLS enforced for all ESO-to-provider communication
- ✅ Mutual authentication prevents impersonation
- ✅ Certificate validation ensures proper identity
- ⚠️ NetworkPolicies should be configured to restrict communication paths

**Residual Risk:** Low (with proper NetworkPolicy configuration)

---

### T2: Certificate Authority Compromise

**Threat:** Attacker gains access to CA private key and can issue arbitrary certificates.

**Attack Vector:**
- Compromise ESO Core pod
- Direct access to ESO namespace secrets
- Kubernetes API server compromise

**Mitigation:**
- ✅ CA private key only stored in ESO namespace
- ✅ CA private key never distributed to provider namespaces
- ✅ Strict RBAC limits access to ESO namespace
- ⚠️ Consider using external system for CA key storage (KMS/HSM)

**Impact:** Critical - full compromise of provider authentication

**Residual Risk:** Medium (depends on cluster hardening)

---

### T3: Rogue Provider Registration

**Threat:** Attacker deploys malicious provider and gets ESO to communicate with it.

**Attack Vector:**
- Deploy service with `external-secrets.io/provider: "true"` label
- ESO generates certificates for the rogue service
- Create Provider resource pointing to rogue service
- Intercept or manipulate secret requests

**Mitigation:**
- ✅ Provider-specific configuration still required (can't serve AWS secrets without AWS credentials)
- ✅ RBAC controls who can create Provider resources
- ⚠️ Consider admission webhook to validate Provider resources
- ⚠️ Monitor for unexpected service labels

**Impact:** Low - attacker can't access external secrets without proper credentials. Provider impersonation may be a DOS vector.

**Residual Risk:** Low

---

### T4: Provider Impersonation

**Threat:** Attacker impersonates legitimate provider to return malicious secrets.

**Attack Vector:**
- Steal provider TLS certificates
- Deploy service at same address as legitimate provider
- Respond to ESO requests with malicious data

**Mitigation:**
- ✅ mTLS with certificate validation prevents impersonation
- ✅ Certificates tied to specific DNS names
- ✅ NetworkPolicies can restrict communication paths
- ⚠️ Certificate rotation limits window of compromised certs

**Residual Risk:** Low

---

### T5: Supply Chain Attack via Provider Dependencies

**Threat:** Malicious code in provider dependencies exfiltrates secrets or credentials.

**Attack Vector:**
- Compromised provider SDK
- Vulnerable dependencies in provider binary
- Backdoored base images

**Mitigation:**
- ✅ Providers built with minimal dependencies
- ✅ Each provider isolated - compromise doesn't affect others
- ⚠️ Dependency scanning and SBOM generation recommended
- ⚠️ Image signing and verification
- ⚠️ Regular security audits of provider code

**Impact:** High - provider can access external secret systems and have a lot of RBAC permissions within Kubernetes.

**Residual Risk:** Medium (ongoing supply chain risk)

---

### T6: Denial of Service via Certificate Exhaustion

**Threat:** Attacker creates many labeled services to exhaust ESO Core resources.

**Attack Vector:**
- Deploy numerous services with provider label
- ESO generates certificates for all services
- Resource exhaustion prevents legitimate operations

**Mitigation:**
- ⚠️ Rate limiting on certificate generation (not yet implemented)
- ⚠️ RBAC limits who can create services with eso-provider label
- ⚠️ Monitoring and alerting on certificate generation rate
- ✅ Certificate creation is managed by a separate pod. Core ESO day-to-day operations will not be impacted.

**Residual Risk:** Medium

---

### T7: Secrets at Rest Exposure

**Threat:** Attacker gains access to secrets stored in Kubernetes etcd.

**Attack Vector:**
- Direct etcd access
- Backup compromise
- Kubernetes API server vulnerability

**Mitigation:**
- ⚠️ Kubernetes encryption at rest (cluster configuration)
- ⚠️ Secure etcd access controls
- ⚠️ Regular key rotation
- ✅ Provider TLS certificates are short-lived (30 days)

**Impact:** Critical - all certificates and potentially secret data exposed

**Residual Risk:** Medium (depends on cluster configuration)

---

### T8: Malicious Secret Injection

**Threat:** Provider returns malicious or incorrect secrets to ESO Core.

**Attack Vector:**
- Compromised provider pod
- Vulnerability in provider code
- Misconfiguration in provider

**Mitigation:**
- ⚠️ No built-in validation of secret content (by design)
- ✅ Provider compromise doesn't grant access to ESO's secrets
- ✅ Isolation between providers limits blast radius
- ⚠️ External system auditing (e.g., CloudTrail for AWS)
- ⚠️ Secret validation at application level

**Impact:** Medium - workloads receive incorrect secrets

**Residual Risk:** Medium

---

### T9: RBAC Misconfiguration

**Threat:** Overly permissive RBAC allows unauthorized access to Provider resources or secrets.

**Attack Vector:**
- Misconfigured ClusterRole bindings
- Overly broad service account permissions
- Default namespaces with excessive permissions

**Mitigation:**
- ✅ Clear RBAC requirements documented
- ⚠️ Principle of least privilege enforcement recommended
- ⚠️ Regular RBAC audits
- ⚠️ Use of namespace-scoped resources where possible

**Residual Risk:** Medium (depends on user configuration)

---

### T10: Certificate Rotation Failure

**Threat:** Certificate rotation fails, causing service disruption or security degradation.

**Attack Vector:**
- ESO Core pod failure during rotation
- Bugs in rotation logic
- Resource constraints preventing secret updates

**Mitigation:**
- ✅ 35-day rotation lookahead provides recovery window
- ✅ Prometheus metrics for monitoring rotation status
- ⚠️ Automated alerts on rotation failures
- ⚠️ Manual intervention procedures documented

**Impact:** High - service disruption and potential security degradation

**Residual Risk:** Low

## Assumptions

This threat model assumes:

1. Kubernetes cluster is properly hardened (CIS benchmarks, etc.)
2. Network infrastructure is trusted within cluster
3. Kubernetes RBAC is properly configured
4. Users follow deployment best practices
5. External secret systems (AWS, Azure, etc.) have their own security controls
6. Workloads consuming secrets perform their own validation
