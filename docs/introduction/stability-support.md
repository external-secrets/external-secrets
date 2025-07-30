---
hide:
  - toc
---

This page lists the status, timeline and policy for currently supported ESO releases and its providers. Please also see our [deprecation policy](deprecation-policy.md) that describes API versioning, deprecation and API surface.

## Supported Versions

external-secrets only supports the most-up-to date, current minor version. Any other minor version releases are automatically deprecated as soon as a new minor version comes.

During a minor version support time, we cover:
- regular image rebuilds to update OS dependencies
- regular go dependency updates

We do not do test coverage for any other kubernetes version than the ones running on our test suites.
As of version 0.14.x , this is the only kubernetes version that we will guarantee support for.

| ESO Version | Kubernetes Version | Release Date | End of Life     |
| ----------- | ------------------ | ------------ | --------------- |
| 0.18.x      | 1.33               | July 17, 2025  | Release of 0.19 |
| 0.17.x      | 1.33               | May 14, 2025  | July 17, 2025    |
| 0.16.x      | 1.32               | Apr 14, 2025  | May 14, 2025    |
| 0.15.x      | 1.32               | Mar 19, 2025  | Apr 14, 2025    |
| 0.14.x      | 1.32               | Feb 4, 2025  | Mar 19, 2025    |
| 0.13.x      | 1.19 → 1.31        | Jan 21, 2025 | Feb 4, 2025     |
| 0.12.x      | 1.19 → 1.31        | Dec 24, 2024 | Jan 21, 2025    |
| 0.11.x      | 1.19 → 1.31        | Dec 2, 2024  | Dec 24, 2024    |
| 0.10.x      | 1.19 → 1.31        | Aug 3, 2024  | Dec 24, 2024    |
| 0.9.x       | 1.19 → 1.30        | Jun 22, 2023 | Dec 2, 2024     |
| 0.8.x       | 1.19 → 1.28        | Mar 16, 2023 | Aug 3, 2024     |
| 0.7.x       | 1.19 → 1.26        | Dec 11, 2022 | Jun 22, 2023    |
| 0.6.x       | 1.19 → 1.24        | Oct 9, 2022  | Mar 16, 2023    |
| 0.5.x       | 1.19 → 1.24        | Apr 6, 2022  | Dec 11, 2022    |
| 0.4.x       | 1.16 → 1.24        | Feb 2, 2022  | Oct 9, 2022     |
| 0.3.x       | 1.16 → 1.24        | Jul 25, 2021 | Apr 6, 2022     |

## Provider Stability and Support Level

The following table describes the stability level of each provider and who's responsible.

| Provider                                                                                                   | Stability | Maintainer                                                                                          |
|------------------------------------------------------------------------------------------------------------|-:-:-------|--:--------------------------------------------------------------------------------------------------|
| [AWS Secrets Manager](https://external-secrets.io/latest/provider/aws-secrets-manager/)                    | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [AWS Parameter Store](https://external-secrets.io/latest/provider/aws-parameter-store/)                    | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [Hashicorp Vault](https://external-secrets.io/latest/provider/hashicorp-vault/)                            | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [GCP Secret Manager](https://external-secrets.io/latest/provider/google-secrets-manager/)                  | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [Azure Keyvault](https://external-secrets.io/latest/provider/azure-key-vault/)                             | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [IBM Cloud Secrets Manager](https://external-secrets.io/latest/provider/ibm-secrets-manager/)              | stable    | [@IdanAdar](https://github.com/IdanAdar)                                                            |
| [Kubernetes](https://external-secrets.io/latest/provider/kubernetes)                                       | beta      | [external-secrets](https://github.com/external-secrets)                                             |
| [Yandex Lockbox](https://external-secrets.io/latest/provider/yandex-lockbox/)                              | alpha     | [@AndreyZamyslov](https://github.com/AndreyZamyslov) [@knelasevero](https://github.com/knelasevero) |
| [GitLab Variables](https://external-secrets.io/latest/provider/gitlab-variables/)                          | alpha     | [@Jabray5](https://github.com/Jabray5)                                                              |
| Alibaba Cloud KMS                                                                                          | alpha     | **UNMAINTAINED**                                                                                    |
| [Oracle Vault](https://external-secrets.io/latest/provider/oracle-vault)                                   | alpha     | **UNMAINTAINED**                                                                                    |
| [Akeyless](https://external-secrets.io/latest/provider/akeyless)                                           | stable    | [external-secrets](https://github.com/external-secrets)                                             |
| [1Password](https://external-secrets.io/latest/provider/1password-automation)                              | alpha     | [@SimSpaceCorp](https://github.com/Simspace) [@snarlysodboxer](https://github.com/snarlysodboxer)   |
| [1Password SDK](https://external-secrets.io/latest/provider/1password-sdk)                                 | alpha     | [@Skarlso](https://github.com/Skarlso)                                                              |
| [Generic Webhook](https://external-secrets.io/latest/provider/webhook)                                     | alpha     | [@willemm](https://github.com/willemm)                                                              |
| [senhasegura DevOps Secrets Management (DSM)](https://external-secrets.io/latest/provider/senhasegura-dsm) | alpha     | [@lfraga](https://github.com/lfraga)                                                                |
| [Doppler SecretOps Platform](https://external-secrets.io/latest/provider/doppler)                          | alpha     | [@ryan-blunden](https://github.com/ryan-blunden/) [@nmanoogian](https://github.com/nmanoogian/)     |
| [Keeper Security](https://www.keepersecurity.com/)                                                         | alpha     | [@ppodevlab](https://github.com/ppodevlab)                                                          |
| [Scaleway](https://external-secrets.io/latest/provider/scaleway)                                           | alpha     | [@azert9](https://github.com/azert9/)                                                               |
| [Conjur](https://external-secrets.io/latest/provider/conjur)                                               | stable    | [@davidh-cyberark](https://github.com/davidh-cyberark/) [@szh](https://github.com/szh)              |
| [Delinea](https://external-secrets.io/latest/provider/delinea)                                             | alpha     | [@michaelsauter](https://github.com/michaelsauter/)                                                 |
| [Beyondtrust](https://external-secrets.io/latest/provider/beyondtrust)                                     | alpha     | [@btfhernandez](https://github.com/btfhernandez/)                                                   |
| [SecretServer](https://external-secrets.io/latest/provider/secretserver)                                   | alpha     | [@billhamilton](https://github.com/pacificcode/)                                                    |
| [Pulumi ESC](https://external-secrets.io/latest/provider/pulumi)                                           | alpha     | [@dirien](https://github.com/dirien)                                                                |
| [Passbolt](https://external-secrets.io/latest/provider/passbolt)                                           | alpha     | **UNMAINTAINED**                                                                                    |
| [Infisical](https://external-secrets.io/latest/provider/infisical)                                         | alpha     | [@akhilmhdh](https://github.com/akhilmhdh)                                                          |
| [Device42](https://external-secrets.io/latest/provider/device42)                                           | alpha     | **UNMAINTAINED**                                                                                    |
| [Bitwarden Secrets Manager](https://external-secrets.io/latest/provider/bitwarden-secrets-manager)         | alpha     | [@skarlso](https://github.com/Skarlso)                                                              |
| [Previder](https://external-secrets.io/latest/provider/previder)                                           | stable    | [@previder](https://github.com/previder)                                                            |
| [Cloud.ru](https://external-secrets.io/latest/provider/cloudru)                                            | alpha     | [@default23](https://github.com/default23)                                                          |


## Provider Feature Support

The following table show the support for features across different providers.

| Provider                  | find by name | find by tags | metadataPolicy Fetch | referent authentication | store validation | push secret | DeletionPolicy Merge/Delete |
|---------------------------|:------------:|:------------:|:--------------------:|:-----------------------:|:----------------:|:-----------:|:---------------------------:|
| AWS Secrets Manager       |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| AWS Parameter Store       |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| Hashicorp Vault           |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| GCP Secret Manager        |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| Azure Keyvault            |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| Kubernetes                |      x       |      x       |          x           |            x            |        x         |      x      |              x              |
| IBM Cloud Secrets Manager |      x       |              |          x           |                         |        x         |             |                             |
| Yandex Lockbox            |              |              |                      |                         |        x         |             |                             |
| GitLab Variables          |      x       |      x       |                      |                         |        x         |             |                             |
| Alibaba Cloud KMS         |              |              |                      |                         |        x         |             |                             |
| Oracle Vault              |              |              |                      |                         |        x         |             |                             |
| Akeyless                  |      x       |      x       |                      |            x            |        x         |      x      |              x              |
| 1Password                 |      x       |      x       |                      |                         |        x         |      x      |              x              |
| 1Password SDK             |              |              |                      |                         |        x         |      x      |              x              |
| Generic Webhook           |              |              |                      |                         |                  |             |              x              |
| senhasegura DSM           |              |              |                      |                         |        x         |             |                             |
| Doppler                   |      x       |              |                      |                         |        x         |             |                             |
| Keeper Security           |      x       |              |                      |                         |        x         |      x      |                             |
| Scaleway                  |      x       |      x       |                      |                         |        x         |      x      |              x              |
| Conjur                    |      x       |      x       |                      |                         |        x         |             |                             |
| Delinea                   |      x       |              |                      |                         |        x         |             |                             |
| Beyondtrust               |      x       |              |                      |                         |        x         |             |                             |
| SecretServer              |      x       |              |                      |                         |        x         |             |                             |
| Pulumi ESC                |      x       |              |                      |                         |        x         |             |                             |
| Passbolt                  |      x       |              |                      |                         |        x         |             |                             |
| Infisical                 |      x       |              |                      |            x            |        x         |             |                             |
| Device42                  |              |              |                      |                         |        x         |             |                             |
| Bitwarden Secrets Manager |      x       |              |                      |                         |        x         |      x      |              x              |
| Previder                  |      x       |              |                      |                         |        x         |             |                             |
| Cloud.ru                  |      x       |      x       |                      |            x            |        x         |             |              x              |

## Support Policy

We provide technical support and security / bug fixes for the above listed versions.

### Technical support

We provide assistance for deploying/upgrading etc. on a best-effort basis. You can request support through the following channels:

- [Kubernetes Slack
  #external-secrets](https://kubernetes.slack.com/messages/external-secrets)
- GitHub [Issues](https://github.com/external-secrets/external-secrets/issues)
- GitHub [Discussions](https://github.com/external-secrets/external-secrets/discussions)

Even though we have active maintainers and people assigned to this project, we kindly ask for patience when asking for support. We will try to get to priority issues as fast as possible, but there may be some delays.
