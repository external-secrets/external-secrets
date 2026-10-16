/*
Copyright © The ESO Authors

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

package feature

import (
	"sync/atomic"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var unsafeAllowSecretStoreHostAccess atomic.Bool

// SetUnsafeAllowSecretStoreHostAccess configures whether a namespaced
// SecretStore may select controller-local files. It is intended to be called
// once, after command-line flags have been parsed.
func SetUnsafeAllowSecretStoreHostAccess(allow bool) {
	unsafeAllowSecretStoreHostAccess.Store(allow)
}

// HostAccessAllowed reports whether the given store kind may select files in
// the controller container. Unknown store kinds fail closed.
func HostAccessAllowed(storeKind string) bool {
	switch storeKind {
	case esv1.ClusterSecretStoreKind:
		return true
	case esv1.SecretStoreKind:
		return unsafeAllowSecretStoreHostAccess.Load()
	default:
		return false
	}
}
