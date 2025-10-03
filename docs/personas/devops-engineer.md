# DevOps Engineer Guide

As a **DevOps Engineer**, you're responsible for configuring secret stores, managing external secret resources, and ensuring smooth integration between ESO and your infrastructure. This guide covers configuration and operational aspects.

## Getting Started

1. **Architecture Overview**: [ESO Components](../api/components.md)
2. **Configuration Basics**: [SecretStore vs ClusterSecretStore](../api/secretstore.md)
3. **Advanced Configuration**: [ClusterExternalSecret](../api/clusterexternalsecret.md)

## Secret Store Configuration

### Core Resources
- [SecretStore](../api/secretstore.md) - Namespace-scoped secret stores
- [ClusterSecretStore](../api/clustersecretstore.md) - Cluster-wide secret stores
- [ExternalSecret](../api/externalsecret.md) - Individual secret retrieval

### Provider Setup

#### Cloud Providers
- [AWS Secrets Manager](../provider/aws-secrets-manager.md)
- [AWS Parameter Store](../provider/aws-parameter-store.md)
- [Azure Key Vault](../provider/azure-key-vault.md)
- [Google Cloud Secret Manager](../provider/google-secrets-manager.md)

#### Enterprise Providers
- [HashiCorp Vault](../provider/hashicorp-vault.md)
- [CyberArk Conjur](../provider/conjur.md)
- [IBM Secrets Manager](../provider/ibm-secrets-manager.md)

#### Other Providers
- [Kubernetes API](../provider/kubernetes.md)
- [Webhook Provider](../provider/webhook.md)
- [Fake Provider](../provider/fake.md) (for testing)

## Advanced Features

### Data Transformation
- [Key Rewriting](../guides/datafrom-rewrite.md)
- [Advanced Templating](../guides/templating.md)
- [Secret Generators](../guides/generator.md)

### Dynamic Secrets
- [Vault Dynamic Secrets](../api/generator/vault.md)
- [AWS STS Tokens](../api/generator/sts.md)
- [Container Registry Tokens](../api/generator/ecr.md)

### Push Secrets (Bidirectional)
- [PushSecret Resource](../api/pushsecret.md)
- [Push Secrets Guide](../guides/pushsecrets.md)

## Operational Excellence

### Monitoring & Observability
- [Metrics](../api/metrics.md)
- [esoctl Tool](../guides/using-esoctl-tool.md)

### Security & Compliance
- [Security Best Practices](../guides/security-best-practices.md)
- [Multi-tenancy](../guides/multi-tenancy.md)

### Troubleshooting
- [FAQ](../introduction/faq.md)
- [Finding Secrets](../guides/getallsecrets.md)

## Migration & Integration

- [Migrating from v1beta1](../guides/v1beta1.md)
- [GitOps Integration](../examples/gitops-using-fluxcd.md)

## Reference

- [Complete API Reference](../api/spec.md)
- [Controller Options](../api/controller-options.md)
