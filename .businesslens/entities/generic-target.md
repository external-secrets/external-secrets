---
domain: core-resources
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_manifest.go
  - kind: doc
    role: intent
    target: docs/guides/targeting-custom-resources.md
---

# Generic target

A ConfigMap or custom resource the Product writes for an ExternalSecret instead
of a Secret. Generic targets are written only while the cluster operator
allows them in the controller options, and the Product's own permissions must
cover the resource kind. A template cannot change the kind of resource
written.

## Information kept

- **Kind** — the API version and kind of the resource written
- **Content** — the fields rendered from the ExternalSecret's template and fetched values
- **Owner** — the ExternalSecret that owns it, when its creation policy is Owner
