//Copyright External Secrets Inc. All Rights Reserved

package suite

import (

	// import different e2e test suites.
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws/parameterstore"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws/secretsmanager"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/azure"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/conjur"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/delinea"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/gcp"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/kubernetes"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/scaleway"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/secretserver"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/template"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/vault"
)
