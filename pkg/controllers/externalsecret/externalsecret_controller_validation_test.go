/*
Copyright © The ESO Authors

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

package externalsecret

import (
	"strings"
	"testing"

	v1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestValidateNullBytePolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  esv1.ExternalSecretNullBytePolicy
		data    map[string][]byte
		wantErr string
	}{
		{
			name:   "zero value policy behaves like ignore",
			policy: "",
			data: map[string][]byte{
				"payload": []byte(nullByteSecretVal),
			},
		},
		{
			name:   "ignores null bytes when policy is not fail",
			policy: esv1.ExternalSecretNullBytePolicyIgnore,
			data: map[string][]byte{
				"payload": []byte(nullByteSecretVal),
			},
		},
		{
			name:   "allows nil secret data map",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data:   nil,
		},
		{
			name:   "allows rendered data without null bytes",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data: map[string][]byte{
				"payload": []byte("QQBC"),
			},
		},
		{
			name:   "fails on the offending key when rendered data contains null bytes",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data: map[string][]byte{
				"safe":    []byte("value"),
				"payload": []byte(nullByteSecretVal),
			},
			wantErr: `target secret key "payload" contains null bytes`,
		},
		{
			name:   "reports the first offending key in sorted order",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data: map[string][]byte{
				"zeta":  []byte(nullByteSecretVal),
				"alpha": []byte("C\x00D"),
			},
			wantErr: `target secret key "alpha" contains null bytes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			es := &esv1.ExternalSecret{
				Spec: esv1.ExternalSecretSpec{
					Target: esv1.ExternalSecretTarget{
						NullBytePolicy: tt.policy,
					},
				},
			}
			secret := &v1.Secret{
				Data: tt.data,
			}

			assertValidateNullBytePolicyError(t, validateNullBytePolicy(es, secret), tt.wantErr)
		})
	}
}

func assertValidateNullBytePolicyError(t *testing.T, err error, wantErr string) {
	t.Helper()

	if wantErr == "" {
		if err != nil {
			t.Fatalf("validateNullBytePolicy() unexpected error = %v", err)
		}
		return
	}

	if err == nil {
		t.Fatalf("validateNullBytePolicy() error = nil, want substring %q", wantErr)
	}
	if got := err.Error(); !strings.Contains(got, wantErr) {
		t.Fatalf("validateNullBytePolicy() error = %q, want substring %q", got, wantErr)
	}
}
