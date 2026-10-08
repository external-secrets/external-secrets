---
appliesTo:
  - { type: entity, id: generic-target, effect: creates }
permits:
  - { configuredBy: kubernetes-role, when: [{ entity: controller-options, fact: Generic targets, is: Allowed }] }
  - { unattended: true, when: [{ entity: controller-options, fact: Generic targets, is: Allowed }] }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_manifest.go
  - kind: doc
    role: intent
    target: docs/guides/targeting-custom-resources.md
  - kind: code
    role: context
    target: deploy/charts/external-secrets/values.yaml
---

# Generic targets are written only while the controller options allow them

The Product writes a ConfigMap or custom resource for an ExternalSecret only
while its controller options allow generic targets; otherwise the
ExternalSecret reports that they are disabled and nothing is written.

## Rationale

A generic target lets anyone who can create an ExternalSecret write other kinds
of resources with the Product's permissions, so it is off unless a cluster
operator turns it on.
