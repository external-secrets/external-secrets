
## senhasegura DevOps Secrets Management (DSM)

External Secrets Operator integrates with senhasegura DevOps Secrets Management (DSM) module to sync application secrets to secrets held on the Kubernetes cluster.

### Requirements

  - senhasegura with DSM module enabled
  - senhasegura DSM Application Authorization with secrets


### SecretStore example

``` yaml
{% include 'senhasegura-dsm-store.yaml' %}
```

### External Secret example

``` yaml
{% include 'senhasegura-dsm-external-secret.yaml' %}
```