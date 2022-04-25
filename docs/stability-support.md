This page lists the status, timeline and policy for currently supported ESO releases and its providers. Please also see our [deprecation policy](deprecation-policy.md) that describes API versioning, deprecation and API surface.

## External Secrets Operator

We are currently in beta and support **only the latest release** for the time being.

| ESO Version | Kubernetes Version |
| ----------- | ------------------ |
| 0.5.x       | 1.19 → 1.23        |
| 0.4.x       | 1.16 → 1.23        |
| 0.3.x       | 1.16 → 1.23        |

## Provider Stability and Support Level

The following table describes the stability level of each provider and who's responsible.

| Provider                                                                                          | Stability |                                                                                                                                                                   Maintainer |
| ------------------------------------------------------------------------------------------------- | :-------: | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
| [AWS Secrets Manager](https://external-secrets.io/latest/provider-aws-secrets-manager/)           |  stable   |                                                                                                                     [external-secrets](https://github.com/external-secrets) |
| [AWS Parameter Store](https://external-secrets.io/latest/provider-aws-parameter-store/)           |  stable   |                                                                                                                     [external-secrets](https://github.com/external-secrets) |
| [Hashicorp Vault](https://external-secrets.io/latest/provider-hashicorp-vault/)                   |  stable   |                                                                                                                     [external-secrets](https://github.com/external-secrets) |
| [GCP Secret Manager](https://external-secrets.io/latest/provider-google-secrets-manager/)         |  stable   |                                                                                                                     [external-secrets](https://github.com/external-secrets) |
| [Azure Keyvault](https://external-secrets.io/latest/provider-azure-key-vault/)                    |   beta    | [@ahmedmus-1A](https://github.com/ahmedmus-1A) [@asnowfix](https://github.com/asnowfix) [@ncourbet-1A](https://github.com/ncourbet-1A) [@1A-mj](https://github.com/1A-mj) |
| [IBM Secrets Manager](https://external-secrets.io/latest/provider-ibm-secrets-manager/)           |   alpha   |                            [@knelasevero](https://github.com/knelasevero) [@sebagomez](https://github.com/sebagomez) [@ricardoptcosta](https://github.com/ricardoptcosta) |
| [Yandex Lockbox](https://external-secrets.io/latest/provider-yandex-lockbox/)                     |   alpha   |                                                                       [@AndreyZamyslov](https://github.com/AndreyZamyslov) [@knelasevero](https://github.com/knelasevero) |
| [Gitlab Project Variables](https://external-secrets.io/latest/provider-gitlab-project-variables/) |   alpha   |                                                                                                                                    [@Jabray5](https://github.com/Jabray5) |
| Alibaba Cloud KMS                                                                                 |   alpha   |                                                                                                                            [@ElsaChelala](https://github.com/ElsaChelala) |
| [Oracle Vault](https://external-secrets.io/latest/provider-oracle-vault)                          |   alpha   |                                                                                   [@KianTigger](https://github.com/KianTigger) [@EladGabay](https://github.com/EladGabay) |
| [Akeyless](https://external-secrets.io/latest/provider-akeyless)                                  |   alpha   |                                                                                                                      [@renanaAkeyless](https://github.com/renanaAkeyless) |
| [Generic Webhook](https://external-secrets.io/latest/provider-webhook)                            |   alpha   |                                                                                                                                    [@willemm](https://github.com/willemm) |


## Support Policy

We provide technical support and provide security & bug fixes for the above listed versions.

### Technical support
We provide assistance for deploying/upgrading etc. on a best-effort basis. You can request support through the following channels:
* [Kubernetes Slack
  #external-secrets](https://kubernetes.slack.com/messages/external-secrets)
* GitHub [Issues](https://github.com/external-secrets/external-secrets/issues)
* GitHub [Discussions](https://github.com/external-secrets/external-secrets/discussions)

Even though we have active maintainers and people assigned to this project, we kindly ask for patience when asking for support. We will try to get to priority issues as fast as possible, but there may be some delays.
