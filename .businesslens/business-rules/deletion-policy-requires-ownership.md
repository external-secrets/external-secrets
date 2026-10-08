---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: edit-external-secret }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go
  - kind: doc
    role: intent
    target: docs/guides/ownership-deletion-policy.md
---

# Deletion policy Delete requires creation policy Owner

The Product refuses an ExternalSecret that asks to delete its target while creation policy is Merge, CreateOrMerge or None, and one that asks to merge on deletion while creation policy is None.
