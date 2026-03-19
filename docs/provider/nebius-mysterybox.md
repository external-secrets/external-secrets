## Nebius MysteryBox

External Secrets Operator integrates with [Nebius MysteryBox](https://docs.nebius.com/mysterybox/overview).

### Authentication

Nebius MysteryBox supports the following authentication methods:

- `auth.serviceAccountRef`: request a Kubernetes service account token via the `TokenRequest` API and exchange it for a Nebius IAM token using workload federation.
- `auth.serviceAccountCredsSecretRef`: read Nebius service account credentials JSON from a Kubernetes `Secret` and exchange it for a Nebius IAM token.
- `auth.tokenSecretRef`: read an already issued Nebius IAM token from a Kubernetes `Secret`.

#### Kubernetes service account federation

Before you start, configure the Nebius federated credentials for the Kubernetes service account outside External Secrets Operator. ESO assumes that this Nebius-side federation setup already exists and only performs runtime token exchange.

Then create a Kubernetes `ServiceAccount` and grant the impersonated Nebius service account permission to read desired secrets in MysteryBox.
For details on required roles and permissions, see [MysteryBox get method](https://docs.nebius.com/mysterybox/secrets/get).

The SecretStore example below uses this authentication method.

#### Nebius service account credentials JSON

If you prefer static Nebius service account credentials, create a service account and grant it permission to read desired secrets in MysteryBox.

You will need to create a Kubernetes Secret with desired auth parameters and structure.
The Kubernetes secret must be in a Subject Credentials format:

```json
{
  "subject-credentials": {
    "alg": "RS256",
    "private-key": "-----BEGIN PRIVATE KEY-----\n<private-key>\n-----END PRIVATE KEY-----\n",
    "kid": "<public-key-ID>",
    "iss": "<service_account_ID>",
    "sub": "<service_account_ID>"
  }
}
```

Follow the [instruction](https://docs.nebius.com/iam/service-accounts/authorized-keys#create) to generate the secret.

### Examples

#### SecretStore

First, create a SecretStore with a Nebius MysteryBox backend.

```yaml
{% include 'nebius-mysterybox-secret-store.yaml' %}
```

#### Getting a secret by key

You can get a secret by its secretID and key.

```yaml
{% include 'nebius-mysterybox-external-secret-by-key.yaml' %}
```

#### Getting a full secret (all keys retrieved)

Another way is to get a full secret that will be imported. When fetching the full secret, each key–value pair from MysteryBox is mapped to a separate entry in the target Kubernetes Secret’s `data` field.


```yaml
{% include 'nebius-mysterybox-external-secret-all.yaml' %}
```

Example of a target secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <your-k8s-secret-name>
type: Opaque
data:
  <entry-key-1>: <base64-of-value-1>
  <entry-key-2>: <base64-of-value-2>
```

#### Additional usage

There is also a possibility to specify Version variable to get a secret.

```yaml
...
 data:
    - secretKey: <secretKey>
      remoteRef:
        key: <secretID>
        version: <secretVersion>

```

!!! tip inline end
    When the `version` field is not specified, a primary version of the secret will be retrieved.

