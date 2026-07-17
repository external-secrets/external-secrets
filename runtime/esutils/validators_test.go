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
			name:  "allows optional value",
			value: "value",
			policy: ValueOrRefPolicy[string]{
				Presence: AllowValueOrRef,
			},
		},
		{
			name: "allows optional ref",
			ref:  &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: AllowValueOrRef,
			},
		},
		{
			name:  "allows optional pair: rejects both",
			value: "value",
			ref:   &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: AllowValueOrRef,
			},
			wantErr: ErrValueAndRefConflict,
		},
		{
			name: "requires ref only",
			ref:  &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireRefOnly,
			},
		},
		{
			name: "requires ref only: rejects missing ref",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireRefOnly,
			},
			wantErr: ErrRefRequired,
		},
		{
			name:  "requires ref only: rejects value",
			value: "value",
			ref:   &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireRefOnly,
			},
			wantErr: ErrValueNotAllowed,
		},
		{
			name:  "requires value only",
			value: "value",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOnly,
			},
		},
		{
			name: "requires value only: rejects missing value",
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOnly,
			},
			wantErr: ErrValueRequired,
		},
		{
			name:  "requires value only: rejects ref",
			value: "value",
			ref:   &ref,
			policy: ValueOrRefPolicy[string]{
				Presence: RequireValueOnly,
			},
			wantErr: ErrRefNotAllowed,
		},
		{
			name: "rejects unknown policy",
			policy: ValueOrRefPolicy[string]{
				Presence: RefPresencePolicy(99),
			},
			wantErr: errors.New("unknown value/reference presence policy: 99"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateValueOrRef(tt.value, tt.ref, tt.policy)
			if tt.wantErr == nil {
				if got != nil {
					t.Errorf("ValidateValueOrRef() got = %v, want no error", got)
				}
				return
			}
			if !errors.Is(got, tt.wantErr) && (got == nil || got.Error() != tt.wantErr.Error()) {
				t.Errorf("ValidateValueOrRef() got = %v, want = %v", got, tt.wantErr)
			}
		})
	}
}
