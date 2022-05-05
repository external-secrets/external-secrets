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
package oracle

import (
	"context"
	"encoding/base64"
	"fmt"
	"reflect"
	"strings"
	"testing"

	secrets "github.com/oracle/oci-go-sdk/v56/secrets"
	utilpointer "k8s.io/utils/pointer"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakeoracle "github.com/external-secrets/external-secrets/pkg/provider/oracle/fake"
)

type vaultTestCase struct {
	mockClient     *fakeoracle.OracleMockClient
	apiInput       *secrets.GetSecretBundleByNameRequest
	apiOutput      *secrets.GetSecretBundleByNameResponse
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidVaultTestCase() *vaultTestCase {
	smtc := vaultTestCase{
		mockClient:     &fakeoracle.OracleMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.WithValue(*smtc.apiInput, *smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "default",
	}
}

func makeValidAPIInput() *secrets.GetSecretBundleByNameRequest {
	return &secrets.GetSecretBundleByNameRequest{
		SecretName: utilpointer.StringPtr("test-secret"),
		VaultId:    utilpointer.StringPtr("test-vault"),
	}
}

func makeValidAPIOutput() *secrets.GetSecretBundleByNameResponse {
	return &secrets.GetSecretBundleByNameResponse{
		SecretBundle: secrets.SecretBundle{},
	}
}

func makeValidVaultTestCaseCustom(tweaks ...func(smtc *vaultTestCase)) *vaultTestCase {
	smtc := makeValidVaultTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(*smtc.apiInput, *smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *vaultTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *vaultTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedOracleProvider
}

func TestOracleVaultGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *vaultTestCase) {
		smtc.apiOutput = &secrets.GetSecretBundleByNameResponse{
			SecretBundle: secrets.SecretBundle{
				SecretId:      utilpointer.StringPtr("test-id"),
				VersionNumber: utilpointer.Int64(1),
				SecretBundleContent: secrets.Base64SecretBundleContentDetails{
					Content: utilpointer.StringPtr(base64.StdEncoding.EncodeToString([]byte(secretValue))),
				},
			},
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*vaultTestCase{
		makeValidVaultTestCaseCustom(setAPIErr),
		makeValidVaultTestCaseCustom(setNilMockClient),
		makeValidVaultTestCaseCustom(setSecretString),
	}

	sm := VaultManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
		fmt.Println(*v.ref)
		out, err := sm.GetSecret(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *vaultTestCase) {
		smtc.apiOutput.SecretBundleContent = secrets.Base64SecretBundleContentDetails{
			Content: utilpointer.StringPtr(base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar"}`))),
		}
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *vaultTestCase) {
		smtc.apiOutput.SecretBundleContent = secrets.Base64SecretBundleContentDetails{
			Content: utilpointer.StringPtr(base64.StdEncoding.EncodeToString([]byte(`-----------------`))),
		}
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*vaultTestCase{
		makeValidVaultTestCaseCustom(setDeserialization),
		makeValidVaultTestCaseCustom(setInvalidJSON),
		makeValidVaultTestCaseCustom(setNilMockClient),
		makeValidVaultTestCaseCustom(setAPIErr),
	}

	sm := VaultManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(vault, region string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Oracle: &esv1beta1.OracleProvider{
					Vault:  vault,
					Region: region,
				},
			},
		},
	}

	for _, f := range fn {
		store = f(store)
	}
	return store
}
func withSecretAuth(user, tenancy string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth = &esv1beta1.OracleAuth{
			User:    user,
			Tenancy: tenancy,
		}
		return store
	}
}
func withPrivateKey(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth.SecretRef.PrivateKey = v1.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}
func withFingerprint(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Oracle.Auth.SecretRef.Fingerprint = v1.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	namespace := "my-namespace"
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", "some-region"),
			err:   fmt.Errorf("vault cannot be empty"),
		},
		{
			store: makeSecretStore("some-OCID", ""),
			err:   fmt.Errorf("region cannot be empty"),
		},
		{
			store: makeSecretStore("some-OCID", "some-region", withSecretAuth("", "a-tenant")),
			err:   fmt.Errorf("user cannot be empty"),
		},
		{
			store: makeSecretStore("some-OCID", "some-region", withSecretAuth("user-OCID", "")),
			err:   fmt.Errorf("tenant cannot be empty"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("", "key", nil)),
			err:   fmt.Errorf("privateKey.name cannot be empty"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("bob", "key", &namespace)),
			err:   fmt.Errorf("namespace not allowed with namespaced SecretStore"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("bob", "", nil)),
			err:   fmt.Errorf("privateKey.key cannot be empty"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("bob", "key", nil), withFingerprint("", "key", nil)),
			err:   fmt.Errorf("fingerprint.name cannot be empty"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("bob", "key", nil), withFingerprint("kelly", "key", &namespace)),
			err:   fmt.Errorf("namespace not allowed with namespaced SecretStore"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region", withSecretAuth("user-OCID", "a-tenant"), withPrivateKey("bob", "key", nil), withFingerprint("kelly", "", nil)),
			err:   fmt.Errorf("fingerprint.key cannot be empty"),
		},
		{
			store: makeSecretStore("vault-OCID", "some-region"),
			err:   nil,
		},
	}
	p := VaultManagementService{}
	for _, tc := range testCases {
		err := p.ValidateStore(tc.store)
		if tc.err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		}
	}
}
