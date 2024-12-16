/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package suite

import (

	// import different e2e test suites.
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws/parameterstore"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/aws/secretsmanager"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/azure"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/delinea"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/gcp"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/kubernetes"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/scaleway"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/template"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/vault"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/conjur"
	_ "github.com/external-secrets/external-secrets-e2e/suites/provider/cases/secretserver"
)
