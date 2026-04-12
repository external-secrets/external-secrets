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

package crd

import (
	"fmt"
	"strings"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

// parseRemoteRefKey interprets ExternalSecret remoteRef.key (and PushSecret remote key).
//
// SecretStore: '/' is not allowed; the object name is the full key. The API namespace
// comes only from the store (ExternalSecret / service account namespace and optional
// remoteNamespace), never from the key.
//
// ClusterSecretStore: if the key contains '/', it must be namespace/objectName (first
// slash separates). If there is no '/', the key is the object name for a cluster-scoped
// resource only.
//
// Returns objectName, keyNamespace (non-nil when namespace/objectName form was used), err.
func parseRemoteRefKey(storeKind, remoteKey string) (objectName string, keyNamespace *string, err error) {
	if remoteKey == "" {
		return "", nil, fmt.Errorf("crd: remoteRef.key must not be empty")
	}
	switch storeKind {
	case esv1.SecretStoreKind:
		if strings.Contains(remoteKey, "/") {
			return "", nil, fmt.Errorf("crd: remoteRef.key must not contain '/' for SecretStore; namespace is fixed to the store namespace")
		}
		return remoteKey, nil, nil
	case esv1.ClusterSecretStoreKind:
		ns, name, ok := strings.Cut(remoteKey, "/")
		if !ok {
			return remoteKey, nil, nil
		}
		if ns == "" {
			return "", nil, fmt.Errorf("crd: invalid remoteRef.key %q: namespace segment before '/' must not be empty", remoteKey)
		}
		if name == "" {
			return "", nil, fmt.Errorf("crd: invalid remoteRef.key %q: object name after '/' must not be empty", remoteKey)
		}
		if strings.Contains(name, "/") {
			return "", nil, fmt.Errorf("crd: invalid remoteRef.key %q: must be in \"namespace/objectName\" form (exactly one '/')", remoteKey)
		}
		return name, &ns, nil
	default:
		return remoteKey, nil, nil
	}
}
