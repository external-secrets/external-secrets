/*
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
package alibaba

import (
	"fmt"
	"github.com/pkg/errors"
	"testing"
)

type AlibabaProvider struct {
	RegionID string
	Auth     AlibabaAuth
}

type AlibabaAuth struct {
	SecretRef SecretReference
}

type SecretReference struct {
	AccessKeyID string
}

type GenericStore struct {
	Spec *SecretStoreSpec
}

type SecretStoreSpec struct {
	Provider *SecretStoreProvider
}

type SecretStoreProvider struct {
	Alibaba *AlibabaProvider
}

func (kms *KeyManagementService) ValidateStore(store GenericStore) error {
	storeSpec := store.Spec
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Alibaba == nil {
		return fmt.Errorf("no store type or wrong store type")
	}

	alibabaSpec := storeSpec.Provider.Alibaba

	regionID := alibabaSpec.RegionID

	if regionID == "" {
		return fmt.Errorf("missing alibaba region")
	}

	accessKeyID := alibabaSpec.Auth.SecretRef.AccessKeyID

	if accessKeyID == "" {
		return fmt.Errorf("missing access key ID")
	}

	return nil
}

func TestValidateStore(t *testing.T) {
	tests := []struct {
		name     string
		store    *GenericStore
		expected error
	}{
		{
			name: "Valid store should pass validation",
			store: &GenericStore{
				Spec: &SecretStoreSpec{
					Provider: &SecretStoreProvider{
						Alibaba: &AlibabaProvider{
							RegionID: "mockRegionID",
							Auth: AlibabaAuth{
								SecretRef: SecretReference{
									AccessKeyID: "mockAccessKeyID",
									// Add other required fields for testing
								},
							},
						},
					},
				},
			},
			expected: nil,
		},
		{
			name: "Invalid store with missing region should fail validation",
			store: &GenericStore{
				Spec: &SecretStoreSpec{
					Provider: &SecretStoreProvider{
						Alibaba: &AlibabaProvider{
							// Missing RegionID intentionally
							Auth: AlibabaAuth{
								SecretRef: SecretReference{
									AccessKeyID: "mockAccessKeyID",
									// Add other required fields for testing
								},
							},
						},
					},
				},
			},
			expected: errors.New("Missing region ID"),
		},
		// Add more test cases as needed
	}

	kms := &KeyManagementService{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := kms.ValidateStore(*tc.store)
			if !errors.Is(err, tc.expected) {
				t.Errorf("ValidateStore() failed, expected: %v, got: %v", tc.expected, err)
			}
		})
	}
}
