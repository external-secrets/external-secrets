# Module Import Boundaries

## Intent

https://github.com/external-secrets/external-secrets/issues/5494 introduced
the split of our go module into multiple go modules.

The module structure has been exercised and can now be moved to a design
document.

## Principles

- Packages in `apis` are for CRDs and a limited set of interfaces.
  They may import the standard library, packages in the `apis`
  module and an explicit allowlist of API dependencies (for example, k8s.io
  modules).
- Packages in `runtime` are common utilies. They contain shared code
  (validation utilities, webhook helpers, metrics/logging tooling, ...).
  They may import the standard library, packages in `apis` or
  `runtime`, and an explicit allowlist of runtime dependencies.
  As the dependencies are included in provider modules, the addition of
  a module into a `runtime` package needs to be carefully analysed.
- Packages in `pkg` are for the core orchestration. It's where the
  controllers and binary reside. They can depend on `apis` and `runtime`.
- Packages in `e2e` are for the e2e testing, and can require any of the
  above packages. They are built in isolation, so they should not leak
  dependencies everywhere.
- Transitive dependencies are not evaluated: If some module
  imports the whole world, then we will have to deal with it.

## Enforcement

In a first stage, we use code reviews to highlight discrepencies and preserve
the module layout expressed above.

In a second stage, we enable golangci-lint's existing `depguard` linter.
It defines strict rules selected by path like `**/apis/**/*.go` or `**/runtime/**/*.go`.
Each rule allows `$gostd`, the module's own package prefix, and the package prefixes belonging to its allowed current direct dependencies.

Depguard analyzes import declarations rather than the module graph, so it
checks only direct imports. The file globs include tests. An import such as an
AWS or Oracle SDK from a runtime test therefore fails the existing `make lint`
and `make reviewable` gates without constraining dependencies loaded
transitively by an allowed package.

The existing `go mod tidy` and `make check-diff` flow remains responsible for
removing stale `go.mod` requirements. No new command, dependency, or CI job is
needed.
