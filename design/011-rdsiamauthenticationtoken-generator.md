```yaml
---
title: RDSIAMAuthenticationToken Generator
version: v1alpha1
authors: Lee Zhen Yong
creation-date: 2024-08-08
status: draft
---
```

# RDSIAMAuthenticationToken Generator

## Table of Contents

<!-- toc -->
// autogen please
<!-- /toc -->

## Summary

A new [external secrets generator](https://external-secrets.io/latest/guides/generator/) that generates ephemeral tokens for RDS databases that uses [IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html).

## Motivation

[RDS IAM database authentication](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.IAMDBAuth.html) is a feature that facilitates short lived credentials that are generated on-demand. Although a more secure option than long lived credentials, it requires changes in application code to generate the token prior to connect to the database.

With `RDSIAMAuthenticationToken` generator, we can use external secrets operator to generate the tokens on our behalf. As far as the application is concerned, it is still using username and password (read from environment variables from kubernetes secret).

### Goals

 - Easier adoption of RDS IAM database authentication
 - Better performance, since tokens are cached instead of needing to be generated on-demand in the application per connection

## Proposal

A new resource kind `RDSIAMAuthenticationToken` of type `generators.external-secrets.io`.

### Output Keys and Values

| Key      | Description                       |
| -------- | --------------------------------- |
| user     | database user.                    |
| host     | database host.                    |
| port     | database port.                    |
| region   | AWS region.                       |
| token    | Generated authentication token.   |

### Example CRD

The authentication parameters follows [`ECRAuthorizationToken`](https://external-secrets.io/latest/api/generator/ecr/):

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: RDSIAMAuthenticationToken
metadata:
  name: postgres-creds
spec:

  # specify aws region (mandatory)
  region: us-east-1

  user: "bruceoutdoors"
  host: "mysqldb.123456789012.us-east-1.rds.amazonaws.com"
  port: "3306"

  # choose an authentication strategy
  # if no auth strategy is defined it falls back to using
  # credentials from the environment of the controller.
  auth:

    # 1: static credentials
    # point to a secret that contains static credentials
    # like AWS_ACCESS_KEY_ID / AWS_SECRET_ACCESS_KEY
    secretRef:
      accessKeyIDSecretRef:
        name: "my-aws-creds"
        key: "key-id"
      secretAccessKeySecretRef:
        name: "my-aws-creds"
        key: "access-secret"

    # option 2: IAM Roles for Service Accounts
    # point to a service account that should be used
    # that is configured for IAM Roles for Service Accounts (IRSA)
    jwt:
      serviceAccountRef:
        name: "my-aws-service"
```
