You can test a feature that was not yet released using the following method:

1. Create a `values.yaml` file with the following content:
   
```
replicaCount: 1

image:
  repository: ghcr.io/external-secrets/external-secrets
  pullPolicy: IfNotPresent
  # -- The image tag to use. The default is the chart appVersion.
  tag: "main"

# -- If set, install and upgrade CRDs through helm chart.
installCRDs: false
```

2. Install the crds
```
make crds.install
```

3. Install the external-secrets Helm chart indicating the values file created before:
```
helm install external-secrets external-secrets/external-secrets -f values.yaml
``` 