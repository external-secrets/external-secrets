# Provider Code Generation

## Context

V2 providers run as standalone gRPC server processes, each requiring a `main.go` file with startup logic and a `Dockerfile` for containerization. This startup code follows a consistent pattern across all providers:

- Parse command-line flags (port, TLS settings, metrics)
- Initialize Kubernetes client with appropriate schemes
- Set up metrics collection and health checks
- Configure gRPC server with TLS/mTLS
- Register provider and generator services
- Implement graceful shutdown handling

Provider-specific configuration includes:
- Which v1 provider implementations to wrap
- GVK (Group/Version/Kind) mappings for stores and generators
- Package imports for API types and implementations
- Spec mapper logic to convert provider resources to `SecretStoreSpec`

Manually maintaining this boilerplate across multiple providers creates maintenance burden, increases error probability, and makes cross-cutting changes (like adding new flags or improving shutdown logic) require updates to every provider.

## Problem Description

Without code generation, each provider requires:

1. **Repetitive Boilerplate**: 150+ lines of identical startup code duplicated across providers
2. **Maintenance Overhead**: Changes to common patterns require updating every provider's `main.go`
3. **Error Susceptibility**: Manual construction of import statements and GVK mappings is error-prone
4. **Inconsistency Risk**: Providers may drift from standard patterns over time
5. **Slow Provider Addition**: Creating new providers requires copying and adapting existing `main.go` files

The only true differences between providers are:
- Provider name and display name
- Which v1 implementations to instantiate
- GVK mappings for stores and generators
- Import paths for API types

## Decision

Implement a template-based code generator that produces `main.go` and `Dockerfile` from declarative YAML configuration.

### Architecture

Introduce a **Generator Tool**:
- Discovers all `provider.yaml` files in the providers directory
- Validates each against JSON schema
- Executes Go templates with provider-specific data
- Manages import aliases to avoid conflicts
- Formats generated code with `goimports`

**Provider Configuration** (`provider.yaml`):
```yaml
provider:
  name: aws
  displayName: "AWS Provider"
  v2Package: "github.com/.../apis/provider/aws/v2alpha1"

stores:
  - gvk:
      group: "provider.external-secrets.io"
      version: "v2alpha1"
      kind: "SecretsManager"
    v1Provider: "github.com/.../providers/v2/aws/store"
    v1ProviderFunc: "NewProvider"

generators:
  - gvk:
      group: "generators.external-secrets.io"
      version: "v1alpha1"
      kind: "ECRAuthorizationToken"
    v1Generator: "github.com/.../providers/v2/aws/generator"
    v1GeneratorFunc: "NewECRGenerator"

configPackage: "."
```

**Manual Component** (`config.go`):

Provider-specific logic that cannot be templated remains in manually written `config.go`:

```go
func GetSpecMapper(kubeClient client.Client) func(*pb.ProviderReference) (*v1.SecretStoreSpec, error) {
    return func(ref *pb.ProviderReference) (*v1.SecretStoreSpec, error) {
        var provider awsv2alpha1.SecretsManager
        err := kubeClient.Get(context.Background(), client.ObjectKey{
            Namespace: ref.Namespace,
            Name:      ref.Name,
        }, &provider)
        if err != nil {
            return nil, err
        }
        return &v1.SecretStoreSpec{
            Provider: &v1.SecretStoreProvider{
                AWS: &provider.Spec,
            },
        }, nil
    }
}
```

### Generation Process

1. **Discovery**: Walk `providers/v2/` to find all `provider.yaml` files
2. **Validation**: Validate YAML against JSON schema to ensure correctness
3. **Template Data Preparation**:
   - Parse YAML into structured configuration
   - Deduplicate imports and generate aliases
   - Build template data with GVK mappings and import information
4. **Code Generation**:
   - Execute `main.go.tmpl` template
   - Execute `Dockerfile.tmpl` template
5. **Formatting**: Run `goimports` to format and organize imports
6. **Output**: Write generated files to provider directory

### Schema Validation

The JSON schema enforces:
- Required fields (`provider.name`, `provider.displayName`)
- At least one of `stores` or `generators` must be defined
- Proper GVK structure for all mappings
- Valid package paths

### Integration

Makefile targets provide interface to the generator:

```bash
make generate-providers  # Generate all provider files
make verify-providers    # Check if files are up-to-date
```

CI verification ensures generated files remain synchronized with configuration.

## Consequences

### Positive

- **Reduced Boilerplate**: Eliminates 150+ lines of repetitive code per provider
- **Centralized Evolution**: Improvements to startup logic propagate to all providers via template updates
- **Type Safety**: Schema validation catches configuration errors before code generation
- **Consistency**: All providers follow identical patterns, reducing cognitive load
- **Fast Onboarding**: New providers require only YAML configuration and spec mapper logic
- **Import Management**: Generator handles deduplication and aliasing automatically
- **Verifiable**: CI can detect drift between configuration and generated code

### Negative

- **Indirection**: Debugging requires understanding template system
- **Build Complexity**: Additional step in development workflow
- **Tool Dependency**: Requires `goimports` for formatting
- **Schema Maintenance**: Changes to common patterns require schema and template updates
- **Generated Code Friction**: Cannot directly edit `main.go`—must modify template or configuration

### Neutral

- **Hybrid Approach**: Provider-specific logic (`GetSpecMapper`) remains manual, requiring judgment about what to template
- **Template Language**: Go templates have limitations compared to programmatic generation
- **Verification Required**: CI must enforce that generated files match configuration

## Alternatives Considered

### Full Manual Implementation
Rejected because maintenance burden scales linearly with provider count and cross-cutting changes become expensive.

### Pure Library Approach
Rejected because providers need different combinations of stores and generators, and compile-time type safety requires different imports and initialization code per provider.

### Runtime Configuration
Rejected because Go's static typing requires compile-time knowledge of which provider implementations to link, and dynamic loading has security and deployment implications.

## Notes

The generator intentionally keeps the spec mapper logic manual because it involves provider-specific type conversions that vary significantly between providers. Templating this logic would create more complexity than it eliminates.

Future enhancements may include automatic discovery of v1 providers to reduce configuration requirements further.
