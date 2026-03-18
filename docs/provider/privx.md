# PrivX

External Secrets Operator integrates with SSH PrivX for secret management.
See [PrivX](https://www.ssh.com/products/privileged-access-management-privx)

This provider uses the PrivX Vault API to read and write secrets. See [PrivX API](https://privx.docs.ssh.com/v42/api)

Secrets are stored as objects within PrivX Vault.

## Building and Deploying from Source

This section documents how to build the External Secrets Operator with the PrivX provider from source and deploy it to a Kubernetes cluster.

### Prerequisites

Install required tools:

```bash
sudo dnf install wget make docker
```

Install Go:

```bash
wget https://go.dev/dl/go1.26.1.linux-amd64.tar.gz
cd /usr/local; sudo tar -xzvf ~/go1.26.1.linux-amd64.tar.gz
cd /usr/local/bin; sudo ln -s ../go/bin/go
```

Install yq:

```bash
VERSION=v4.52.4
PLATFORM=linux_amd64
sudo wget https://github.com/mikefarah/yq/releases/download/${VERSION}/yq_${PLATFORM} -O /usr/local/bin/yq && \
  sudo chmod +x /usr/local/bin/yq
```

Install Helm:

```bash
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-4
chmod 700 get_helm.sh
./get_helm.sh
```

### Build

Generate CRDs and manifests:

```bash
make manifests
```

Set the build to include only the PrivX provider (optional, reduces binary size):

```bash
sed -i 's/PROVIDER ?= all_providers/PROVIDER ?= privx/' Makefile
```

Build the binary and Docker image:

```bash
make build
make docker.build IMAGE_NAME=external-secrets IMAGE_TAG=latest
```

Tag the image for your target version:

```bash
docker image tag localhost/external-secrets:latest ghcr.io/external-secrets/external-secrets:v2.1.0
```

### Deploy to Kubernetes

Apply the CRDs using server-side apply (required for large CRDs):

```bash
kubectl apply --server-side --force-conflicts -f config/crds/bases/
```

Ignore the `kustomization.yaml` error — it is not a Kubernetes resource.

If using containerd as the container runtime, load the image into containerd's store (podman and containerd use separate image stores):

```bash
podman save ghcr.io/external-secrets/external-secrets:v2.1.0 | sudo ctr -n k8s.io images import -
```

Install with Helm (skip CRD install since they were applied above):

```bash
cd deploy/charts/external-secrets
helm dependency build
helm install external-secrets . \
  -n external-secrets \
  --create-namespace \
  --set installCRDs=false
```

### Create the connection credentials (this example uses OAuth):

```bash
kubectl create secret generic privx-secret \
--from-literal=privx_api_oauth_client_id='<SECRET-VALUE>' \
--from-literal=privx_api_oauth_client_secret='<SECRET-VALUE>' \
--from-literal=privx_api_client_id='<SECRET-VALUE>' \
--from-literal=privx_api_client_secret='<SECRET-VALUE>'
```

## Usage example

### Create a SecretStore with a PrivX backend. Enter following with `kubectl apply -f`

```yaml
apiVersion: external-secrets.io/v1
kind: SecretStore
metadata:
  name: secretstore-privx
spec:
  provider:
    privx:
      host: <privx host url>
      defaultReadRoles: [<default_read_role>]
      defaultWriteRoles: [<default_write_role>]
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
Note that the *OAuth user* must have a *role* in PrivX that is listed in the *readers of the secret*.

### Verify

Create the secret credentials, SecretStore, and ExternalSecret as described in the [Usage example](#usage-example) below, then verify:

```bash
kubectl get externalsecret
kubectl get secret privx-test-secret -o jsonpath='{.data.test_value}' | base64 -d
```


### Fetching Multiple Secrets

The PrivX provider supports dataFrom.find.

```yaml
Find by Name (RegExp)
dataFrom:
- find:
    name:
      regexp: "app-.*"
```

Returns all secrets whose name matches the regular expression.


# PushSecret

PrivX supports PushSecret to write Kubernetes Secret values into PrivX.

```yaml
apiVersion: external-secrets.io/v1
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
