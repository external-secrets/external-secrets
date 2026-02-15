![Infisical k8s Diagram](../pictures/external-secrets-operator.png)

Sync secrets from [Infisical](https://www.infisical.com) to your Kubernetes cluster using External Secrets Operator.

## Authentication

In order for the operator to fetch secrets from Infisical, it needs to first authenticate with Infisical using a [Machine Identity](https://infisical.com/docs/documentation/platform/identities/machine-identities).

The Infisical provider supports multiple authentication methods to accommodate different deployment environments:

| Authentication Method | Use Case |
|----------------------|----------|
| [Universal Auth](#universal-auth) | Platform-agnostic authentication using Client ID and Client Secret |
| [Kubernetes Auth](#kubernetes-auth) | Native authentication for Kubernetes workloads using service account tokens |
| [AWS Auth](#aws-iam-auth) | Native authentication for AWS resources (EC2, Lambda, EKS) using IAM |
| [Azure Auth](#azure-auth) | Native authentication for Azure resources using managed identities |
| [GCP ID Token Auth](#gcp-id-token-auth) | Native authentication for GCP resources (GCE, Cloud Functions, Cloud Run) |
| [GCP IAM Auth](#gcp-iam-auth) | Authentication using GCP service account credentials |
| [JWT Auth](#jwt-auth) | Authentication using JSON Web Tokens from any JWT issuer |
| [LDAP Auth](#ldap-auth) | Authentication using LDAP credentials |
| [OCI Auth](#oci-auth) | Native authentication for Oracle Cloud Infrastructure resources |
| [Token Auth](#token-auth) | Simple authentication using a pre-generated access token |

---

## Universal Auth

[Universal Auth](https://infisical.com/docs/documentation/platform/identities/universal-auth) is a platform-agnostic authentication method that uses a Client ID and Client Secret to authenticate with Infisical.

### Prerequisites

1. Create a Machine Identity in Infisical with Universal Auth enabled
2. Obtain the Client ID and Client Secret
3. Add the identity to your Infisical project with appropriate permissions

### Storing Credentials

Create a Kubernetes secret containing your Universal Auth credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: universal-auth-credentials
type: Opaque
stringData:
  clientId: <your-client-id>
  clientSecret: <your-client-secret>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        universalAuthCredentials:
          clientId:
            name: universal-auth-credentials
            key: clientId
          clientSecret:
            name: universal-auth-credentials
            key: clientSecret
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! note
    For `ClusterSecretStore`, be sure to set `namespace` in `universalAuthCredentials.clientId` and `universalAuthCredentials.clientSecret`.

---

## Kubernetes Auth

[Kubernetes Auth](https://infisical.com/docs/documentation/platform/identities/kubernetes-auth) is a Kubernetes-native authentication method that validates service account tokens to authenticate with Infisical.

### Prerequisites

1. Create a Machine Identity in Infisical with Kubernetes Auth enabled
2. Configure allowed service account names and namespaces in Infisical
3. Set up a Token Reviewer service account (or use the client's own token for self-validation)

!!! note
    Infisical requires access to the Kubernetes TokenReview API for authentication. Follow the [Infisical Kubernetes Auth guide](https://infisical.com/docs/documentation/platform/identities/kubernetes-auth) to set up the required `system:auth-delegator` ClusterRoleBinding.

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubernetes-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
```

Optionally, if using a custom service account token path:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: kubernetes-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  serviceAccountTokenPath: /var/run/secrets/kubernetes.io/serviceaccount/token
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        kubernetesAuthCredentials:
          identityId:
            name: kubernetes-auth-credentials
            key: identityId
          # Optional: specify a custom service account token path
          # serviceAccountTokenPath:
          #   name: kubernetes-auth-credentials
          #   key: serviceAccountTokenPath
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

---

## AWS IAM Auth

[AWS Auth](https://infisical.com/docs/documentation/platform/identities/aws-auth) is an AWS-native authentication method for IAM principals like EC2 instances, Lambda functions, or EKS pods to authenticate with Infisical.

### Prerequisites

1. Create a Machine Identity in Infisical with AWS Auth enabled
2. Configure allowed Principal ARNs or Account IDs in Infisical
3. Ensure your AWS resource has an IAM role attached

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        awsAuthCredentials:
          identityId:
            name: aws-auth-credentials
            key: identityId
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! note
    The operator must run in an AWS environment with IAM credentials available (EC2 instance profile, EKS IRSA, Lambda execution role, etc.). The operator will automatically construct and sign the `GetCallerIdentity` request.

---

## Azure Auth

[Azure Auth](https://infisical.com/docs/documentation/platform/identities/azure-auth) is an Azure-native authentication method for Azure resources with managed identities to authenticate with Infisical.

### Prerequisites

1. Create a Machine Identity in Infisical with Azure Auth enabled
2. Configure allowed Service Principal IDs in Infisical
3. Ensure your Azure resource has a managed identity attached

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
```

Optionally, specify a custom resource/audience:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: azure-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  resource: https://management.azure.com/
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        azureAuthCredentials:
          identityId:
            name: azure-auth-credentials
            key: identityId
          # Optional: specify a custom resource/audience (default: https://management.azure.com/)
          # resource:
          #   name: azure-auth-credentials
          #   key: resource
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! note
    The operator must run in an Azure environment with a managed identity attached. The operator will automatically obtain the access token from the Azure Instance Metadata Service (IMDS).

---

## GCP ID Token Auth

[GCP ID Token Auth](https://infisical.com/docs/documentation/platform/identities/gcp-auth) is for GCP services like Compute Engine, App Engine, Cloud Functions, Cloud Run, and GKE to authenticate with Infisical using instance identity tokens.

### Prerequisites

1. Create a Machine Identity in Infisical with GCP Auth (ID Token type) enabled
2. Configure allowed service account emails in Infisical
3. Ensure your GCP resource has a service account attached

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcp-id-token-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        gcpIdTokenAuthCredentials:
          identityId:
            name: gcp-id-token-auth-credentials
            key: identityId
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! note
    The operator must run in a GCP environment with access to the metadata server. The operator will automatically obtain the ID token from the GCP metadata service.

---

## GCP IAM Auth

[GCP IAM Auth](https://infisical.com/docs/documentation/platform/identities/gcp-auth) allows GCP service accounts to authenticate with Infisical by signing a JWT token using the service account's credentials.

### Prerequisites

1. Create a Machine Identity in Infisical with GCP Auth (IAM type) enabled
2. Configure allowed service account emails in Infisical
3. Have a GCP service account key file available
4. The service account must have the `iam.serviceAccounts.signJwt` permission

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID and the path to the service account key:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcp-iam-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  serviceAccountKeyFilePath: /path/to/service-account-key.json
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        gcpIamAuthCredentials:
          identityId:
            name: gcp-iam-auth-credentials
            key: identityId
          serviceAccountKeyFilePath:
            name: gcp-iam-auth-credentials
            key: serviceAccountKeyFilePath
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

---

## JWT Auth

[JWT Auth](https://infisical.com/docs/documentation/platform/identities/jwt-auth) is a platform-agnostic authentication method that validates JSON Web Tokens from any JWT issuer.

### Prerequisites

1. Create a Machine Identity in Infisical with JWT Auth enabled
2. Configure the JWT validation settings (JWKS URL or public keys, issuer, audience, etc.)
3. Have a valid JWT available from your identity provider

### Storing Credentials

Create a Kubernetes secret containing your Machine Identity ID and the JWT:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: jwt-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  jwt: <your-jwt-token>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        jwtAuthCredentials:
          identityId:
            name: jwt-auth-credentials
            key: identityId
          jwt:
            name: jwt-auth-credentials
            key: jwt
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

---

## LDAP Auth

[LDAP Auth](https://infisical.com/docs/documentation/platform/identities/ldap-auth/general) allows authentication using LDAP credentials (username and password).

### Prerequisites

1. Create a Machine Identity in Infisical with LDAP Auth enabled
2. Configure LDAP settings in Infisical (server, bind DN, etc.)

### Storing Credentials

Create a Kubernetes secret containing your LDAP credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ldap-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  ldapUsername: <your-ldap-username>
  ldapPassword: <your-ldap-password>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        ldapAuthCredentials:
          identityId:
            name: ldap-auth-credentials
            key: identityId
          ldapUsername:
            name: ldap-auth-credentials
            key: ldapUsername
          ldapPassword:
            name: ldap-auth-credentials
            key: ldapPassword
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

---

## OCI Auth

[OCI Auth](https://infisical.com/docs/documentation/platform/identities/oci-auth) is an Oracle Cloud Infrastructure-native authentication method that verifies OCI users through signature validation.

### Prerequisites

1. Create a Machine Identity in Infisical with OCI Auth enabled
2. Configure allowed usernames and tenancy OCID in Infisical
3. Have OCI user credentials (private key, fingerprint, etc.)

### Storing Credentials

Create a Kubernetes secret containing your OCI credentials:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: oci-auth-credentials
type: Opaque
stringData:
  identityId: <your-machine-identity-id>
  privateKey: |
    -----BEGIN PRIVATE KEY-----
    ...
    -----END PRIVATE KEY-----
  fingerprint: <your-api-key-fingerprint>
  userId: <your-oci-user-ocid>
  tenancyId: <your-oci-tenancy-ocid>
  region: <your-oci-region>
  # Optional: only if the private key is encrypted
  # privateKeyPassphrase: <your-passphrase>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        ociAuthCredentials:
          identityId:
            name: oci-auth-credentials
            key: identityId
          privateKey:
            name: oci-auth-credentials
            key: privateKey
          fingerprint:
            name: oci-auth-credentials
            key: fingerprint
          userId:
            name: oci-auth-credentials
            key: userId
          tenancyId:
            name: oci-auth-credentials
            key: tenancyId
          region:
            name: oci-auth-credentials
            key: region
          # Optional: only if the private key is encrypted
          # privateKeyPassphrase:
          #   name: oci-auth-credentials
          #   key: privateKeyPassphrase
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

---

## Token Auth

[Token Auth](https://infisical.com/docs/documentation/platform/identities/token-auth) is the simplest authentication method that uses a pre-generated access token to authenticate directly with Infisical. This is similar to using an API key.

### Prerequisites

1. Create a Machine Identity in Infisical with Token Auth enabled
2. Generate an access token from the Infisical UI

### Storing Credentials

Create a Kubernetes secret containing your access token:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: token-auth-credentials
type: Opaque
stringData:
  accessToken: <your-access-token>
```

### SecretStore Configuration

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://app.infisical.com
      auth:
        tokenAuthCredentials:
          accessToken:
            name: token-auth-credentials
            key: accessToken
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! warning
    Token Auth tokens do not automatically renew. When the token expires, you will need to generate a new one and update the Kubernetes secret.

---

## Fetching Secrets

For the following examples, it assumes we have a secret structure in an Infisical project with the following structure:

```plaintext
/API_KEY
/DB_PASSWORD
/JSON_BLOB
/my-app
  /SERVICE_PASSWORD
  /ADMIN_PASSWORD
```

Where `JSON_BLOB` is a JSON string like `{"key": "value"}`.

### Fetch Individual Secret(s)

To sync one or more secrets individually, use the following YAML:

```yaml
{% include 'infisical-fetch-secret.yaml' %}
```

### Fetch All Secrets

To sync all secrets from an Infisical project, use the following YAML:

``` yaml
{% include 'infisical-fetch-all-secrets.yaml' %}
```

### Filtering Secrets

To filter secrets by `path` (path prefix) and `name` (regular expression):

``` yaml
{% include 'infisical-filtered-secrets.yaml' %}
```

---

## Custom CA Certificates

If you are using a self-hosted Infisical instance with a self-signed certificate or a certificate signed by a private CA, you can configure the provider to trust it.

### Using caBundle (inline)

You can provide the CA certificate directly as a base64-encoded PEM bundle:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://my-infisical.example.com
      # Base64-encoded PEM certificate
      caBundle: "LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0t..."
      auth:
        universalAuthCredentials:
          clientId:
            key: clientId
            name: universal-auth-credentials
          clientSecret:
            key: clientSecret
            name: universal-auth-credentials
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

### Using caProvider (from Secret or ConfigMap)

Alternatively, you can reference a Secret or ConfigMap containing the CA certificate:

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: infisical
spec:
  provider:
    infisical:
      hostAPI: https://my-infisical.example.com
      caProvider:
        type: Secret
        name: infisical-ca
        key: ca.crt
      auth:
        universalAuthCredentials:
          clientId:
            key: clientId
            name: universal-auth-credentials
          clientSecret:
            key: clientSecret
            name: universal-auth-credentials
      secretsScope:
        projectSlug: my-project
        environmentSlug: dev
```

!!! note
    For `ClusterSecretStore`, be sure to set `namespace` in `caProvider`.

---

## Secrets Scope Configuration

The `secretsScope` configuration controls which secrets are accessible:

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `projectSlug` | Yes | - | The slug identifier for your Infisical project |
| `environmentSlug` | Yes | - | The environment slug (e.g., `dev`, `staging`, `prod`) |
| `secretsPath` | No | `/` | The base path for secrets retrieval |
| `recursive` | No | `false` | When true, fetches secrets recursively from subfolders |
| `expandSecretReferences` | No | `true` | When true, expands secret references (e.g., `${SECRET_NAME}`) |

!!! tip
    To get your project slug from Infisical, head over to the project settings and click the button `Copy Project Slug`.
