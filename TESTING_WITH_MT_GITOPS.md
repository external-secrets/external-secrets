# Testing Pulumi OIDC with mt-gitops

This guide shows how to test the OIDC changes in mt-gitops.

## Quick Start

### 1. Build and Push Custom Image

```bash
cd external-secrets

# Build image
export IMAGE_TAG=pulumi-oidc-$(git rev-parse --short HEAD)
make docker.build IMAGE_NAME=ghcr.io/johnstonmatt/external-secrets IMAGE_TAG=${IMAGE_TAG}

# Push to registry (make sure you're logged in to ghcr.io)
docker push ghcr.io/johnstonmatt/external-secrets:${IMAGE_TAG}
```

### 2. Update mt-gitops External Secrets Chart

In `mt-gitops/charts/k8s/ExternalSecretsChart.ts`, add custom image support:

```typescript
export interface ExternalSecretsChartProps extends SupaBaseChartProps {
  chartVersion: string;
  customImage?: {
    repository: string;
    tag: string;
  };
}

export class ExternalSecretsChart extends HelmChart {
  constructor(scope: Construct, id: string, props: ExternalSecretsChartProps) {
    const values: any = {
      // ... existing values
    };

    // Add custom image if specified
    if (props.customImage) {
      values.image = {
        repository: props.customImage.repository,
        tag: props.customImage.tag,
        pullPolicy: "Always",
      };
    }

    super(scope, id, {
      ...props,
      // ... rest of config
    });
  }
}
```

### 3. Update main.ts to Use Custom Image (Temporary)

In `mt-gitops/main.ts`:

```typescript
// Add this near the top for testing
const useCustomExternalSecrets = true;
const customImage = useCustomExternalSecrets ? {
  repository: "ghcr.io/johnstonmatt/external-secrets",
  tag: "pulumi-oidc-bd0015b",  // Your git SHA
} : undefined;

// Update the createHelmChart call
createHelmChart(ExternalSecretsChart, {
  env: EnvType.staging,
  regions: ["ap-southeast-1"], // Test in one region first
  name: "external-secrets",
  props: {
    chartVersion: "0.11.0",
    customImage,  // Add this
  },
});
```

### 4. Update PulumiEscSecretsChart for OIDC

Create a new version that supports OIDC:

```typescript
// In mt-gitops/charts/k8s/PulumiEscSecretsChart.ts
import { Construct } from "constructs";
import { SupaBaseChart, SupaBaseChartProps } from "@/lib/BaseChart.ts";
import { ClusterSecretStore } from "@/imports/external-secrets-external-secrets.io.ts";

export interface PulumiEscSecretsChartProps extends SupaBaseChartProps {
  organization: string;
  project: string;
  environment: string;

  // Auth configuration - either accessToken or oidc
  auth: {
    accessToken?: {
      secretName?: string;
      secretKey?: string;
      secretNamespace?: string;
    };
    oidc?: {
      serviceAccountName: string;
      serviceAccountNamespace?: string;
      expirationSeconds?: number;
    };
  };
}

export class PulumiEscSecretsChart extends SupaBaseChart {
  constructor(scope: Construct, id: string, props: PulumiEscSecretsChartProps) {
    super(scope, id, props);

    // Build auth config
    const authConfig: any = {};

    if (props.auth.accessToken) {
      const secretName = props.auth.accessToken.secretName ?? "pulumi-esc-credentials";
      const secretKey = props.auth.accessToken.secretKey ?? "accessToken";
      const secretNamespace = props.auth.accessToken.secretNamespace ?? "external-secrets";

      authConfig.accessToken = {
        secretRef: {
          name: secretName,
          key: secretKey,
          namespace: secretNamespace,
        },
      };
    } else if (props.auth.oidc) {
      authConfig.oidcConfig = {
        organization: props.organization,
        serviceAccountRef: {
          name: props.auth.oidc.serviceAccountName,
          namespace: props.auth.oidc.serviceAccountNamespace ?? "external-secrets",
        },
        expirationSeconds: props.auth.oidc.expirationSeconds ?? 600,
      };
    }

    new ClusterSecretStore(this, "pulumi-esc-store", {
      metadata: {
        name: "pulumi-esc-store",
        annotations: {
          "argocd.argoproj.io/sync-options": "SkipDryRunOnMissingResource=true",
        },
      },
      spec: {
        provider: {
          pulumi: {
            organization: props.organization,
            project: props.project,
            environment: props.environment,
            auth: authConfig,
          },
        },
      },
    });

    this.applyValidation();
  }
}
```

### 5. Update main.ts to Use OIDC

```typescript
// In main.ts, update the PulumiEscSecretsChart call:

createClusterManifestChart(PulumiEscSecretsChart, {
  env: EnvType.staging,
  regions: ["ap-southeast-1"],
  name: "pulumi-esc-store",
  props: {
    organization: "supabase",
    project: "mt-services-combined",
    environment: "staging-apse1",
    auth: {
      // Use OIDC instead of access token
      oidc: {
        serviceAccountName: "pulumi-esc-oidc",
        serviceAccountNamespace: "external-secrets",
        expirationSeconds: 600,
      },
      // Old way (comment out when testing OIDC):
      // accessToken: {
      //   secretName: "pulumi-esc-credentials",
      //   secretKey: "accessToken",
      //   secretNamespace: "external-secrets",
      // },
    },
  },
});
```

### 6. Create ServiceAccount Chart (New)

Create `mt-gitops/charts/k8s/PulumiEscServiceAccountChart.ts`:

```typescript
import { Construct } from "constructs";
import { SupaBaseChart, SupaBaseChartProps } from "@/lib/BaseChart.ts";
import * as k8s from "@/imports/k8s.ts";

export class PulumiEscServiceAccountChart extends SupaBaseChart {
  constructor(scope: Construct, id: string, props: SupaBaseChartProps) {
    super(scope, id, props);

    new k8s.ServiceAccount(this, "pulumi-esc-oidc", {
      metadata: {
        name: "pulumi-esc-oidc",
        namespace: "external-secrets",
      },
    });

    this.applyValidation();
  }
}
```

Add to main.ts:

```typescript
import { PulumiEscServiceAccountChart } from "./charts/k8s/PulumiEscServiceAccountChart.ts";

createClusterManifestChart(PulumiEscServiceAccountChart, {
  env: EnvType.staging,
  regions: ["ap-southeast-1"],
  name: "pulumi-esc-service-account",
  props: {},
});
```

### 7. Configure Pulumi ESC OIDC

Before deploying, set up OIDC in Pulumi:

```bash
# Get your EKS OIDC issuer
CLUSTER_NAME="staging-aws-apse1-mt"
REGION="ap-southeast-1"
OIDC_ISSUER=$(aws eks describe-cluster --name $CLUSTER_NAME --region $REGION \
  --query "cluster.identity.oidc.issuer" --output text)

echo "OIDC Issuer: $OIDC_ISSUER"
```

Then in Pulumi Console:
1. Go to Organization Settings → OIDC
2. Add new OIDC provider:
   - Name: `eks-staging-apse1`
   - Issuer: `$OIDC_ISSUER`
   - Audience: `https://api.pulumi.com`
3. Create Service Account Identity
   - Name: `mt-external-secrets`
   - OIDC Provider: Select the one you just created
   - Subject: `system:serviceaccount:external-secrets:pulumi-esc-oidc`
4. Note the Identity ID (looks like `oidc-12345678`)

### 8. Deploy and Test

```bash
cd mt-gitops

# Generate manifests
npm run synth

# Check generated files
ls -la dist/staging/ap-southeast-1/pulumi-esc-store/
ls -la dist/staging/ap-southeast-1/pulumi-esc-service-account/

# Deploy via ArgoCD (or directly for testing)
kubectl apply -f dist/staging/ap-southeast-1/pulumi-esc-service-account/
kubectl apply -f dist/staging/ap-southeast-1/pulumi-esc-store/

# Wait for sync
kubectl get clustersecretstore pulumi-esc-store -o yaml

# Check for errors
kubectl get clustersecretstore pulumi-esc-store -o jsonpath='{.status.conditions}'
```

### 9. Verify OIDC Token Exchange

```bash
# Check external-secrets operator logs
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets --tail=50 -f

# Look for messages like:
# - "creating service account token"
# - "exchanging token with Pulumi"
# - "Pulumi OIDC auth failed" (if there's an issue)
```

### 10. Test with a Real ExternalSecret

```bash
# Create a test ExternalSecret
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: pulumi-oidc-test
  namespace: default
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: pulumi-esc-store
    kind: ClusterSecretStore
  target:
    name: pulumi-test-secret
    creationPolicy: Owner
  data:
  - secretKey: test-value
    remoteRef:
      key: environmentVariables.SOME_KEY
EOF

# Check status
kubectl get externalsecret pulumi-oidc-test -n default -o yaml
kubectl describe externalsecret pulumi-oidc-test -n default

# If successful, the secret should be created
kubectl get secret pulumi-test-secret -n default
```

## Rollback

If something goes wrong:

```bash
# Revert to access token auth in main.ts
# Comment out OIDC config, uncomment accessToken config
# Re-run npm run synth
# Sync with ArgoCD or kubectl apply

# Or rollback external-secrets image
kubectl set image deployment/external-secrets \
  -n external-secrets \
  external-secrets=ghcr.io/external-secrets/external-secrets:v0.11.0
```

## Debugging Tips

### Check ServiceAccount Token Creation
```bash
# The operator should be able to create tokens
kubectl auth can-i create serviceaccounts/token \
  --as=system:serviceaccount:external-secrets:external-secrets \
  -n external-secrets
```

### Check OIDC Configuration
```bash
# Verify audience in Pulumi matches
echo $OIDC_ISSUER
# Should match what you configured in Pulumi Console
```

### Check Cache
```bash
# Check if caching is working (look for repeat token fetches)
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets \
  | grep -i "oidc token"
```

## Performance Comparison

Test performance difference between access token and OIDC:

```bash
# Create 10 ExternalSecrets and measure time
time for i in {1..10}; do
  kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: perf-test-$i
  namespace: default
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: pulumi-esc-store
    kind: ClusterSecretStore
  target:
    name: perf-test-secret-$i
  data:
  - secretKey: value
    remoteRef:
      key: environmentVariables.SOME_KEY
EOF
done

# Check how many token exchanges happened
kubectl logs -n external-secrets -l app.kubernetes.io/name=external-secrets \
  | grep "exchanging token with Pulumi" | wc -l
# Should be 1 (cached) or small number
```
