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

import (
	_ "github.com/external-secrets/external-secrets/pkg/generator/acr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/ecr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/fake"
	_ "github.com/external-secrets/external-secrets/pkg/generator/gcr"
	_ "github.com/external-secrets/external-secrets/pkg/generator/github"
	_ "github.com/external-secrets/external-secrets/pkg/generator/password"
	_ "github.com/external-secrets/external-secrets/pkg/generator/vault"
	_ "github.com/external-secrets/external-secrets/pkg/generator/webhook"
)
