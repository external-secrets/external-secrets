```yaml
---
title: Feature Flag Consolidation
version: v1alpha1
authors: Jean-Philippe Evrard
creation-date: 2026-01-05
status: draft
---
```

# Feature Flag Consolidation

## Table of Contents

<!-- toc -->
<!-- /toc -->

## Summary

Consolidate the 76+ configuration points scattered across build tags, CLI flags, environment variables, and feature registration into a unified, documented configuration system. This proposal addresses broken registrations, deprecated dead code, and inconsistent patterns while maintaining full backward compatibility.

## Motivation

The External Secrets Operator has accumulated configuration sprawl across multiple mechanisms:

| Type                     | Count     | Location                      |
|--------------------------|-----------|-------------------------------|
| Build tags               | 30+       | `pkg/register/*.go`           |
| CLI flags (root)         | 40        | `cmd/controller/root.go`      |
| Feature-registered flags | 6         | Distributed across 5 packages |
| Environment variables    | 2         | `providers/v1/doppler/`       |
| **Total**                | around 80 | 6+ locations                  |

**Critical issues identified:**

1. **Broken registration**: Doppler's `InitializeFlags()` function (doppler/provider.go:65-78) is never called, making `--doppler-oidc-cache-size` unavailable
2. **Dead code**: AWS `--experimental-enable-aws-session-cache` flag is marked DEPRECATED but still registered (aws/auth/auth.go:68-74)
3. **Inconsistent patterns**: Four different registration mechanisms across providers
4. **Undocumented flags**: No central documentation of all available configuration options
5. **Mixed naming**: `enable-*`, `experimental-*`, `unsafe-*` prefixes used inconsistently

### Goals

1. Establish a single, consistent pattern for feature flag registration
2. Fix broken Doppler flag registration
3. Remove deprecated AWS session cache flag
4. Document all configuration options in a central location
5. Maintain 100% backward compatibility for existing flags

### Non-Goals

- Changing build tag patterns (38 provider tags work correctly)
- Introducing a configuration file format (future enhancement)
- Deprecating existing CLI flags (only fixing broken/dead ones)

## Proposal

### Phase 1: Fix Critical Bugs

#### 1.1 Fix Doppler Flag Registration

**Current (broken):**
```go
// providers/v1/doppler/provider.go:65-78
func InitializeFlags() *feature.Feature {  // Never called!
    var dopplerOIDCCacheSize int
    fs := pflag.NewFlagSet("doppler", pflag.ExitOnError)
    fs.IntVar(&dopplerOIDCCacheSize, "doppler-oidc-cache-size", defaultCacheSize, "...")
    return &feature.Feature{
        Flags: fs,
        Initialize: func() { initCache(dopplerOIDCCacheSize) },
    }
}
```

**Fixed:**
```go
// providers/v1/doppler/provider.go
func init() {
    var dopplerOIDCCacheSize int
    fs := pflag.NewFlagSet("doppler", pflag.ExitOnError)
    fs.IntVar(&dopplerOIDCCacheSize, "doppler-oidc-cache-size", defaultCacheSize,
        "Size of the Doppler OIDC token cache")
    feature.Register(feature.Feature{
        Flags:      fs,
        Initialize: func() { initCache(dopplerOIDCCacheSize) },
    })
}
```

#### 1.2 Remove Deprecated AWS Flag

**Current (dead code):**
```go
// providers/v1/aws/auth/auth.go:68-74
func init() {
    fs := pflag.NewFlagSet("aws-auth", pflag.ExitOnError)
    fs.BoolVar(&enableSessionCache, "experimental-enable-aws-session-cache", false,
        "DEPRECATED: this flag is no longer used and will be removed...")
    feature.Register(feature.Feature{Flags: fs})
}
```

**Action:** Remove entirely. The flag is documented as deprecated and the variable `enableSessionCache` is never read.

### Phase 2: Standardize Registration Pattern

Establish a single pattern for all feature-registered flags:

```go
// Standard pattern for provider-specific flags
func init() {
    fs := pflag.NewFlagSet("<provider-name>", pflag.ExitOnError)

    // Define flags
    fs.TypeVar(&variable, "flag-name", defaultValue, "Description")

    // Register with optional late initialization
    feature.Register(feature.Feature{
        Flags:      fs,
        Initialize: func() { /* optional setup */ },
    })
}
```

**Current state by provider:**

| Provider     | Pattern                          | Status                |
|--------------|----------------------------------|-----------------------|
| Vault        | `init()` + `feature.Register()`  | Correct               |
| AWS Auth     | `init()` + `feature.Register()`  | Has deprecated flag   |
| Doppler      | `InitializeFlags()` (standalone) | Broken - never called |
| Template     | `init()` + `feature.Register()`  | Correct               |
| StateManager | `init()` + `feature.Register()`  | Correct               |

**After Phase 2:** All use `init()` + `feature.Register()` pattern.

### Phase 3: Naming Convention Standardization

Establish clear naming conventions for flags:

| Prefix           | Meaning                           | Example                                   |
|------------------|-----------------------------------|-------------------------------------------|
| `experimental-*` | Experimental feature (may change) | `--experimental-enable-vault-token-cache` |
| `enable-*`       | Core feature toggle (stable)      | `--enable-leader-election`                |
| `unsafe-*`       | Potentially dangerous option      | `--unsafe-allow-generic-targets`          |
| `<provider>-*`   | Provider-specific setting         | `--doppler-oidc-cache-size`               |

We do not have a policy when two or more are combined.
My suggestion is to take the previous table from top to bottom, and use them from left to right.
It means some flags could, in the future, look like this:
`--experimental-enable-unsafe-vault-provider-specific-flag-name`
When the flag would not be experimental anymore, it would become:
`--enable-unsafe-vault-providerspecificflagname`

Next to that, some internal features are NOT provider specific, but match the pattern:
`--template-left-delimiter` / `--template-right-delimiter`.
I suggest we keep them as is, as they are okayish (provider-style naming)

I propose the following:
- Any new flag must follow the conventions
- When migrating a flag to a new flag (e.g. when leaving experimental state, or not marked unsafe anymore), we normalize and deprecate the flag in the flagset.

Info:
- We can use `SetNormalizeFunc` to call a function to normalize flag names which would register a deprecated state (it will be useful in later phases)
- The register method of `feature` can then ensure flags are consistent with the maturity/safety level of the feature, until a compile time manner is implemented.

### Phase 4: Central Documentation

Create `docs/flags.md` (or equivalent in documentation site) listing all flags:

```markdown
# Configuration Reference


## Controller Flags

| Feature name    | Flag                       | Default    | Description                 |
|-----------------|----------------------------|------------|-----------------------------|
| Metrics         | `--metrics-addr`           | `:8080`    | Metrics server bind address |
| Leader election | `--enable-leader-election` | `true`     | Enable leader election      |
| ...             | ...                        | ...        |                             |

## Provider Flags

### Vault
| Feature name | Flag                                       | Default  | Description          |
|--------------|--------------------------------------------|----------|----------------------|
| Token Cache  | `--experimental-enable-vault-token-cache`  | `false`  | Enable token caching |
| Token Cache  | `--experimental-vault-token-cache-size`    | `1000`   | Cache size           |

### Doppler
| Feature name | Flag                        | Default  | Description           |
|--------------|-----------------------------|----------|-----------------------|
| OIDC  Cache  | `--doppler-oidc-cache-size` | `1000`   | OIDC token cache size |
```

### Phase 5: Feature Registry Enhancement

Enhance the `runtime/feature` package to support introspection:

```go
// runtime/feature/feature.go

type Feature struct {
    Name        string           // Human-readable/understandable short name?
    Flags       *pflag.FlagSet
    Initialize  func()
    Maturity    Maturity         // Experimental, Stable, Deprecated
    Safety      Safety           // Safe, Unsafe
}

type Maturity int
type Safety int

const (
    Experimental Maturity = iota
    Stable
	Deprecated
	Unknown
)

const (
	Unsafe Safety = iota
    Safe
	Unknown
)

// ListFeatures returns all registered features for documentation/help
func ListFeatures() []Feature {
    return features
}
```

This enables:
- Auto-generated feature flag documentation
- `--help` output grouped by feature
- Safety-based warnings are possible.

### User Stories

**As a contributor**, I want a clear pattern for adding provider flags, so I don't repeat existing mistakes.

**As a user**, I want to see all available flags documented, so I can configure the operator correctly.

**As an operator**, I want `--doppler-oidc-cache-size` to actually work, so I can tune Doppler performance.

### API

No CRD or API changes.

### Behavior changes

- `--doppler-oidc-cache-size` becomes functional (currently silently ignored)
- `--experimental-enable-aws-session-cache` removed (was already non-functional)
- All other behavior unchanged

### Drawbacks

1. Users specifying `--experimental-enable-aws-session-cache` will get "unknown flag" error
2. Documentation maintenance: Central docs must be kept in sync manually until phase 5.

### Acceptance Criteria

**Phase 1 (Bug Fixes)**:
- [ ] Doppler `--doppler-oidc-cache-size` flag is functional
- [ ] AWS deprecated flag removed
- [ ] No other flags affected

**Phase 2 (Standardization)**:
- [ ] All provider flags use `init()` + `feature.Register()` pattern
- [ ] Linting rule added to prevent `InitializeFlags()` pattern

**Phase 3 (Naming)**:
- [ ] Naming convention documented in CONTRIBUTING.md or other poolicy document (depending on diataxis progress)
- [ ] New flags follow convention (enforced in PR review)

**Phase 4 (Documentation)**:
- [ ] Flags documented
- [ ] Documentation linked from README and CLI help

**Phase 5 (Automation)**:
- [ ] Feature struct extended with metadata
- [ ] Generated documentation from code
- [ ] Existing startup logs show which features are enabled, and their maturity/safety level

**Rollout:**
- Phase 1: Immediate bug fix release
- Phase 2-3: Next minor release
- Phase 4: Documentation PR (no code change)
- Phase 5: Code and Documentation PR

## Alternatives

### Alternative 1: Configuration File

Support YAML/TOML configuration file instead of CLI flags.

```yaml
# eso-config.yaml
controller:
  metricsAddr: ":8080"
  leaderElection: true
providers:
  vault:
    tokenCache:
      enabled: true
      size: 1000
```
Pros:
- Easier to manage complex configurations
- supports comments

Cons:
- Breaking change
- requires migration tooling
- two sources of truth (config/cli)

**Decision**: Deferred - valuable but higher effort. CLI flags remain primary interface.

### Alternative 3: Keep Current State

Accept the scattered configuration. No migration effort.
We can fix the Doppler flag and AWS dead code.
However,the contributors and our users will be confused about the maturity of the flags, and the docs might become outdated if we forget to update it.

## Compatibility with Other Designs

- **design/007-provider-versioning-strategy**: When providers become separate CRDs, their flags may move to CRD spec fields. The feature registration pattern supports this transition - flags can be deprecated in favor of CRD configuration.

- **design/006-LTS-release**: Flag documentation is essential for LTS releases. Users need to know which flags are stable vs experimental when planning upgrades.
