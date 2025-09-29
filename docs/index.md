# [Sponsored by](https://opencollective.com/external-secrets-org)

[![cs-logo](./pictures/cs_logo.png)](https://container-solutions.com)
[![External Secrets inc.](./pictures/ESI_Logo.svg)](https://externalsecrets.com)
[![Form3](./pictures/form3_logo.png)](https://www.form3.tech/)
[![Pento](./pictures/pento_logo.png)](https://www.pento.io)

# External Secrets Operator

![high-level](./pictures/diagrams-high-level-simple.png)

**External Secrets Operator** is a Kubernetes operator that integrates external
secret management systems like [AWS Secrets
Manager](https://aws.amazon.com/secrets-manager/), [HashiCorp
Vault](https://www.vaultproject.io/), [Google Secrets
Manager](https://cloud.google.com/secret-manager), [Azure Key
Vault](https://azure.microsoft.com/en-us/services/key-vault/), [IBM Cloud Secrets Manager](https://www.ibm.com/cloud/secrets-manager), [CyberArk Secrets Manager](https://www.cyberark.com/products/secrets-management/), [Pulumi ESC](https://www.pulumi.com/product/esc/) and many more. The
operator reads information from external APIs and automatically injects the
values into a [Kubernetes
Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

## What is External Secrets Operator?

The goal of External Secrets Operator is to synchronize secrets from external
APIs into Kubernetes. ESO is a collection of custom API resources -
`ExternalSecret`, `SecretStore` and `ClusterSecretStore` that provide a
user-friendly abstraction for the external API that stores and manages the
lifecycle of the secrets for you.

## Find Your Path

ESO serves different roles in different organizations. Choose your path:

### 👨‍💼 **Decision Maker**
Evaluating ESO for your organization? Understanding business value and capabilities?

→ [**Start Here**](./personas/decision-maker.md)
- Business value and ROI
- Architecture overview
- Security and compliance
- Adoption considerations

### 🏗️ **Platform Administrator**
Installing and maintaining ESO in production clusters?

→ [**Platform Admin Guide**](./personas/platform-admin.md)
- Installation and deployment
- Operations and maintenance
- Monitoring and troubleshooting

### 🔧 **DevOps Engineer**
Configuring secret stores and managing ESO resources?

→ [**DevOps Engineer Guide**](./personas/devops-engineer.md)
- Secret store configuration
- Provider setup and integration
- Advanced features and automation

### 👩‍💻 **Application Developer**
Building applications that need to access secrets via ESO?

→ [**Application Developer Guide**](./personas/app-developer.md)
- Using ESO in your applications
- Secret retrieval patterns
- Provider integration

### 🔒 **Security Professional**
Evaluating ESO for security compliance and risk management?

→ [**Security Team Guide**](./personas/security-team.md)
- Security assessment and compliance
- Threat modeling and mitigation
- Access control and auditing

## Quick Start

Want to try ESO immediately?

- [**Getting Started**](./introduction/getting-started.md) - Deploy ESO and create your first secret
- [**Overview**](./introduction/overview.md) - Understand ESO architecture and concepts
- [**Find Your Path**](./personas/index.md) - More detailed persona guidance

## Community & Support

This project is driven by its users and contributors. Join our community:

- **Bi-weekly Development Meeting**: Every odd Wednesday at [8:00 PM Berlin Time](https://dateful.com/time-zone-converter?t=20:00&tz=Europe/Berlin)
  ([agenda](https://hackmd.io/GSGEpTVdRZCP6LDxV3FHJA), [jitsi call](https://meet.jit.si/eso-community-meeting))
- [**Kubernetes Slack #external-secrets**](https://kubernetes.slack.com/messages/external-secrets)
- [**Contributing Process**](./contributing/process.md)
- [**Twitter**](https://twitter.com/ExtSecretsOptr)

## Kicked off by

![godaddy-logo](./pictures/godaddy_logo.png)

