You can test a feature that was not yet released using the following methods, use them at your own discretion:

### Helm
1. Create a `values.yaml` file with the following content:
```yaml
replicaCount: 1

image:
  repository: ghcr.io/external-secrets/external-secrets
  pullPolicy: IfNotPresent
  # -- The image tag to use. The default is the chart appVersion.
  tag: "main"

# -- If set, install and upgrade CRDs through helm chart.
installCRDs: false
```
1. Install the crds
```shell
make crds.install
```
1. Install the external-secrets Helm chart indicating the values file created before:
```
helm install external-secrets external-secrets/external-secrets -f values.yaml
```


### Manual
1. Build the Docker image
```shell
docker build -f Dockerfile.standalone -t my-org/external-secrets:latest .
```
1. Apply the `bundle.yaml`
```shell
kubectl apply -f deploy/crds/bundle.yaml
```
1. Modify your configs to use the image
```yaml
kind: Deployment
metadata:
  name: external-secrets|external-secrets-webhook|external-secrets-cert-controller
...
        image: my-org/external-secrets:latest
```
