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
	"fmt"
)

var (
	// ErrValueAndRefConflict is returned when both a value and reference are set.
	ErrValueAndRefConflict = errors.New("cannot specify both value and reference")
	// ErrValueOrRefMissing is returned when neither a value nor reference is set.
	ErrValueOrRefMissing = errors.New("must specify either value or reference")
	// ErrRefRequired is returned when a reference is required but missing.
	ErrRefRequired = errors.New("reference is required")
	// ErrValueNotAllowed is returned when a value is set but only a reference is allowed.
	ErrValueNotAllowed = errors.New("value must not be specified")
	// ErrValueRequired is returned when a value is required but missing.
	ErrValueRequired = errors.New("value is required")
	// ErrRefNotAllowed is returned when a reference is set but only a value is allowed.
	ErrRefNotAllowed = errors.New("reference must not be specified")
)

// RefPresencePolicy describes which side of a value/reference pair is allowed.
type RefPresencePolicy int

const (
	// RequireValueOrRef requires exactly one of value or reference to be set.
	RequireValueOrRef RefPresencePolicy = iota
	// AllowValueOrRef allows neither value nor reference to be set, but rejects both being set.
	AllowValueOrRef
	// RequireRefOnly requires reference to be set and value to be empty.
	RequireRefOnly
	// RequireValueOnly requires value to be set and reference to be empty.
	RequireValueOnly
)

// ValueOrRefPolicy configures ValidateValueOrRef.
type ValueOrRefPolicy[T any] struct {
	Presence    RefPresencePolicy
	ValidateRef func(T) error
}

// ValidateValueOrRef validates fields that allow a direct value or a reference.
func ValidateValueOrRef[T any](value string, ref *T, policy ValueOrRefPolicy[T]) error {
	switch policy.Presence {
	case RequireValueOrRef:
		if value != "" && ref != nil {
			return ErrValueAndRefConflict
		}
		if value == "" && ref == nil {
			return ErrValueOrRefMissing
		}
	case AllowValueOrRef:
		if value != "" && ref != nil {
			return ErrValueAndRefConflict
		}
	case RequireRefOnly:
		if ref == nil {
			return ErrRefRequired
		}
		if value != "" {
			return ErrValueNotAllowed
		}
	case RequireValueOnly:
		if value == "" {
			return ErrValueRequired
		}
		if ref != nil {
			return ErrRefNotAllowed
		}
	default:
		return fmt.Errorf("unknown value/reference presence policy: %d", policy.Presence)
	}

	if ref != nil && policy.ValidateRef != nil {
		return policy.ValidateRef(*ref)
	}
	return nil
}
