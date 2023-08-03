<p align="center">
    <img src="assets/eso-logo-large.png" width="30%" align="center" alt="external-secrets">
</p>

# External Secrets

![ci](https://github.com/external-secrets/external-secrets/actions/workflows/ci.yml/badge.svg?branch=main)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/5327/badge)](https://bestpractices.coreinfrastructure.org/projects/5947)
[![Go Report Card](https://goreportcard.com/badge/github.com/external-secrets/external-secrets)](https://goreportcard.com/report/github.com/external-secrets/external-secrets)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fexternal-secrets%2Fexternal-secrets.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fexternal-secrets%2Fexternal-secrets?ref=badge_shield)
<a href="https://artifacthub.io/packages/helm/external-secrets-operator/external-secrets"><img alt="Artifact Hub" src="https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/external-secrets" /></a>
<a href="https://operatorhub.io/operator/external-secrets-operator"><img alt="operatorhub.io" src="https://img.shields.io/badge/operatorhub.io-external--secrets-brightgreen" /></a>

**External Secrets Operator** is a Kubernetes operator that integrates external
secret management systems like [AWS Secrets
Manager](https://aws.amazon.com/secrets-manager/), [HashiCorp
Vault](https://www.vaultproject.io/), [Google Secrets
Manager](https://cloud.google.com/secret-manager), [Azure Key
Vault](https://azure.microsoft.com/en-us/services/key-vault/), [IBM Cloud Secrets Manager](https://www.ibm.com/cloud/secrets-manager), [Akeyless](https://akeyless.io), [CyberArk Conjur](https://www.conjur.org) and many more. The
operator reads information from external APIs and automatically injects the
values into a [Kubernetes
Secret](https://kubernetes.io/docs/concepts/configuration/secret/).

Multiple people and organizations are joining efforts to create a single External Secrets solution based on existing projects. If you are curious about the origins of this project, check out [this issue](https://github.com/external-secrets/kubernetes-external-secrets/issues/47) and [this PR](https://github.com/external-secrets/kubernetes-external-secrets/pull/477).

## Documentation

External Secrets Operator guides and reference documentation is available at [external-secrets.io](https://external-secrets.io). Also see our [stability and support](https://external-secrets.io/main/introduction/stability-support/) policy.

## Contributing

We welcome and encourage contributions to this project! Please read the [Developer](https://www.external-secrets.io/main/contributing/devguide/) and [Contribution process](https://www.external-secrets.io/main/contributing/process/) guides. Also make sure to check the [Code of Conduct](https://www.external-secrets.io/main/contributing/coc/) and adhere to its guidelines.

### Sponsoring

Please consider sponsoring this project, there are many ways you can help us with: engineering time, providing infrastructure, donating money, etc. We are open to cooperations, feel free to approach as and we discuss how this could look like. We can keep your contribution anonymized if that's required (depending on the type of contribution), and anonymous donations are possible inside [Opencollective](https://opencollective.com/external-secrets-org).

## Bi-weekly Development Meeting

We host our development meeting every odd wednesday at [8:00 PM Berlin Time](https://dateful.com/time-zone-converter?t=20:00&tz=Europe/Berlin) on [Jitsi](https://meet.jit.si/eso-community-meeting). Meeting notes are recorded on [hackmd](https://hackmd.io/GSGEpTVdRZCP6LDxV3FHJA).

Anyone is welcome to join. Feel free to ask questions, request feedback, raise awareness for an issue, or just say hi. ;)

## Security

Please report vulnerabilities by email to contact@external-secrets.io (or cncf-ExternalSecretsOp-maintainers@lists.cncf.io). Also see our [SECURITY.md file](SECURITY.md) for details.

## software bill of materials
We attach SBOM and provenance file to our GitHub release. Also, they are attached to container images.

## Adopters

Please create a PR and add your company or project to our [ADOPTERS.md file](ADOPTERS.md) if you are using our project!

## Roadmap

You can find the roadmap in our documentation: https://external-secrets.io/main/contributing/roadmap/

## Kicked off by

![](assets/Godaddylogo_2020.png)

## Sponsored by

![](assets/CS_logo_1.png)
![](assets/form3_logo.png)
![](assets/pento_logo.png)


## License
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fexternal-secrets%2Fexternal-secrets.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fexternal-secrets%2Fexternal-secrets?ref=badge_large)
