---
appliesTo:
  - { type: entity, id: provider-secret, effect: changes }
permits:
  - { configuredBy: kubernetes-role, when: [{ entity: controller-options, fact: Push secrets, is: Enabled }] }
  - { unattended: true, when: [{ entity: controller-options, fact: Push secrets, is: Enabled }] }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/pushsecret/pushsecret_controller.go#Reconciler.Reconcile
  - kind: doc
    role: intent
    target: docs/guides/disable-cluster-features.md
  - kind: code
    role: context
    target: cmd/controller/root.go
---

# The Product updates a provider secret only for a PushSecret, while it processes PushSecrets

Only a PushSecret makes the Product write to a provider, on behalf of whoever
may manage that PushSecret or on its own schedule as it pushes again — and only
while its controller options have PushSecrets processed. The store's
credentials and the provider decide whether the write succeeds.

## Rationale

Writing into providers is the riskiest thing the Product does, so an
installation can run with it switched off entirely.
