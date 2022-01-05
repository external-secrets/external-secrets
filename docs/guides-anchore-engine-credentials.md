# Getting started

Anchore engine is one of the best of breed security scanning used for scanning docker images, filesystems and with Harbor allowing a complete solution to container scanning at build time, nightly and with the admission controller the prevention of unscanned images from being deployed into kubernetes clusters.

## Installing with Helm

There are several parts of the installation that require credentials these being :-

ANCHORE_ADMIN_USERNAME
ANCHORE_ADMIN_PASSWORD
ANCHORE_DB_PASSWORD
db-url
db-user
postgres-password


Creating the following external secret ensure the credentials are drawn from the backend provider of choice example for Hashicorp Vault and AWS Secrets Manager providers

#### Hashicorp Vault

``` yaml
{% include 'vault-anchore-engine-access-credentials-external-secret.yaml' %}
```


#### AWS Secrets Manager

``` yaml
{% include 'aws-anchore-engine-access-credentials-external-secret.yaml' %}
```

