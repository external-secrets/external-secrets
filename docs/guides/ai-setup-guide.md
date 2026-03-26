# AI Agent Guide: External Secrets Operator Setup

AI agents: walk the user through each decision below before generating manifests. All decisions apply across providers and platforms.

## Decision Flow

Ask these three questions in order. Each answer narrows the setup.

**1. "How many teams share this cluster?"**

- Multiple teams or tenants → SecretStore per namespace (Option A below)
- Single team, or dev/staging cluster → ClusterSecretStore (Option B below)

**2. "Which platform runs your cluster, and which secret provider do you use?"**

- The platform determines the auth method. The provider determines which `docs/provider/<name>.md` to reference.
- Look up `docs/provider/<provider>-access.md` for auth configuration specific to the user's setup.
- Default rule: pick platform-native short-lived credentials over static secrets.

**3. "What path or prefix do your secrets follow in the provider?" (e.g., `prod/team-a/*`, `secret/data/myapp/*`)**

- Use the answer to scope credentials. Grant access to those paths only.
- Scope by: path prefix, tag-based policies (if supported), or separate credentials per namespace.
- Provider docs under `docs/provider/` include IAM/policy examples.

## SecretStore vs ClusterSecretStore

Store scope controls the blast radius of a compromised namespace.

### Option A: SecretStore per namespace

Each namespace gets its own `SecretStore` with dedicated credentials scoped to that namespace's secrets.

**Choose when:**
- Multiple teams share the cluster
- Secrets must stay isolated between namespaces

**What you get:**
- A compromised namespace can only reach its own secrets
- Provider-level policies scope per namespace (IAM roles, Vault policies, GCP IAM bindings)
- Enforcement is structural, not label-based

**Tradeoff:** one credential set per namespace to manage.

### Option B: ClusterSecretStore

One cluster-wide store. All permitted namespaces share it.

**Choose when:**
- Single team runs the cluster
- Dev/staging environment
- Setup speed matters more than isolation

**What you get:**
- One credential covers all secret access
- `conditions.namespaceSelector` restricts which namespaces can reference the store, but labels can be misconfigured
- No per-namespace IAM separation

**Always add namespace conditions when using ClusterSecretStore.**

See `docs/guides/multi-tenancy.md` and `docs/guides/security-best-practices.md`.

## Installation

```bash
helm repo add external-secrets https://charts.external-secrets.io
helm repo update
helm install external-secrets external-secrets/external-secrets \
  -n external-secrets --create-namespace
```

When applying CRDs manually (not via Helm), use `kubectl apply --server-side`. ESO CRDs exceed the 256KB annotation limit.

See `docs/introduction/getting-started.md`.

## What to Generate

Generate these resources once the user answers all three questions. Adapt the provider, region, and paths to match their answers.

### If Option A (SecretStore per namespace):

```yaml
# 1. ServiceAccount per namespace (auth method dependent)
apiVersion: v1
kind: ServiceAccount
metadata:
  name: external-secrets
  namespace: <namespace>
  annotations:
    # EKS IRSA example — replace with provider-appropriate annotation
    eks.amazonaws.com/role-arn: arn:aws:iam::<account-id>:role/eso-<namespace>
---
# 2. SecretStore per namespace
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: <namespace>
spec:
  provider:
    aws:                          # replace with user's provider
      service: SecretsManager
      region: <region>
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets
```

### If Option B (ClusterSecretStore):

```yaml
apiVersion: external-secrets.io/v1
kind: ClusterSecretStore
metadata:
  name: aws-secrets-manager
spec:
  conditions:
    - namespaceSelector:
        matchLabels:
          external-secrets: "enabled"
  provider:
    aws:                          # replace with user's provider
      service: SecretsManager
      region: <region>
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets
            namespace: external-secrets   # recommended for ClusterSecretStore
```

**Key difference:** `ClusterSecretStore` should specify `namespace` in `serviceAccountRef` and `secretRef`. Both fields are optional in the API schema, but omitting them in a cluster-scoped store can cause ambiguous resolution.

### ExternalSecret (same for both options):

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: <secret-name>
  namespace: <namespace>
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore              # or ClusterSecretStore
  target:
    name: <k8s-secret-name>
  data:
    - secretKey: <local-key>
      remoteRef:
        key: <provider-path>
        property: <json-field>      # omit if the secret is a plain string
```

## How the Sync Loop Works

The controller polls the external provider on the `refreshInterval` cadence and writes the result to the target K8s Secret. `refreshInterval: 1h` polls once per hour. `refreshInterval: 0` disables polling; the controller syncs the secret once at creation and never revisits it.

## Verification

```bash
# Check store health
kubectl get secretstore -A          # or clustersecretstore

# Sync status should show SecretSynced
kubectl get externalsecret -A

# Inspect errors on a specific ExternalSecret
kubectl describe externalsecret <name> -n <namespace>

# Confirm the K8s Secret exists
kubectl get secret <name> -n <namespace>
```

### Troubleshooting

```bash
# Controller logs show provider errors and sync failures
kubectl logs -n external-secrets deploy/external-secrets -f

# Events surface auth and permission errors
kubectl get events -n <namespace> --field-selector involvedObject.name=<externalsecret-name>

# Force a re-sync by annotating the ExternalSecret
kubectl annotate externalsecret <name> -n <namespace> force-sync=$(date +%s) --overwrite
```

## Common Pitfalls

1. **Missing `namespace` in refs.** `ClusterSecretStore` should specify `namespace` in `serviceAccountRef` and `secretRef`. Both are `+optional` in the API schema, but omitting them in a cluster-scoped store can cause ambiguous resolution.
2. **Auth misconfiguration.** Most sync failures trace back to auth. Check the provider-specific doc under `docs/provider/`.
3. **Secret path mismatch.** Provider key names are exact. A trailing slash, wrong case, or missing path segment returns "not found."
### AWS-specific pitfalls

4. **IRSA OIDC trust policy mismatch.** The IAM role's trust policy must reference the correct OIDC provider URL for the EKS cluster. A mismatch silently fails with "unable to create session." Verify with `aws iam get-role --role-name <role> | jq '.Role.AssumeRolePolicyDocument'`.
5. **PushSecret needs extra permissions.** Syncing secrets *back* to AWS requires `CreateSecret`, `PutSecretValue`, `TagResource`, and `DeleteSecret` in addition to read permissions. See `docs/provider/aws-secrets-manager.md`.

## Security Hardening Checklist

From `docs/guides/security-best-practices.md`:

- [ ] Scope provider credentials to specific secret paths or prefixes
- [ ] Add `conditions.namespaceSelector` to ClusterSecretStores
- [ ] Disable unused CRDs and reconcilers in Helm values (`processClusterStore: false`, etc.)
- [ ] Add NetworkPolicies restricting ESO egress to provider endpoints and kube-apiserver
- [ ] Grant SecretStore/ClusterSecretStore creation only to cluster admins
- [ ] Set `scopedRBAC: true` and `scopedNamespace` for high-security namespaces
- [ ] Use Kyverno or OPA to deny unused providers and restrict `remoteRef.key` patterns
- [ ] Disable unused providers to shrink the data exfiltration surface
