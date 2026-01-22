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

package ovh

import (
	"context"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// GetSecret retrieves a single secret from the provider.
// The created secret will store the entire secret value under the specified key.
// You can specify a key, a property and a version.
func (cl *ovhClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	// Retrieve the KMS secret using the OVH SDK.
	secretData, _, err := getSecretWithOvhSDK(ctx, cl.okmsClient, cl.okmsID, ref)
	if err != nil {
		return []byte{}, err
	}

	return secretData, nil
}
