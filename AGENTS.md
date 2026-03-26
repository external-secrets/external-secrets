# External Secrets Operator

Kubernetes operator that synchronizes secrets from external providers (AWS Secrets Manager, Vault, GCP Secret Manager, Azure Key Vault, etc.) into Kubernetes Secrets.

## Build and Test

Use `make` targets. Do not run `go test`, `golangci-lint`, or `helm` commands directly.

```bash
make generate        # Generate code and CRDs
make test            # Run unit tests (requires envtest)
make test.e2e        # Run end-to-end tests
make lint            # Run golangci-lint
make reviewable      # Full pre-PR check: generate, lint, docs, manifests, helm, tests
make check-diff      # Verify branch is clean after reviewable
make docker.build    # Build Docker image for all architectures
make docs.serve      # Serve docs locally
```

CRDs exceed the 256KB annotation limit. Apply them with `kubectl apply --server-side`.

## Project Layout

Three binaries in `cmd/`: the **controller** reconciles ExternalSecrets into K8s Secrets, the **webhook** validates and defaults CRDs, and the **certcontroller** manages TLS certificates for the webhook.

```
cmd/                  # Entry points (controller, webhook, certcontroller)
pkg/                  # Core logic (controllers, provider clients, template engine)
apis/                 # API types (v1, v1beta1) — separate Go module
providers/            # Provider implementations — each is a separate Go module
generators/           # Generator implementations — separate Go modules
runtime/              # Shared runtime utilities — separate Go module
e2e/                  # End-to-end tests — separate Go module
deploy/charts/        # Helm chart
config/crds/          # CRD source
docs/                 # MkDocs documentation site
docs/provider/        # Provider-specific setup and auth docs
docs/guides/          # Guides (multi-tenancy, security, templating)
hack/                 # Build tooling, API doc generation
terraform/            # Terraform modules
```

Multi-module repo: `apis/`, `runtime/`, `e2e/`, and each `providers/v1/*/` have their own `go.mod`.

## Key Docs for Setup Tasks

When a user asks to set up ESO, read these before generating manifests:

- `docs/guides/ai-setup-guide.md` — decision flow for store scope, auth, and credential scoping
- `docs/guides/multi-tenancy.md` — SecretStore vs ClusterSecretStore tradeoffs
- `docs/guides/security-best-practices.md` — hardening checklist
- `docs/provider/aws-access.md` — AWS auth methods (IRSA, Pod Identity, static keys)
- `docs/introduction/getting-started.md` — installation and first ExternalSecret

## Non-Obvious Patterns

- `make reviewable` is the gate for PRs. Run it, not individual checks.
- Helm chart is the source of truth for deploy manifests. `make manifests` generates static YAML from it.
- Provider docs use MkDocs snippet transclusion (`--8<--`). Auth docs are shared across providers via `docs/snippets/`.
- CRD tests use snapshot testing. Run `make test.crds.update` to update snapshots after CRD changes.
- `make update-deps` updates dependencies across all modules at once.
