package alibaba

import (
	"testing"
	"fmt"
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
	// Add other required fields
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

type KeyManagementService struct {
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
	kms := &KeyManagementService{}

	// Test case: Valid store should pass validation
	validStore := &GenericStore{
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
	}

	err := kms.ValidateStore(*validStore)
	if err != nil {
		t.Errorf("ValidateStore() failed, expected: nil, got: %v", err)
	}

	// Test case: Invalid store with missing region should fail validation
	invalidStoreMissingRegion := &GenericStore{
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
	}

	err = kms.ValidateStore(*invalidStoreMissingRegion)
	if err == nil {
		t.Error("ValidateStore() failed, expected: error, got: nil")
	}

	
}

