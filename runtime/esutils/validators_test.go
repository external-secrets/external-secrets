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

package esutils

import (
	"errors"
	"testing"
)

func TestValidateValueOrRef(t *testing.T) {
	errRef := errors.New("ref")
	ref := "ref"

	tests := []struct {
		name    string
		value   string
		ref     *string
		policy  ValueOrRefPolicy[string]
		wantErr error
	}{
		{
			name:  "requires exactly one: value",
			value: "value",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOrRef,
			},
		},
		{
			name: "requires exactly one: ref",
			ref:  &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOrRef,
			},
		},
		{
			name:  "requires exactly one: rejects both",
			value: "value",
			ref:   &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOrRef,
			},
			wantErr: ErrValueAndRefConflict,
		},
		{
			name: "requires exactly one: rejects missing both",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOrRef,
			},
			wantErr: ErrValueOrRefMissing,
		},
		{
			name: "validates ref when present",
			ref:  &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOrRef,
				ValidateRef: func(string) error {
					return errRef
				},
			},
			wantErr: errRef,
		},
		{
			name:  "allows optional empty pair",
			value: "",
			ref:   nil,
			policy: ValueOrRefPolicy[string]{
				Presence: AllowValueOrRef,
			},
		},
		{
			name: "requires ref only",
			ref:  &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireRefOnly,
			},
		},
		{
			name:  "requires value only",
			value: "value",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOnly,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateValueOrRef(tt.value, tt.ref, tt.policy)
			if got == nil && tt.wantErr == nil {
				return
			}
			if got == nil || tt.wantErr == nil || got.Error() != tt.wantErr.Error() {
				t.Errorf("ValidateValueOrRef() got = %v, want = %v", got, tt.wantErr)
			}
		})
	}
}
