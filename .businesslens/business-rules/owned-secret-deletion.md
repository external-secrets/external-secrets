---
appliesTo:
  - { type: entity, id: secret, effect: removes }
permits:
  - { configuredBy: kubernetes-role, when: [{ fact: Owner, present: true }] }
  - { unattended: true, when: [{ fact: Owner, present: true }] }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.cleanupManagedSecrets
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller.go#Reconciler.deleteOrphanedSecrets
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
---

# The Product deletes a target Secret only when its ExternalSecret owns it

Whether an ExternalSecret is deleted, renames its target or loses its provider
values, a target Secret goes only when that ExternalSecret owns it.

## Rationale

Targets created with Orphan, Merge, CreateOrMerge or None may hold data others
manage; deleting them would destroy it.
