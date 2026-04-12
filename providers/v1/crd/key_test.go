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
	"strings"
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestParseRemoteRefKey(t *testing.T) {
	tests := []struct {
		name        string
		storeKind   string
		key         string
		wantObj     string
		wantNS      *string
		wantErrSubs string
	}{
		{name: "SecretStore bare name", storeKind: esv1.SecretStoreKind, key: "myobj", wantObj: "myobj"},
		{name: "SecretStore rejects slash", storeKind: esv1.SecretStoreKind, key: "ns/obj", wantErrSubs: "must not contain '/'"},
		{name: "SecretStore empty key", storeKind: esv1.SecretStoreKind, key: "", wantErrSubs: "must not be empty"},
		{name: "ClusterSecretStore bare name", storeKind: esv1.ClusterSecretStoreKind, key: "cluster-only", wantObj: "cluster-only"},
		{name: "ClusterSecretStore namespace/name", storeKind: esv1.ClusterSecretStoreKind, key: "default/myobj", wantObj: "myobj", wantNS: strPtr("default")},
		{name: "ClusterSecretStore empty namespace segment", storeKind: esv1.ClusterSecretStoreKind, key: "/obj", wantErrSubs: "namespace segment"},
		{name: "ClusterSecretStore empty name segment", storeKind: esv1.ClusterSecretStoreKind, key: "ns/", wantErrSubs: "object name after '/'"},
		{name: "ClusterSecretStore extra slash", storeKind: esv1.ClusterSecretStoreKind, key: "ns/foo/bar", wantErrSubs: "exactly one '/'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj, ns, err := parseRemoteRefKey(tt.storeKind, tt.key)
			if tt.wantErrSubs != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubs) {
					t.Fatalf("parseRemoteRefKey() error = %v, want substring %q", err, tt.wantErrSubs)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseRemoteRefKey() unexpected error: %v", err)
			}
			if obj != tt.wantObj {
				t.Fatalf("objectName = %q, want %q", obj, tt.wantObj)
			}
			if tt.wantNS == nil {
				if ns != nil {
					t.Fatalf("keyNamespace = %v, want nil", ns)
				}
				return
			}
			if ns == nil || *ns != *tt.wantNS {
				t.Fatalf("keyNamespace = %v, want %q", ns, *tt.wantNS)
			}
		})
	}
}

func strPtr(s string) *string { return &s }
