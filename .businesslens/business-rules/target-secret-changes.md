---
appliesTo:
  - { type: entity, id: secret, effect: changes }
permits:
  - { configuredBy: kubernetes-role, when: [{ fact: Type, is-not: bootstrap.kubernetes.io/token }, { fact: Type, is-not: kubernetes.io/service-account-token }] }
  - { configuredBy: kubernetes-role, when: [{ fact: Type, is: kubernetes.io/service-account-token }, { fact: Service account, absent: true }] }
  - { unattended: true, when: [{ fact: Type, is-not: bootstrap.kubernetes.io/token }, { fact: Type, is-not: kubernetes.io/service-account-token }] }
  - { unattended: true, when: [{ fact: Type, is: kubernetes.io/service-account-token }, { fact: Service account, absent: true }] }
references:
  - kind: code
    role: implementation
    target: apis/externalsecrets/v1/externalsecret_validator.go#validatePrivilegedTemplate
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_template.go#validateSecretCandidate
  - kind: code
    role: implementation
    target: pkg/controllers/externalsecret/externalsecret_controller_manifest.go
---

# The Product changes a target Secret for whoever may manage its ExternalSecret, never as a privileged token

The Product writes a target Secret on behalf of anyone whose Kubernetes roles
let them manage the ExternalSecret, and on its own schedule as it refreshes —
but never a bootstrap token Secret, and never a service-account token Secret
bound to a service account. The admission webhook refuses such templates, and
the Product checks the rendered Secret again before writing it, whether it is
the default target or named as a manifest.

## Rationale

The Product writes with its own permissions, so a privileged token Secret would
let anyone who can create an ExternalSecret mint long-lived cluster credentials
beyond their own roles.
