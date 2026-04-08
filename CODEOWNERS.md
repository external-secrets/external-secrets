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
providers/v1/                         @external-secrets/providers-reviewers
providers/v1/akeyless/                @external-secrets/provider-akeyless-reviewers
providers/v1/aws/                     @external-secrets/provider-aws-reviewers
providers/v1/azure/                   @external-secrets/provider-azure-reviewers
providers/v1/barbican/                @external-secrets/provider-barbican-reviewers
providers/v1/beyondtrust/             @external-secrets/provider-beyondtrust-reviewers
providers/v1/bitwarden/               @external-secrets/provider-bitwarden-reviewers
providers/v1/chef/                    @external-secrets/provider-chef-reviewers
providers/v1/cloudru/                 @external-secrets/provider-cloudru-reviewers
providers/v1/conjur/                  @external-secrets/provider-conjur-reviewers
providers/v1/delinea/                 @external-secrets/provider-delinea-reviewers
providers/v1/doppler/                 @external-secrets/provider-doppler-reviewers
providers/v1/dvls/                    @external-secrets/provider-dvls-reviewers
providers/v1/fake/                    @external-secrets/provider-fake-reviewers
providers/v1/fortanix/                @external-secrets/provider-fortanix-reviewers
providers/v1/gcp/                     @external-secrets/provider-gcp-reviewers
providers/v1/github/                  @external-secrets/provider-github-reviewers
providers/v1/gitlab/                  @external-secrets/provider-gitlab-reviewers
providers/v1/ibm/                     @external-secrets/provider-ibm-reviewers
providers/v1/infisical/               @external-secrets/provider-infisical-reviewers
providers/v1/keepersecurity/          @external-secrets/provider-keepersecurity-reviewers
providers/v1/kubernetes/              @external-secrets/provider-kubernetes-reviewers
providers/v1/nebius/                  @external-secrets/provider-nebius-reviewers
providers/v1/ngrok/                   @external-secrets/provider-ngrok-reviewers
providers/v1/onboardbase/             @external-secrets/provider-onboardbase-reviewers
providers/v1/onepassword/             @external-secrets/provider-onepassword-reviewers
providers/v1/onepasswordsdk/          @external-secrets/provider-onepasswordsdk-reviewers
providers/v1/oracle/                  @external-secrets/provider-oracle-reviewers
pkg/provider/v1/ovh/                  @external-secrets/provider-ovh-reviewers
providers/v1/passbolt/                @external-secrets/provider-passbolt-reviewers
providers/v1/passworddepot/           @external-secrets/provider-passworddepot-reviewers
providers/v1/previder/                @external-secrets/provider-previder-reviewers
providers/v1/pulumi/                  @external-secrets/provider-pulumi-reviewers
providers/v1/scaleway/                @external-secrets/provider-scaleway-reviewers
providers/v1/secretserver/            @external-secrets/provider-secretserver-reviewers
providers/v1/senhasegura/             @external-secrets/provider-senhasegura-reviewers
providers/v1/vault/                   @external-secrets/provider-vault-reviewers
providers/v1/volcengine/              @external-secrets/provider-volcengine-reviewers
providers/v1/webhook/                 @external-secrets/provider-webhook-reviewers
providers/v1/yandex/                  @external-secrets/provider-yandex-reviewers


# --- Maintainers (project-wide) ---
*                            @external-secrets/maintainers @external-secrets/interim-maintainers
