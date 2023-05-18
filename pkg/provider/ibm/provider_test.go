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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/ibm/fake"
)

const (
	errExpectedErr = "wanted error got nil"
	secretKey      = "test-secret"
	secretUUID     = "d5deb37a-7883-4fe2-a5e7-3c15420adc76"
)

type secretManagerTestCase struct {
	name           string
	mockClient     *fakesm.IBMMockClient
	apiInput       *sm.GetSecretOptions
	apiOutput      sm.SecretIntf
	listInput      *sm.ListSecretsOptions
	listOutput     *sm.SecretMetadataPaginatedCollection
	listError      error
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
		listInput:      makeValidListInput(),
		listOutput:     makeValidListSecretsOutput(),
		listError:      nil,
		serviceURL:     nil,
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	mcParams := fakesm.IBMMockClientParams{
		GetSecretOptions:   smtc.apiInput,
		GetSecretOutput:    smtc.apiOutput,
		GetSecretErr:       smtc.apiErr,
		ListSecretsOptions: smtc.listInput,
		ListSecretsOutput:  smtc.listOutput,
		ListSecretsErr:     smtc.listError,
	}
	smtc.mockClient.WithValue(mcParams)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     secretUUID,
		Version: "default",
	}
}

func makeValidAPIInput() *sm.GetSecretOptions {
	return &sm.GetSecretOptions{
		ID: utilpointer.String(secretUUID),
	}
}

func makeValidAPIOutput() sm.SecretIntf {
	secret := &sm.Secret{
		SecretType: utilpointer.String(sm.Secret_SecretType_Arbitrary),
		Name:       utilpointer.String("testyname"),
		ID:         utilpointer.String(secretUUID),
	}
	var i sm.SecretIntf = secret
	return i
}

func makeValidListSecretsOutput() *sm.SecretMetadataPaginatedCollection {
	list := sm.SecretMetadataPaginatedCollection{}
	return &list
}

func makeValidListInput() *sm.ListSecretsOptions {
	listOpt := sm.ListSecretsOptions{}
	return &listOpt
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	mcParams := fakesm.IBMMockClientParams{
		GetSecretOptions:   smtc.apiInput,
		GetSecretOutput:    smtc.apiOutput,
		GetSecretErr:       smtc.apiErr,
		ListSecretsOptions: smtc.listInput,
		ListSecretsOutput:  smtc.listOutput,
		ListSecretsErr:     smtc.listError,
	}
	smtc.mockClient.WithValue(mcParams)
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
	var nilProfile esv1beta1.IBMAuthContainerAuth
	store.Spec.Provider.IBM.Auth.ContainerAuth = nilProfile
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

	// add container auth test
	store.Spec.Provider.IBM = &esv1beta1.IBMProvider{}
	store.Spec.Provider.IBM.ServiceURL = &url
	store.Spec.Provider.IBM.Auth.ContainerAuth.Profile = "Trusted IAM Profile"
	store.Spec.Provider.IBM.Auth.ContainerAuth.TokenLocation = "/a/path/to/nowhere/that/should/exist"
	err = p.ValidateStore(store)
	expected := "cannot read container auth token"
	if !ErrorContains(err, expected) {
		t.Errorf("ProfileSelector test failed: %s, expected: '%s'", err.Error(), expected)
	}
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestIBMSecretManagerGetSecret(t *testing.T) {
	secretString := "changedvalue"
	secretUsername := "userName"
	secretPassword := "P@ssw0rd"
	secretAPIKey := "01234567890"
	secretCertificate := "certificate_value"

	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretManagerTestCase) {
		secret := &sm.ArbitrarySecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Arbitrary),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Payload:    &secretString,
		}
		smtc.name = "good case: default version is set"
		smtc.apiOutput = secret
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.expectedSecret = secretString
	}

	// good case: custom version set
	setCustomKey := func(smtc *secretManagerTestCase) {
		secret := &sm.ArbitrarySecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Arbitrary),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Payload:    &secretString,
		}
		smtc.name = "good case: custom version set"
		smtc.ref.Key = "arbitrary/" + secretUUID
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.expectedSecret = secretString
	}

	// bad case: username_password type without property
	secretUserPass := "username_password/" + secretUUID
	badSecretUserPass := func(smtc *secretManagerTestCase) {
		secret := &sm.UsernamePasswordSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_UsernamePassword),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Username:   &secretUsername,
			Password:   &secretPassword,
		}
		smtc.name = "bad case: username_password type without property"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUserPass
		smtc.expectError = "remoteRef.property required for secret type username_password"
	}

	// good case: username_password type with property
	funcSetUserPass := func(secretName, property, name string) func(smtc *secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.UsernamePasswordSecret{
				SecretType: utilpointer.String(sm.Secret_SecretType_UsernamePassword),
				Name:       utilpointer.String("testyname"),
				ID:         utilpointer.String(secretUUID),
				Username:   &secretUsername,
				Password:   &secretPassword,
			}
			secretMetadata := &sm.UsernamePasswordSecretMetadata{
				Name: utilpointer.String("testyname"),
				ID:   utilpointer.String(secretUUID),
			}
			smtc.name = name
			smtc.apiInput.ID = utilpointer.String(secretUUID)
			smtc.apiOutput = secret
			smtc.listInput.Search = utilpointer.String("testyname")
			smtc.listOutput.Secrets = make([]sm.SecretMetadataIntf, 1)
			smtc.listOutput.Secrets[0] = secretMetadata
			smtc.ref.Key = "username_password/" + secretName
			smtc.ref.Property = property
			if property == "username" {
				smtc.expectedSecret = secretUsername
			} else {
				smtc.expectedSecret = secretPassword
			}
		}
	}
	setSecretUserPassByID := funcSetUserPass(secretUUID, "username", "good case: username_password type - get username by ID")
	setSecretUserPassUsername := funcSetUserPass("testyname", "username", "good case: username_password type - get username by secret name")
	setSecretUserPassPassword := funcSetUserPass("testyname", "password", "good case: username_password type - get password by secret name")

	// good case: iam_credentials type
	funcSetSecretIam := func(secretName, name string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.IAMCredentialsSecret{
				SecretType: utilpointer.String(sm.Secret_SecretType_IamCredentials),
				Name:       utilpointer.String("testyname"),
				ID:         utilpointer.String(secretUUID),
				ApiKey:     utilpointer.String(secretAPIKey),
			}
			secretMetadata := &sm.IAMCredentialsSecretMetadata{
				Name: utilpointer.String("testyname"),
				ID:   utilpointer.String(secretUUID),
			}
			smtc.apiInput.ID = utilpointer.String(secretUUID)
			smtc.name = name
			smtc.apiOutput = secret
			smtc.listInput.Search = utilpointer.String("testyname")
			smtc.listOutput.Secrets = make([]sm.SecretMetadataIntf, 1)
			smtc.listOutput.Secrets[0] = secretMetadata
			smtc.ref.Key = "iam_credentials/" + secretName
			smtc.expectedSecret = secretAPIKey
		}
	}

	setSecretIamByID := funcSetSecretIam(secretUUID, "good case: iam_credenatials type - get API Key by ID")
	setSecretIamByName := funcSetSecretIam("testyname", "good case: iam_credenatials type - get API Key by name")

	funcSetCertSecretTest := func(secret sm.SecretIntf, name, certType string, good bool) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			smtc.name = name
			smtc.apiInput.ID = utilpointer.String(secretUUID)
			smtc.apiOutput = secret
			smtc.ref.Key = certType + "/" + secretUUID
			if good {
				smtc.ref.Property = "certificate"
				smtc.expectedSecret = secretCertificate
			} else {
				smtc.expectError = "remoteRef.property required for secret type " + certType
			}
		}
	}

	// good case: imported_cert type with property
	importedCert := &sm.ImportedCertificate{
		SecretType:   utilpointer.String(sm.Secret_SecretType_ImportedCert),
		Name:         utilpointer.String("testyname"),
		ID:           utilpointer.String(secretUUID),
		Certificate:  utilpointer.String(secretCertificate),
		Intermediate: utilpointer.String("intermediate"),
		PrivateKey:   utilpointer.String("private_key"),
	}
	setSecretCert := funcSetCertSecretTest(importedCert, "good case: imported_cert type with property", sm.Secret_SecretType_ImportedCert, true)

	// bad case: imported_cert type without property
	badSecretCert := funcSetCertSecretTest(importedCert, "bad case: imported_cert type without property", sm.Secret_SecretType_ImportedCert, false)

	// good case: public_cert type with property
	publicCert := &sm.PublicCertificate{
		SecretType:   utilpointer.String(sm.Secret_SecretType_PublicCert),
		Name:         utilpointer.String("testyname"),
		ID:           utilpointer.String(secretUUID),
		Certificate:  utilpointer.String(secretCertificate),
		Intermediate: utilpointer.String("intermediate"),
		PrivateKey:   utilpointer.String("private_key"),
	}
	setSecretPublicCert := funcSetCertSecretTest(publicCert, "good case: public_cert type with property", sm.Secret_SecretType_PublicCert, true)

	// bad case: public_cert type without property
	badSecretPublicCert := funcSetCertSecretTest(publicCert, "bad case: public_cert type without property", sm.Secret_SecretType_PublicCert, false)

	// good case: private_cert type with property
	privateCert := &sm.PrivateCertificate{
		SecretType:  utilpointer.String(sm.Secret_SecretType_PublicCert),
		Name:        utilpointer.String("testyname"),
		ID:          utilpointer.String(secretUUID),
		Certificate: utilpointer.String(secretCertificate),
		PrivateKey:  utilpointer.String("private_key"),
	}
	setSecretPrivateCert := funcSetCertSecretTest(privateCert, "good case: private_cert type with property", sm.Secret_SecretType_PrivateCert, true)

	// bad case: private_cert type without property
	badSecretPrivateCert := funcSetCertSecretTest(privateCert, "bad case: private_cert type without property", sm.Secret_SecretType_PrivateCert, false)

	secretDataKV := make(map[string]interface{})
	secretDataKV["key1"] = "val1"

	secretDataKVComplex := make(map[string]interface{})
	secretKVComplex := `{"key1":"val1","key2":"val2","key3":"val3","keyC":{"keyC1":"valC1","keyC2":"valC2"},"special.log":"file-content"}`
	json.Unmarshal([]byte(secretKVComplex), &secretDataKVComplex)

	secretKV := "kv/" + secretUUID

	// bad case: kv type with key which is not in payload
	badSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKV,
		}
		smtc.name = "bad case: kv type with key which is not in payload"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "other-key"
		smtc.expectError = "key other-key does not exist in secret kv/" + secretUUID
	}

	// good case: kv type with property
	setSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKV,
		}
		smtc.name = "good case: kv type with property"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key1"
		smtc.expectedSecret = "val1"
	}

	// good case: kv type with property, returns specific value
	setSecretKVWithKey := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property, returns specific value"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key2"
		smtc.expectedSecret = "val2"
	}

	// good case: kv type with property and path, returns specific value
	setSecretKVWithKeyPath := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property and path, returns specific value"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "keyC.keyC2"
		smtc.expectedSecret = "valC2"
	}

	// good case: kv type with property and dot, returns specific value
	setSecretKVWithKeyDot := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property and dot, returns specific value"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "special.log"
		smtc.expectedSecret = "file-content"
	}

	// good case: kv type without property, returns all
	setSecretKVWithOutKey := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type without property, returns all"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.expectedSecret = secretKVComplex
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setCustomKey),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(badSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretUserPassByID),
		makeValidSecretManagerTestCaseCustom(setSecretUserPassUsername),
		makeValidSecretManagerTestCaseCustom(setSecretUserPassPassword),
		makeValidSecretManagerTestCaseCustom(setSecretIamByID),
		makeValidSecretManagerTestCaseCustom(setSecretIamByName),
		makeValidSecretManagerTestCaseCustom(setSecretCert),
		makeValidSecretManagerTestCaseCustom(setSecretKV),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKey),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyPath),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyDot),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithOutKey),
		makeValidSecretManagerTestCaseCustom(badSecretKV),
		makeValidSecretManagerTestCaseCustom(badSecretCert),
		makeValidSecretManagerTestCaseCustom(setSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(badSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(setSecretPrivateCert),
		makeValidSecretManagerTestCaseCustom(badSecretPrivateCert),
	}

	sm := providerIBM{}
	for k, v := range successCases {
		t.Run(v.name, func(t *testing.T) {
			sm.IBMClient = v.mockClient
			sm.cache = NewCache(10, 1*time.Minute)
			out, err := sm.GetSecret(context.Background(), *v.ref)
			if !ErrorContains(err, v.expectError) {
				t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), v.expectError)
			}
			if string(out) != v.expectedSecret {
				t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
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

	// good case: arbitrary
	setArbitrary := func(smtc *secretManagerTestCase) {
		payload := `{"foo":"bar"}`
		secret := &sm.ArbitrarySecret{
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_Arbitrary),
			Payload:    &payload,
		}
		smtc.name = "good case: arbitrary"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUUID
		smtc.expectedData["arbitrary"] = []byte(payload)
	}

	// good case: username_password
	setSecretUserPass := func(smtc *secretManagerTestCase) {
		secret := &sm.UsernamePasswordSecret{
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_UsernamePassword),
			Username:   &secretUsername,
			Password:   &secretPassword,
		}
		smtc.name = "good case: username_password"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "username_password/" + secretUUID
		smtc.expectedData["username"] = []byte(secretUsername)
		smtc.expectedData["password"] = []byte(secretPassword)
	}

	// good case: iam_credentials
	setSecretIam := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_IamCredentials),
			ApiKey:     utilpointer.String(secretAPIKey),
		}
		smtc.name = "good case: iam_credentials"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "iam_credentials/" + secretUUID
		smtc.expectedData["apikey"] = []byte(secretAPIKey)
	}

	funcCertTest := func(secret sm.SecretIntf, name, certType string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			smtc.name = name
			smtc.apiInput.ID = utilpointer.String(secretUUID)
			smtc.apiOutput = secret
			smtc.ref.Key = certType + "/" + secretUUID
			smtc.expectedData["certificate"] = []byte(secretCertificate)
			smtc.expectedData["private_key"] = []byte(secretPrivateKey)
			smtc.expectedData["intermediate"] = []byte(secretIntermediate)
		}
	}

	// good case: imported_cert
	importedCert := &sm.ImportedCertificate{
		SecretType:   utilpointer.String(sm.Secret_SecretType_ImportedCert),
		Name:         utilpointer.String("testyname"),
		ID:           utilpointer.String(secretUUID),
		Certificate:  utilpointer.String(secretCertificate),
		Intermediate: utilpointer.String(secretIntermediate),
		PrivateKey:   utilpointer.String(secretPrivateKey),
	}
	setSecretCert := funcCertTest(importedCert, "good case: imported_cert", sm.Secret_SecretType_ImportedCert)

	// good case: public_cert
	publicCert := &sm.PublicCertificate{
		SecretType:   utilpointer.String(sm.Secret_SecretType_PublicCert),
		Name:         utilpointer.String("testyname"),
		ID:           utilpointer.String(secretUUID),
		Certificate:  utilpointer.String(secretCertificate),
		Intermediate: utilpointer.String(secretIntermediate),
		PrivateKey:   utilpointer.String(secretPrivateKey),
	}
	setSecretPublicCert := funcCertTest(publicCert, "good case: public_cert", sm.Secret_SecretType_PublicCert)

	// good case: private_cert
	setSecretPrivateCert := func(smtc *secretManagerTestCase) {
		secret := &sm.PrivateCertificate{
			Name:        utilpointer.String("testyname"),
			ID:          utilpointer.String(secretUUID),
			SecretType:  utilpointer.String(sm.Secret_SecretType_PrivateCert),
			Certificate: &secretCertificate,
			PrivateKey:  &secretPrivateKey,
		}
		smtc.name = "good case: private_cert"
		smtc.apiInput.ID = utilpointer.String(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "private_cert/" + secretUUID
		smtc.expectedData["certificate"] = []byte(secretCertificate)
		smtc.expectedData["private_key"] = []byte(secretPrivateKey)
	}

	secretKeyKV := "kv/" + secretUUID
	// good case: kv, no property, return entire payload as key:value pairs
	setSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, no property, return entire payload as key:value pairs"
		smtc.apiInput.ID = core.StringPtr(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["key1"] = []byte("val1")
		smtc.expectedData["key2"] = []byte("val2")
		smtc.expectedData["keyC"] = []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`)
	}

	// good case: kv, with property
	setSecretKVWithProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       utilpointer.String("d5deb37a-7883-4fe2-a5e7-3c15420adc76"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, with property"
		smtc.apiInput.ID = core.StringPtr(secretUUID)
		smtc.ref.Property = "keyC"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["keyC1"] = []byte(`{"keyA":"valA","keyB":"valB"}`)
	}

	// good case: kv, with property and path
	setSecretKVWithPathAndProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       utilpointer.String(secretUUID),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, with property and path"
		smtc.apiInput.ID = core.StringPtr(secretUUID)
		smtc.ref.Property = "keyC.keyC1"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["keyA"] = []byte("valA")
		smtc.expectedData["keyB"] = []byte("valB")
	}

	// bad case: kv, with property and path
	badSecretKVWithUnknownProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       utilpointer.String("testyname"),
			ID:         utilpointer.String(secretUUID),
			SecretType: utilpointer.String(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "bad case: kv, with property and path"
		smtc.apiInput.ID = core.StringPtr(secretUUID)
		smtc.ref.Property = "unknown.property"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectError = "key unknown.property does not exist in secret " + secretKeyKV
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setArbitrary),
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
		makeValidSecretManagerTestCaseCustom(setSecretPrivateCert),
	}

	sm := providerIBM{}
	for k, v := range successCases {
		t.Run(v.name, func(t *testing.T) {
			sm.IBMClient = v.mockClient
			out, err := sm.GetSecretMap(context.Background(), *v.ref)
			if !ErrorContains(err, v.expectError) {
				t.Errorf(" unexpected error: %s, expected: '%s'", err.Error(), v.expectError)
			}
			if err == nil && !reflect.DeepEqual(out, v.expectedData) {
				t.Errorf("[%d] unexpected secret data: expected %+v, got %v", k, v.expectedData, out)
			}
		})
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
	kube := clientfake.NewClientBuilder().WithObjects(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "fake-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"fake-key": []byte("ImAFakeApiKey"),
		},
	}).Build()

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
