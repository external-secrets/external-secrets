## Akeyless Secrets Management Platform

External Secrets Operator integrates with the [Akeyless Secrets Management Platform](https://www.akeyless.io/).

### Create Secret Store

SecretStore resource specifies how to access Akeyless. This resource is namespaced.

**NOTE:** Make sure the Akeyless provider is listed in the Kind=SecretStore.
If you use a customer fragment, define the value of akeylessGWApiURL as the URL of your Akeyless Gateway in the following format: https://your.akeyless.gw:8080/v2.

When using an Akeyless Gateway, you can set `ignoreCache: true` on the SecretStore to bypass the Gateway cache and fetch the latest secret value from Akeyless SaaS on every sync. This matches the `ignore_cache` option in the Akeyless Kubernetes Secrets Injector. Use this only when you need fresh values on each reconcile: it increases load on the Gateway and upstream Akeyless API compared to serving cached responses.

**NOTE:** `ignoreCache` affects Gateway-side caching only. ESO still reuses the provider client between reconciles when the SecretStore spec is unchanged.

Akeyless provides several Authentication Methods:

### Authentication with Kubernetes

Options for obtaining Kubernetes credentials include:

1. Using a service account jwt referenced in serviceAccountRef
2. Using the jwt from a Kind=Secret referenced by the secretRef
3. Using transient credentials from the mounted service account token within the external-secrets operator

#### Create the Akeyless Secret Store Provider with Kubernetes Auth-Method

```yaml
{% include 'akeyless-secret-store-k8s-auth.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `serviceAccountRef` and `secretRef` according to the namespaces where the secrets reside.

### Authentication with OAuth2.0/JWT

Use an [Akeyless OAuth2.0/JWT](https://docs.akeyless.io/docs/auth-with-oauth-jwt) auth method. The provider calls Akeyless `Auth` with `access-type=jwt` and the bearer JWT.

Preferred SecretStore shape is `authSecretRef.jwtAuth`:

```yaml
{% include 'akeyless-secret-store-jwt-auth.yaml' %}
```

JWT material:

```yaml
{% include 'akeyless-credentials-jwt-secret.yaml' %}
```

`jwtAuth.serviceAccountRef` uses the TokenRequest API (`create` on `serviceaccounts/token`). The Helm chart grants this when `rbac.serviceAccountTokenCreate` is `true` (the default). `serviceAccountRef.audiences` is required and must match the audience configured on the Akeyless OAuth2.0/JWT auth method. This keeps the token from being scoped to the Kubernetes API server, so it cannot be replayed against the cluster API.

When `jwtAuth` is set, it takes precedence: `authSecretRef.secretRef` and `authSecretRef.serviceAccountRef` are ignored, and ESO returns a warning if they are also set.

#### Akeyless setup for ServiceAccount tokens

For `serviceAccountRef`, the Akeyless auth method must trust your cluster's ServiceAccount token issuer:

1. Export the cluster JWKS and note the issuer. Refresh the JWKS in Akeyless when the cluster signing keys rotate.

    ```bash
    kubectl get --raw /openid/v1/jwks > cluster-jwks.json
    kubectl get --raw /.well-known/openid-configuration  # "issuer" field
    ```

2. Create an OAuth2.0/JWT auth method that trusts that JWKS, with `sub` as the unique identifier and the same audience you set in `serviceAccountRef.audiences`:

    ```bash
    akeyless auth-method create oauth2 \
      --name /k8s/cluster-oidc \
      --jwks-json-file ./cluster-jwks.json \
      --issuer <issuer-from-openid-configuration> \
      --unique-identifier sub \
      --audience akeyless.io
    ```

    If the cluster JWKS is publicly reachable, you can use `--jwks-uri` instead of `--jwks-json-file`.

3. Scope access per ServiceAccount with role associations on the `sub` claim:

    ```bash
    akeyless assoc-role-am --role-name <role> --am-name /k8s/cluster-oidc \
      --sub-claims sub=system:serviceaccount:<namespace>:<serviceaccount>
    ```

Alternatively, store `accessType=jwt` and put the JWT in `accessTypeParam` via the generic credentials Secret below.

### Authentication with Cloud-Identity or Api-Access-Key

Akeyless providers require an access-id, access-type and access-type-param
to set your SecretStore with an authentication method from Akeyless.

The supported auth-methods and their parameters are:

| accessType     | accessTypeParam                    |
| -------------- | ---------------------------------- |
| `aws_iam`      | - (optionally `serviceAccountRef`) |
| `gcp`          | The GCP audience                   |
| `azure_ad`     | Azure object ID (optional)         |
| `api_key`      | The access key                     |
| `access_key`   | The access key (alias for api_key) |
| `k8s`          | The k8s configuration name         |
| `jwt`          | The bearer JWT                     |

For `azure_ad` on AKS Workload Identity, set `authSecretRef.serviceAccountRef` to a ServiceAccount annotated with `azure.workload.identity/client-id` and `azure.workload.identity/tenant-id`. This field is only used when `accessType` is `azure_ad` or `aws_iam`; other access types ignore it.

```yaml
{% include 'akeyless-secret-store-azure-ad-wi.yaml' %}
```

For `ClusterSecretStore`, set `serviceAccountRef.namespace` when the ServiceAccount is not in the same namespace as the consuming `ExternalSecret`; otherwise the ServiceAccount is resolved from that namespace. Sovereign Azure clouds (US Government, China) are supported when `AZURE_ENVIRONMENT` or `AZURE_CLOUD` is set accordingly, matching the non-WI `GetCloudId` path.

#### aws_iam on EKS with IAM Roles for Service Accounts (IRSA)

By default, `aws_iam` authenticates with the AWS identity of the ESO controller pod, so every SecretStore appears in Akeyless as the same IAM role. To give each SecretStore its own identity, set `authSecretRef.serviceAccountRef` to a ServiceAccount annotated with `eks.amazonaws.com/role-arn`. ESO then:

1. Requests a token for that ServiceAccount via the TokenRequest API, with audience `sts.amazonaws.com` (or the value of the `eks.amazonaws.com/audience` annotation).
2. Calls `AssumeRoleWithWebIdentity` for the annotated role.
3. Uses the resulting temporary credentials to authenticate to the Akeyless AWS IAM auth method.

```yaml
{% include 'akeyless-secret-store-aws-iam-irsa.yaml' %}
```

Requirements:

- The IAM role's trust policy must allow `sts:AssumeRoleWithWebIdentity` from the cluster's OIDC provider for the subject `system:serviceaccount:<namespace>:<serviceaccount>`.
- The Akeyless AWS IAM auth method referenced by `accessID` must be bound to that role (for example with `bound-role-name` or `bound-arn`).
- The ESO controller needs `get` on `serviceaccounts` (to read the role annotation) and `create` on `serviceaccounts/token`. The Helm chart grants both by default (`create` is controlled by `rbac.serviceAccountTokenCreate: true`); a custom role must grant both.
- STS is called in the controller's `AWS_REGION`. If it is not set, ESO uses the default region of the role's partition: `us-east-1` for `aws`, `us-gov-west-1` for `aws-us-gov` and `cn-north-1` for `aws-cn`. For any other partition, set `AWS_REGION` on the ESO controller.

The STS session name is `eso-akeyless-<namespace>-<serviceaccount>`. Names longer than the STS limit of 64 characters are shortened and end in a hash of the namespace and ServiceAccount. Shorter names are not hashed, so different namespace and ServiceAccount pairs can produce the same session name (for example `a-b`/`c` and `a`/`b-c`). Use the role ARN together with the session name to attribute events when ServiceAccounts share a role.

!!! note "Upgrading"
    Before this release, `serviceAccountRef` was ignored for `aws_iam`. A store that already sets `serviceAccountRef` together with `accessType: aws_iam` will now authenticate as that ServiceAccount's IAM role instead of the controller's. Remove `serviceAccountRef` from such stores before upgrading if you want to keep the controller identity.

A `SecretStore` can only reference a ServiceAccount in its own namespace. For a `ClusterSecretStore`, set `serviceAccountRef.namespace`; otherwise the ServiceAccount is resolved from the namespace of the consuming `ExternalSecret`. If `serviceAccountRef` is not set, the controller pod identity is used as before. EKS Pod Identity associations are not used on this path because they are not exposed through the ServiceAccount.

For more information see [Akeyless Authentication Methods](https://docs.akeyless.io/docs/access-and-authentication-methods)

#### Creating an Akeyless Credentials Secret

Create a secret containing your credentials using the following example as a guide:

```yaml
{% include 'akeyless-credentials-secret.yaml' %}
```

#### Create the Akeyless Secret Store Provider with the Credentials Secret

```yaml
{% include 'akeyless-secret-store.yaml' %}
```

**NOTE:** In case of a `ClusterSecretStore`, be sure to provide `namespace` for `accessID`, `accessType` and `accessTypeParam` according to the namespaces where the secrets reside.

#### Create the Akeyless Secret Store With CAs for TLS handshake

```yaml
....
spec:
  provider:
    akeyless:
      akeylessGWApiURL: "https://your.akeyless.gw:8080/v2"
      ignoreCache: true

      # Optional caBundle - PEM/base64 encoded CA certificate
      caBundle: "<base64 encoded cabundle>"
      # Optional caProvider:
      # Instead of caBundle you can also specify a caProvider
      # this will retrieve the cert from a Secret or ConfigMap
      caProvider:
        type: Secret  # Can be Secret or ConfigMap
        name: "<name of secret or configmap>"
        key: "<key inside secret>"
        # namespace is mandatory for ClusterSecretStore and not relevant for SecretStore
        namespace: "my-cert-secret-namespace"
  ....
```

### Supported Secret Types

The provider supports the following Akeyless item types:

- **Static Secret** -- standard key/value secret
- **Dynamic Secret** -- ephemeral credentials generated on demand
- **Rotated Secret** -- automatically rotated credentials
- **Certificate** -- TLS/SSH certificates

### Creating an external secret

To get a secret from Akeyless and create it as a secret on the Kubernetes cluster, a `Kind=ExternalSecret` is needed.

```yaml
{% include 'akeyless-external-secret.yaml' %}
```

#### Fetching a specific version

Use `remoteRef.version` to pin a specific secret version (integer). Omit the field or set it to `0` to get the latest version.

```yaml
data:
  - secretKey: password
    remoteRef:
      key: /path/to/secret
      version: "3"  # fetch version 3 specifically
```

#### Extracting a property from a JSON secret

If the secret value is a JSON object, use `remoteRef.property` to extract a single key. Nested keys can be addressed with dot notation; literal dots in key names are escaped with a backslash (`key\.with\.dots`).

```yaml
data:
  - secretKey: db-password
    remoteRef:
      key: /path/to/json-secret
      property: password  # extracts {"password": "..."} from the JSON value
```

#### Using DataFrom

DataFrom can be used to get a secret as a JSON string and attempt to parse it, creating one Kubernetes secret key per JSON field.

```yaml
{% include 'akeyless-external-secret-json.yaml' %}
```

When the secret contains nested JSON objects, use `extract.property` to select a subtree before parsing. This is useful when only part of the secret should be synced (for example, extracting a `db` object while ignoring other top-level keys like `apiKey`).

```yaml
{% include 'akeyless-external-secret-json-property.yaml' %}
```

Given an Akeyless static secret with value:

```json
{
  "db": {
    "username": "my_user",
    "password": "my_pass"
  },
  "apiKey": "myApiKey"
}
```

The example above creates a Kubernetes secret with keys `username` and `password`. Non-string JSON values (numbers, booleans, nested objects) are preserved as their JSON representation.

**Note:** `extract.property` requires the secret value to be valid JSON. It works reliably with **Static Secrets**. Dynamic and Rotated secrets may return provider-specific JSON envelopes; use `remoteRef.property` with `spec.data` for individual fields in those cases.

#### Finding secrets by name or tag

Use `dataFrom.find` to bulk-fetch secrets matching a name pattern or tag:

```yaml
# by name regex
dataFrom:
  - find:
      path: /my/path/         # optional path prefix
      name:
        regexp: ".*db.*"

# by tag
dataFrom:
  - find:
      tags:
        env: production
```

### Getting the Kubernetes Secret

The operator will fetch the secret and inject it as a `Kind=Secret`.

```bash
kubectl get secret database-credentials -o jsonpath='{.data.db-password}' | base64 -d
```

```bash
kubectl get secret database-credentials-json -o jsonpath='{.data}'
```

### Pushing a secret

To push a secret from Kubernetes cluster and create it as a secret to Akeyless, a `Kind=PushSecret` resource is needed.

```yaml
{% include 'akeyless-push-secret.yaml' %}
```

Then when you create a matching secret as follows:

```bash
kubectl create secret generic --from-literal=cache-pass=mypassword k8s-created-secret
```

Then it will create a secret in akeyless `eso-created/my-secret` with value `{"cache-pass":"mypassword"}`
