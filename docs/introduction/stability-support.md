---
hide:
  - toc
---

This page lists the status, timeline and policy for currently supported ESO releases and its providers. Please also see our [deprecation policy](deprecation-policy.md) that describes API versioning, deprecation and API surface.

## External Secrets Operator

We are currently in beta and support **only the latest release** for the time being.

| ESO Version | Kubernetes Version |
| ----------- | ------------------ |
| 0.7.x       | 1.19 → 1.26        |
| 0.6.x       | 1.19 → 1.24        |
| 0.5.x       | 1.19 → 1.24        |
| 0.4.x       | 1.16 → 1.24        |
| 0.3.x       | 1.16 → 1.24        |

## Provider Stability and Support Level

The following table describes the stability level of each provider and who's responsible.

| Provider                                                                                                   | Stability |                                                                                                                                     Maintainer |
|------------------------------------------------------------------------------------------------------------|:---------:|-----------------------------------------------------------------------------------------------------------------------------------------------:|
| [AWS Secrets Manager](https://external-secrets.io/latest/provider/aws-secrets-manager/)                    |  stable   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [AWS Parameter Store](https://external-secrets.io/latest/provider/aws-parameter-store/)                    |  stable   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [Hashicorp Vault](https://external-secrets.io/latest/provider/hashicorp-vault/)                            |  stable   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [GCP Secret Manager](https://external-secrets.io/latest/provider/google-secrets-manager/)                  |  stable   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [Azure Keyvault](https://external-secrets.io/latest/provider/azure-key-vault/)                             |  stable   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [IBM Cloud Secrets Manager](https://external-secrets.io/latest/provider/ibm-secrets-manager/)              |  stable   | [@knelasevero](https://github.com/knelasevero) [@sebagomez](https://github.com/sebagomez) [@ricardoptcosta](https://github.com/ricardoptcosta) [@IdanAdar](https://github.com/IdanAdar) |
| [Kubernetes](https://external-secrets.io/latest/provider/kubernetes)                                       |   alpha   |                                                                                        [external-secrets](https://github.com/external-secrets) |
| [Yandex Lockbox](https://external-secrets.io/latest/provider/yandex-lockbox/)                              |   alpha   |                                            [@AndreyZamyslov](https://github.com/AndreyZamyslov) [@knelasevero](https://github.com/knelasevero) |
| [Gitlab Variables](https://external-secrets.io/latest/provider/gitlab-variables/)                          |   alpha   |                                                                                                         [@Jabray5](https://github.com/Jabray5) |
| Alibaba Cloud KMS                                                                                          |   alpha   |                                                                                                 [@ElsaChelala](https://github.com/ElsaChelala) |
| [Oracle Vault](https://external-secrets.io/latest/provider/oracle-vault)                                   |   alpha   |                                                        [@KianTigger](https://github.com/KianTigger) [@EladGabay](https://github.com/EladGabay) |
| [Akeyless](https://external-secrets.io/latest/provider/akeyless)                                           |   alpha   |                                                                                           [@renanaAkeyless](https://github.com/renanaAkeyless) |
| [1Password](https://external-secrets.io/latest/provider/1password-automation)                              |   alpha   |                                              [@SimSpaceCorp](https://github.com/Simspace) [@snarlysodboxer](https://github.com/snarlysodboxer) |
| [Generic Webhook](https://external-secrets.io/latest/provider/webhook)                                     |   alpha   |                                                                                                         [@willemm](https://github.com/willemm) |
| [senhasegura DevOps Secrets Management (DSM)](https://external-secrets.io/latest/provider/senhasegura-dsm) |   alpha   |                                                                                                           [@lfraga](https://github.com/lfraga) |
| [Doppler SecretOps Platform](https://external-secrets.io/latest/provider/doppler)                          |   alpha   |                                                [@ryan-blunden](https://github.com/ryan-blunden/) [@nmanoogian](https://github.com/nmanoogian/) |

## Provider Feature Support

The following table show the support for features across different providers.

| Provider                  | find by name | find by tags | metadataPolicy Fetch | referent authentication | store validation | push secret |
|---------------------------|:------------:|:------------:| :------------------: | :---------------------: | :--------------: | :---------: |
| AWS Secrets Manager       |      x       |      x       |                      |            x            |        x         |             |
| AWS Parameter Store       |      x       |      x       |                      |            x            |        x         |             |
| Hashicorp Vault           |      x       |      x       |                      |                         |        x         |             |
| GCP Secret Manager        |      x       |      x       |                      |            x            |        x         |             |
| Azure Keyvault            |      x       |      x       |          x           |            x            |        x         |     x        |
| Kubernetes                |      x       |      x       |                      |            x            |        x         |             |
| IBM Cloud Secrets Manager |              |              |                      |                         |        x         |             |
| Yandex Lockbox            |              |              |                      |                         |        x         |             |
| Gitlab Variables          |      x       |      x       |                      |                         |        x         |             |
| Alibaba Cloud KMS         |              |              |                      |                         |        x         |             |
| Oracle Vault              |              |              |                      |                         |        x         |             |
| Akeyless                  |              |              |                      |                         |        x         |             |
| 1Password                 |      x       |              |                      |                         |        x         |             |
| Generic Webhook           |              |              |                      |                         |                  |             |
| senhasegura DSM           |              |              |                      |                         |        x         |             |
| Doppler                   |      x       |              |                      |                         |        x         |             |


## Support Policy

We provide technical support and security / bug fixes for the above listed versions.

### Technical support
We provide assistance for deploying/upgrading etc. on a best-effort basis. You can request support through the following channels:

* [Kubernetes Slack
  #external-secrets](https://kubernetes.slack.com/messages/external-secrets)
* GitHub [Issues](https://github.com/external-secrets/external-secrets/issues)
* GitHub [Discussions](https://github.com/external-secrets/external-secrets/discussions)

Even though we have active maintainers and people assigned to this project, we kindly ask for patience when asking for support. We will try to get to priority issues as fast as possible, but there may be some delays.
