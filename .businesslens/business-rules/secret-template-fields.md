---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: edit-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go#ValidateSecretTemplateFromTargets
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go#Reconciler.ApplyTemplate
---

# Templates for a Secret write only its data, labels and annotations

A template kept in a ConfigMap or Secret may fill only the data, labels and
annotations of a target Secret; any other part, such as its type or owners, is
refused at admission and again before the Product renders the target. Templates
for a custom resource may fill other parts.

## Rationale

Writing other fields would let a template turn a Secret into a privileged one
without the checks on its type.
