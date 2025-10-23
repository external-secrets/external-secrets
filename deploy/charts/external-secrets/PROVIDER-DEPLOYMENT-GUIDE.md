# External Secrets Monolithic Helm Chart - Provider Support

## Overview

The External Secrets Helm chart has been enhanced to support deploying one or multiple secret providers alongside the controller in a single, monolithic installation. This provides a unified deployment model where both the controller and providers are managed through a single Helm release.

## What's New

### Unified Deployment Model
- Deploy External Secrets controller and providers in a single Helm chart
- Each provider runs as an independent deployment with dedicated resources
- Support for multiple providers simultaneously (AWS, GCP, Azure, Vault, etc.)
- Per-provider configuration for authentication, resources, scaling, and security

### Template Structure

New template files added:
- `templates/provider-deployment.yaml` - Provider deployment with full configuration
- `templates/provider-service.yaml` - Provider gRPC service
- `templates/provider-serviceaccount.yaml` - Provider service accounts with cloud IAM annotations
- `templates/provider-poddisruptionbudget.yaml` - Pod disruption budgets for HA
- `templates/provider-hpa.yaml` - Horizontal Pod Autoscaler
- `templates/provider-servicemonitor.yaml` - Prometheus ServiceMonitor

Helper templates in `_helpers.tpl`:
- `external-secrets.provider.fullname` - Generate provider resource names
- `external-secrets.provider.labels` - Generate provider labels
- `external-secrets.provider.selectorLabels` - Generate selector labels
- `external-secrets.provider.serviceAccountName` - Get service account name
- `external-secrets.provider.image` - Construct provider image name

## Configuration

### Enabling Providers

Set `providers.enabled: true` and define providers in the `providers.list` array:

```yaml
providers:
  enabled: true
  list:
    - name: aws-primary
      type: aws
      enabled: true
      # ... configuration
```

### Provider Configuration Schema

Each provider supports:

**Identity & Metadata:**
- `name` - Unique identifier for the provider instance
- `type` - Provider type (aws, gcp, azure, vault, etc.)
- `enabled` - Enable/disable the provider

**Container Configuration:**
- `image.repository` - Container image repository
- `image.pullPolicy` - Pull policy (IfNotPresent, Always, Never)
- `image.tag` - Image tag (defaults to chart appVersion)
- `imagePullSecrets` - Pull secrets for private registries
- `replicaCount` - Number of replicas (default: 2)

**Service Account & Authentication:**
- `serviceAccount.create` - Create service account
- `serviceAccount.annotations` - Annotations for cloud IAM (IRSA, Workload Identity, etc.)
- `serviceAccount.name` - Custom service account name
- `serviceAccount.automount` - Automount service account token

**Security:**
- `podSecurityContext` - Pod-level security settings
- `securityContext` - Container-level security settings
- Both contexts support OpenShift compatibility via global settings

**Networking:**
- `service.type` - Service type (ClusterIP, LoadBalancer, etc.)
- `service.port` - gRPC port (default: 8080)
- `service.annotations` - Service annotations

**Resources & Scaling:**
- `resources.limits` - CPU and memory limits
- `resources.requests` - CPU and memory requests
- `autoscaling.enabled` - Enable HPA
- `autoscaling.minReplicas` - Minimum replicas
- `autoscaling.maxReplicas` - Maximum replicas
- `autoscaling.targetCPUUtilizationPercentage` - CPU target
- `autoscaling.targetMemoryUtilizationPercentage` - Memory target

**High Availability:**
- `podDisruptionBudget.enabled` - Enable PDB
- `podDisruptionBudget.minAvailable` - Minimum available pods
- `podDisruptionBudget.maxUnavailable` - Maximum unavailable pods
- `affinity` - Pod affinity/anti-affinity rules
- `topologySpreadConstraints` - Topology spread constraints
- `tolerations` - Node taints tolerations
- `nodeSelector` - Node selection constraints
- `priorityClassName` - Priority class

**TLS Configuration:**
- `tls.enabled` - Enable TLS
- `tls.certPath` - Certificate mount path
- `tls.caSecretName` - CA certificate secret name
- `tls.mountCA` - Mount CA certificate

**Provider-Specific Config:**
- `config` - Key-value map for provider settings
  - Converted to environment variables (uppercased, dots to underscores)
  - Example: `config.region: us-east-1` → `REGION=us-east-1`

**Logging:**
- `logging.level` - Log level (debug, info, warn, error)
- `logging.format` - Log format (json, console)
- `logging.development` - Development mode

**Metrics:**
- `metrics.enabled` - Enable metrics endpoint
- `metrics.port` - Metrics port (default: 8081)
- `metrics.serviceMonitor.enabled` - Create Prometheus ServiceMonitor
- `metrics.serviceMonitor.namespace` - ServiceMonitor namespace
- `metrics.serviceMonitor.interval` - Scrape interval
- `metrics.serviceMonitor.scrapeTimeout` - Scrape timeout
- `metrics.serviceMonitor.labels` - Additional labels

**Health Checks:**
- `health.port` - Health check port (default: 8082)
- `health.livenessProbe.enabled` - Enable liveness probe
- `health.livenessProbe.*` - Liveness probe settings
- `health.readinessProbe.enabled` - Enable readiness probe
- `health.readinessProbe.*` - Readiness probe settings

**Extra Configuration:**
- `extraEnv` - Additional environment variables
- `extraVolumes` - Additional volumes
- `extraVolumeMounts` - Additional volume mounts
- `podAnnotations` - Pod annotations
- `podLabels` - Pod labels

## Example Configurations

### Single AWS Provider with IRSA

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
          eks.amazonaws.com/role-arn: arn:aws:iam::123456789012:role/eso-provider-aws
      resources:
        limits:
          cpu: 200m
          memory: 256Mi
        requests:
          cpu: 50m
          memory: 64Mi
      config:
        region: us-east-1
        authMethod: irsa
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
          azure.workload.identity/client-id: "client-id"
      podLabels:
        azure.workload.identity/use: "true"
```

## Installation

### Install with providers

```bash
helm install external-secrets ./deploy/charts/external-secrets \
  -f values-with-providers.yaml
```

### Upgrade existing installation

```bash
helm upgrade external-secrets ./deploy/charts/external-secrets \
  -f values-with-providers.yaml
```

### Dry-run to test configuration

```bash
helm template test ./deploy/charts/external-secrets \
  -f values-with-providers.yaml
```

## Files Reference

### New Files
- `deploy/charts/external-secrets/templates/provider-deployment.yaml`
- `deploy/charts/external-secrets/templates/provider-service.yaml`
- `deploy/charts/external-secrets/templates/provider-serviceaccount.yaml`
- `deploy/charts/external-secrets/templates/provider-poddisruptionbudget.yaml`
- `deploy/charts/external-secrets/templates/provider-hpa.yaml`
- `deploy/charts/external-secrets/templates/provider-servicemonitor.yaml`
- `deploy/charts/external-secrets/values-with-providers-example.yaml`
- `deploy/charts/external-secrets/values-test.yaml`
- `deploy/charts/external-secrets/PROVIDERS.md`

### Modified Files
- `deploy/charts/external-secrets/values.yaml` - Added providers section
- `deploy/charts/external-secrets/templates/_helpers.tpl` - Added provider helpers
- `deploy/charts/external-secrets/README.md` - Added provider documentation

## Resource Naming Convention

Resources are named using the pattern:
```
<release-name>-external-secrets-provider-<provider-name>
```

For example, with release name "test" and provider name "aws":
- Deployment: `test-external-secrets-provider-aws`
- Service: `test-external-secrets-provider-aws`
- ServiceAccount: `test-external-secrets-provider-aws`

## Labels

All provider resources include:
- `app.kubernetes.io/name: external-secrets-provider-<provider-name>`
- `app.kubernetes.io/instance: <release-name>`
- `app.kubernetes.io/component: provider`
- `external-secrets.io/provider: <provider-type>`
- Standard Helm labels (chart, version, managed-by)
- User-defined common labels

## Best Practices

1. **Always use at least 2 replicas** for production deployments
2. **Enable pod disruption budgets** to maintain availability during updates
3. **Use cloud workload identity** (IRSA, Workload Identity) instead of static credentials
4. **Set resource limits** appropriate for your workload
5. **Enable metrics and ServiceMonitor** for observability
6. **Use anti-affinity rules** to spread replicas across nodes/zones
7. **Keep TLS enabled** for secure communication with the controller
8. **Enable health checks** for better reliability
9. **Use HPA** for automatic scaling based on load
10. **Configure security contexts** appropriately (non-root, read-only filesystem)

## Testing

Test configuration rendering:
```bash
cd deploy/charts/external-secrets
helm template test . -f values-test.yaml
```

Validate against Kubernetes:
```bash
helm template test . -f values-test.yaml | kubectl apply --dry-run=client -f -
```

## Future Enhancements

Possible future improvements:
- Auto-discovery of provider types from installed CRDs
- Provider-specific default values
- Support for provider-specific RBAC
- Integration with external certificate management (cert-manager)
- Provider health monitoring and automated failover
- Cross-namespace provider sharing

## Migration from Separate Provider Charts

If you're currently using separate provider Helm charts, you can migrate to this monolithic chart by:

1. Extracting your provider configuration from the separate chart values
2. Adding it to the `providers.list` array in the monolithic chart
3. Ensuring service account annotations and other cloud-specific settings are preserved
4. Uninstalling the separate provider charts
5. Installing/upgrading the monolithic chart with provider configuration

## Troubleshooting

### Check provider status
```bash
kubectl get deployments -l app.kubernetes.io/component=provider
kubectl get pods -l app.kubernetes.io/component=provider
```

### View provider logs
```bash
kubectl logs -l external-secrets.io/provider=aws -f
```

### Check metrics
```bash
kubectl port-forward svc/external-secrets-provider-aws 8081:8081
curl http://localhost:8081/metrics
```

### Verify service account
```bash
kubectl describe sa external-secrets-provider-aws
```

### Check TLS certificates
```bash
kubectl get secrets -l app.kubernetes.io/component=provider
```
