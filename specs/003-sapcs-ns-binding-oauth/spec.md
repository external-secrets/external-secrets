# Feature Specification: SAP Credential Store — Namespace, BTP Binding, OAuth & E2E Tests

**Feature Branch**: `003-sapcs-ns-binding-oauth`
**Created**: 2026-07-14
**Status**: Draft
**Input**: User description: "Make namespace configurable per ExternalSecret (or support both per-store and per-secret). Support using a BTP Service Binding Secret directly as the source of connection details. Ensure robust OAuth token caching and refresh. Add end-to-end integration tests against a live SAP Credential Store instance (or a representative test environment). Document authentication flows, namespace behavior, and expected Service Binding contents in detail."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Per-ExternalSecret Namespace Override (Priority: P1)

A platform engineer creates multiple `ExternalSecret` resources that each need to read credentials from different namespaces within the same SAP Credential Store instance. Today the namespace is fixed on the `SecretStore` / `ClusterSecretStore`, so every secret in that store shares one namespace. The engineer wants to override the namespace on a per-`ExternalSecret` basis without having to create a separate store per namespace.

**Why this priority**: Unblocking multi-namespace usage is the most direct enhancement to day-to-day usability and is a prerequisite for any team that segments credentials by environment or application domain.

**Independent Test**: Deploy a `ClusterSecretStore` with a default namespace and two `ExternalSecret` resources each specifying a different namespace override; verify each resolves credentials from its respective namespace.

**Acceptance Scenarios**:

1. **Given** a `ClusterSecretStore` with a default namespace `"shared"`, **When** an `ExternalSecret` specifies a per-secret namespace override `"team-a"`, **Then** the provider fetches the credential from namespace `"team-a"` instead of `"shared"`.
2. **Given** an `ExternalSecret` with no namespace override, **When** it syncs, **Then** the provider uses the namespace defined on the `SecretStore` as the default.
3. **Given** an `ExternalSecret` with an empty-string namespace override, **When** it syncs, **Then** the provider treats it as "no override" and falls back to the store-level namespace.
4. **Given** the namespace field is set on both the store and the `ExternalSecret`, **When** the `ExternalSecret` value is non-empty, **Then** the per-secret value takes precedence.

---

### User Story 2 — BTP Service Binding Secret as Connection Source (Priority: P1)

A developer running the External Secrets Operator on a BTP Kyma runtime already has a Kubernetes `Secret` containing a BTP Service Binding (injected by the SAP BTP Operator). Instead of copying credentials into the `SecretStore` spec manually, they want to point the store directly at that existing binding secret.

**Why this priority**: BTP Service Bindings are the standard credential delivery mechanism on Kyma/BTP; supporting them natively removes a manual copy step that is error-prone and complicates rotation.

**Independent Test**: Create a `ClusterSecretStore` referencing a BTP binding secret by name and namespace; sync an `ExternalSecret`; confirm it fetches from the Credential Store without any manually specified `clientId`, `clientSecret`, or `tokenUrl`.

**Acceptance Scenarios**:

1. **Given** a `ClusterSecretStore` with a `serviceBindingSecretRef` pointing to a valid BTP Service Binding `Secret`, **When** the operator starts, **Then** it reads `clientId`, `clientSecret`, `tokenUrl`, and `url` from the binding secret without error.
2. **Given** the binding secret is updated with new credentials (rotation), **When** the next reconcile runs, **Then** the provider uses the refreshed values.
3. **Given** both `serviceBindingSecretRef` and explicit credential fields are set, **When** the operator initializes the provider, **Then** `serviceBindingSecretRef` takes precedence and a warning is logged about the ignored inline credentials.
4. **Given** the referenced binding secret does not exist, **When** the store is reconciled, **Then** the store transitions to an error condition with a human-readable status message.
5. **Given** required fields are missing from the binding secret (e.g., `tokenUrl` is absent), **When** the store is reconciled, **Then** the store transitions to an error condition listing the missing fields.

---

### User Story 3 — Robust OAuth Token Caching and Refresh (Priority: P2)

An operations team runs hundreds of `ExternalSecret` reconciles per minute. Each reconcile must not trigger a new OAuth token request; instead, a valid cached token should be reused until it nears expiry, and the provider should refresh transparently without causing reconcile failures.

**Why this priority**: Without caching, high-frequency reconciles would exhaust OAuth rate limits and cause unnecessary latency. This story underpins the reliability of all other stories.

**Independent Test**: Trigger 100 rapid reconciles pointing at a store; verify that the number of token endpoint calls is bounded (no more than a small constant for the duration of a short-lived token), and that reconciles continue successfully across a token expiry boundary.

**Acceptance Scenarios**:

1. **Given** a valid cached token with more than the refresh-before-expiry window remaining, **When** multiple reconciles run concurrently, **Then** only the first reconcile fetches a new token; all others reuse the cached token.
2. **Given** a cached token within the refresh window (e.g., < 60 seconds to expiry), **When** the next reconcile runs, **Then** the provider proactively refreshes the token before using it for an API call.
3. **Given** a token refresh call fails but the existing token is still valid, **When** the reconcile runs, **Then** the provider continues using the existing token and retries refresh on the next reconcile.
4. **Given** a token refresh call fails and the token has expired, **When** the reconcile runs, **Then** the `ExternalSecret` transitions to an error state with a clear message identifying the OAuth failure.
5. **Given** multiple stores sharing the same OAuth credentials reconcile concurrently, **When** the token is not yet cached, **Then** the cache is populated by a single token fetch and reused by all concurrent callers (no duplicate requests).

---

### User Story 4 — End-to-End Integration Tests (Priority: P2)

A contributor wants to validate that the SAP Credential Store provider works correctly against a real (or representative) SAP Credential Store endpoint before merging changes. Unit tests already cover individual functions, but no test exercises the full flow from `ExternalSecret` creation to Kubernetes `Secret` population against a live or mock service.

**Why this priority**: Integration tests are the safety net that catches regressions that mocks miss; they are required before the feature can be declared production-ready.

**Independent Test**: Run the e2e suite (e.g., `make e2e-sapcs`) in a CI environment with appropriate secrets injected; confirm all test cases pass and the suite can be run locally against a live instance using environment variables.

**Acceptance Scenarios**:

1. **Given** valid SAP CS credentials in environment variables, **When** the e2e suite runs, **Then** it creates an `ExternalSecret` and verifies the populated Kubernetes `Secret` matches the value stored in SAP CS.
2. **Given** an `ExternalSecret` referencing a namespace override, **When** the e2e suite runs, **Then** it verifies the correct namespace-scoped credential is retrieved.
3. **Given** a `ClusterSecretStore` using a BTP binding secret, **When** the e2e suite runs, **Then** it verifies credentials are resolved from the binding without inline credentials.
4. **Given** an invalid or missing key reference, **When** the e2e suite runs, **Then** the `ExternalSecret` transitions to a failed state and the test asserts the correct error message.
5. **Given** the SAP CS endpoint is unreachable, **When** the e2e suite runs, **Then** the test asserts a connection-failure error condition on the store.

---

### User Story 5 — Comprehensive Documentation (Priority: P3)

A new adopter wants to onboard the SAP Credential Store provider on their BTP/Kyma cluster. They need a single place to understand: how OAuth authentication works, what fields a BTP Service Binding secret must contain, how namespace configuration interacts between stores and individual secrets, and how to run the integration tests.

**Why this priority**: Without clear documentation, all the above features are effectively invisible to new users; documentation is the multiplier for adoption.

**Independent Test**: A person unfamiliar with the provider follows the documentation from scratch on a BTP Kyma cluster and successfully syncs their first credential within 30 minutes.

**Acceptance Scenarios**:

1. **Given** the updated provider documentation, **When** a user reads the authentication-flow section, **Then** they can identify every OAuth step: token URL discovery, client credentials grant, token caching, and refresh.
2. **Given** the documentation on Service Binding contents, **When** a user inspects their BTP-injected secret, **Then** they can map each JSON key to the required field in the `ClusterSecretStore` spec.
3. **Given** the namespace configuration section, **When** a user sets up both a store-level and a per-secret namespace, **Then** they understand the precedence rules and can predict the behavior without running the operator.
4. **Given** the integration test guide, **When** a contributor wants to run e2e tests locally, **Then** the guide lists all required environment variables and explains how to obtain them from a BTP service binding.

---

### Edge Cases

- What happens when the namespace in the SAP CS API path contains special characters or is empty after trimming?
- How does the system behave when the BTP binding secret is in a different Kubernetes namespace than the `ClusterSecretStore`?
- What happens if the token cache is stale because the system clock was skewed?
- How are concurrent requests handled if the cache entry is being refreshed while another goroutine tries to read it?
- What if the BTP Service Binding contains additional unrecognized fields (forward compatibility)?
- What if a `SecretStore` (namespaced) and a `ClusterSecretStore` reference the same binding secret but are operated by different controller replicas?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The provider MUST support a per-`ExternalSecret` namespace field that, when set to a non-empty value, overrides the namespace defined on the parent `SecretStore` or `ClusterSecretStore`.
- **FR-002**: The provider MUST fall back to the store-level namespace when no per-secret namespace is specified or the per-secret field is empty.
- **FR-003**: The `SecretStore` / `ClusterSecretStore` spec MUST accept a `serviceBindingSecretRef` field (name + namespace) that points to a Kubernetes `Secret` containing a BTP Service Binding.
- **FR-004**: When `serviceBindingSecretRef` is set, the provider MUST derive `clientId`, `clientSecret`, `tokenUrl`, and the SAP CS `url` from the referenced secret, without requiring any inline credential fields.
- **FR-005**: When `serviceBindingSecretRef` is set alongside inline credential fields, the binding ref MUST take precedence and a warning MUST be emitted.
- **FR-006**: The provider MUST cache OAuth access tokens in memory, keyed by credential identity, and reuse them across reconciles until they are within a configurable expiry window (default: 60 seconds before expiry).
- **FR-007**: The provider MUST proactively refresh a token when it is within the expiry window, before using it for an API call.
- **FR-008**: If a proactive refresh fails but the existing token is still valid, the provider MUST continue using the existing token and attempt a refresh on the next reconcile.
- **FR-009**: If a refresh fails and the token has expired, the provider MUST surface an error on the `ExternalSecret` with a message that identifies the OAuth failure.
- **FR-010**: The token cache MUST be safe for concurrent use (multiple goroutines reconciling simultaneously must not race on cache reads/writes).
- **FR-011**: The repository MUST include an end-to-end integration test suite exercisable against a live or mock SAP Credential Store, covering at minimum: basic secret fetch, namespace override, BTP binding secret resolution, missing-key error, and connection failure.
- **FR-012**: The integration tests MUST be runnable using environment variables that supply SAP CS credentials, enabling execution both locally and in CI without code changes.
- **FR-013**: Provider documentation MUST cover: OAuth authentication flow (token URL discovery, grant type, caching, refresh), Service Binding secret structure and required keys, namespace precedence rules (store-level vs. per-secret), and how to run the integration tests.

### Quality & Operational Requirements

- **QR-001**: The feature MUST follow existing repository conventions for APIs, field names, events, metrics, logs, and package structure unless an explicit exception is approved.
- **QR-002**: The feature MUST define the automated test coverage required to verify the behavior change.
- **QR-003**: The feature MUST define documentation, examples, or generated artifacts that need to change with the implementation.
- **QR-004**: If the feature can affect latency, throughput, memory, reconcile cost, or external API usage, it MUST define measurable performance expectations.
- **QR-005**: Any change that handles credential material, calls external APIs, modifies RBAC permissions, or adds new dependencies MUST document how it satisfies security and compliance requirements: no credential logging, TLS enforcement, least-privilege RBAC, and CVE-clean dependencies.

### Key Entities

- **SecretStore / ClusterSecretStore**: The resource holding provider-level connection configuration; extended with `serviceBindingSecretRef` and a default `namespace` field.
- **ExternalSecret**: The per-workload resource that requests one or more credentials; extended with an optional namespace override field scoped to the SAP CS provider.
- **BTP Service Binding Secret**: A Kubernetes `Secret` produced by the SAP BTP Operator containing a service binding with fields: `clientid`, `clientsecret`, `url`, `tokenurl` (mTLS variants with `certificate`/`key` are out of scope for this iteration).
- **OAuth Token Cache**: An in-memory, concurrency-safe structure storing tokens keyed by credential identity with associated expiry timestamps.
- **SAP Credential Store Namespace**: A logical partition within a SAP CS service instance; acts as a path segment in API calls, resolved from the store-level default or the per-secret override.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can configure separate SAP CS namespaces for individual `ExternalSecret` resources without creating additional `SecretStore` resources — verified by a passing acceptance test covering at least two distinct namespaces from one store.
- **SC-002**: Configuring a `ClusterSecretStore` using only a BTP Service Binding `Secret` reference (no inline credentials) results in successful credential retrieval — verified by the integration test suite.
- **SC-003**: Under a workload of 100 concurrent reconciles sharing the same credentials, the number of OAuth token-endpoint calls does not exceed 3 for any 5-minute window with a token lifetime of at least 5 minutes — verified by an integration or load test.
- **SC-004**: Zero reconcile failures attributable to token expiry occur during normal operation (proactive refresh eliminates mid-reconcile token expiry) — verified by a time-skipping unit test and confirmed in e2e.
- **SC-005**: The end-to-end test suite completes (pass or documented failure) in under 5 minutes on standard CI infrastructure.
- **SC-006**: A user unfamiliar with the provider can successfully sync their first credential on BTP/Kyma by following only the updated documentation — validated by a documentation review against the acceptance scenarios in User Story 5.

## Assumptions

- The SAP Credential Store API uses the OAuth 2.0 client credentials grant as used in the existing provider implementation; mTLS-based authentication is out of scope for this iteration.
- The BTP Service Binding `Secret` produced by the SAP BTP Operator always contains at minimum: `clientid`, `clientsecret`, `tokenurl`, and `url` as top-level keys or nested under a `credentials` key.
- The External Secrets Operator controller has RBAC permission to read `Secret` resources in the namespace where the BTP binding secret resides; if not, extending the ClusterRole is a documented prerequisite.
- Integration tests are opt-in (gated by an environment variable or build tag) and not run as part of the default unit-test suite, to avoid requiring live credentials in every CI run.
- mTLS-based BTP Service Bindings (using `certificate` + `key` instead of `clientsecret`) are explicitly out of scope and noted as a future extension point in documentation.
- The per-secret namespace override is expressed via the `ExternalSecret` `spec.data[].remoteRef` structure (or an equivalent provider-specific extension); the exact field name will be aligned with ESO conventions during design.
