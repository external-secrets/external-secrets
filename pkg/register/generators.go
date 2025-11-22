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

package register

import (
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"

	acr "github.com/external-secrets/external-secrets/generators/v1/acr"
	cloudsmith "github.com/external-secrets/external-secrets/generators/v1/cloudsmith"
	ecr "github.com/external-secrets/external-secrets/generators/v1/ecr"
	fakegen "github.com/external-secrets/external-secrets/generators/v1/fake"
	gcr "github.com/external-secrets/external-secrets/generators/v1/gcr"
	githubgen "github.com/external-secrets/external-secrets/generators/v1/github"
	grafana "github.com/external-secrets/external-secrets/generators/v1/grafana"
	mfa "github.com/external-secrets/external-secrets/generators/v1/mfa"
	password "github.com/external-secrets/external-secrets/generators/v1/password"
	quay "github.com/external-secrets/external-secrets/generators/v1/quay"
	sshkey "github.com/external-secrets/external-secrets/generators/v1/sshkey"
	sts "github.com/external-secrets/external-secrets/generators/v1/sts"
	uuid "github.com/external-secrets/external-secrets/generators/v1/uuid"
	vaultgen "github.com/external-secrets/external-secrets/generators/v1/vault"
	webhookgen "github.com/external-secrets/external-secrets/generators/v1/webhook"
)

func init() {
	// Register all generators
	genv1alpha1.Register(acr.Kind(), acr.NewGenerator())
	genv1alpha1.RegisterGeneric(acr.Kind(), &genv1alpha1.ACRAccessToken{})

	genv1alpha1.Register(cloudsmith.Kind(), cloudsmith.NewGenerator())
	genv1alpha1.RegisterGeneric(cloudsmith.Kind(), &genv1alpha1.CloudsmithAccessToken{})

	genv1alpha1.Register(ecr.Kind(), ecr.NewGenerator())
	genv1alpha1.RegisterGeneric(ecr.Kind(), &genv1alpha1.ECRAuthorizationToken{})

	genv1alpha1.Register(fakegen.Kind(), fakegen.NewGenerator())
	genv1alpha1.RegisterGeneric(fakegen.Kind(), &genv1alpha1.Fake{})

	genv1alpha1.Register(gcr.Kind(), gcr.NewGenerator())
	genv1alpha1.RegisterGeneric(gcr.Kind(), &genv1alpha1.GCRAccessToken{})

	genv1alpha1.Register(githubgen.Kind(), githubgen.NewGenerator())
	genv1alpha1.RegisterGeneric(githubgen.Kind(), &genv1alpha1.GithubAccessToken{})

	genv1alpha1.Register(grafana.Kind(), grafana.NewGenerator())
	genv1alpha1.RegisterGeneric(grafana.Kind(), &genv1alpha1.Grafana{})

	genv1alpha1.Register(mfa.Kind(), mfa.NewGenerator())
	genv1alpha1.RegisterGeneric(mfa.Kind(), &genv1alpha1.MFA{})

	genv1alpha1.Register(password.Kind(), password.NewGenerator())
	genv1alpha1.RegisterGeneric(password.Kind(), &genv1alpha1.Password{})

	genv1alpha1.Register(quay.Kind(), quay.NewGenerator())
	genv1alpha1.RegisterGeneric(quay.Kind(), &genv1alpha1.QuayAccessToken{})

	genv1alpha1.Register(sts.Kind(), sts.NewGenerator())
	genv1alpha1.RegisterGeneric(sts.Kind(), &genv1alpha1.STSSessionToken{})

	genv1alpha1.Register(vaultgen.Kind(), vaultgen.NewGenerator())
	genv1alpha1.RegisterGeneric(vaultgen.Kind(), &genv1alpha1.VaultDynamicSecret{})

	genv1alpha1.Register(sshkey.Kind(), sshkey.NewGenerator())
	genv1alpha1.RegisterGeneric(sshkey.Kind(), &genv1alpha1.SSHKey{})

	genv1alpha1.Register(uuid.Kind(), uuid.NewGenerator())
	genv1alpha1.RegisterGeneric(uuid.Kind(), &genv1alpha1.UUID{})

	genv1alpha1.Register(webhookgen.Kind(), webhookgen.NewGenerator())
	genv1alpha1.RegisterGeneric(webhookgen.Kind(), &genv1alpha1.Webhook{})
}
