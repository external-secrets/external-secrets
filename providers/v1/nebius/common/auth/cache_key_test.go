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

package auth

import (
	"testing"

	tassert "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

func Test_ServiceAccountCredentials_CacheKeyInvariants(t *testing.T) {
	t.Parallel()

	baseStore := newCacheKeyTestStore("store-uid", 1)
	baseKey := getServiceAccountCredsCacheKey(
		baseStore,
		"namespace",
		"key-id",
		"subject",
		"secret-key-material",
	)

	tests := []struct {
		name        string
		store       esv1.GenericStore
		namespace   string
		keyID       string
		subject     string
		privateKey  string
		wantSameKey bool
	}{
		{
			name:        "same cache inputs",
			store:       newCacheKeyTestStore("store-uid", 1),
			namespace:   "namespace",
			keyID:       "key-id",
			subject:     "subject",
			privateKey:  "secret-key-material",
			wantSameKey: true,
		},
		{
			name:       "different store UID",
			store:      newCacheKeyTestStore("another-store-uid", 1),
			namespace:  "namespace",
			keyID:      "key-id",
			subject:    "subject",
			privateKey: "secret-key-material",
		},
		{
			name:       "different store generation",
			store:      newCacheKeyTestStore("store-uid", 2),
			namespace:  "namespace",
			keyID:      "key-id",
			subject:    "subject",
			privateKey: "secret-key-material",
		},
		{
			name:       "different namespace",
			store:      newCacheKeyTestStore("store-uid", 1),
			namespace:  "another-namespace",
			keyID:      "key-id",
			subject:    "subject",
			privateKey: "secret-key-material",
		},
		{
			name:       "different key ID",
			store:      newCacheKeyTestStore("store-uid", 1),
			namespace:  "namespace",
			keyID:      "another-key-id",
			subject:    "subject",
			privateKey: "secret-key-material",
		},
		{
			name:       "different subject",
			store:      newCacheKeyTestStore("store-uid", 1),
			namespace:  "namespace",
			keyID:      "key-id",
			subject:    "another-subject",
			privateKey: "secret-key-material",
		},
		{
			name:       "different private key",
			store:      newCacheKeyTestStore("store-uid", 1),
			namespace:  "namespace",
			keyID:      "key-id",
			subject:    "subject",
			privateKey: "rotated-key-material",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := getServiceAccountCredsCacheKey(
				tt.store,
				tt.namespace,
				tt.keyID,
				tt.subject,
				tt.privateKey,
			)

			if tt.wantSameKey {
				tassert.Equal(t, baseKey, key)
			} else {
				tassert.NotEqual(t, baseKey, key)
			}
		})
	}

	tassert.NotContains(t, baseKey, "secret-key-material")
	tassert.Contains(t, baseKey, HashBytes([]byte("secret-key-material")))
}

func Test_WorkloadIdentity_CacheKeyInvariants(t *testing.T) {
	t.Parallel()

	baseKey := getWICacheKey("namespace", newCacheKeyTestStore("store-uid", 1))

	tests := []struct {
		name        string
		store       esv1.GenericStore
		namespace   string
		wantSameKey bool
	}{
		{
			name:        "same cache inputs",
			store:       newCacheKeyTestStore("store-uid", 1),
			namespace:   "namespace",
			wantSameKey: true,
		},
		{
			name:      "different store UID",
			store:     newCacheKeyTestStore("another-store-uid", 1),
			namespace: "namespace",
		},
		{
			name:      "different store generation",
			store:     newCacheKeyTestStore("store-uid", 2),
			namespace: "namespace",
		},
		{
			name:      "different namespace",
			store:     newCacheKeyTestStore("store-uid", 1),
			namespace: "another-namespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := getWICacheKey(tt.namespace, tt.store)
			if tt.wantSameKey {
				tassert.Equal(t, baseKey, key)
			} else {
				tassert.NotEqual(t, baseKey, key)
			}
		})
	}
}

func TestCacheKeysAreDistinctAcrossCredentialTypes(t *testing.T) {
	t.Parallel()

	store := newCacheKeyTestStore("store-uid", 1)
	serviceAccountKey := getServiceAccountCredsCacheKey(
		store,
		"namespace",
		"key-id",
		"subject",
		"secret-key-material",
	)
	workloadIdentityKey := getWICacheKey("namespace", store)

	tassert.NotEqual(t, serviceAccountKey, workloadIdentityKey)
}

func newCacheKeyTestStore(uid string, generation int64) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			UID:        types.UID(uid),
			Generation: generation,
		},
	}
}
