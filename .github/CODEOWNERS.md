# External Secrets CODEOWNERS
# This file maps repository paths to GitHub teams for review.
# Syntax: https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/about-code-owners

# --- CI / Infrastructure ---
.github/                      @external-secrets/ci-reviewers
scripts/                      @external-secrets/ci-reviewers
build/                        @external-secrets/ci-reviewers
hack/                         @external-secrets/ci-reviewers

# --- Testing ---
test/                         @external-secrets/testing-reviewers
e2e/                          @external-secrets/testing-reviewers
tests/                        @external-secrets/testing-reviewers
hack/                         @external-secrets/testing-reviewers

# --- Core Controllers ---
apis/                         @external-secrets/core-reviewers
pkg/controllers/              @external-secrets/core-reviewers

# --- Providers ---
pkg/provider/                 @external-secrets/providers-reviewers
pkg/provider/akeyless/        @external-secrets/provider-akeyless-reviewers
pkg/provider/alibaba/         @external-secrets/provider-alibaba-reviewers
pkg/provider/aws/             @external-secrets/provider-aws-reviewers
pkg/provider/azure/           @external-secrets/provider-azure-reviewers
pkg/provider/beyondtrust/     @external-secrets/provider-beyondtrust-reviewers
pkg/provider/bitwarden/       @external-secrets/provider-bitwarden-reviewers
pkg/provider/chef/            @external-secrets/provider-chef-reviewers
pkg/provider/cloudru/         @external-secrets/provider-cloudru-reviewers
pkg/provider/conjur/          @external-secrets/provider-conjur-reviewers
pkg/provider/delinea/         @external-secrets/provider-delinea-reviewers
pkg/provider/device42/        @external-secrets/provider-device42-reviewers
pkg/provider/doppler/         @external-secrets/provider-doppler-reviewers
pkg/provider/fake/            @external-secrets/provider-fake-reviewers
pkg/provider/fortanix/        @external-secrets/provider-fortanix-reviewers
pkg/provider/gcp/             @external-secrets/provider-gcp-reviewers
pkg/provider/github/          @external-secrets/provider-github-reviewers
pkg/provider/gitlab/          @external-secrets/provider-gitlab-reviewers
pkg/provider/ibm/             @external-secrets/provider-ibm-reviewers
pkg/provider/infisical/       @external-secrets/provider-infisical-reviewers
pkg/provider/keepersecurity/  @external-secrets/provider-keepersecurity-reviewers
pkg/provider/kubernetes/      @external-secrets/provider-kubernetes-reviewers
pkg/provider/onboardbase/     @external-secrets/provider-onboardbase-reviewers
pkg/provider/onepassword/     @external-secrets/provider-onepassword-reviewers
pkg/provider/onepasswordsdk/  @external-secrets/provider-onepasswordsdk-reviewers
pkg/provider/oracle/          @external-secrets/provider-oracle-reviewers
pkg/provider/ovh/             @external-secrets/provider-ovh-reviewers
pkg/provider/passbolt/        @external-secrets/provider-passbolt-reviewers
pkg/provider/passworddepot/   @external-secrets/provider-passworddepot-reviewers
pkg/provider/previder/        @external-secrets/provider-previder-reviewers
pkg/provider/pulumi/          @external-secrets/provider-pulumi-reviewers
pkg/provider/scaleway/        @external-secrets/provider-scaleway-reviewers
pkg/provider/secretserver/    @external-secrets/provider-secretserver-reviewers
pkg/provider/senhasegura/     @external-secrets/provider-senhasegura-reviewers
pkg/provider/vault/           @external-secrets/provider-vault-reviewers
pkg/provider/webhook/         @external-secrets/provider-webhook-reviewers
pkg/provider/yandex/          @external-secrets/provider-yandex-reviewers


# --- Maintainers (project-wide) ---
*                            @external-secrets/maintainers @external-secrets/interim-maintainers
