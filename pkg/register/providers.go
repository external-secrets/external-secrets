/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package register provides explicit registration of all providers and generators.
package register

import (
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"

	akeyless "github.com/external-secrets/external-secrets/providers/v1/akeyless"
	alibaba "github.com/external-secrets/external-secrets/providers/v1/alibaba"
	aws "github.com/external-secrets/external-secrets/providers/v1/aws"
	azure "github.com/external-secrets/external-secrets/providers/v1/azure/keyvault"
	beyondtrust "github.com/external-secrets/external-secrets/providers/v1/beyondtrust"
	bitwarden "github.com/external-secrets/external-secrets/providers/v1/bitwarden"
	chef "github.com/external-secrets/external-secrets/providers/v1/chef"
	cloudru "github.com/external-secrets/external-secrets/providers/v1/cloudru/secretmanager"
	conjur "github.com/external-secrets/external-secrets/providers/v1/conjur"
	delinea "github.com/external-secrets/external-secrets/providers/v1/delinea"
	device42 "github.com/external-secrets/external-secrets/providers/v1/device42"
	doppler "github.com/external-secrets/external-secrets/providers/v1/doppler"
	fake "github.com/external-secrets/external-secrets/providers/v1/fake"
	fortanix "github.com/external-secrets/external-secrets/providers/v1/fortanix"
	gcp "github.com/external-secrets/external-secrets/providers/v1/gcp/secretmanager"
	github "github.com/external-secrets/external-secrets/providers/v1/github"
	gitlab "github.com/external-secrets/external-secrets/providers/v1/gitlab"
	ibm "github.com/external-secrets/external-secrets/providers/v1/ibm"
	infisical "github.com/external-secrets/external-secrets/providers/v1/infisical"
	keepersecurity "github.com/external-secrets/external-secrets/providers/v1/keepersecurity"
	kubernetes "github.com/external-secrets/external-secrets/providers/v1/kubernetes"
	ngrok "github.com/external-secrets/external-secrets/providers/v1/ngrok"
	onboardbase "github.com/external-secrets/external-secrets/providers/v1/onboardbase"
	onepassword "github.com/external-secrets/external-secrets/providers/v1/onepassword"
	onepasswordsdk "github.com/external-secrets/external-secrets/providers/v1/onepasswordsdk"
	oracle "github.com/external-secrets/external-secrets/providers/v1/oracle"
	passbolt "github.com/external-secrets/external-secrets/providers/v1/passbolt"
	passworddepot "github.com/external-secrets/external-secrets/providers/v1/passworddepot"
	previder "github.com/external-secrets/external-secrets/providers/v1/previder"
	pulumi "github.com/external-secrets/external-secrets/providers/v1/pulumi"
	scaleway "github.com/external-secrets/external-secrets/providers/v1/scaleway"
	secretserver "github.com/external-secrets/external-secrets/providers/v1/secretserver"
	senhasegura "github.com/external-secrets/external-secrets/providers/v1/senhasegura"
	vault "github.com/external-secrets/external-secrets/providers/v1/vault"
	volcengine "github.com/external-secrets/external-secrets/providers/v1/volcengine"
	webhook "github.com/external-secrets/external-secrets/providers/v1/webhook"
	yandexcert "github.com/external-secrets/external-secrets/providers/v1/yandex/certificatemanager"
	yandexlock "github.com/external-secrets/external-secrets/providers/v1/yandex/lockbox"
)

func init() {
	// Register all providers
	esv1.Register(akeyless.NewProvider(), akeyless.ProviderSpec(), akeyless.MaintenanceStatus())
	esv1.Register(alibaba.NewProvider(), alibaba.ProviderSpec(), alibaba.MaintenanceStatus())
	esv1.Register(aws.NewProvider(), aws.ProviderSpec(), aws.MaintenanceStatus())
	esv1.Register(azure.NewProvider(), azure.ProviderSpec(), azure.MaintenanceStatus())
	esv1.Register(beyondtrust.NewProvider(), beyondtrust.ProviderSpec(), beyondtrust.MaintenanceStatus())
	esv1.Register(bitwarden.NewProvider(), bitwarden.ProviderSpec(), bitwarden.MaintenanceStatus())
	esv1.Register(chef.NewProvider(), chef.ProviderSpec(), chef.MaintenanceStatus())
	esv1.Register(cloudru.NewProvider(), cloudru.ProviderSpec(), cloudru.MaintenanceStatus())
	esv1.Register(conjur.NewProvider(), conjur.ProviderSpec(), conjur.MaintenanceStatus())
	esv1.Register(delinea.NewProvider(), delinea.ProviderSpec(), delinea.MaintenanceStatus())
	esv1.Register(device42.NewProvider(), device42.ProviderSpec(), device42.MaintenanceStatus())
	esv1.Register(doppler.NewProvider(), doppler.ProviderSpec(), doppler.MaintenanceStatus())
	esv1.Register(fake.NewProvider(), fake.ProviderSpec(), fake.MaintenanceStatus())
	esv1.Register(fortanix.NewProvider(), fortanix.ProviderSpec(), fortanix.MaintenanceStatus())
	esv1.Register(gcp.NewProvider(), gcp.ProviderSpec(), gcp.MaintenanceStatus())
	esv1.Register(github.NewProvider(), github.ProviderSpec(), github.MaintenanceStatus())
	esv1.Register(gitlab.NewProvider(), gitlab.ProviderSpec(), gitlab.MaintenanceStatus())
	esv1.Register(ibm.NewProvider(), ibm.ProviderSpec(), ibm.MaintenanceStatus())
	esv1.Register(infisical.NewProvider(), infisical.ProviderSpec(), infisical.MaintenanceStatus())
	esv1.Register(keepersecurity.NewProvider(), keepersecurity.ProviderSpec(), keepersecurity.MaintenanceStatus())
	esv1.Register(kubernetes.NewProvider(), kubernetes.ProviderSpec(), kubernetes.MaintenanceStatus())
	esv1.Register(ngrok.NewProvider(), ngrok.ProviderSpec(), ngrok.MaintenanceStatus())
	esv1.Register(onboardbase.NewProvider(), onboardbase.ProviderSpec(), onboardbase.MaintenanceStatus())
	esv1.Register(onepassword.NewProvider(), onepassword.ProviderSpec(), onepassword.MaintenanceStatus())
	esv1.Register(onepasswordsdk.NewProvider(), onepasswordsdk.ProviderSpec(), onepasswordsdk.MaintenanceStatus())
	esv1.Register(oracle.NewProvider(), oracle.ProviderSpec(), oracle.MaintenanceStatus())
	esv1.Register(passbolt.NewProvider(), passbolt.ProviderSpec(), passbolt.MaintenanceStatus())
	esv1.Register(passworddepot.NewProvider(), passworddepot.ProviderSpec(), passworddepot.MaintenanceStatus())
	esv1.Register(previder.NewProvider(), previder.ProviderSpec(), previder.MaintenanceStatus())
	esv1.Register(pulumi.NewProvider(), pulumi.ProviderSpec(), pulumi.MaintenanceStatus())
	esv1.Register(scaleway.NewProvider(), scaleway.ProviderSpec(), scaleway.MaintenanceStatus())
	esv1.Register(secretserver.NewProvider(), secretserver.ProviderSpec(), secretserver.MaintenanceStatus())
	esv1.Register(senhasegura.NewProvider(), senhasegura.ProviderSpec(), senhasegura.MaintenanceStatus())
	esv1.Register(vault.NewProvider(), vault.ProviderSpec(), vault.MaintenanceStatus())
	esv1.Register(volcengine.NewProvider(), volcengine.ProviderSpec(), volcengine.MaintenanceStatus())
	esv1.Register(webhook.NewProvider(), webhook.ProviderSpec(), webhook.MaintenanceStatus())
	esv1.Register(yandexcert.NewProvider(), yandexcert.ProviderSpec(), yandexcert.MaintenanceStatus())
	esv1.Register(yandexlock.NewProvider(), yandexlock.ProviderSpec(), yandexlock.MaintenanceStatus())
}
