# External Secrets

<img src="assets/round_eso_logo.png" width="100">

----

The External Secrets Operator reads information from a third party service
like [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/) and automatically injects the values as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

Multiple people and organizations are joining efforts to create a single External Secrets solution based on existing projects. If you are curious about the origins of this project, check out this [issue](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and this [PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477).

# Supported Backends

- [AWS Secrets Manager](https://external-secrets.io/provider-aws-secrets-manager/)
- [AWS Parameter Store](https://external-secrets.io/provider-aws-parameter-store/)
- [Hashicorp Vault](https://www.vaultproject.io/)
- [Azure Key Vault](https://external-secrets.io/provider-azure-key-vault/) (being implemented)
- [Google Cloud Secrets Manager](https://external-secrets.io/provider-google-secrets-manager/) (being implemented)

## ESO installation with an AWS example


If you want to use Helm:

```shell
helm repo add external-secrets https://charts.external-secrets.io

helm install external-secrets \
   external-secrets/external-secrets \
    -n external-secrets \
    --create-namespace \
  # --set installCRDs=true
```

If you want to run it locally against the active Kubernetes cluster context:

```shell
git clone https://github.com/external-secrets/external-secrets.git
make crds.install
make run
```

Create a secret containing your AWS credentials:

```shell
echo -n 'KEYID' > ./access-key
echo -n 'SECRETKEY' > ./secret-access-key
kubectl create secret generic awssm-secret --from-file=./access-key  --from-file=./secret-access-key
```

Create a secret inside AWS Secret Manager with name `my-json-secret` with the following data:

```json
{
  "name": {"first": "Tom", "last": "Anderson"},
  "friends": [
    {"first": "Dale", "last": "Murphy"},
    {"first": "Roger", "last": "Craig"},
    {"first": "Jane", "last": "Murphy"}
  ]
}
```

Apply the sample resources (omitting role and controller keys here, you should not omit them in production):

```yaml
# secretstore.yaml
apiVersion: external-secrets.io/v1alpha1
kind: SecretStore
metadata:
  name: secretstore-sample
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-2
      auth:
        secretRef:
          accessKeyIDSecretRef:
            name: awssm-secret
            key: access-key
          secretAccessKeySecretRef:
            name: awssm-secret
            key: secret-access-key
```

```yaml
# externalsecret.yaml
apiVersion: external-secrets.io/v1alpha1
kind: ExternalSecret
metadata:
  name: example
spec:
  refreshInterval: 1m
  secretStoreRef:
    name: secretstore-sample
    kind: SecretStore
  target:
    name: secret-to-be-created
    creationPolicy: Owner
  data:
  - secretKey: firstname
    remoteRef:
      key: my-json-secret
      property: name.first # Tom
  - secretKey: first_friend
    remoteRef:
      key: my-json-secret
      property: friends.1.first # Roger
```

```shell
kubectl apply -f secretstore.yaml
kubectl apply -f externalsecret.yaml
```

Running `kubectl get secret secret-to-be-created` should return a new secret created by the operator.

You can get one of its values with jsonpath (This should return `Roger`):

```shell
kubectl get secret secret-to-be-created   -o jsonpath='{.data.first_friend}' | base64 -d
```

We will add more documentation once we have the implementation for the different providers. You can find some here: https://external-secrets.io


## Contributing

We welcome and encourage contributions to this project! Please read the [Developer](https://www.external-secrets.io/contributing-devguide/) and [Contribution process](https://www.external-secrets.io/contributing-process/) guides. Also make sure to check the [Code of Conduct](https://www.external-secrets.io/contributing-coc/) and adhere to its guidelines.

## Kicked off by

![](assets/CS_logo_1.png)
![](assets/Godaddylogo_2020.png)
