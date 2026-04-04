# External Secrets Operator

Kubernetes operator that synchronizes secrets from external providers (AWS Secrets Manager, Vault, GCP Secret Manager, Azure Key Vault, etc.) into Kubernetes Secrets.

## Build and Test

Use `make` targets — refer to the Makefile for available commands. Do not run `go test`, `golangci-lint`, or `helm` directly.

## Project Layout

Single binary built from `main.go`. The **controller** reconciles ExternalSecrets into K8s Secrets. The **webhook** (validates and defaults CRDs) and **certcontroller** (manages webhook TLS) are subcommands registered via `rootCmd.AddCommand()`.

Multi-module repo: `apis/`, `runtime/`, `e2e/`, and each `providers/v1/*/` have their own `go.mod`.

## Non-Obvious Patterns

- `make reviewable` is the gate for PRs. Run it, not individual checks.
- Helm chart is the source of truth for deploy manifests. `make manifests` generates static YAML from it.
- Provider docs use MkDocs snippet transclusion (`--8<--`). Auth docs are shared across providers via `docs/snippets/`.
- CRD tests use snapshot testing. Run `make test.crds.update` to update snapshots after CRD changes.
- `make update-deps` updates dependencies across all modules at once.
