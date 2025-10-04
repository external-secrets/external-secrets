The OIDC generator provides OAuth2/OIDC access tokens using password grants and token exchange. It's designed to work with OIDC providers like Dex.

!!! note "Token Expiration"
    Generated tokens have limited lifespans. Set `refreshInterval` appropriately to refresh tokens before they expire.

## Output Keys and Values

| Key           | Description                                    |
| ------------- | ---------------------------------------------- |
| access_token  | The OAuth2 access token                       |
| token_type    | The token type (usually "Bearer")             |
| expires_in    | Token expiration time in seconds              |
| refresh_token | Refresh token (if provided by the server)     |
| id_token      | OpenID Connect ID token (if provided)         |
| expiry        | JWT expiration time in RFC3339 format         |

## Parameters

### Common Parameters

| Key                  | Required | Description                                                                          |
| -------------------- | -------- | ------------------------------------------------------------------------------------ |
| tokenUrl             | Yes      | OAuth2 token endpoint URL                                                            |
| clientId             | Yes      | OAuth2 client ID                                                                     |
| clientSecretRef      | No       | Reference to secret containing OAuth2 client secret                                  |
| scopes               | No       | List of OAuth2 scopes to request (defaults to ["openid"])                           |
| additionalParameters | No       | Provider-specific form parameters (e.g., Dex's connector_id)                        |
| additionalHeaders    | No       | Additional HTTP headers for requests                                                 |

### Grant Types

**Password Grant:**
- `usernameRef` (required): Reference to secret containing username
- `passwordRef` (required): Reference to secret containing password

**Token Exchange Grant:**
- `subjectTokenRef` or `serviceAccountRef` (required): Token source (mutually exclusive)
- `subjectTokenType` (required): Type of the subject token
- `requestedTokenType` (optional): Type of token to request
- `actorTokenRef` (optional): Reference to actor token for delegation
- `actorTokenType` (optional): Type of the actor token
- `audience` (optional): Target service logical name
- `resource` (optional): Target service URI
- `additionalParameters` (optional): Grant-specific provider parameters
- `additionalHeaders` (optional): Grant-specific HTTP headers

## Dex Setup

### Prerequisites

1. **Enable OIDC discovery:**
   ```bash
   kubectl create clusterrolebinding oidc-reviewer \
     --clusterrole=system:service-account-issuer-discovery \
     --group=system:unauthenticated
   ```

2. **Configure Dex connector:**
   ```yaml
   connectors:
     - name: kubernetes
       type: oidc
       id: kubernetes
       config:
         issuer: "https://kubernetes.default.svc.cluster.local"
         rootCAs:
           - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
   ```

3. **For password authentication, enable password database:**
   ```yaml
   enablePasswordDB: true
   staticPasswords:
     - email: "user@example.com"
       hash: "$2a$10$..."
       username: "user"
   ```

4. **Configure Dex Client:**
   ```yaml
   staticClients:
     - id: example-client
       name: "Example Client"
       secret: secret
       public: true   
   ```

## Examples

### Password Grant

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dex-credentials
  namespace: default
type: Opaque
stringData:
  client-secret: "your-client-secret"
  username: "user@example.com"
  password: "your-password"
---
apiVersion: generators.external-secrets.io/v1alpha1
kind: OIDC
metadata:
  name: dex-password-auth
  namespace: default
spec:
  tokenUrl: "https://dex.example.com/token"
  clientId: "example-client"
  clientSecretRef:
    name: dex-credentials
    key: client-secret
  scopes: ["openid", "profile"]
  grant:
    password:
      usernameRef:
        name: dex-credentials
        key: username
      passwordRef:
        name: dex-credentials
        key: password
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: dex-password-example
  namespace: default
spec:
  refreshInterval: 30m
  target:
    name: dex-password-token
    creationPolicy: Owner
    template:
      type: Opaque
      metadata:
        labels:
          auth-provider: "dex"
          auth-method: "password"
      data:
        access-token: "{{ .access_token }}"
        token-type: "{{ .token_type }}"
        expires-in: "{{ .expires_in }}"
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: OIDC
          name: dex-password-auth
```

### Token Exchange with Service Account

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: dex-workload-sa
  namespace: default
---
apiVersion: v1
kind: Secret
metadata:
  name: dex-client-secret
  namespace: default
type: Opaque
stringData:
  client-secret: "your-client-secret"
---
apiVersion: generators.external-secrets.io/v1alpha1
kind: OIDC
metadata:
  name: dex-sa-token-exchange
  namespace: default
spec:
  tokenUrl: "https://dex.example.com/token"
  clientId: "example-client"
  clientSecretRef:
    name: dex-client-secret
    key: client-secret
  scopes: ["openid", "profile", "email"]
  grant:
    tokenExchange:
      serviceAccountRef:
        name: dex-workload-sa
        audiences: ["https://dex.example.com"]
      subjectTokenType: "urn:ietf:params:oauth:token-type:id_token"
      requestedTokenType: "urn:ietf:params:oauth:token-type:access_token"
      additionalParameters:
        connector_id: "kubernetes"
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: dex-token-exchange-example
  namespace: default
spec:
  refreshInterval: 30m
  target:
    name: dex-access-token
    creationPolicy: Owner
    template:
      type: Opaque
      metadata:
        labels:
          auth-provider: "dex"
          auth-method: "token-exchange"
      data:
        access-token: "{{ .access_token }}"
        expires-at: "{{ .expiry }}"
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: OIDC
          name: dex-sa-token-exchange
```



