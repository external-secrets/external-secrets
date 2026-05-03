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

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func TestValidateFetchedSecretValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  esv1.ExternalSecretNullBytePolicy
		key     string
		value   []byte
		wantErr string
	}{
		{
			name:   "zero value policy behaves like ignore",
			policy: "",
			key:    "payload",
			value:  []byte(nullByteSecretVal),
		},
		{
			name:   "ignores null bytes when policy is not fail",
			policy: esv1.ExternalSecretNullBytePolicyIgnore,
			key:    "payload",
			value:  []byte(nullByteSecretVal),
		},
		{
			name:   "allows nil values",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			key:    "payload",
			value:  nil,
		},
		{
			name:   "allows fetched data without null bytes",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			key:    "payload",
			value:  []byte("QQBC"),
		},
		{
			name:    "fails on fetched data containing null bytes",
			policy:  esv1.ExternalSecretNullBytePolicyFail,
			key:     "payload",
			value:   []byte(nullByteSecretVal),
			wantErr: `fetched secret value for key "payload" contains null bytes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertFetchedSecretValidationError(t, validateFetchedSecretValue(tt.policy, tt.key, tt.value), tt.wantErr)
		})
	}
}

func TestValidateFetchedSecretMap(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  esv1.ExternalSecretNullBytePolicy
		data    map[string][]byte
		wantErr string
	}{
		{
			name:   "allows nil secret data map",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data:   nil,
		},
		{
			name:   "reports the first offending key in sorted order",
			policy: esv1.ExternalSecretNullBytePolicyFail,
			data: map[string][]byte{
				"zeta":  []byte(nullByteSecretVal),
				"alpha": []byte("C\x00D"),
			},
			wantErr: `fetched secret value for key "alpha" contains null bytes`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertFetchedSecretValidationError(t, validateFetchedSecretMap(tt.policy, tt.data), tt.wantErr)
		})
	}
}

func assertFetchedSecretValidationError(t *testing.T, err error, wantErr string) {
	t.Helper()

	if wantErr == "" {
		if err != nil {
			t.Fatalf("unexpected error = %v", err)
		}
		return
	}

	if err == nil {
		t.Fatalf("error = nil, want substring %q", wantErr)
	}
	if got := err.Error(); !strings.Contains(got, wantErr) {
		t.Fatalf("error = %q, want substring %q", got, wantErr)
	}
}
