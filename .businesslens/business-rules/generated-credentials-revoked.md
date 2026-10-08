---
appliesTo:
  - { type: capability, id: refresh-external-secret }
  - { type: capability, id: delete-external-secret }
  - { type: capability, id: refresh-push-secret }
  - { type: capability, id: delete-push-secret }
references:
  - kind: code
    role: implementation
    target: runtime/statemanager/statemanager.go
  - kind: code
    role: implementation
    target: pkg/controllers/generatorstate/generatorstate_controller.go
  - kind: code
    role: implementation
    target: generators/v1/gitlab/gitlab.go
  - kind: code
    role: implementation
    target: generators/v1/grafana/grafana.go
---

# A credential a generator issued is revoked once it is replaced or no longer used

When a GitlabDeployToken or Grafana generator issues a new credential for an
ExternalSecret or PushSecret, the Product revokes the one it issued before,
after a grace period of two minutes by default, and revokes the last one when
the ExternalSecret or PushSecret is deleted. A credential issued during a sync
that then fails is revoked straight away. For ExternalSecrets this holds while
the controller options keep generator state on; other generator kinds issue
nothing that needs revoking.

## Rationale

Every refresh issues a fresh credential; without revocation the old ones would
stay valid at GitLab or Grafana.
