// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Copyright External Secrets Inc. 2025
// All Rights Reserved

package job

import (
	"fmt"
	"slices"
	"strings"

	scanv1alpha1 "github.com/external-secrets/external-secrets/apis/scan/v1alpha1"
)

// Sanitize converts a secret reference to a sanitized string.
func Sanitize(ref scanv1alpha1.SecretInStoreRef) string {
	cleanedName := strings.ToLower(strings.TrimSpace(ref.Name))
	cleanedKind := strings.ToLower(strings.TrimSpace(ref.Kind))
	cleanedKey := strings.TrimSuffix(strings.TrimPrefix(ref.RemoteRef.Key, "/"), "/")
	ans := cleanedKind + "." + cleanedName + "." + cleanedKey
	if ref.RemoteRef.Property != "" {
		cleanedProperty := strings.TrimSuffix(strings.TrimPrefix(ref.RemoteRef.Property, "/"), "/")
		ans += "." + cleanedProperty
	}
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(ans, "_", "-"), "/", "-"), ":", "-"))
}

// EqualLocations checks if two secret locations are equal.
func EqualLocations(a, b scanv1alpha1.SecretInStoreRef) bool {
	return a.Name == b.Name && a.Kind == b.Kind && a.APIVersion == b.APIVersion && a.RemoteRef.Key == b.RemoteRef.Key && a.RemoteRef.Property == b.RemoteRef.Property
}

// CompareLocations compares two secret locations.
func CompareLocations(a, b scanv1alpha1.SecretInStoreRef) int {
	aIdx := fmt.Sprintf("%s.%s", a.RemoteRef.Key, a.RemoteRef.Property)
	if a.RemoteRef.Property == "" {
		aIdx = a.RemoteRef.Key
	}
	bIdx := fmt.Sprintf("%s.%s", b.RemoteRef.Key, b.RemoteRef.Property)
	if b.RemoteRef.Property == "" {
		bIdx = b.RemoteRef.Key
	}
	return strings.Compare(aIdx, bIdx)
}

// SortLocations sorts a slice of secret locations.
func SortLocations(loc []scanv1alpha1.SecretInStoreRef) {
	slices.SortFunc(loc, CompareLocations)
}

// EqualSecretUpdateRecord checks if two secret update records are equal.
func EqualSecretUpdateRecord(a, b scanv1alpha1.SecretUpdateRecord) bool {
	return a.SecretHash == b.SecretHash && a.Timestamp.Equal(&b.Timestamp)
}
