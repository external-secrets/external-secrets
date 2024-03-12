## Fortanix DSM / SDKMS

Populate kubernetes secrets from OPAQUE or SECRET security objects in Fortanix.

### Authentication

SDKMS [Application API Key](https://support.fortanix.com/hc/en-us/articles/360015941132-Authentication)

### Creating a SecretStore

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: secret-store
spec:
  provider:
    fortanix:
      apiUrl: <HOST_OF_SDKMS_API>
      apiKey:
        secretRef:
          name: <NAME_OF_KUBE_SECRET>
          key: <KEY_IN_KUBE_SECRET>
```

### Referencing Secrets

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: secret
spec:
  refreshInterval: 1h
  secretStoreRef:
    kind: SecretStore
    name: secret-store
  data:

  # Raw stored value
  - secretKey: <KEY_IN_KUBE_SECRET>
    remoteRef:
      key: <SDKMS_SECURITY_OBJECT_NAME>

  # From stored key-value JSON
  - secretKey: <KEY_IN_KUBE_SECRET>
    remoteRef:
      key: <SDKMS_SECURITY_OBJECT_NAME>
      property: <SECURITY_OBJECT_VALUE_INNER_PROPERTY>
```
