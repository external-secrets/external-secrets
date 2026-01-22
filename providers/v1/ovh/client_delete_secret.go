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
	"errors"
	"fmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// If deletionPolicy is set to Delete, the Secret Manager Secret
// created from the Push Secret will be automatically removed
// when the associated Push Secret is deleted.
func (cl *ovhClient) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	err := cl.okmsClient.DeleteSecretV2(ctx, cl.okmsID, remoteRef.GetRemoteKey())

	if err != nil {
		err = handleOkmsError(err)
		if errors.Is(err, esv1.NoSecretErr) {
			return nil
		}
		return fmt.Errorf("failed to delete secret at path %q: %w", remoteRef.GetRemoteKey(), err)
	}

	return nil
}
