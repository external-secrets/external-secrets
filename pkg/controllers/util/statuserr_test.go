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

package ctrlutil

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// providerErr stands in for an error returned by provider code, which may embed
// secret material and must never reach a status condition.
const providerErr = "json: cannot unmarshal number 8019210420527506405 into Go value of type float64"

func TestSafeMessage(t *testing.T) {
	sentinel := errors.New("secret is immutable")

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error yields nothing",
			err:  nil,
			want: "",
		},
		{
			name: "unmarked error yields nothing",
			err:  errors.New(providerErr),
			want: "",
		},
		{
			name: "marked error is published",
			err:  Safe(errors.New("could not update secret foo: already exists")),
			want: "could not update secret foo: already exists",
		},
		{
			name: "marking nil stays nil",
			err:  Safe(nil),
			want: "",
		},
		{
			name: "wrapping a marked error keeps it publishable",
			err:  fmt.Errorf("could not update secret: %w", Safe(sentinel)),
			want: "secret is immutable",
		},
		{
			name: "marked twice is not duplicated",
			err:  Safe(Safe(errors.New("target is owned by another ExternalSecret"))),
			want: "target is owned by another ExternalSecret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SafeMessage(tt.err); got != tt.want {
				t.Errorf("SafeMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

// Text composed around a marked error must never be published, whichever way it
// was composed. Each shape here leaked before SafeMessage took the innermost mark.
func TestSafeMessageDoesNotPublishWrapperText(t *testing.T) {
	marked := Safe(errors.New("connection refused"))

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "unmarked provider wrapper",
			err:  fmt.Errorf("%s: %w", providerErr, marked),
		},
		{
			name: "marked provider wrapper",
			err:  Safe(fmt.Errorf("%s: %w", providerErr, marked)),
		},
		{
			name: "joined with a provider error",
			err:  Safe(errors.Join(errors.New(providerErr), marked)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeMessage(tt.err)
			if got != "connection refused" {
				t.Errorf("SafeMessage() = %q, want %q", got, "connection refused")
			}
			if strings.Contains(got, "8019210420527506405") {
				t.Errorf("SafeMessage() leaked wrapper text: %q", got)
			}
		})
	}
}

func TestSafePreservesErrorsIs(t *testing.T) {
	sentinel := errors.New("sentinel")

	if !errors.Is(Safe(sentinel), sentinel) {
		t.Error("Safe() broke errors.Is against the wrapped error")
	}
	if Safe(sentinel).Error() != "sentinel" {
		t.Errorf("Safe().Error() = %q, want %q", Safe(sentinel).Error(), "sentinel")
	}
}

func TestSafeMessageTruncates(t *testing.T) {
	long := strings.Repeat("a", MaxConditionMessageLength+50)

	got := SafeMessage(Safe(errors.New(long)))
	want := strings.Repeat("a", MaxConditionMessageLength) + "..."
	if got != want {
		t.Errorf("SafeMessage() length = %d, want %d", len(got), len(want))
	}
}

// Truncation counts runes, so a multi-byte message is not cut mid-character.
func TestSafeMessageTruncatesOnRunes(t *testing.T) {
	long := strings.Repeat("é", MaxConditionMessageLength+10)

	got := SafeMessage(Safe(errors.New(long)))
	if runes := []rune(strings.TrimSuffix(got, "...")); len(runes) != MaxConditionMessageLength {
		t.Errorf("truncated to %d runes, want %d", len(runes), MaxConditionMessageLength)
	}
	if !strings.HasPrefix(got, "é") {
		t.Errorf("truncation split a multi-byte rune: %q", got[:8])
	}
}
