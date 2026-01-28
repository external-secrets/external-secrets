# Testing External Secrets Fork with mt-gitops

This guide explains how to build and test your forked external-secrets operator with Pulumi OIDC support.

## Prerequisites

1. Docker installed and authenticated to a container registry
2. Access to a Kubernetes cluster (EKS)
3. kubectl configured
4. Pulumi ESC environment configured for OIDC

## Step 1: Build Custom Image

```bash
# Set your image name
export IMAGE_NAME=ghcr.io/johnstonmatt/external-secrets
export IMAGE_TAG=pulumi-oidc-$(git rev-parse --short HEAD)

# Build for your platform (AMD64 for most EKS)
make docker.build IMAGE_NAME=${IMAGE_NAME} IMAGE_TAG=${IMAGE_TAG}

# Or build multi-platform (if you have ARM nodes)
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ${IMAGE_NAME}:${IMAGE_TAG} \
  --push .
```

**Alternative: Use GitHub Actions**
```bash
# Push your branch and let GitHub Actions build it
git push origin add-pulumi-oidc-support

# Then find the image at:
# ghcr.io/johnstonmatt/external-secrets:<git-sha>
```

## Step 2: Deploy to Your Cluster

### Option A: Update Existing Deployment

```bash
# Get current deployment
kubectl get deployment -n external-secrets external-secrets -o yaml > /tmp/eso-deployment.yaml

# Update the image
kubectl set image deployment/external-secrets \
  -n external-secrets \
  external-secrets=${IMAGE_NAME}:${IMAGE_TAG}

# Verify it's running
kubectl rollout status deployment/external-secrets -n external-secrets
kubectl get pods -n external-secrets
```

### Option B: Install Fresh with Helm

```bash
# Add your custom values
cat > /tmp/eso-custom-values.yaml <<EOF
image:
  repository: ${IMAGE_NAME}
  tag: ${IMAGE_TAG}
  pullPolicy: Always
EOF

# Install/Upgrade
helm upgrade --install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets \
  --create-namespace \
  -f /tmp/eso-custom-values.yaml
```

## Step 3: Configure Pulumi OIDC in Your ESC Organization

Before testing, configure OIDC in Pulumi:

1. Go to Pulumi Console → Organization Settings → OIDC
2. Add new OIDC provider:
   - Provider: Custom
   - Issuer URL: Your EKS cluster's OIDC issuer
   - Audience: `https://api.pulumi.com`
3. Create a Service Account Identity and note the identity ID

## Step 4: Create Test Resources in Your Cluster

```bash
# Create ServiceAccount
kubectl create serviceaccount pulumi-oidc-test -n default

# Create ClusterSecretStore with OIDC
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: pulumi-oidc-test
  namespace: default
spec:
  provider:
    pulumi:
      organization: supabase  # Your Pulumi org
      project: mt-services-combined  # Your ESC project
      environment: staging-apse1  # Your ESC environment
      auth:
        oidcConfig:
          organization: supabase
          serviceAccountRef:
            name: pulumi-oidc-test
          expirationSeconds: 600
EOF
```

## Step 5: Verify OIDC Authentication

```bash
# Check SecretStore status
kubectl get secretstore pulumi-oidc-test -n default -o yaml

# Check operator logs for OIDC token exchange
kubectl logs -n external-secrets deployment/external-secrets -f | grep -i oidc

# Create a test ExternalSecret
cat <<EOF | kubectl apply -f -
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: pulumi-oidc-test
  namespace: default
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: pulumi-oidc-test
    kind: SecretStore
  target:
    name: pulumi-test-secret
  data:
  - secretKey: test-key
    remoteRef:
      key: environmentVariables.SOME_KEY  # Path to a value in your ESC environment
EOF

# Check if secret was created
kubectl get externalsecret pulumi-oidc-test -n default
kubectl get secret pulumi-test-secret -n default
```

## Step 6: Test from mt-gitops

Update your mt-gitops configuration to use OIDC:

```bash
cd ../mt-gitops

# Update your Pulumi stack to create ServiceAccount for ESO
# In mt/cluster/external-secrets/pulumi-esc-store.ts or similar:
```

```typescript
// Create ServiceAccount for Pulumi ESC OIDC
const pulumiOidcSA = new k8s.core.v1.ServiceAccount("pulumi-esc-oidc", {
  metadata: {
    name: "pulumi-esc-oidc",
    namespace: "external-secrets",
  },
});

// Create ClusterSecretStore with OIDC
const pulumiEscStore = new k8s.apiextensions.CustomResource("pulumi-esc-store", {
  apiVersion: "external-secrets.io/v1",
  kind: "ClusterSecretStore",
  metadata: {
    name: "pulumi-esc-store",
  },
  spec: {
    provider: {
      pulumi: {
        organization: cfg.require("organization"),
        project: cfg.require("project"),
        environment: cfg.require("environment"),
        auth: {
          oidcConfig: {
            organization: cfg.require("organization"),
            serviceAccountRef: {
              name: "pulumi-esc-oidc",
              namespace: "external-secrets",
            },
            expirationSeconds: 600,
          },
        },
      },
    },
  },
});
```

## Debugging

### Check Operator Logs
```bash
kubectl logs -n external-secrets deployment/external-secrets --tail=100 -f
```

### Check SecretStore Status
```bash
kubectl describe secretstore pulumi-oidc-test -n default
```

### Check ExternalSecret Status
```bash
kubectl describe externalsecret pulumi-oidc-test -n default
```

### Common Issues

1. **"failed to create service account token"**
   - Ensure ServiceAccount exists in the correct namespace
   - Check RBAC permissions

2. **"Pulumi OIDC auth failed with status 401"**
   - Verify OIDC configuration in Pulumi Console
   - Check that the issuer URL matches your EKS cluster
   - Verify the audience includes `https://api.pulumi.com`

3. **"failed to refresh OIDC token"**
   - Check token expiration settings
   - Verify ServiceAccount still exists

4. **CRD validation errors**
   - Make sure you've installed the CRDs from your fork:
     ```bash
     kubectl apply -f config/crds/bases/
     ```

## Rollback

If you need to rollback to the official image:

```bash
kubectl set image deployment/external-secrets \
  -n external-secrets \
  external-secrets=ghcr.io/external-secrets/external-secrets:v0.11.0
```

## Performance Testing

Monitor cache effectiveness:

```bash
# Check operator metrics (if Prometheus is enabled)
kubectl port-forward -n external-secrets svc/external-secrets-metrics 8080:8080

# In another terminal
curl localhost:8080/metrics | grep pulumi
```
