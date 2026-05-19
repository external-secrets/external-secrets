# PrivX

External Secrets Operator integrates with SSH PrivX for secret management.
See [PrivX](https://www.ssh.com/products/privileged-access-management-privx)

This provider uses the PrivX Vault API to read and write secrets. See [PrivX API](https://privx.docs.ssh.com/api)

Secrets are stored as objects within PrivX Vault.

## Authentication

The PrivX provider supports two authentication methods. Use only one per SecretStore.

### Method 1: API Client with OAuth2

This method uses an API client registered in PrivX with OAuth2 credentials. It is the recommended approach for automated integrations.

See [PrivX API-Client Integration](https://privx.docs.ssh.com/v43/docs/advanced-configuration/api-client-integration/) for setup details.

**How it works:**

1. Create a PrivX role with the required permissions in Administration → Roles.
2. Create an API client in Administration → Deployment → Integrate with PrivX using API clients, and assign the role.
3. Expand the API client's Credentials to obtain: `oauth_client_id`, `oauth_client_secret`, `api_client_id`, and `api_client_secret`.
4. The provider authenticates by posting these credentials to the `/auth/api/v1/oauth/token` endpoint using the OAuth2 `grant_type=password` flow, receiving a bearer access token.

**Create the Kubernetes secret:**

```bash
kubectl create secret generic privx-secret \
  --from-literal=privx_api_oauth_client_id='<OAUTH_CLIENT_ID>' \
  --from-literal=privx_api_oauth_client_secret='<OAUTH_CLIENT_SECRET>' \
  --from-literal=privx_api_client_id='<API_CLIENT_ID>' \
  --from-literal=privx_api_client_secret='<API_CLIENT_SECRET>'
```

**SecretStore configuration:**

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secretstore-privx
spec:
  provider:
    privx:
      host: <privx host url>
      defaultReadRoles:
        [
          "d0f6418b-728b-4ffc-8b9e-4f01ba1e659b",
          "bcd9dd57-abce-5e6c-6e12-1c988e1d7bc9",
        ]
      defaultWriteRoles:
        [
          "d0f6418b-728b-4ffc-8b9e-4f01ba1e659b",
          "bcd9dd57-abce-5e6c-6e12-1c988e1d7bc9",
        ]
      auth:
        oauth:
          clientIDRef:
            name: privx-secret
            key: privx_api_oauth_client_id
          clientSecretRef:
            name: privx-secret
            key: privx_api_oauth_client_secret
          apiClientIDRef:
            name: privx-secret
            key: privx_api_client_id
          apiClientSecretRef:
            name: privx-secret
            key: privx_api_client_secret
```

| Field | Description |
|---|---|
| `clientIDRef` | Reference to the OAuth client ID credential |
| `clientSecretRef` | Reference to the OAuth client secret credential |
| `apiClientIDRef` | Reference to the API client ID (used as `username` in the token request) |
| `apiClientSecretRef` | Reference to the API client secret (used as `password` in the token request) |

> **Note:** The OAuth user must have a role in PrivX that is listed in the readers (or writers) of the target secret.

---

### Method 2: External JWT Authentication

This method authenticates using an externally created JSON Web Token (JWT) signed with a private key. The provider generates a JWT locally and exchanges it for a PrivX access token via the `/auth/api/v1/token/login` endpoint.

See [PrivX External JWT Authentication](https://privx.docs.ssh.com/v43/docs/users-and-permissions/additional-authentication-methods/external-jwt-authentication) for setup details.

**Prerequisites in PrivX:**

1. Configure an external JWT identity provider in Administration → Deployment → External Token Authentication.
2. Set the **Public key method** to "Use Token Provider Public Key" and provide the public key corresponding to the private key used for signing.
3. Set the **Issuer** to match the `iss` claim in the JWT.
4. Configure a user directory from which PrivX resolves users by the JWT `sub` claim.

**How it works:**

1. The provider signs a JWT using the private key stored in the referenced Kubernetes secret.
2. The JWT contains `iss` (issuer) and `sub` (subject/username) claims.
3. The signed JWT is sent to PrivX's `/auth/api/v1/token/login` endpoint.
4. PrivX validates the JWT signature against the configured public key, resolves the user from the `sub` claim, and returns a bearer access token.

**Create the Kubernetes secret containing the private key:**

```bash
kubectl create secret generic privx-jwt-secret \
  --from-file=jwt_private_key=./path/to/private-key.pem
```

**SecretStore configuration:**

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secretstore-privx-jwt
spec:
  provider:
    privx:
      host: <privx host url>
      defaultReadRoles:
        [
          "d0f6418b-728b-4ffc-8b9e-4f01ba1e659b",
        ]
      defaultWriteRoles:
        [
          "d0f6418b-728b-4ffc-8b9e-4f01ba1e659b",
        ]
      auth:
        jwtAuth:
          publicKeyRef:
            name: privx-jwt-secret
            key: jwt_private_key
          iss: "issuer12345"
          sub: "alice"
```

| Field | Description |
|---|---|
| `publicKeyRef` | Reference to the Kubernetes secret containing the PEM-encoded private key used for signing the JWT |
| `iss` | Issuer claim — must match the issuer configured in PrivX's External Token Authentication settings |
| `sub` | Subject claim — must match a user name resolvable in the PrivX user directory |

> **Note:** Despite the field name `publicKeyRef`, this should reference the **private key** used for signing. The corresponding public key must be configured in PrivX for signature verification.

---

## Usage

### Create the external secret definition

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: privx-test
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: secretstore-privx
    kind: SecretStore
  target:
    name: privx-test-secret
    creationPolicy: Owner
  data:
    - secretKey: test_value
      remoteRef:
        key: <name of secret in PrivX>
```

The secret from PrivX is now available in Kubernetes secret `privx-test-secret`, with key `test_value`.

### Verify

```bash
kubectl get externalsecret
kubectl get secret privx-test-secret -o jsonpath='{.data.test_value}' | base64 -d
```

### Fetching Multiple Secrets

The PrivX provider supports `dataFrom.find`.

Find by name (RegExp):

```yaml
dataFrom:
- find:
    name:
      regexp: "app-.*"
```

Returns all secrets whose name matches the regular expression.

---

## PushSecret

PrivX supports PushSecret to write Kubernetes Secret values into PrivX.

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-to-privx
spec:
  secretStoreRefs:
  - name: privx-backend
  selector:
    secret:
      name: source-secret
  data:
  - match:
      secretKey: password
      remoteRef:
        remoteKey: my-app-secret
        property: password
```

### Deletion Policy

When a PushSecret resource is deleted from Kubernetes, the corresponding secret in PrivX Vault is **not** deleted by default. This means removing the PushSecret or the source Kubernetes Secret will leave the secret in PrivX, potentially resulting in orphaned secrets.

This behavior is controlled by the `deletionPolicy` parameter:

| Value | Behavior |
|---|---|
| `None` (default) | Secret is **not** deleted from PrivX when the PushSecret resource is removed from Kubernetes. |
| `Delete` | Secret **is** deleted from PrivX when the PushSecret resource is removed from Kubernetes. |

Example with `deletionPolicy: Delete`:

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: push-to-privx
spec:
  deletionPolicy: Delete
  secretStoreRefs:
    - name: secretstore-privx
      kind: SecretStore
  selector:
    secret:
      name: my-local-secret
  data:
    - match:
        secretKey: password
        remoteRef:
          remoteKey: my-app-secret
          property: password
```

> **Note:** For deletion to succeed, the PrivX API user's role must have **writer** access on the target secret in PrivX Vault.
