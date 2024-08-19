---
hide:
  - toc
---

This page lists the status, timeline and policy for currently supported ESO releases and its providers. Please also see our [deprecation policy](deprecation-policy.md) that describes API versioning, deprecation and API surface.

## Supported Versions

We want to provide security patches and critical bug fixes in a timely manner to our users.
To do so, we offer long-term support for our latest two (N, N-1) software releases.
We aim for a 2-3 month minor release cycle, i.e. a given release is supported for about 4-6 months.

We want to cover the following cases:

- regular image rebuilds to update OS dependencies
- regular go dependency updates
- backport bug fixes on demand

| ESO Version | Kubernetes Version | Release Date | End of Life    |
|-------------|--------------------|--------------| -------------- |
| 0.10.x      | 1.19 → 1.31        | Aug 3, 2024  | Release of 1.1 |
| 0.9.x       | 1.19 → 1.30        | Jun 22, 2023 | Release of 1.1 |
| 0.8.x       | 1.19 → 1.28        | Mar 16, 2023 | Release of 1.0 |
| 0.7.x       | 1.19 → 1.26        | Dec 11, 2022 | Jun 22, 2023   |
| 0.6.x       | 1.19 → 1.24        | Oct 9, 2022  | Mar 16, 2023   |
| 0.5.x       | 1.19 → 1.24        | Apr 6, 2022  | Dec 11, 2022   |
| 0.4.x       | 1.16 → 1.24        | Feb 2, 2022  | Oct 9, 2022    |
| 0.3.x       | 1.16 → 1.24        | Jul 25, 2021 | Apr 6, 2022    |

## Provider Stability and Support Level

The following table describes the stability level of each provider and who's responsible.

| Provider                                                                                                   | Stability |                                                                                                                                                                              Maintainer |
|------------------------------------------------------------------------------------------------------------| :-------: | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------: |
| [AWS Secrets Manager](https://external-secrets.io/latest/provider/aws-secrets-manager/)                    |  stable   |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [AWS Parameter Store](https://external-secrets.io/latest/provider/aws-parameter-store/)                    |  stable   |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [Hashicorp Vault](https://external-secrets.io/latest/provider/hashicorp-vault/)                            |  stable   |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [GCP Secret Manager](https://external-secrets.io/latest/provider/google-secrets-manager/)                  |  stable   |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [Azure Keyvault](https://external-secrets.io/latest/provider/azure-key-vault/)                             |  stable   |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [IBM Cloud Secrets Manager](https://external-secrets.io/latest/provider/ibm-secrets-manager/)              |  stable   | [@knelasevero](https://github.com/knelasevero) [@sebagomez](https://github.com/sebagomez) [@ricardoptcosta](https://github.com/ricardoptcosta) [@IdanAdar](https://github.com/IdanAdar) |
| [Kubernetes](https://external-secrets.io/latest/provider/kubernetes)                                       |   beta    |                                                                                                                                 [external-secrets](https://github.com/external-secrets) |
| [Yandex Lockbox](https://external-secrets.io/latest/provider/yandex-lockbox/)                              |   alpha   |                                                                                     [@AndreyZamyslov](https://github.com/AndreyZamyslov) [@knelasevero](https://github.com/knelasevero) |
| [GitLab Variables](https://external-secrets.io/latest/provider/gitlab-variables/)                          |   alpha   |                                                                                                                                                  [@Jabray5](https://github.com/Jabray5) |
| Alibaba Cloud KMS                                                                                          |   alpha   |                                                                                                                                          [@ElsaChelala](https://github.com/ElsaChelala) |
| [Oracle Vault](https://external-secrets.io/latest/provider/oracle-vault)                                   |   alpha   |                                                                                                 [@KianTigger](https://github.com/KianTigger) [@EladGabay](https://github.com/EladGabay) |
| [Akeyless](https://external-secrets.io/latest/provider/akeyless)                                           |   alpha   |                                                                                                                                    [@renanaAkeyless](https://github.com/renanaAkeyless) |
| [1Password](https://external-secrets.io/latest/provider/1password-automation)                              |   alpha   |                                                                                       [@SimSpaceCorp](https://github.com/Simspace) [@snarlysodboxer](https://github.com/snarlysodboxer) |
| [Generic Webhook](https://external-secrets.io/latest/provider/webhook)                                     |   alpha   |                                                                                                                                                  [@willemm](https://github.com/willemm) |
| [senhasegura DevOps Secrets Management (DSM)](https://external-secrets.io/latest/provider/senhasegura-dsm) |   alpha   |                                                                                                                                                    [@lfraga](https://github.com/lfraga) |
| [Doppler SecretOps Platform](https://external-secrets.io/latest/provider/doppler)                          |   alpha   |                                                                                         [@ryan-blunden](https://github.com/ryan-blunden/) [@nmanoogian](https://github.com/nmanoogian/) |
| [Keeper Security](https://www.keepersecurity.com/)                                                         |   alpha   |                                                                                                                                              [@ppodevlab](https://github.com/ppodevlab) |
| [Scaleway](https://external-secrets.io/latest/provider/scaleway)                                           |   alpha   |                                                                                                                                                   [@azert9](https://github.com/azert9/) |
| [Conjur](https://external-secrets.io/latest/provider/conjur)                                               |   stable   |                                                                                                                                 [@davidh-cyberark](https://github.com/davidh-cyberark/) [@szh](https://github.com/szh) |
| [Delinea](https://external-secrets.io/latest/provider/delinea)                                             |   alpha   |                                                                                                                                     [@michaelsauter](https://github.com/michaelsauter/) |
| [Beyondtrust](https://external-secrets.io/latest/provider/beyondtrust)                                     |   alpha   |                                                                                                                                       [@btfhernandez](https://github.com/btfhernandez/) |
| [SecretServer](https://external-secrets.io/latest/provider/secretserver)                                   |   alpha   |                                                                                                                                     [@billhamilton](https://github.com/pacificcode/) |
| [Pulumi ESC](https://external-secrets.io/latest/provider/pulumi)                                           |   alpha   |                                                                                                                                                  [@dirien](https://github.com/dirien) |
| [Passbolt](https://external-secrets.io/latest/provider/passbolt)                                           |   alpha   |                                                                                                                                                   |
| [Infisical](https://external-secrets.io/latest/provider/infisical)                                         |   alpha   | [@akhilmhdh](https://github.com/akhilmhdh)                                                                                       |
| [Device42](https://external-secrets.io/latest/provider/device42)                                           |   alpha   |                                                                                                                                                   |
| [Bitwarden Secrets Manager](https://external-secrets.io/latest/provider/bitwarden-secrets-manager)         |   alpha   | [@skarlso](https://github.com/Skarlso)                                                                                           |

## Provider Feature Support

The following table show the support for features across different providers.

| Provider                  | find by name | find by tags | metadataPolicy Fetch | referent authentication | store validation | push secret | DeletionPolicy Merge/Delete |
|---------------------------|:------------:| :----------: | :------------------: | :---------------------: | :--------------: |:-----------:|:---------------------------:|
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
| Akeyless                  |      x       |      x       |                      |                         |        x         |             |                             |
| 1Password                 |      x       |              |                      |                         |        x         |      x      |              x              |
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

## Support Policy

We provide technical support and security / bug fixes for the above listed versions.

### Technical support

We provide assistance for deploying/upgrading etc. on a best-effort basis. You can request support through the following channels:

- [Kubernetes Slack
  #external-secrets](https://kubernetes.slack.com/messages/external-secrets)
- GitHub [Issues](https://github.com/external-secrets/external-secrets/issues)
- GitHub [Discussions](https://github.com/external-secrets/external-secrets/discussions)

Even though we have active maintainers and people assigned to this project, we kindly ask for patience when asking for support. We will try to get to priority issues as fast as possible, but there may be some delays.
