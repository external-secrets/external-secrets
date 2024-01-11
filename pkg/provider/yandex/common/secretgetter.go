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

package common

import (
	"context"
)

// Adapts the secrets received from a remote Yandex.Cloud service for the format expected by v1beta1.SecretsClient.
type SecretGetter interface {
	GetSecret(ctx context.Context, iamToken, resourceID, versionID, property string) ([]byte, error)
	GetSecretMap(ctx context.Context, iamToken, resourceID, versionID string) (map[string][]byte, error)
}
