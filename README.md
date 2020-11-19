# External Secrets

This operator reads information from a third party service
like [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/)  and automatically injects the values as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) (other secret targets are also planned).

This is a new joint effort to consolidate a single central solution that delivers on most of the requirements gathered from multiple other external secrets projects out there. If you are curious about the origins of this project, check out this issue [kubernetes-external-secrets/issues/47](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and this [PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477) and the discussion that triggered this.


<a name="features"></a>

# Planned Features

- Multiple provider stores supported simultaneously in your cluster.
- Multiple External Secrets instances, each for a different context/environment in your cluster (dev/prod).
- Secrets being refreshed from time to time allowing you to rotate secrets in your Providers and still keep everything up to date inside your k8s cluster.
- Changing the refresh interval of the secrets to match your needs. You can even make it 10s if you need to debug something (beware of API rate limits).
- Using speciffic versions of the secrets or just gettting latest versions of them.
- Changing something in your ExternalSecret CR will trigger a reconcile it (Even if your refresh interval is big).
- AWS Secret Manager, Google Secret Manager, Gitlab, Vault, Azure and many other backends planned!


<a name="partners"></a>

<!-- Not sure how to word this properly. -->

# Partner Companies Maintaining this repository/org

![](assets/CS_logo_1.png)

![](assets/Godaddylogo_2020.png)

<!-- Who else? Please add here. -->

<a name="original-projects"></a>

# While this project is being maintained

While this project is not stable and we don't have feature parity with the original projects, maybe you would like to consider having a look over these:

[Kubernetes External Secrets](https://github.com/external-secrets/kubernetes-external-secrets)

[Secrets Manager](https://github.com/itscontained/secret-manager)

[External Secrets Operator](https://github.com/ContainerSolutions/externalsecret-operator/)

<!-- Who else? Please add here. -->