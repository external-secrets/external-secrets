---
availability:
  - { place: esoctl }
references:
  - kind: code
    role: implementation
    target: cmd/esoctl/template.go
  - kind: doc
    role: intent
    target: docs/guides/using-esoctl-tool.md
---

# Render a template

Try out the template of an ExternalSecret or PushSecret on a workstation, with
sample data instead of provider values, and see the resulting Kubernetes object
before applying anything to a cluster.
