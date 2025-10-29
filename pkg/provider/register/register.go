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

// Package register imports all provider implementations to register them in the controller schema.
package register

import (
	// To ensure all providers are registered, we import them here.
	_ "github.com/external-secrets/external-secrets/pkg/provider/akeyless"
	_ "github.com/external-secrets/external-secrets/pkg/provider/alibaba"
	_ "github.com/external-secrets/external-secrets/pkg/provider/aws"
	_ "github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/beyondtrust"
	_ "github.com/external-secrets/external-secrets/pkg/provider/bitwarden"
	_ "github.com/external-secrets/external-secrets/pkg/provider/chef"
	_ "github.com/external-secrets/external-secrets/pkg/provider/cloudru/secretmanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/conjur"
	_ "github.com/external-secrets/external-secrets/pkg/provider/delinea"
	_ "github.com/external-secrets/external-secrets/pkg/provider/device42"
	_ "github.com/external-secrets/external-secrets/pkg/provider/doppler"
	_ "github.com/external-secrets/external-secrets/pkg/provider/fake"
	_ "github.com/external-secrets/external-secrets/pkg/provider/fortanix"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/github"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gitlab"
	_ "github.com/external-secrets/external-secrets/pkg/provider/ibm"
	_ "github.com/external-secrets/external-secrets/pkg/provider/infisical"
	_ "github.com/external-secrets/external-secrets/pkg/provider/keepersecurity"
	_ "github.com/external-secrets/external-secrets/pkg/provider/kubernetes"
	_ "github.com/external-secrets/external-secrets/pkg/provider/ngrok"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onboardbase"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onepassword"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onepasswordsdk"
	_ "github.com/external-secrets/external-secrets/pkg/provider/oracle"
	_ "github.com/external-secrets/external-secrets/pkg/provider/ovh"
	_ "github.com/external-secrets/external-secrets/pkg/provider/passbolt"
	_ "github.com/external-secrets/external-secrets/pkg/provider/passworddepot"
	_ "github.com/external-secrets/external-secrets/pkg/provider/previder"
	_ "github.com/external-secrets/external-secrets/pkg/provider/pulumi"
	_ "github.com/external-secrets/external-secrets/pkg/provider/scaleway"
	_ "github.com/external-secrets/external-secrets/pkg/provider/secretserver"
	_ "github.com/external-secrets/external-secrets/pkg/provider/senhasegura"
	_ "github.com/external-secrets/external-secrets/pkg/provider/vault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/volcengine"
	_ "github.com/external-secrets/external-secrets/pkg/provider/webhook"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/certificatemanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox"
)
