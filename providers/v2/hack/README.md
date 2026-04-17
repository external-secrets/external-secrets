# Provider Main Generator

This directory contains the code generation tooling for v2 provider `main.go` and `Dockerfile` files.

## Overview

The generator reduces boilerplate and maintenance burden by centralizing the common provider startup logic (flags, gRPC server setup, health checks, graceful shutdown, etc.) while allowing provider-specific configuration through YAML files.

## Directory Structure

```
providers/v2/hack/
├── generate-provider-main.go      # Generator tool
├── schema/
│   └── provider-config.schema.json  # JSON schema for provider.yaml validation
├── templates/
│   ├── main.go.tmpl               # Template for main.go
│   └── Dockerfile.tmpl            # Template for Dockerfile
└── README.md                      # This file
```

## Usage

### Generate Provider Files

From the repository root:

```bash
make generate-providers
```

This will:
1. Find all `provider.yaml` files in `providers/v2/`
2. Validate each against the JSON schema
3. Generate `main.go` and `Dockerfile` for each provider
4. Format the generated Go code with `goimports`

### Verify Generated Files Are Up-to-Date

```bash
make verify-providers
```

This checks if any generated files are out of sync with their configuration.

## Adding a New Provider

To add a new v2 provider:

1. **Create the provider directory structure:**
   ```
   providers/v2/myprovider/
   ├── provider.yaml     # Configuration (required)
   ├── config.go         # Spec mapper logic (required)
   ├── store/            # v1 store implementation
   └── generator/        # v1 generator implementation (optional)
   ```

2. **Create `provider.yaml`:**

   ```yaml
   provider:
     name: myprovider
     displayName: "My Provider"
     v2Package: "github.com/external-secrets/external-secrets/apis/provider/myprovider/v2alpha1"

   stores:
     - gvk:
         group: "provider.external-secrets.io"
         version: "v2alpha1"
         kind: "MyProvider"
       v1Provider: "github.com/external-secrets/external-secrets/providers/v1/myprovider"
       v1ProviderFunc: "NewProvider"

   # Optional: if provider includes generators
   generators:
     - gvk:
         group: "generators.external-secrets.io"
         version: "v1alpha1"
         kind: "MyGenerator"
       v1Generator: "github.com/external-secrets/external-secrets/providers/v2/myprovider/generator"
       v1GeneratorFunc: "NewGenerator"

   configPackage: "."
   ```

3. **Create `config.go` with GetSpecMapper function:**

   ```go
   package main

   import (
       "context"
       "sigs.k8s.io/controller-runtime/pkg/client"
       v1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
       myproviderv2alpha1 "github.com/external-secrets/external-secrets/apis/provider/myprovider/v2alpha1"
       pb "github.com/external-secrets/external-secrets/proto/provider"
   )

   func GetSpecMapper(kubeClient client.Client) func(*pb.ProviderReference) (*v1.SecretStoreSpec, error) {
       return func(ref *pb.ProviderReference) (*v1.SecretStoreSpec, error) {
           var provider myproviderv2alpha1.MyProvider
           err := kubeClient.Get(context.Background(), client.ObjectKey{
               Namespace: ref.Namespace,
               Name:      ref.Name,
           }, &provider)
           if err != nil {
               return nil, err
           }
           return &v1.SecretStoreSpec{
               Provider: &v1.SecretStoreProvider{
                   MyProvider: &provider.Spec,
               },
           }, nil
       }
   }
   ```

4. **Generate the files:**
   ```bash
   make generate-providers
   ```

5. **Test the provider compiles:**
   ```bash
   cd providers/v2/myprovider && go build
   ```

## Provider Configuration Schema

### Required Fields

- `provider.name`: Provider name (lowercase, alphanumeric with hyphens)
- `provider.displayName`: Human-readable provider name

### Optional Fields

- `provider.v2Package`: Go import path for v2alpha1 API (required if using stores)
- `stores`: Array of store implementations
- `generators`: Array of generator implementations
- `configPackage`: Relative import path for config.go (default: ".")

### Store Configuration

```yaml
stores:
  - gvk:
      group: "provider.external-secrets.io"
      version: "v2alpha1"
      kind: "MyKind"
    v1Provider: "github.com/org/repo/providers/v1/myprovider"
    v1ProviderFunc: "NewProvider"
```

### Generator Configuration

```yaml
generators:
  - gvk:
      group: "generators.external-secrets.io"
      version: "v1alpha1"
      kind: "MyGenerator"
    v1Generator: "github.com/org/repo/providers/v2/myprovider/generator"
    v1GeneratorFunc: "NewMyGenerator"
```

## Examples

### Provider with Single Store (Kubernetes)

```yaml
provider:
  name: kubernetes
  displayName: "Kubernetes Provider"
  v2Package: "github.com/external-secrets/external-secrets/apis/provider/kubernetes/v2alpha1"

stores:
  - gvk:
      group: "provider.external-secrets.io"
      version: "v2alpha1"
      kind: "Kubernetes"
    v1Provider: "github.com/external-secrets/external-secrets/providers/v1/kubernetes"
    v1ProviderFunc: "NewProvider"

configPackage: "."
```

### Provider with Store and Generators (AWS)

```yaml
provider:
  name: aws
  displayName: "AWS Provider"
  v2Package: "github.com/external-secrets/external-secrets/apis/provider/aws/v2alpha1"

stores:
  - gvk:
      group: "provider.external-secrets.io"
      version: "v2alpha1"
      kind: "SecretsManager"
    v1Provider: "github.com/external-secrets/external-secrets/providers/v2/aws/store"
    v1ProviderFunc: "NewProvider"

generators:
  - gvk:
      group: "generators.external-secrets.io"
      version: "v1alpha1"
      kind: "ECRAuthorizationToken"
    v1Generator: "github.com/external-secrets/external-secrets/providers/v2/aws/generator"
    v1GeneratorFunc: "NewECRGenerator"
  - gvk:
      group: "generators.external-secrets.io"
      version: "v1alpha1"
      kind: "STSSessionToken"
    v1Generator: "github.com/external-secrets/external-secrets/providers/v2/aws/generator"
    v1GeneratorFunc: "NewSTSGenerator"

configPackage: "."
```

## Troubleshooting

### Validation Errors

If you see schema validation errors:
1. Check that your `provider.yaml` follows the schema
2. Ensure all required fields are present
3. Verify that at least one of `stores` or `generators` is defined

### Compilation Errors

If generated code doesn't compile:
1. Verify import paths in `provider.yaml` are correct
2. Check that `GetSpecMapper` function signature matches expected format
3. Ensure v1 provider/generator packages export the specified constructor functions

### Import Conflicts

The generator automatically handles import aliases. If you have multiple stores or generators from the same package, they will share the same import alias.

## Development

To modify the generator:

1. Edit the generator logic in `generate-provider-main.go`
2. Update templates in `templates/`
3. Update schema in `schema/provider-config.schema.json`
4. Regenerate all providers to test: `make generate-providers`
5. Verify nothing broke: `make verify-providers`

## Future Enhancements

- Support for custom CLI flags (deferred)
- Support for custom middleware
- Automatic detection of v1 providers to generate configs

