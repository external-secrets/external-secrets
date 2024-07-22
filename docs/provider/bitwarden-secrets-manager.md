## Bitwarden Secrets Manager Provider

This section describes how to set up the Bitwarden Secrets Manager provider for External Secrets Operator (ESO).

### Prerequisites

In order for the bitwarden provider to work, we need a second service. This service is the [Bitwarden SDK Server](https://github.com/external-secrets/bitwarden-sdk-server).
The Bitwarden SDK is Rust based and requires CGO enabled. In order to not restrict the capabilities of ESO, and the image
size ( the bitwarden Rust SDK libraries are over 150MB in size ) it has been decided to create a soft wrapper
around the SDK that runs as a separate service providing ESO with a light REST API to pull secrets through.

#### Bitwarden SDK server

The server itself can be installed together with ESO. The ESO Helm Chart packages this service as a dependency.
The Bitwarden SDK Server's full name is hardcoded to bitwarden-sdk-server. This is so that the exposed service URL
gets a determinable endpoint.

In order to install the service install ESO with the following helm directive:

```
helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
    --set bitwarden-sdk-server.enabled=true
```

##### Certificate

The Bitwarden SDK Server _NEEDS_ to run as an HTTPS service. That means that any installation that wants to communicate with the Bitwarden
provider will need to generate a certificate. The best approach for that is to use cert-manager. It's easy to set up
and can generate a certificate that the store can use to connect with the server.

For a sample set up look at the bitwarden sdk server's test setup. It contains a self-signed certificate issuer for
cert-manager.

### External secret store

With that out of the way, let's take a look at how a secret store would look like.

```yaml
{% include 'bitwarden-secrets-manager-secret-store.yaml' %}
```

The api url and identity url are optional. The secret should contain the token for the Machine account for bitwarden.

!!! note inline end
Make sure that the machine account has Read-Write access to the Project that the secrets are in.

!!! note inline end
A secret store is organization/project dependent. Meaning a 1 store == 1 organization/project. This is so that we ensure
that no other project's secrets can be modified accidentally _or_ intentionally.

### External Secrets

There are two ways to fetch secrets from the provider.

#### Find by UUID

In order to fetch a secret by using its UUID simply provide that as remote key in the external secrets like this:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: bitwarden
spec:
  refreshInterval: 10s
  secretStoreRef:
    # This name must match the metadata.name in the `SecretStore`
    name: bitwarden-secretsmanager
    kind: SecretStore
  data:
  - secretKey: test
    remoteRef:
      key: "339062b8-a5a1-4303-bf1d-b1920146a622"
```

#### Find by Name

To find a secret using its name, we need a bit more information. Mainly, these are the rules to find a secret:

- if name is a UUID get the secret
- if name is NOT a UUID Property is mandatory that defines the projectID to look for
- if name + projectID + organizationID matches, we return that secret
- if more than one name exists for the same projectID within the same organization we error

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: bitwarden
spec:
  refreshInterval: 10s
  secretStoreRef:
    # This name must match the metadata.name in the `SecretStore`
    name: bitwarden-secretsmanager
    kind: SecretStore
  data:
  - secretKey: test
    remoteRef:
      key: "secret-name"
```

### Push Secret

Pushing a secret is also implemented. Pushing a secret requires even more restrictions because Bitwarden Secrets Manager
allows creating the same secret with the same key multiple times. In order to avoid overwriting, or potentially, returning
the wrong secret, we restrict push secret with the following rules:

- name, projectID, organizationID and value AND NOTE equal, we won't push it again.
- name, projectID, organizationID and ONLY the value does not equal ( INCLUDING THE NOTE ) we update
- any of the above isn't true, we create the secret ( this means that it will create a secret in a separate project )

```yaml
apiVersion: external-secrets.io/v1alpha1
kind: PushSecret
metadata:
  name: pushsecret-bitwarden # Customisable
spec:
  refreshInterval: 10s # Refresh interval for which push secret will reconcile
  secretStoreRefs: # A list of secret stores to push secrets to
    - name: bitwarden-secretsmanager
      kind: SecretStore
  selector:
    secret:
      name: my-secret # Source Kubernetes secret to be pushed
  data:
    - match:
        secretKey: key # Source Kubernetes secret key to be pushed
        remoteRef:
          remoteKey: remote-key-name # Remote reference (where the secret is going to be pushed)
      metadata:
        note: "Note of the secret to add."
```
