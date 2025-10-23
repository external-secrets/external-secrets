# Provider Deployment Guide

This guide explains how to deploy External Secrets with integrated provider deployments using the monolithic Helm chart.

## Overview

The External Secrets Helm chart now supports deploying one or multiple secret providers alongside the controller in a single installation. Each provider runs as an independent deployment with its own configuration, allowing you to:

- Deploy multiple providers simultaneously (AWS, GCP, Azure, Vault, etc.)
- Configure each provider independently with specific resource limits, security contexts, and authentication
- Scale providers independently based on workload requirements
- Enable high availability with pod disruption budgets and anti-affinity rules

## Basic Configuration

### Enable Provider Deployments

Set `providers.enabled` to `true` and define your providers in the `providers.list` array:

```yaml
providers:
  enabled: true
  list:
    - name: aws
      type: aws
      enabled: true
```

### Provider Structure

Each provider in the list supports the following configuration:

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `name` | string | Unique name for this provider instance | Yes |
| `type` | string | Provider type (aws, gcp, azure, vault, etc.) | Yes |
| `enabled` | boolean | Enable/disable this provider | Yes |
| `replicaCount` | int | Number of replicas (default: 2) | No |

## Configuration Sections

### Image Configuration

```yaml
image:
  repository: oci.external-secrets.io/external-secrets/provider-aws
  pullPolicy: IfNotPresent
  tag: ""  # Defaults to chart appVersion
```

### Service Account

```yaml
serviceAccount:
  create: true
  annotations:
    # Example: AWS IRSA
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-provider-aws
    # Example: GCP Workload Identity
    # iam.gke.io/gcp-service-account: eso-provider@project.iam.gserviceaccount.com
    # Example: Azure Workload Identity
    # azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
  name: ""  # Auto-generated if empty
  automount: true
```

### Security Contexts

```yaml
podSecurityContext:
  enabled: true
  runAsNonRoot: true
  runAsUser: 65532
  fsGroup: 65532
  seccompProfile:
    type: RuntimeDefault

securityContext:
  enabled: true
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 65532
  capabilities:
    drop:
    - ALL
```

### Resources

```yaml
resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 50m
    memory: 64Mi
```

### Service Configuration

```yaml
service:
  type: ClusterIP
  port: 8080
  annotations: {}
```

### High Availability

```yaml
# Pod Disruption Budget
podDisruptionBudget:
  enabled: true
  minAvailable: 1
  # maxUnavailable: 1

# Affinity rules for spreading pods
affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchLabels:
            app.kubernetes.io/component: provider
            external-secrets.io/provider: aws
        topologyKey: kubernetes.io/hostname

# Topology spread constraints
topologySpreadConstraints:
- maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
  labelSelector:
    matchLabels:
      external-secrets.io/provider: aws
```

### Auto-scaling

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

### TLS Configuration

```yaml
tls:
  enabled: true
  certPath: /etc/provider/certs
  caSecretName: external-secrets-v2-ca
  mountCA: true
```

### Provider-Specific Configuration

Use the `config` section to pass provider-specific settings:

```yaml
config:
  # AWS provider example
  region: us-east-1
  authMethod: irsa
  assumeRoleARN: ""
  externalID: ""
  
  # GCP provider example
  # projectID: my-project
  
  # Azure provider example
  # vaultURL: https://my-vault.vault.azure.net
  # tenantID: "00000000-0000-0000-0000-000000000000"
  
  # Vault provider example
  # vaultAddr: https://vault.example.com
  # authMethod: kubernetes
```

### Logging

```yaml
logging:
  level: info  # debug, info, warn, error
  format: json  # json, console
  development: false
```

### Metrics

```yaml
metrics:
  enabled: true
  port: 8081
  serviceMonitor:
    enabled: true
    namespace: ""  # Defaults to release namespace
    interval: 30s
    scrapeTimeout: 10s
    labels: {}
```

### Health Checks

```yaml
health:
  port: 8082
  livenessProbe:
    enabled: true
    initialDelaySeconds: 10
    periodSeconds: 20
    timeoutSeconds: 5
    failureThreshold: 3
  readinessProbe:
    enabled: true
    initialDelaySeconds: 5
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 3
```

### Extra Configuration

```yaml
# Additional environment variables
extraEnv:
- name: CUSTOM_VAR
  value: "custom-value"
- name: SECRET_VAR
  valueFrom:
    secretKeyRef:
      name: my-secret
      key: password

# Additional volumes
extraVolumes:
- name: custom-config
  configMap:
    name: provider-config

# Additional volume mounts
extraVolumeMounts:
- name: custom-config
  mountPath: /etc/config
  readOnly: true

# Pod annotations
podAnnotations:
  prometheus.io/scrape: "true"

# Pod labels
podLabels:
  environment: production

# Node selector
nodeSelector:
  cloud.provider/instance-type: standard

# Tolerations
tolerations:
- key: "provider-workload"
  operator: "Equal"
  value: "true"
  effect: "NoSchedule"

# Priority class
priorityClassName: high-priority
```

## Examples

### AWS Provider with IRSA

```yaml
providers:
  enabled: true
  list:
    - name: aws-us-east-1
      type: aws
      enabled: true
      replicaCount: 3
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-aws
      
      serviceAccount:
        create: true
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-provider-aws
      
      resources:
        limits:
          cpu: 300m
          memory: 512Mi
        requests:
          cpu: 100m
          memory: 128Mi
      
      config:
        region: us-east-1
        authMethod: irsa
      
      podDisruptionBudget:
        enabled: true
        minAvailable: 2
      
      metrics:
        enabled: true
        serviceMonitor:
          enabled: true
```

### Multiple Providers

```yaml
providers:
  enabled: true
  list:
    # AWS Provider
    - name: aws
      type: aws
      enabled: true
      replicaCount: 2
      image:
        repository: oci.external-secrets.io/external-secrets/provider-aws
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-aws
      config:
        region: us-east-1
        authMethod: irsa
    
    # GCP Provider
    - name: gcp
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
    
    # Azure Provider
    - name: azure
      type: azure
      enabled: true
      replicaCount: 2
      image:
        repository: oci.external-secrets.io/external-secrets/provider-azure
      serviceAccount:
        annotations:
          azure.workload.identity/client-id: "00000000-0000-0000-0000-000000000000"
      podLabels:
        azure.workload.identity/use: "true"
      config:
        vaultURL: https://my-vault.vault.azure.net
        tenantID: "00000000-0000-0000-0000-000000000000"
```

### Provider with Custom Authentication Secret

```yaml
providers:
  enabled: true
  list:
    - name: vault
      type: vault
      enabled: true
      
      image:
        repository: oci.external-secrets.io/external-secrets/provider-vault
      
      config:
        vaultAddr: https://vault.example.com
        authMethod: token
      
      extraEnv:
      - name: VAULT_TOKEN
        valueFrom:
          secretKeyRef:
            name: vault-token
            key: token
      
      extraVolumes:
      - name: vault-ca
        secret:
          secretName: vault-ca-cert
      
      extraVolumeMounts:
      - name: vault-ca
        mountPath: /etc/vault/ca
        readOnly: true
```

## Installation

### Install with providers

```bash
helm install external-secrets external-secrets/external-secrets \
  -f values-with-providers.yaml
```

### Upgrade existing installation

```bash
helm upgrade external-secrets external-secrets/external-secrets \
  -f values-with-providers.yaml
```

### Install with specific provider enabled

```bash
helm install external-secrets external-secrets/external-secrets \
  --set providers.enabled=true \
  --set providers.list[0].name=aws \
  --set providers.list[0].type=aws \
  --set providers.list[0].enabled=true \
  --set providers.list[0].replicaCount=2 \
  --set providers.list[0].image.repository=oci.external-secrets.io/external-secrets/provider-aws
```

## Troubleshooting

### Check provider deployment status

```bash
kubectl get deployments -l app.kubernetes.io/component=provider
```

### View provider logs

```bash
kubectl logs -l external-secrets.io/provider=aws -f
```

### Check provider metrics

```bash
kubectl port-forward svc/external-secrets-provider-aws 8081:8081
curl http://localhost:8081/metrics
```

### Verify TLS connectivity

```bash
kubectl exec -it deployment/external-secrets-provider-aws -- sh
# Check if certificates are mounted
ls -la /etc/provider/certs
```

## Best Practices

1. **High Availability**: Always use `replicaCount >= 2` for production
2. **Resource Limits**: Set appropriate resource limits based on your workload
3. **Pod Disruption Budgets**: Enable PDBs to prevent all replicas from being evicted
4. **Anti-Affinity**: Use pod anti-affinity to spread replicas across nodes/zones
5. **Monitoring**: Enable metrics and ServiceMonitor for observability
6. **Authentication**: Use workload identity (IRSA, Workload Identity) instead of static credentials
7. **TLS**: Keep TLS enabled for secure provider-controller communication
8. **Auto-scaling**: Use HPA for dynamic scaling based on load
9. **Health Checks**: Enable liveness and readiness probes for better reliability
10. **Security Context**: Use restrictive security contexts (non-root, read-only filesystem)
