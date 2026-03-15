# Getting started

**Anchore Engine** is an open-source platform that provides centralized inspection, analysis, and certification of container images. When integrated with Kubernetes, it adds powerful features—such as preventing unscanned images from being deployed into your clusters.

## Installation with Helm

There are several parts of the installation that require credentials these being:

- `ANCHORE_ADMIN_USERNAME`
- `ANCHORE_ADMIN_PASSWORD`
- `ANCHORE_DB_PASSWORD`
- `db-url`
- `db-user`
- `postgres-password`

You can use an **ExternalSecret** to automatically fetch these credentials from your preferred backend provider. The following examples demonstrate how to configure it with **HashiCorp Vault** and **AWS Secrets Manager**.


#### Hashicorp Vault

``` yaml
{% include 'vault-anchore-engine-access-credentials-external-secret.yaml' %}
```


#### AWS Secrets Manager

``` yaml
{% include 'aws-anchore-engine-access-credentials-external-secret.yaml' %}
```

