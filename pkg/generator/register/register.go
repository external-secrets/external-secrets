//Copyright External Secrets Inc. All Rights Reserved

package register

// packages imported here are registered to the controller schema.

import (
	_ "github.com/external-secrets/external-secrets/pkg/generator/acr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/aws/ecr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/aws/iam"
	_ "github.com/external-secrets/external-secrets/pkg/generator/fake"
	_ "github.com/external-secrets/external-secrets/pkg/generator/gcr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/github"
	_ "github.com/external-secrets/external-secrets/pkg/generator/password"
	_ "github.com/external-secrets/external-secrets/pkg/generator/vault"
	_ "github.com/external-secrets/external-secrets/pkg/generator/webhook"
)
