# External Secrets

The External Secrets Kubernetes operator reads information from a third party service
like [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/) and automatically injects the values as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

Multiple people and organizations are joining efforts to create a single External Secrets solution based on existing projects. If you are curious about the origins of this project, check out this [issue](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and this [PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477).

<a name="original-projects"></a>

# ⚠️ Please bear in mind

While this project is not ready, you might consider using the following: 

- [Kubernetes External Secrets](https://github.com/external-secrets/kubernetes-external-secrets)
- [Secrets Manager](https://github.com/itscontained/secret-manager)
- [External Secrets Operator](https://github.com/ContainerSolutions/externalsecret-operator/)

## Installation
Clone this repository:
```shell
git clone https://github.com/external-secrets/external-secrets.git
```

Install the Custom Resource Definitions:
```shell
make install
```

Run the controller against the active Kubernetes cluster context:
```shell
make run
```

Apply the sample resources:
```shell
kubectl apply -f config/samples/external-secrets_v1alpha1_secretstore.yaml
kubectl applt -f config/samples/external-secrets_v1alpha1_externalsecret.yaml
```

We will add more documentation once we have the implementation for the different providers.

<a name="features"></a>

## Planned Features

- Support to multiple Provider stores (AWS Secret Manager, GCP Secret Manger, Vault and more) simultaneously.
- Multiple External Secrets operator instances for different contexts/environments.
- A custom refresh interval to sync the data from the Providers, syncing your Kubernetes Secrets up to date.
- Select specific versions of the Provider data.

<a name="partners"></a>

## Kicked off by

![](assets/CS_logo_1.png)
![](assets/Godaddylogo_2020.png)
