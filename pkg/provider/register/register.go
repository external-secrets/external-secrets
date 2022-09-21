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

package register

// packages imported here are registered to the controller schema.
// nolint:revive
import (
	_ "github.com/external-secrets/external-secrets/pkg/provider/akeyless"
	_ "github.com/external-secrets/external-secrets/pkg/provider/alibaba"
	_ "github.com/external-secrets/external-secrets/pkg/provider/aws"
	_ "github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/fake"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gcp/secretmanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/gitlab"
	_ "github.com/external-secrets/external-secrets/pkg/provider/ibm"
	_ "github.com/external-secrets/external-secrets/pkg/provider/kubernetes"
	_ "github.com/external-secrets/external-secrets/pkg/provider/onepassword"
	_ "github.com/external-secrets/external-secrets/pkg/provider/oracle"
	_ "github.com/external-secrets/external-secrets/pkg/provider/senhasegura"
	_ "github.com/external-secrets/external-secrets/pkg/provider/vault"
	_ "github.com/external-secrets/external-secrets/pkg/provider/webhook"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/certificatemanager"
	_ "github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox"
	_ "github.com/external-secrets/external-secrets/pkg/provider/conjur"
)
