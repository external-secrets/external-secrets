//Copyright External Secrets Inc. All Rights Reserved

package register

// packages imported here are registered to the controller schema.

import (
	_ "github.com/external-secrets/external-secrets/pkg/provider/akeyless"
	_ "github.com/external-secrets/external-secrets/pkg/provider/alibaba"
	_ "github.com/external-secrets/external-secrets/pkg/provider/aws"
	_ "github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/beyondtrust"
	_ "github.com/external-secrets/external-secrets/pkg/provider/bitwarden"
	_ "github.com/external-secrets/external-secrets/pkg/provider/chef"
	_ "github.com/external-secrets/external-secrets/pkg/provider/conjur"
	_ "github.com/external-secrets/external-secrets/pkg/provider/delinea"
	_ "github.com/external-secrets/external-secrets/pkg/provider/device42"
	_ "github.com/external-secrets/external-secrets/pkg/provider/doppler"
	_ "github.com/external-secrets/external-secrets/pkg/provider/fake"
	_ "github.com/external-secrets/external-secrets/pkg/provider/fortanix"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gitlab"
	_ "github.com/external-secrets/external-secrets/pkg/provider/ibm"
	_ "github.com/external-secrets/external-secrets/pkg/provider/infisical"
	_ "github.com/external-secrets/external-secrets/pkg/provider/keepersecurity"
	_ "github.com/external-secrets/external-secrets/pkg/provider/kubernetes"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onboardbase"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onepassword"
	_ "github.com/external-secrets/external-secrets/pkg/provider/oracle"
	_ "github.com/external-secrets/external-secrets/pkg/provider/passbolt"
	_ "github.com/external-secrets/external-secrets/pkg/provider/passworddepot"
	_ "github.com/external-secrets/external-secrets/pkg/provider/pulumi"
	_ "github.com/external-secrets/external-secrets/pkg/provider/scaleway"
	_ "github.com/external-secrets/external-secrets/pkg/provider/secretserver"
	_ "github.com/external-secrets/external-secrets/pkg/provider/senhasegura"
	_ "github.com/external-secrets/external-secrets/pkg/provider/vault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/webhook"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/certificatemanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox"
)
