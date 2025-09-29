# Application Developer Guide

As an **Application Developer**, you want to integrate External Secrets Operator (ESO) into your applications to securely access secrets from external providers. This guide focuses on using ESO rather than operating it.

## Quick Start

Get up and running quickly:

1. **What is ESO?**: [Overview](../introduction/overview.md)
2. **Basic Usage**: [Introduction to Guides](../guides/introduction.md)
3. **Your First Secret**: [Getting Started](../introduction/getting-started.md)

## Accessing Secrets

### Basic Secret Retrieval
- [ExternalSecret Resource](../api/externalsecret.md)
- [SecretStore Configuration](../api/secretstore.md)
- [Common Kubernetes Secret Types](../guides/common-k8s-secret-types.md)

### Advanced Features
- [Extracting Structured Data](../guides/all-keys-one-secret.md)
- [Templating Secrets](../guides/templating.md)
- [Decoding Strategies](../guides/decoding-strategy.md)

## Provider Integration

Choose your secret provider:

### Cloud Providers
- [AWS Secrets Manager](../provider/aws-secrets-manager.md)
- [AWS Parameter Store](../provider/aws-parameter-store.md)
- [Google Cloud Secret Manager](../provider/google-secrets-manager.md)
- [Azure Key Vault](../provider/azure-key-vault.md)

### Other Providers
- [HashiCorp Vault](../provider/hashicorp-vault.md)
- [GitLab Variables](../provider/gitlab-variables.md)
- [GitHub Actions](../provider/github.md)

## Examples & Tutorials

- [Jenkins Credentials](../examples/jenkins-kubernetes-credentials.md)
- [Anchore Engine](../examples/anchore-engine-credentials.md)
- [GitOps with FluxCD](../examples/gitops-using-fluxcd.md)

## Best Practices

- [Security Considerations](../guides/security-best-practices.md)
- [Secret Lifecycle Management](../guides/ownership-deletion-policy.md)

## API Reference

- [Complete API Specification](../api/spec.md)
- [Selectable Fields](../api/selectable-fields.md)
