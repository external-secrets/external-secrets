# External Secrets Operator V2 - Helm Charts

This directory contains production-ready Helm charts for External Secrets Operator V2.

## Available Charts

### [external-secrets-v2](./external-secrets-v2/)

Main controller chart for External Secrets Operator V2.

**Install**:
```bash
helm install external-secrets-v2 ./external-secrets-v2 \
  --namespace external-secrets-system \
  --create-namespace
```

**Features**:
- Automatic TLS certificate management
- Leader election for HA
- Prometheus metrics
- Security hardening
- Flexible RBAC

[📖 Chart Documentation](./external-secrets-v2/README.md)

### [external-secrets-v2-provider-aws](./external-secrets-v2-provider-aws/)

AWS Secrets Manager provider for External Secrets Operator V2.

**Install**:
```bash
helm install aws-provider ./external-secrets-v2-provider-aws \
  --namespace external-secrets-system \
  --set aws.region=us-east-1 \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::ACCOUNT:role/ROLE"
```

**Features**:
- IRSA (IAM Roles for Service Accounts) support
- Connection pooling (50x faster)
- Auto-scaling support
- High availability

[📖 Chart Documentation](./external-secrets-v2-provider-aws/README.md)

## Quick Start

### 1. Install Controller

```bash
helm install external-secrets-v2 ./external-secrets-v2 \
  --namespace external-secrets-system \
  --create-namespace \
  --wait
```

### 2. Install Provider

```bash
helm install aws-provider ./external-secrets-v2-provider-aws \
  --namespace external-secrets-system \
  --set aws.region=us-east-1 \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"="arn:aws:iam::123456789012:role/eso-aws" \
  --wait
```

### 3. Verify

```bash
kubectl get pods -n external-secrets-system
```

## Documentation

- 📘 [Quick Start Guide](../../examples/v2/helm-quick-start.md)
- 📗 [Installation Guide](../../docs/guides/helm-v2-installation.md)
- 📙 [Design Document](../../design/014-helm-charts-implementation.md)

## Testing

Run automated tests:

```bash
../../hack/test-helm-charts.sh all
```

## Development

### Lint Charts

```bash
helm lint ./external-secrets-v2
helm lint ./external-secrets-v2-provider-aws
```

### Template Rendering

```bash
helm template test ./external-secrets-v2 > rendered-controller.yaml
helm template test ./external-secrets-v2-provider-aws > rendered-provider.yaml
```

### Dry Run

```bash
helm install --dry-run test ./external-secrets-v2
helm install --dry-run test ./external-secrets-v2-provider-aws
```

## Production Deployment

### High Availability

```yaml
# values-ha.yaml
replicaCount: 3

podDisruptionBudget:
  enabled: true
  minAvailable: 2

metrics:
  enabled: true
  serviceMonitor:
    enabled: true

affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
    - labelSelector:
        matchLabels:
          app.kubernetes.io/name: external-secrets-v2
      topologyKey: kubernetes.io/hostname
```

```bash
helm install external-secrets-v2 ./external-secrets-v2 \
  --namespace external-secrets-system \
  --create-namespace \
  -f values-ha.yaml
```

## GitOps

### ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: external-secrets-v2
spec:
  project: default
  source:
    repoURL: https://charts.external-secrets.io
    chart: external-secrets-v2
    targetRevision: 0.1.0-alpha.1
  destination:
    server: https://kubernetes.default.svc
    namespace: external-secrets-system
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

### Flux

```yaml
apiVersion: helm.toolkit.fluxcd.io/v2beta1
kind: HelmRelease
metadata:
  name: external-secrets-v2
  namespace: flux-system
spec:
  interval: 10m
  chart:
    spec:
      chart: external-secrets-v2
      version: 0.1.0-alpha.1
      sourceRef:
        kind: HelmRepository
        name: external-secrets
  targetNamespace: external-secrets-system
  install:
    createNamespace: true
```

## Chart Versions

| Chart | Version | App Version | Status |
|-------|---------|-------------|--------|
| external-secrets-v2 | 0.1.0-alpha.1 | v0.1.0-alpha.1 | Alpha |
| external-secrets-v2-provider-aws | 0.1.0-alpha.1 | v0.1.0-alpha.1 | Alpha |

## Support

- 🐛 [Report Issues](https://github.com/external-secrets/external-secrets/issues)
- 💬 [Slack](https://kubernetes.slack.com/messages/external-secrets)
- 📚 [Documentation](https://external-secrets.io)

## License

Apache 2.0 - See [LICENSE](../../LICENSE)
