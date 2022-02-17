# Getting started

Anchore Engine is an open-source project that provides a centralized service for inspection, analysis, and certification of container images. With Kubernetes, it also brings nice features like preventing unscanned images from being deployed into your clusters

## Installing with Helm

There are several parts of the installation that require credentials these being :-

ANCHORE_ADMIN_USERNAME
ANCHORE_ADMIN_PASSWORD
ANCHORE_DB_PASSWORD
db-url
db-user
postgres-password


Creating the following external secret ensure the credentials are drawn from the backend provider of choice. The example shown here works with Hashicorp Vault and AWS Secrets Manager providers.

#### Hashicorp Vault

``` yaml
{% include 'vault-anchore-engine-access-credentials-external-secret.yaml' %}
```


#### AWS Secrets Manager

``` yaml
{% include 'aws-anchore-engine-access-credentials-external-secret.yaml' %}
```

