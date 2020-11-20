# External Secrets

The External Secrets Kubernetes operator reads information from a third party service
like [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/)  and automatically injects the values as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

Multiple people and organizations are joining efforts to create a single External Secrets solution based on existing projects. If you are curious about the origins of this project, check out this issue [kubernetes-external-secrets/issues/47](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and this [PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477) and the discussion that triggered this.


<a name="features"></a>

# Planned Features

- Support to multiple Provider stores (AWS Secret Manager, GCP Secret Manger, Vault and more) simultaneously.
- Multiple External Secrets operator instances for different contexts/environments.
- A custom refresh interval to sync the data from the Providers, syncing your Kubernetes Secrets up to date.
- Select specific versions of the Provider data.

<a name="partners"></a>

<!-- Not sure how to word this properly. -->

# Kicked off by

![](assets/CS_logo_1.png)

![](assets/Godaddylogo_2020.png)

<!-- Who else? Please add here. -->

<a name="original-projects"></a>

# Please bear in mind

While this project is not stable and we don't have feature parity with the original projects, maybe you would like to consider having a look over these:

[Kubernetes External Secrets](https://github.com/external-secrets/kubernetes-external-secrets)

[Secrets Manager](https://github.com/itscontained/secret-manager)

[External Secrets Operator](https://github.com/ContainerSolutions/externalsecret-operator/)

<!-- Who else? Please add here. -->