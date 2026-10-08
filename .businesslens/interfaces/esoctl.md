---
type: cli
actors: [application-developer]
references:
  - kind: doc
    role: intent
    target: docs/guides/using-esoctl-tool.md
  - kind: code
    role: implementation
    target: cmd/esoctl/template.go
  - kind: doc
    role: context
    target: cmd/esoctl/README.md
  - kind: code
    role: context
    target: .github/workflows/release_esoctl.yml
---

# esoctl

A released command-line tool for working with the Product's resources away
from a cluster, used to try out the templates of ExternalSecrets and
PushSecrets before applying them.
