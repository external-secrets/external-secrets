The `BeyondtrustSecretsDynamicSecret` Generator provides an interface to BeyondTrust Secrets Manager's
dynamic secret generation capabilities. This enables obtaining temporary, short-lived credentials.

Dynamic secret definitions must be created in BeyondTrust Secrets Manager before they can be
referenced by the generator. The generator calls the generation endpoint to produce fresh credentials
each time it is invoked.

Any authentication method supported by the BeyondTrust Secrets Manager provider can be used here
(`provider` block of the spec).

## Example manifest

```yaml
{% include 'beyondtrustsecrets-dynamic-secret.yaml' %}
```

Example `ExternalSecret` that references the BeyondTrust Secrets Manager generator:
```yaml
{% include 'beyondtrustsecrets-dynamic-external-secret.yaml' %}
```

## Configuration

### Folder Path

The `folderPath` in the generator spec uses the format `{folder}/{secretName}`:
- `folder`: The folder containing the dynamic secret definition (e.g., `eso`)
- `secretName`: The name of the dynamic secret definition (e.g., `dynamic`)

For example, if your dynamic secret is stored at path `my/dynamic` in BeyondTrust Secrets Manager:

```yaml
spec:
  provider:
    folderPath: "my/dynamic"
```
### Generated Secret Fields
The generator returns different fields depending on the type of dynamic secret:
#### AWS Dynamic Secrets
```yaml
data:
  accessKeyId: ASIAIOSFODNN7EXAMPLE
  secretAccessKey: wJal...YEKY
  sessionToken: IQoJ...Ek8=
  leaseId: 84038398-ec0f-417d-9a0f-02494fd7d22c
  expiration: 2025-12-29T22:35:29Z
```
All fields are automatically populated in the target Kubernetes secret.
### Credential Refresh and Expiration
**Important:** External Secrets Operator does NOT automatically handle credential expiration/TTL from BeyondTrust Secrets Manager. The refresh is controlled solely by the `refreshInterval` specified in the ExternalSecret spec.

#### Setting Refresh Interval

You should set `refreshInterval` to **less than** the credential lifetime to ensure credentials are refreshed before expiration:

```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: aws-credentials
spec:
  refreshInterval: 45m  # If credentials expire in 1 hour
  target:
    name: aws-temp-creds
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: BeyondtrustSecretsDynamicSecret
          name: beyondtrustsecrets-ds
```

#### What happens if refreshInterval > credential expiration?

Credentials will expire before being refreshed. Users will see:
- ExternalSecret status: `SecretSyncError`
- Logs/events: Authorization errors when the application tries to use expired credentials
- The application will fail to authenticate with the target service

#### What happens if refreshInterval << credential expiration?

For example, if credentials expire in 1 hour but `refreshInterval: 1m`:
- New credentials are generated every minute
- Old credentials remain valid until their expiration time
- Multiple valid credential sets may exist simultaneously
- **Credential revocation is handled automatically by AWS** (for assume role credentials).

**Recommendation:** Set `refreshInterval` to 75-80% of the credential lifetime. For example:
- 1-hour credentials → `refreshInterval: 45m`
- 12-hour credentials → `refreshInterval: 9h`
- 24-hour credentials → `refreshInterval: 18h`

### Generator Reusability

Generators are reusable Custom Resources. You can reference the same generator from multiple ExternalSecrets:

```yaml
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-1-aws-creds
spec:
  refreshInterval: 45m
  target:
    name: app-1-aws-credentials
  dataFrom:
    - sourceRef:
        generatorRef:
          kind: BeyondtrustSecretsDynamicSecret
          name: beyondtrustsecrets-ds
---
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-2-aws-creds
spec:
  refreshInterval: 45m
  target:
    name: app-2-aws-credentials
  dataFrom:
    - sourceRef:
        generatorRef:
          kind: BeyondtrustSecretsDynamicSecret
          name: beyondtrustsecrets-ds
```

**Important:** Each reference triggers a **new credential generation**. In the example above, `app-1` and `app-2` will receive different, independent sets of credentials.

### Authentication

The generator uses the same authentication mechanism as the BeyondTrust Secrets Manager provider (API key authentication):

```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
metadata:
  name: beyondtrustsecrets-ds
  namespace: external-secrets
spec:
  provider:
    auth:
      apikey:
        token:
          name: api-token
          key: token
```

Create the API token secret:
```bash
kubectl create secret generic api-token \
  --from-literal=token=<YOUR_API_TOKEN> \
  -n external-secrets
```

### Certificate Trust

If using self-signed certificates, configure trust using `caProvider`:

```yaml
spec:
  provider:
    # ... other config ...
    caProvider:
      type: Secret
      name: my-ca-bundle
      key: ca.crt
```

Create the CA bundle secret:
```bash
kubectl create secret generic my-ca-bundle \
  --from-file=ca.crt="/path/to/ca.crt" \
  -n external-secrets
```

### Server Configuration

Configure the BeyondTrust Secrets Manager API endpoint:

```yaml
spec:
  provider:
    server:
      apiUrl: "https://api.beyondtrust.io/site"
      siteId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
```

- `apiUrl`: The base URL of your BeyondTrust Secrets Manager API
- `siteId`: Your BeyondTrust site identifier (UUID format)

### Complete Example

Here's a complete example for AWS dynamic credentials:

1. Create the API token and CA bundle secrets:
```bash
kubectl create secret generic api-token \
  --from-literal=token=<YOUR_API_TOKEN> \
  -n external-secrets
kubectl create secret generic my-ca-bundle \
  --from-file=ca.crt="/path/to/ca.crt" \
  -n external-secrets
```

2. Create the generator:
```yaml
apiVersion: generators.external-secrets.io/v1alpha1
kind: BeyondtrustSecretsDynamicSecret
metadata:
  name: aws-dynamic-generator
  namespace: external-secrets
spec:
  provider:
    auth:
      apikey:
        token:
          name: api-token
          key: token
    server:
      apiUrl: "https://api.beyondtrust.io/site"
      siteId: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    folderPath: "production/aws-temp"
```

3. Create an ExternalSecret that uses the generator:
```yaml
apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata:
  name: app-aws-credentials
  namespace: production
spec:
  refreshInterval: 45m  # Refresh before 1-hour expiration
  target:
    name: aws-temp-credentials
    creationPolicy: Owner
  dataFrom:
    - sourceRef:
        generatorRef:
          apiVersion: generators.external-secrets.io/v1alpha1
          kind: BeyondtrustSecretsDynamicSecret
          name: aws-dynamic-generator
```

4. The resulting Kubernetes secret will contain:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-temp-credentials
  namespace: production
data:
  accessKeyId: QVNJ...R04=
  secretAccessKey: Z3dk...WFk=
  sessionToken: SVFv...Ek8=
  leaseId: NTdk...Nm1j
  expiration: MjAy...OVo=
```

### Troubleshooting

#### Empty Credential Fields

If the generated secret has empty values:
1. Verify the dynamic secret exists in BeyondTrust Secrets Manager at the specified path
2. Check the API token has permissions to generate credentials
3. Verify the `folderPath` format is correct (`folder/secretName`)
4. Check controller logs: `kubectl logs -l app.kubernetes.io/name=external-secrets -n external-secrets`

#### Authentication Errors

If you see 403/401 errors:
1. Verify the API token is valid and not expired
2. Check the token has `generate` permissions for the dynamic secret
3. Ensure the `caProvider` or `caBundle` is configured correctly if using self-signed certificates

#### Timeout Errors

If credential generation times out:
1. Check network connectivity from the cluster to BeyondTrust Secrets Manager API
2. Verify the API endpoint is responsive
3. Check if there are firewall rules blocking the connection

#### Credential Expiration Issues

If applications report authentication failures:
1. Check if `refreshInterval` is greater than credential lifetime
2. Review the `expiration` field in the secret to see when credentials expire
3. Adjust `refreshInterval` to be 75-80% of the credential lifetime
4. Check ExternalSecret status: `kubectl describe externalsecret <name> -n <namespace>`