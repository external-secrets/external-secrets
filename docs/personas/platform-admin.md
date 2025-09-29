# Platform Administrator Guide

As a **Platform Administrator**, you're responsible for installing, configuring, and maintaining External Secrets Operator (ESO) in your Kubernetes clusters. This guide will help you get started with deployment and operational aspects.

## Quick Start

If you're new to ESO, start here:

1. **Installation**: [Getting Started Guide](../introduction/getting-started.md)
2. **Prerequisites**: [System Requirements](../introduction/prerequisites.md)
3. **Overview**: [What is ESO?](../introduction/overview.md)

## Installation & Setup

### Deploying ESO
- [Helm Installation](../introduction/getting-started.md#install-external-secrets-operator)
- [Custom Resource Definitions](../api/spec.md)
- [Controller Options](../api/controller-options.md)

### Configuration
- [Multi-tenancy Setup](../guides/multi-tenancy.md)
- [Controller Classes](../guides/controller-class.md)
- [Disable Cluster Features](../guides/disable-cluster-features.md)

## Operations & Maintenance

### Monitoring
- [Metrics](../api/metrics.md)
- [Health Checks](../api/controller-options.md)

### Security
- [Security Best Practices](../guides/security-best-practices.md)
- [Threat Model](../guides/threat-model.md)

### Upgrades
- [Upgrading Guide](../guides/v1beta1.md)
- [Deprecation Policy](../introduction/deprecation-policy.md)
- [Stability & Support](../introduction/stability-support.md)

## Advanced Topics

- [Push Secrets](../guides/pushsecrets.md) - For bidirectional secret synchronization
- [Secret Generators](../guides/generator.md) - Dynamic secret generation
- [esoctl Tool](../guides/using-esoctl-tool.md) - CLI management tool

## Need Help?

- [FAQ](../introduction/faq.md)
- [Community Resources](../eso-tools.md)
- [Contributing](../contributing/process.md)
