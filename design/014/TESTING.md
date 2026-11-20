# V2 Provider Testing Strategy

## Context

The v2 provider architecture introduces out-of-process providers communicating via gRPC. This architectural shift adds network communication, connection pooling, TLS/mTLS handling, and distributed system concerns that were absent in the v1 in-process model.

Testing must validate:
- **Functional Equivalence**: V2 providers produce identical results to v1 providers
- **Performance Characteristics**: Quantify the impact of the network hop and identify bottlenecks
- **Reliability**: Handle network failures, certificate rotation, and provider restarts gracefully
- **Security**: Enforce TLS requirements, namespace isolation, and certificate validation
- **Operational Patterns**: Support rolling deployments, scaling, and common operational scenarios

The existing e2e test suite validates provider behavior in the v1 in-process model. We must ensure these tests pass against v2 providers while adding new tests for v2-specific concerns.

## Problem Description

V2 testing faces challenges that v1 testing did not:

1. **Distributed System Complexity**: Failures can occur in network communication, TLS handshakes, connection pooling, or remote provider processes
2. **Performance Unknowns**: The network hop introduces latency, but connection pooling and caching should mitigate impact. We lack quantitative data.
3. **Operational Scenarios**: Certificate rotation, rolling updates, and provider scaling are new operational concerns requiring validation
4. **Conformance Across Providers**: With out-of-tree providers, we need standardized validation that each provider correctly implements the protocol
5. **Test Environment Complexity**: Tests must deploy and manage provider processes, service networking, and TLS infrastructure

Without comprehensive testing, we risk:
- Silent correctness issues where v2 produces different results than v1
- Performance regressions that degrade user experience
- Production failures during certificate rotation or rolling updates
- Provider implementations that deviate from expected behavior
- Inability to confidently recommend v2 adoption

## Decision

Implement a multi-layered testing strategy covering unit, integration, e2e, conformance, performance, and disruption testing.

### 1. E2E Test Migration

**Objective**: Ensure existing provider behavior works identically with v2 architecture.

**Approach**:
- Run the complete existing e2e test suite against v2 provider configurations
- Each test case that currently uses v1 `SecretStore` gets a v2 equivalent using `Provider` resources
- Deploy providers as separate services in the test environment
- Configure TLS for provider communication in test clusters

**Test Matrix**:
```
Provider (AWS, GCP, Vault, etc.)
  × Store Type (SecretStore, ClusterSecretStore)
  × Auth Mode (ManifestNamespace, ProviderNamespace)
  × Feature (GetSecret, GetAllSecrets, PushSecret, DeleteSecret)
  × Data Format (string, binary, JSON templating, dataFrom)
```

**Implementation**:
- Create test utilities that generate v2 provider configurations from existing v1 tests
- Establish provider deployment helpers for test environments
- Run v1 and v2 tests in parallel during transition period to detect regressions

**Success Criteria**:
- 100% of v1 e2e tests pass with v2 providers
- No behavioral differences between v1 and v2 results

### 2. Certificate Management Tests

**Objective**: Validate TLS certificate lifecycle operations work without service disruption.

**Test Scenarios**:

**2.1 Certificate Rotation**
- Issue initial certificates for ESO→Provider communication
- Rotate server certificate while maintaining client connections
- Rotate client certificate while maintaining connectivity
- Rotate CA certificate with overlap period
- Verify: Zero failed reconciliations during rotation
- Verify: Connection pool handles certificate updates gracefully

**2.2 Certificate Expiration**
- Configure short-lived certificates (5 minutes)
- Run reconciliation loop through multiple certificate renewals
- Verify: Automatic certificate refresh before expiration
- Verify: Clear error messages if certificate expires

**2.3 Certificate Validation Failures**
- Present invalid server certificate (wrong CN, expired, self-signed)
- Present mismatched CA certificate
- Present revoked certificate
- Verify: Connections rejected with clear error messages
- Verify: Controller status reflects TLS validation failures

**2.4 mTLS Configuration**
- Enable mutual TLS with client certificates
- Verify: Provider rejects connections without valid client cert
- Verify: ESO successfully authenticates with client cert
- Rotate both client and server certificates

**Implementation**:
- Integrate cert-manager in test environments for automated certificate issuance
- Create test scenarios with Kubernetes TLS secrets
- Use short certificate lifetimes to accelerate rotation testing

### 3. Provider Conformance Suite

**Objective**: Standardize validation that providers correctly implement the v2 protocol.

**Approach**: Create a reusable test library (`providers/v2/conformance`) that provider implementers run against their implementations.

**Core Conformance Tests**:

**3.1 Secret Operations**
- `GetSecret`: Retrieve secret by key, handle missing secrets, decode strategies
- `GetSecretMap`: Retrieve key-value maps (deprecated but supported)
- `GetAllSecrets`: Find secrets by path, tags, regex, conversion strategies
- `PushSecret`: Write secrets with properties, metadata, idempotency
- `DeleteSecret`: Remove secrets, handle non-existent deletions
- `SecretExists`: Check existence without retrieving data

**3.2 Error Semantics**
- Return `NoSecretError` for missing secrets with correct deletion policy behavior
- Return validation errors for malformed requests
- Return permission errors when authentication fails
- Propagate timeouts and retryable vs non-retryable errors correctly

**3.3 Authentication**
- Respect namespace boundaries for secret references
- Support multiple authentication methods (IAM, service accounts, static credentials)
- Handle credential refresh and expiration
- Reject cross-namespace access when not permitted

**3.4 Protocol Compliance**
- Respond to health checks correctly
- Implement graceful shutdown
- Handle concurrent requests safely
- Respect context cancellation
- Return proper gRPC status codes

**3.5 Provider Metadata**
- Return correct capabilities (ReadOnly, WriteOnly, ReadWrite)
- Validate provider configuration
- Provide meaningful validation warnings

**Implementation**:
- Conformance tests as Go package: `import "github.com/external-secrets/external-secrets/providers/v2/conformance"`
- Provider tests instantiate conformance suite with their provider implementation
- CI integration: gate provider releases on conformance pass
- Versioned conformance suite (v2alpha1, v2beta1) for compatibility testing

**Success Criteria**:
- All official providers pass 100% of conformance tests
- Conformance suite executable in <2 minutes
- Clear failure messages identifying non-compliant behavior

### 4. Performance and Load Tests

**Objective**: Quantify performance impact of v2 architecture and identify breaking points.

**4.1 Baseline Performance Comparison**

Compare v1 (in-process) vs v2 (gRPC) on identical workloads:

**Metrics**:
- Secret fetch latency (p50, p95, p99)
- Reconciliation latency (ExternalSecret update to Secret ready)
- Throughput (secrets/second)
- CPU usage (controller and provider)
- Memory usage (controller and provider)
- Network bandwidth
- Connection pool statistics (active, idle, created, reused)

**Test Scenarios**:
```
Small: 100 ExternalSecrets, 1 secret each, 5m refresh
Medium: 1000 ExternalSecrets, 3 secrets each, 5m refresh
Large: 5000 ExternalSecrets, 10 secrets each, 5m refresh
Burst: 1000 ExternalSecrets created simultaneously
```

**Implementation**:
- Deploy both v1 and v2 configurations in identical clusters
- Use Prometheus to capture metrics
- Generate ExternalSecrets with controlled characteristics
- Run tests for 30 minutes to measure steady-state performance
- Export results to comparable format (CSV, JSON)

**4.2 Connection Pool Behavior**

Validate connection pooling effectiveness:

**Test Cases**:
- Measure connection reuse rate under steady load
- Verify connection pool respects max connection limits
- Validate idle connection timeout and cleanup
- Measure connection establishment latency
- Test connection recovery after network blip

**4.3 Cache Effectiveness**

Measure client manager cache hit rates:

**Metrics**:
- Cache hit ratio by provider type
- Cache invalidation frequency and causes
- Memory consumption by cache size
- Impact of generation changes on cache invalidation

**4.4 Breaking Point Analysis**

Identify system limits:

**Approach**:
- Incrementally increase load until degradation or failure
- Vary: Number of ExternalSecrets, secrets per ExternalSecret, refresh frequency
- Measure: When does latency exceed SLO? When do errors begin? What fails first?
- Compare: Where does v2 break compared to v1?

**Implementation**:
- Use load generation tools (custom or k6/locust adapted for Kubernetes)
- Monitor resource exhaustion (CPU, memory, file descriptors, connections)
- Capture system behavior at breaking point (logs, metrics, traces)

**Success Criteria**:
- v2 adds ≤50ms p95 latency compared to v1 under medium load
- v2 throughput within 80% of v1 throughput
- v2 handles ≥1000 concurrent ExternalSecrets per controller
- Connection pool prevents connection exhaustion
- Clear documentation of performance characteristics and limits

### 5. Disruption and Chaos Tests

**Objective**: Validate system resilience during operational disruptions.

**5.1 Rolling Deployments**

Test rolling updates of ESO controller and providers:

**Scenarios**:
- Roll ESO controller pods while providers run
- Roll provider pods while ESO reconciles
- Roll both simultaneously
- Vary: Deployment strategy (RollingUpdate, Recreate), replica count, update velocity

**Measurements**:
- Reconciliation success rate during rollout
- Latency increase during rollout
- Connection pool behavior during pod replacement
- Error rate and recovery time
- Number of failed secret fetches

**5.2 Network Failures**

Simulate network issues between ESO and providers:

**Test Cases**:
- Complete network partition (10s, 60s, 5m)
- Packet loss (5%, 20%, 50%)
- Latency injection (+50ms, +500ms, +5s)
- DNS resolution failures
- Service endpoint unavailability

**Measurements**:
- Retry behavior and backoff
- Circuit breaking (if implemented)
- Error propagation to ExternalSecret status
- Recovery time after network restoration
- Connection pool health monitoring effectiveness

**5.3 Provider Failures**

Test provider process failures:

**Scenarios**:
- Graceful shutdown (SIGTERM)
- Forced termination (SIGKILL)
- Provider panic/crash
- Provider deadlock/hang
- OOM kill

**Measurements**:
- Health check detection time
- Connection pool marking unhealthy connections
- Automatic reconnection attempts
- User-visible error messages in ExternalSecret status
- Time to recovery after provider restart

**5.4 Certificate Issues**

Inject certificate problems:

**Test Cases**:
- Expire server certificate mid-operation
- Revoke certificate
- Change CA without updating client
- TLS handshake timeout
- Certificate chain validation failure

**Measurements**:
- Error detection latency
- Error message clarity
- Automatic recovery after certificate fix

**5.5 Resource Contention**

Test behavior under resource pressure:

**Scenarios**:
- CPU throttling (limit provider to 100m CPU)
- Memory pressure (limit provider to 128Mi)
- Disk I/O saturation
- High concurrent request load

**Measurements**:
- Graceful degradation vs hard failure
- Request timeout behavior
- Resource limit enforcement
- Queue buildup and backpressure

**Implementation**:
- Use chaos engineering tools (Chaos Mesh, Litmus)
- Automate disruption injection in test suite
- Run continuously in staging environments
- Generate chaos test reports with metrics and logs

**Success Criteria**:
- Zero data corruption during disruptions
- <5% error rate during rolling updates
- Automatic recovery within 2 minutes after disruption ends
- Clear error messages visible in ExternalSecret status
- No panics or crashes in ESO controller

### 6. Additional Recommended Tests

**6.1 Version Skew Tests**

Validate compatibility across version combinations:

**Matrix**:
- ESO version N with Provider version N-1, N, N+1
- Protocol version compatibility
- Deprecated field handling
- Forward/backward compatibility

**6.2 Metrics Validation**

Ensure observability correctness:

**Tests**:
- Verify all metrics are emitted correctly
- Validate metric labels and cardinality
- Check metrics match actual system behavior
- Ensure no metrics memory leaks

**6.3 Concurrent Operations**

Test race conditions and concurrent access:

**Scenarios**:
- Multiple ExternalSecrets referencing same Provider simultaneously
- Rapid ExternalSecret create/delete cycles
- Concurrent provider client cache access
- Connection pool concurrent get/release

**6.4 Error Recovery**

Test recovery from error states:

**Scenarios**:
- Provider becomes healthy after being unhealthy
- Invalid configuration fixed
- Credentials updated after auth failure
- Network restored after partition

**6.5 Migration Tests**

Validate v1 to v2 migration:

**Tests**:
- Switch ExternalSecret from v1 to v2 store without data loss
- Run mixed v1/v2 workloads simultaneously
- Gradual provider migration
- Rollback from v2 to v1

## Consequences

### Positive

- **Confidence**: Comprehensive testing enables confident v2 recommendation to users
- **Quality**: Conformance suite ensures provider consistency
- **Performance Insight**: Quantitative data informs optimization priorities
- **Operational Readiness**: Disruption tests validate production scenarios
- **Regression Prevention**: Automated testing catches regressions early

### Negative

- **Test Infrastructure Complexity**: Managing provider deployments increases test environment complexity
- **Execution Time**: Comprehensive testing takes longer than v1 tests
- **Maintenance Burden**: More tests require ongoing maintenance
- **Resource Cost**: Performance and chaos tests consume significant compute resources

### Neutral

- **Gradual Rollout**: Testing strategy supports phased v2 adoption
- **Provider Responsibility**: Out-of-tree providers own their conformance test execution
- **Tooling Requirements**: Requires investment in test tooling and infrastructure
