# Dex Access Token Generators

External Secrets Operator provides two generators for integrating with [Dex](https://dexidp.io/) to obtain OAuth2 access tokens:

1. **DexTokenExchange**: Uses Kubernetes service account tokens
2. **DexUsernamePassword**: Uses username/password credentials

Both generators create secrets containing OAuth2 access tokens that can be used with other systems.

## Token Exchange Setup

Token exchange uses Kubernetes service account tokens and is the most secure approach as it doesn't require storing user credentials.

1. **Enable OIDC discovery access:**
   ```bash
   kubectl create clusterrolebinding oidc-reviewer \
     --clusterrole=system:service-account-issuer-discovery \
     --group=system:unauthenticated
   ```

2. **Configure Dex with the Kubernetes connector:**
   ```yaml
   connectors:
     - name: kubernetes
       type: oidc
       id: kubernetes
       config:
         issuer: "https://kubernetes.default.svc.cluster.local"
         rootCAs:
           - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
         userNameKey: sub
         scopes:
           - profile
   ```

3. **Create required resources:**
   ```yaml
   # OAuth2 client secret
   apiVersion: v1
   kind: Secret
   metadata:
     name: dex-client-credentials
     namespace: default
   type: Opaque
   stringData:
     client-secret: "your-oauth2-client-secret"
   ---
   # Service account for token exchange
   apiVersion: v1
   kind: ServiceAccount
   metadata:
     name: dex-token-exchange-sa
     namespace: default
   ```

4. **Create the DexTokenExchange generator:**
   ```yaml
   apiVersion: generators.external-secrets.io/v1alpha1
   kind: DexTokenExchange
   metadata:
     name: dex-token-exchange-auth
     namespace: default
   spec:
     dexUrl: "https://dex.example.com"
     clientId: your-client-id
     clientSecretRef:
       name: dex-client-credentials
       key: client-secret
     connectorId: "kubernetes"
     scopes:
       - "openid"
       - "profile"
       - "email"
     serviceAccountRef:
       name: dex-token-exchange-sa
   ```

5. **Create an ExternalSecret:**
   ```yaml
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
           access-token: "{{ .token }}"
           expires-at: "{{ .expiry }}"
     dataFrom:
       - sourceRef:
           generatorRef:
             apiVersion: generators.external-secrets.io/v1alpha1
             kind: DexTokenExchange
             name: dex-token-exchange-auth
   ```

## Password Authentication Setup

Password authentication uses traditional username/password credentials stored in Kubernetes secrets.

1. **Enable password authentication in Dex:**
   ```yaml
   enablePasswordDB: true
   staticPasswords:
     - email: "user@example.com"
       hash: "$2a$10$..."  # bcrypt hash of password
       username: "user"
       userID: "unique-user-id"
   ```

2. **Create required secrets:**
   ```bash
   # Create client credentials secret
   kubectl create secret generic dex-client-credentials \
     --from-literal=client-secret=your-client-secret

   # Create user credentials secret
   kubectl create secret generic dex-user-credentials \
     --from-literal=username=user@example.com \
     --from-literal=password=your-password
   ```

3. **Create the DexUsernamePassword generator:**
   ```yaml
   apiVersion: generators.external-secrets.io/v1alpha1
   kind: DexUsernamePassword
   metadata:
     name: dex-password-auth
     namespace: default
   spec:
     dexUrl: "https://dex.example.com"
     clientId: your-client-id
     clientSecretRef:
       name: dex-client-credentials
       key: client-secret
     scopes:
       - "openid"
       - "profile"
       - "email"
     usernameRef:
       name: dex-user-credentials
       key: username
     passwordRef:
       name: dex-user-credentials
       key: password
   ```

4. **Create an ExternalSecret:**
   ```yaml
   apiVersion: external-secrets.io/v1
   kind: ExternalSecret
   metadata:
     name: dex-password-example
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
             auth-method: "password"
         data:
           access-token: "{{ .token }}"
           expires-at: "{{ .expiry }}"
     dataFrom:
       - sourceRef:
           generatorRef:
             apiVersion: generators.external-secrets.io/v1alpha1
             kind: DexUsernamePassword
             name: dex-password-auth
   ```

## Spec Fields

### DexTokenExchange

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `dexUrl` | Yes | - | Dex server URL (e.g., `https://dex.example.com`) |
| `clientId` | Yes | - | OAuth2 client ID registered in Dex |
| `clientSecretRef` | Yes | - | Reference to secret containing OAuth2 client secret |
| `connectorId` | No | `kubernetes` | Dex connector ID for token exchange |
| `scopes` | No | `["openid"]` | OAuth2 scopes to request |
| `serviceAccountRef` | Yes | - | Service account for token exchange |

### DexUsernamePassword

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `dexUrl` | Yes | - | Dex server URL (e.g., `https://dex.example.com`) |
| `clientId` | Yes | - | OAuth2 client ID registered in Dex |
| `clientSecretRef` | Yes | - | Reference to secret containing OAuth2 client secret |
| `scopes` | No | `["openid"]` | OAuth2 scopes to request |
| `usernameRef` | Yes | - | Reference to secret containing username |
| `passwordRef` | Yes | - | Reference to secret containing password |

## Generated Secret Data

Both generators create secrets with the following data:

| Key | Description |
|-----|-------------|
| `token` | OAuth2 access token (JWT) |
| `expiry` | Token expiration timestamp in RFC3339 format |



