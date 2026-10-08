---
appliesTo:
  - { type: capability, id: create-external-secret }
  - { type: capability, id: edit-external-secret }
  - { type: capability, id: refresh-external-secret }
references:
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go#Reconciler.ApplyTemplate
  - kind: doc
    role: intent
    target: docs/guides/templating.md
---

# A template's own data wins over templates from ConfigMaps and Secrets, which win over fetched values

When the Product renders a target Secret, a key written by the template's own
data takes precedence over the same key from a template kept in a ConfigMap or
Secret, which takes precedence over a fetched value. Fetched values are written
as they are when there is no template, or when the template asks to merge them;
keys others wrote to the Secret are kept only with creation policy Merge or
CreateOrMerge.
