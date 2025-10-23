# Quick Start: Deploy External Secrets with Providers

This guide will help you quickly deploy External Secrets with provider support.

## Prerequisites

- Kubernetes cluster (1.19+)
- Helm 3.x
- kubectl configured

## Quick Start

### 1. Add Helm Repository (if not already added)

```bash
helm repo add external-secrets https://charts.external-secrets.io
helm repo update
```

### 2. Basic Installation with Single Provider

Create a `values.yaml` file:

```yaml
# Basic controller configuration
installCRDs: true
replicaCount: 1

# Enable provider deployments
providers:
  enabled: true
  list:
    - name: aws
      type: aws
      enabled: true
      replicaCount: 2
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-aws
      
      serviceAccount:
        create: true
        # Add your cloud IAM annotations here
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::YOUR-ACCOUNT:role/YOUR-ROLE
      
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
        requests:
          cpu: 50m
          memory: 64Mi
      
      config:
        region: us-east-1
```

### 3. Install

```bash
helm install external-secrets external-secrets/external-secrets -f values.yaml
```

### 4. Verify Installation

```bash
# Check controller
kubectl get pods -l app.kubernetes.io/name=external-secrets

# Check provider
kubectl get pods -l app.kubernetes.io/component=provider

# Check all resources
kubectl get all -l app.kubernetes.io/instance=external-secrets
```

## Provider-Specific Examples

### AWS with IRSA

```yaml
providers:
  enabled: true
  list:
    - name: aws
      type: aws
      enabled: true
      replicaCount: 2
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-aws
      
      serviceAccount:
        create: true
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/external-secrets-provider
      
      config:
        region: us-east-1
        authMethod: irsa
```

**Required AWS IAM Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue",
        "secretsmanager:DescribeSecret"
      ],
      "Resource": "*"
    }
  ]
}
```

### GCP with Workload Identity

```yaml
providers:
  enabled: true
  list:
    - name: gcp
      type: gcp
      enabled: true
      replicaCount: 2
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-gcp
      
      serviceAccount:
        create: true
        annotations:
          iam.gke.io/gcp-service-account: external-secrets@PROJECT-ID.iam.gserviceaccount.com
      
      config:
        projectID: PROJECT-ID
```

**Required GCP IAM Role:**
```bash
gcloud projects add-iam-policy-binding PROJECT-ID \
  --member="serviceAccount:external-secrets@PROJECT-ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### Azure with Workload Identity

```yaml
providers:
  enabled: true
  list:
    - name: azure
      type: azure
      enabled: true
      replicaCount: 2
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-azure
      
      serviceAccount:
        create: true
        annotations:
          azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
      
      podLabels:
        azure.workload.identity/use: "true"
      
      config:
        vaultURL: https://my-keyvault.vault.azure.net
        tenantID: "00000000-0000-0000-0000-000000000000"
```

### HashiCorp Vault

```yaml
providers:
  enabled: true
  list:
    - name: vault
      type: vault
      enabled: true
      replicaCount: 2
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-vault
      
      serviceAccount:
        create: true
      
      config:
        vaultAddr: https://vault.example.com
        authMethod: kubernetes
      
      extraEnv:
      - name: VAULT_NAMESPACE
        value: "admin"
```

## Multiple Providers Example

```yaml
providers:
  enabled: true
  list:
    # AWS Provider
    - name: aws-prod
      type: aws
      enabled: true
      replicaCount: 3
      image:
        repository: oci.external-secrets.io/external-secrets/provider-aws
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-aws
      config:
        region: us-east-1
      
    # GCP Provider
    - name: gcp-prod
      type: gcp
      enabled: true
      replicaCount: 2
      image:
        repository: oci.external-secrets.io/external-secrets/provider-gcp
      serviceAccount:
        annotations:
          iam.gke.io/gcp-service-account: eso@project.iam.gserviceaccount.com
      config:
        projectID: my-project
```

## Common Configurations

### Enable Auto-Scaling

```yaml
providers:
  list:
    - name: aws
      # ...other config...
      autoscaling:
        enabled: true
        minReplicas: 2
        maxReplicas: 10
        targetCPUUtilizationPercentage: 80
```

### Enable Metrics & Monitoring

```yaml
providers:
  list:
    - name: aws
      # ...other config...
      metrics:
        enabled: true
        port: 8081
        serviceMonitor:
          enabled: true
          interval: 30s
```

### High Availability Configuration

```yaml
providers:
  list:
    - name: aws
      replicaCount: 3
      
      podDisruptionBudget:
        enabled: true
        minAvailable: 2
      
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchLabels:
                  external-secrets.io/provider: aws
              topologyKey: kubernetes.io/hostname
      
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: ScheduleAnyway
        labelSelector:
          matchLabels:
            external-secrets.io/provider: aws
```

## Verification Commands

```bash
# Check all resources
kubectl get all -l app.kubernetes.io/instance=external-secrets

# Check provider deployments
kubectl get deployment -l app.kubernetes.io/component=provider

# Check provider pods
kubectl get pods -l app.kubernetes.io/component=provider

# Check provider services
kubectl get svc -l app.kubernetes.io/component=provider

# View provider logs
kubectl logs -l external-secrets.io/provider=aws -f

# Check metrics
kubectl port-forward svc/external-secrets-provider-aws 8081:8081 &
curl http://localhost:8081/metrics

# Check health
kubectl port-forward svc/external-secrets-provider-aws 8082:8082 &
curl http://localhost:8082/healthz
curl http://localhost:8082/readyz
```

## Troubleshooting

### Provider Pod Not Starting

```bash
# Check pod status
kubectl describe pod -l external-secrets.io/provider=aws

# Check logs
kubectl logs -l external-secrets.io/provider=aws --tail=50

# Check events
kubectl get events --sort-by='.lastTimestamp' | grep provider
```

### Authentication Issues

**AWS IRSA:**
```bash
# Check service account annotations
kubectl describe sa external-secrets-provider-aws

# Check if role is properly configured
aws sts get-caller-identity

# Check if pod has the right environment variables
kubectl exec -it deployment/external-secrets-provider-aws -- env | grep AWS
```

**GCP Workload Identity:**
```bash
# Check service account annotations
kubectl describe sa external-secrets-provider-gcp

# Verify workload identity binding
gcloud iam service-accounts get-iam-policy \
  external-secrets@PROJECT-ID.iam.gserviceaccount.com
```

### Check Connectivity

```bash
# Check if provider service is accessible
kubectl get svc -l app.kubernetes.io/component=provider

# Test connectivity from controller to provider
kubectl exec -it deployment/external-secrets -- \
  nc -zv external-secrets-provider-aws 8080
```

## Next Steps

1. **Create a SecretStore** pointing to your provider:
   ```yaml
   apiVersion: external-secrets.io/v1beta1
   kind: SecretStore
   metadata:
     name: aws-secrets
   spec:
     provider:
       aws:
         service: SecretsManager
         region: us-east-1
   ```

2. **Create an ExternalSecret** to sync secrets:
   ```yaml
   apiVersion: external-secrets.io/v1beta1
   kind: ExternalSecret
   metadata:
     name: my-secret
   spec:
     refreshInterval: 1h
     secretStoreRef:
       name: aws-secrets
       kind: SecretStore
     target:
       name: my-k8s-secret
     data:
     - secretKey: password
       remoteRef:
         key: my-secret-name
   ```

3. **Verify the secret was created**:
   ```bash
   kubectl get externalsecret
   kubectl get secret my-k8s-secret
   ```

## Additional Resources

- [Provider Deployment Guide](./PROVIDERS.md) - Comprehensive provider configuration reference
- [Official Documentation](https://external-secrets.io/)
- [GitHub Repository](https://github.com/external-secrets/external-secrets)
- [Example Values Files](./values-with-providers-example.yaml)

## Common Helm Commands

```bash
# Install
helm install external-secrets external-secrets/external-secrets -f values.yaml

# Upgrade
helm upgrade external-secrets external-secrets/external-secrets -f values.yaml

# Uninstall
helm uninstall external-secrets

# Dry-run (test without installing)
helm install external-secrets external-secrets/external-secrets -f values.yaml --dry-run

# Template (see generated manifests)
helm template external-secrets external-secrets/external-secrets -f values.yaml

# Get values
helm get values external-secrets

# Get all information
helm status external-secrets
```
