# External Secrets

<img src="assets/round_eso_logo.png" width="100">

----

The External Secrets Operator reads information from a third party service
like [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/) and automatically injects the values as [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/).

Multiple people and organizations are joining efforts to create a single External Secrets solution based on existing projects. If you are curious about the origins of this project, check out this [issue](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and this [PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477).

# Supported Backends

- [AWS Secrets Manager](https://external-secrets.io/provider-aws-secrets-manager/)
- [AWS Parameter Store](https://external-secrets.io/provider-aws-parameter-store/)
- [Akeyless](https://www.akeyless.io/)
- [Hashicorp Vault](https://www.vaultproject.io/)
- [Google Cloud Secrets Manager](https://external-secrets.io/provider-google-secrets-manager/)
- [Azure Key Vault](https://external-secrets.io/provider-azure-key-vault/)
- [IBM Cloud Secrets Manager](https://external-secrets.io/provider-ibm-secrets-manager/)
- [Yandex Lockbox](https://external-secrets.io/provider-yandex-lockbox/)
- [Gitlab Project Variables](https://external-secrets.io/provider-gitlab-project-variables/)
- [Alibaba Cloud KMS](https://www.alibabacloud.com/product/kms) (Docs still missing, PRs welcomed!)
- [Oracle Vault]( https://external-secrets.io/provider-oracle-vault)

## Stability and Support Level

### Internally maintained:

| Provider                                                                 | Stability |                                        Contact |
| ------------------------------------------------------------------------ | :-------: | ---------------------------------------------: |
| [AWS SM](https://external-secrets.io/provider-aws-secrets-manager/)      |   alpha   | [ESO Org](https://github.com/external-secrets) |
| [AWS PS](https://external-secrets.io/provider-aws-parameter-store/)      |   alpha   | [ESO Org](https://github.com/external-secrets) |
| [Hashicorp Vault](https://external-secrets.io/provider-hashicorp-vault/) |   alpha   | [ESO Org](https://github.com/external-secrets) |
| [GCP SM](https://external-secrets.io/provider-google-secrets-manager/)   |   alpha   | [ESO Org](https://github.com/external-secrets) |

### Community maintained:

| Provider                                                            | Stability |                  Contact                   |
| ------------------------------------------------------------------- | :-------: | :----------------------------------------: |
| [Azure KV](https://external-secrets.io/provider-azure-key-vault/)   |   alpha   | [@ahmedmus-1A](https://github.com/ahmedmus-1A) [@asnowfix](https://github.com/asnowfix) [@ncourbet-1A](https://github.com/ncourbet-1A) [@1A-mj](https://github.com/1A-mj) |
| [IBM SM](https://external-secrets.io/provider-ibm-secrets-manager/) |   alpha   |   [@knelasevero](https://github.com/knelasevero) [@sebagomez](https://github.com/sebagomez) [@ricardoptcosta](https://github.com/ricardoptcosta)  |
| [Yandex Lockbox](https://external-secrets.io/provider-yandex-lockbox/) |   alpha   |   [@AndreyZamyslov](https://github.com/AndreyZamyslov) [@knelasevero](https://github.com/knelasevero)          |
| [Gitlab Project Variables](https://external-secrets.io/provider-gitlab-project-variables/) |   alpha   |   [@Jabray5](https://github.com/Jabray5)          |
| Alibaba Cloud KMS                                                   |   alpha  | [@ElsaChelala](https://github.com/ElsaChelala)                                |
| [Oracle Vault]( https://external-secrets.io/provider-oracle-vault)  |   alpha  | [@KianTigger](https://github.com/KianTigger)                                 |
| [Akeyless]( https://external-secrets.io/provider-akeyless)  |   alpha  | [@renanaAkeyless](https://github.com/renanaAkeyless)                                 |


## Documentation

External Secrets Operator guides and reference documentation is available at [external-secrets.io](https://external-secrets.io).

## Support

You can use GitHub's [issues](https://github.com/external-secrets/external-secrets/issues) to report bugs/suggest features or use GitHub's [discussions](https://github.com/external-secrets/external-secrets/discussions) to ask for help and figure out problems. You can also reach us at our KES and ESO shared [channel in Kubernetes slack](https://kubernetes.slack.com/messages/external-secrets).

Even though we have active maintainers and people assigned to this project, we kindly ask for patience when asking for support. We will try to get to priority issues as fast as possible, but there may be some delays.

## Contributing

We welcome and encourage contributions to this project! Please read the [Developer](https://www.external-secrets.io/contributing-devguide/) and [Contribution process](https://www.external-secrets.io/contributing-process/) guides. Also make sure to check the [Code of Conduct](https://www.external-secrets.io/contributing-coc/) and adhere to its guidelines.

## Security

Please report vulnerabilities by email to contact@external-secrets.io, also see our [security policy](SECURITY.md) for details.

## Kicked off by

![](assets/CS_logo_1.png)
![](assets/Godaddylogo_2020.png)
