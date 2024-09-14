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

package vault

import (
	"context"

	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

func setSecretKeyToken(ctx context.Context, v *client) (bool, error) {
	tokenRef := v.store.Auth.TokenSecretRef
	if tokenRef != nil {
		token, err := resolvers.SecretKeyRef(ctx, v.kube, v.storeKind, v.namespace, tokenRef)
		if err != nil {
			return true, err
		}
		v.client.SetToken(token)
		return true, nil
	}
	return false, nil
}
