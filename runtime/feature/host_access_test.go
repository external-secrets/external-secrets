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
	"testing"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestHostAccessAllowed(t *testing.T) {
	t.Cleanup(func() { SetUnsafeAllowSecretStoreHostAccess(false) })

	for _, tt := range []struct {
		name     string
		kind     string
		unsafe   bool
		expected bool
	}{
		{name: "namespaced denied by default", kind: esv1.SecretStoreKind},
		{name: "namespaced explicitly enabled", kind: esv1.SecretStoreKind, unsafe: true, expected: true},
		{name: "cluster store allowed", kind: esv1.ClusterSecretStoreKind, expected: true},
		{name: "cluster store unaffected by flag", kind: esv1.ClusterSecretStoreKind, unsafe: true, expected: true},
		{name: "empty kind fails closed", unsafe: true},
		{name: "unknown kind fails closed", kind: "Other", unsafe: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			SetUnsafeAllowSecretStoreHostAccess(tt.unsafe)
			if got := HostAccessAllowed(tt.kind); got != tt.expected {
				t.Fatalf("HostAccessAllowed(%q) = %v, want %v", tt.kind, got, tt.expected)
			}
		})
	}
}
