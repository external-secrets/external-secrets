---
kind: alternative
routes:
  api: Kubernetes API
steps:
  - text: The Application Developer applies an SSHKey generator with its settings
    kind: actor
    actor: application-developer
    entities:
      - { entity: ssh-key, effect: creates, facts: [Key type, Key size, Comment] }
    contexts: { api: { place: kubernetes-api::namespace } }
---

# Create an SSHKey generator

## Trigger

A namespace needs a generator that produces an SSH key pair.

## Outcome

The SSHKey generator exists in the namespace; ExternalSecrets and PushSecrets there can name it.
