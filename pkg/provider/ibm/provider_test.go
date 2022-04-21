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
package ibm

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/secretsmanagerv1"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	corev1 "k8s.io/api/core/v1"
	utilpointer "k8s.io/utils/pointer"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/ibm/fake"
)

const (
	errExpectedErr = "wanted error got nil"
)

type secretManagerTestCase struct {
	mockClient     *fakesm.IBMMockClient
	apiInput       *sm.GetSecretOptions
	apiOutput      *sm.GetSecret
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	serviceURL     *string
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockClient:     &fakesm.IBMMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		serviceURL:     nil,
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "default",
	}
}

func makeValidAPIInput() *sm.GetSecretOptions {
	return &sm.GetSecretOptions{
		SecretType: core.StringPtr(sm.GetSecretOptionsSecretTypeArbitraryConst),
		ID:         utilpointer.StringPtr("test-secret"),
	}
}

func makeValidAPIOutput() *sm.GetSecret {
	secretData := make(map[string]interface{})
	secretData["payload"] = ""

	return &sm.GetSecret{
		Resources: []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr("testytype"),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			},
		},
	}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretManagerTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *secretManagerTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedIBMProvider
}

// simple tests for Validate Store.
func TestValidateStore(t *testing.T) {
	p := providerIBM{}
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				IBM: &esv1beta1.IBMProvider{},
			},
		},
	}
	err := p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "serviceURL is required" {
		t.Errorf("service URL test failed")
	}
	url := "my-url"
	store.Spec.Provider.IBM.ServiceURL = &url
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "secretAPIKey.name cannot be empty" {
		t.Errorf("KeySelector test failed: expected secret name is required, got %v", err)
	}
	store.Spec.Provider.IBM.Auth.SecretRef.SecretAPIKey.Name = "foo"
	store.Spec.Provider.IBM.Auth.SecretRef.SecretAPIKey.Key = "bar"
	ns := "ns-one"
	store.Spec.Provider.IBM.Auth.SecretRef.SecretAPIKey.Namespace = &ns
	err = p.ValidateStore(store)
	if err == nil {
		t.Errorf(errExpectedErr)
	} else if err.Error() != "namespace not allowed with namespaced SecretStore" {
		t.Errorf("KeySelector test failed: expected namespace not allowed, got %v", err)
	}
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestIBMSecretManagerGetSecret(t *testing.T) {
	secretData := make(map[string]interface{})
	secretString := "changedvalue"
	secretPassword := "P@ssw0rd"
	secretAPIKey := "01234567890"
	secretCertificate := "certificate_value"

	secretData["payload"] = secretString
	secretData["password"] = secretPassword
	secretData["certificate"] = secretCertificate

	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr("testytype"),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiOutput.Resources = resources
		smtc.expectedSecret = secretString
	}

	// good case: custom version set
	setCustomKey := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr("testytype"),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}
		smtc.ref.Key = "testyname"
		smtc.apiInput.ID = utilpointer.StringPtr("testyname")
		smtc.apiOutput.Resources = resources
		smtc.expectedSecret = secretString
	}

	// bad case: username_password type without property
	secretUserPass := "username_password/test-secret"
	badSecretUserPass := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretUserPass
		smtc.expectError = "remoteRef.property required for secret type username_password"
	}

	// good case: username_password type with property
	setSecretUserPass := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretUserPass
		smtc.ref.Property = "password"
		smtc.expectedSecret = secretPassword
	}

	// good case: iam_credenatials type
	setSecretIam := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeIamCredentialsConst),
				Name:       utilpointer.StringPtr("testyname"),
				APIKey:     utilpointer.StringPtr(secretAPIKey),
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeIamCredentialsConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = "iam_credentials/test-secret"
		smtc.expectedSecret = secretAPIKey
	}

	// good case: imported_cert type with property
	secretCert := "imported_cert/test-secret"
	setSecretCert := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretCert
		smtc.ref.Property = "certificate"
		smtc.expectedSecret = secretCertificate
	}

	// bad case: imported_cert type without property
	badSecretCert := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretCert
		smtc.expectError = "remoteRef.property required for secret type imported_cert"
	}

	// good case: public_cert type with property
	secretPublicCert := "public_cert/test-secret"
	setSecretPublicCert := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretPublicCert
		smtc.ref.Property = "certificate"
		smtc.expectedSecret = secretCertificate
	}

	// bad case: public_cert type without property
	badSecretPublicCert := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretPublicCert
		smtc.expectError = "remoteRef.property required for secret type public_cert"
	}

	secretDataKV := make(map[string]interface{})
	secretKVPayload := make(map[string]interface{})
	secretKVPayload["key1"] = "val1"
	secretDataKV["payload"] = secretKVPayload

	secretDataKVComplex := make(map[string]interface{})
	secretKVComplex := `{"key1":"val1","key2":"val2","key3":"val3","keyC":{"keyC1":"valC1", "keyC2":"valC2"}, "special.log": "file-content"}`

	secretDataKVComplex["payload"] = secretKVComplex

	secretKV := "kv/test-secret"
	// bad case: kv type with key which is not in payload
	badSecretKV := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKV,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.ref.Property = "other-key"
		smtc.expectError = "key other-key does not exist in secret kv/test-secret"
	}

	// good case: kv type with property
	setSecretKV := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKV,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key1"
		smtc.expectedSecret = "val1"
	}

	// good case: kv type with property, returns specific value
	setSecretKVWithKey := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKVComplex,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key2"
		smtc.expectedSecret = "val2"
	}

	// good case: kv type with property and path, returns specific value
	setSecretKVWithKeyPath := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKVComplex,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.ref.Property = "keyC.keyC2"
		smtc.expectedSecret = "valC2"
	}

	// good case: kv type with property and dot, returns specific value
	setSecretKVWithKeyDot := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKVComplex,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.ref.Property = "special.log"
		smtc.expectedSecret = "file-content"
	}

	// good case: kv type without property, returns all
	setSecretKVWithOutKey := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretDataKVComplex,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKV
		smtc.expectedSecret = secretKVComplex
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCase(),
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setCustomKey),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(badSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretIam),
		makeValidSecretManagerTestCaseCustom(setSecretCert),
		makeValidSecretManagerTestCaseCustom(badSecretCert),
		makeValidSecretManagerTestCaseCustom(setSecretKV),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKey),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyPath),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyDot),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithOutKey),
		makeValidSecretManagerTestCaseCustom(badSecretKV),
		makeValidSecretManagerTestCaseCustom(setSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(badSecretPublicCert),
	}

	sm := providerIBM{}
	for k, v := range successCases {
		sm.IBMClient = v.mockClient
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
	secretKeyName := "kv/test-secret"
	secretUsername := "user1"
	secretPassword := "P@ssw0rd"
	secretAPIKey := "01234567890"
	secretCertificate := "certificate_value"
	secretPrivateKey := "private_key_value"
	secretIntermediate := "intermediate_value"

	secretComplex := map[string]interface{}{
		"key1": "val1",
		"key2": "val2",
		"keyC": map[string]interface{}{
			"keyC1": map[string]string{
				"keyA": "valA",
				"keyB": "valB",
			},
		},
	}

	// good case: default version & deserialization
	setDeserialization := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = `{"foo":"bar"}`

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr("testytype"),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiOutput.Resources = resources
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = `-----------------`

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr("testytype"),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiOutput.Resources = resources
		smtc.expectError = "unable to unmarshal secret: invalid character '-' in numeric literal"
	}

	// good case: username_password
	setSecretUserPass := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["username"] = secretUsername
		secretData["password"] = secretPassword
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeUsernamePasswordConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = "username_password/test-secret"
		smtc.expectedData["username"] = []byte(secretUsername)
		smtc.expectedData["password"] = []byte(secretPassword)
	}

	// good case: iam_credentials
	setSecretIam := func(smtc *secretManagerTestCase) {
		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeIamCredentialsConst),
				Name:       utilpointer.StringPtr("testyname"),
				APIKey:     utilpointer.StringPtr(secretAPIKey),
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeIamCredentialsConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = "iam_credentials/test-secret"
		smtc.expectedData["apikey"] = []byte(secretAPIKey)
	}

	// good case: imported_cert
	setSecretCert := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["certificate"] = secretCertificate
		secretData["private_key"] = secretPrivateKey
		secretData["intermediate"] = secretIntermediate

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeImportedCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = "imported_cert/test-secret"
		smtc.expectedData["certificate"] = []byte(secretCertificate)
		smtc.expectedData["private_key"] = []byte(secretPrivateKey)
		smtc.expectedData["intermediate"] = []byte(secretIntermediate)
	}

	// good case: public_cert
	setSecretPublicCert := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["certificate"] = secretCertificate
		secretData["private_key"] = secretPrivateKey
		secretData["intermediate"] = secretIntermediate

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypePublicCertConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = "public_cert/test-secret"
		smtc.expectedData["certificate"] = []byte(secretCertificate)
		smtc.expectedData["private_key"] = []byte(secretPrivateKey)
		smtc.expectedData["intermediate"] = []byte(secretIntermediate)
	}

	// good case: kv, no property, return entire payload as key:value pairs
	setSecretKV := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = secretComplex

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKeyName
		smtc.expectedData["key1"] = []byte("val1")
		smtc.expectedData["key2"] = []byte("val2")
		smtc.expectedData["keyC"] = []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`)
	}

	// good case: kv, with property
	setSecretKVWithProperty := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = secretComplex

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.ref.Property = "keyC"
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKeyName
		smtc.expectedData["keyC1"] = []byte(`{"keyA":"valA","keyB":"valB"}`)
	}

	// good case: kv, with property and path
	setSecretKVWithPathAndProperty := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = secretComplex

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.ref.Property = "keyC.keyC1"
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKeyName
		smtc.expectedData["keyA"] = []byte("valA")
		smtc.expectedData["keyB"] = []byte("valB")
	}

	// bad case: kv, with property and path
	badSecretKVWithUnknownProperty := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretData["payload"] = secretComplex

		resources := []sm.SecretResourceIntf{
			&sm.SecretResource{
				SecretType: utilpointer.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst),
				Name:       utilpointer.StringPtr("testyname"),
				SecretData: secretData,
			}}

		smtc.apiInput.SecretType = core.StringPtr(sm.CreateSecretOptionsSecretTypeKvConst)
		smtc.ref.Property = "unknown.property"
		smtc.apiOutput.Resources = resources
		smtc.ref.Key = secretKeyName
		smtc.expectError = "key unknown.property does not exist in secret kv/test-secret"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretIam),
		makeValidSecretManagerTestCaseCustom(setSecretCert),
		makeValidSecretManagerTestCaseCustom(setSecretKV),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithProperty),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithPathAndProperty),
		makeValidSecretManagerTestCaseCustom(badSecretKVWithUnknownProperty),
		makeValidSecretManagerTestCaseCustom(setSecretPublicCert),
	}

	sm := providerIBM{}
	for k, v := range successCases {
		sm.IBMClient = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.ref)
		if !ErrorContains(err, v.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
		}
		if err == nil && !reflect.DeepEqual(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}

func TestValidRetryInput(t *testing.T) {
	sm := providerIBM{}

	invalid := "Invalid"
	serviceURL := "http://fake-service-url.cool"

	spec := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				IBM: &esv1beta1.IBMProvider{
					Auth: esv1beta1.IBMAuth{
						SecretRef: esv1beta1.IBMAuthSecretRef{
							SecretAPIKey: v1.SecretKeySelector{
								Name: "fake-secret",
								Key:  "fake-key",
							},
						},
					},
					ServiceURL: &serviceURL,
				},
			},
			RetrySettings: &esv1beta1.SecretStoreRetrySettings{
				RetryInterval: &invalid,
			},
		},
	}

	expected := fmt.Sprintf("cannot setup new ibm client: time: invalid duration %q", invalid)
	ctx := context.TODO()
	kube := &test.MockClient{
		MockGet: test.NewMockGetFn(nil, func(obj kclient.Object) error {
			if o, ok := obj.(*corev1.Secret); ok {
				o.Data = map[string][]byte{
					"fake-key": []byte("ImAFakeApiKey"),
				}
				return nil
			}
			return nil
		}),
	}

	_, err := sm.NewClient(ctx, spec, kube, "default")

	if !ErrorContains(err, expected) {
		t.Errorf("CheckValidRetryInput unexpected error: %s, expected: '%s'", err.Error(), expected)
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
