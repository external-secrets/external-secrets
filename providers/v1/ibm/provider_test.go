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

package ibm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	sm "github.com/IBM/secrets-manager-go-sdk/v2/secretsmanagerv2"
	"github.com/go-openapi/strfmt"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/providers/v1/ibm/fake"
)

const (
	errExpectedErr       = "wanted error got nil"
	secretKey            = "test-secret"
	secretUUID           = "d5deb37a-7883-4fe2-a5e7-3c15420adc76"
	iamCredentialsSecret = "iam_credentials/"
)

type secretManagerTestCase struct {
	name            string
	mockClient      *fakesm.IBMMockClient
	apiInput        *sm.GetSecretOptions
	apiOutput       sm.SecretIntf
	getByNameInput  *sm.GetSecretByNameTypeOptions
	getByNameOutput sm.SecretIntf
	getByNameError  error
	ref             *esv1.ExternalSecretDataRemoteRef
	serviceURL      *string
	apiErr          error
	expectError     string
	expectedSecret  string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockClient:      &fakesm.IBMMockClient{},
		apiInput:        makeValidAPIInput(),
		ref:             makeValidRef(),
		apiOutput:       makeValidAPIOutput(),
		getByNameInput:  makeValidGetByNameInput(),
		getByNameOutput: makeValidAPIOutput(),
		getByNameError:  nil,
		serviceURL:      nil,
		apiErr:          nil,
		expectError:     "",
		expectedSecret:  "",
		expectedData:    map[string][]byte{},
	}
	mcParams := fakesm.IBMMockClientParams{
		GetSecretOptions:       smtc.apiInput,
		GetSecretOutput:        smtc.apiOutput,
		GetSecretErr:           smtc.apiErr,
		GetSecretByNameOptions: smtc.getByNameInput,
		GetSecretByNameOutput:  smtc.getByNameOutput,
		GetSecretByNameErr:     smtc.getByNameError,
	}
	smtc.mockClient.WithValue(mcParams)
	return &smtc
}

func makeValidRef() *esv1.ExternalSecretDataRemoteRef {
	return &esv1.ExternalSecretDataRemoteRef{
		Key:     secretUUID,
		Version: "default",
	}
}

func makeValidAPIInput() *sm.GetSecretOptions {
	return &sm.GetSecretOptions{
		ID: new(secretUUID),
	}
}

func makeValidAPIOutput() sm.SecretIntf {
	secret := &sm.Secret{
		SecretType: new(sm.Secret_SecretType_Arbitrary),
		Name:       new("testyname"),
		ID:         new(secretUUID),
	}
	var i sm.SecretIntf = secret
	return i
}

func makeValidGetByNameInput() *sm.GetSecretByNameTypeOptions {
	return &sm.GetSecretByNameTypeOptions{}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	mcParams := fakesm.IBMMockClientParams{
		GetSecretOptions:       smtc.apiInput,
		GetSecretOutput:        smtc.apiOutput,
		GetSecretErr:           smtc.apiErr,
		GetSecretByNameOptions: smtc.getByNameInput,
		GetSecretByNameOutput:  smtc.getByNameOutput,
		GetSecretByNameErr:     smtc.apiErr,
	}
	smtc.mockClient.WithValue(mcParams)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretManagerTestCase) {
	smtc.apiErr = errors.New("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *secretManagerTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitializedIBMProvider
}

// simple tests for Validate Store.
func TestValidateStore(t *testing.T) {
	p := providerIBM{}
	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				IBM: &esv1.IBMProvider{},
			},
		},
	}
	_, err := p.ValidateStore(store)
	if err == nil {
		t.Error(errExpectedErr)
	} else if err.Error() != "serviceURL is required" {
		t.Errorf("service URL test failed")
	}
	url := "my-url"
	store.Spec.Provider.IBM.ServiceURL = &url
	_, err = p.ValidateStore(store)
	if err == nil {
		t.Error(errExpectedErr)
	} else if err.Error() != "missing auth method" {
		t.Errorf("KeySelector test failed: expected missing auth method, got %v", err)
	}
	ns := "ns-one"
	store.Spec.Provider.IBM.Auth.SecretRef = &esv1.IBMAuthSecretRef{
		SecretAPIKey: v1.SecretKeySelector{
			Name:      "foo",
			Key:       "bar",
			Namespace: &ns,
		},
		IAMEndpoint: "https://iam.cloud.ibm.com",
	}
	_, err = p.ValidateStore(store)
	if err == nil {
		t.Error(errExpectedErr)
	} else if err.Error() != "namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore" {
		t.Errorf("KeySelector test failed: expected namespace not allowed, got %v", err)
	}

	// add container auth test
	store = &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				IBM: &esv1.IBMProvider{
					ServiceURL: &url,
					Auth: esv1.IBMAuth{
						ContainerAuth: &esv1.IBMAuthContainerAuth{
							Profile:       "Trusted IAM Profile",
							TokenLocation: "/a/path/to/nowhere/that/should/exist",
						},
					},
				},
			},
		},
	}
	_, err = p.ValidateStore(store)
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
	firstValue := "val1"

	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretManagerTestCase) {
		secret := &sm.ArbitrarySecret{
			SecretType: new(sm.Secret_SecretType_Arbitrary),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Payload:    &secretString,
		}
		smtc.name = "good case: default version is set"
		smtc.apiOutput = secret
		smtc.apiInput.ID = new(secretUUID)
		smtc.expectedSecret = secretString
	}

	// good case: custom version set
	setCustomKey := func(smtc *secretManagerTestCase) {
		secret := &sm.ArbitrarySecret{
			SecretType: new(sm.Secret_SecretType_Arbitrary),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Payload:    &secretString,
		}
		smtc.name = "good case: custom version set"
		smtc.ref.Key = "arbitrary/" + secretUUID
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.expectedSecret = secretString
	}

	// bad case: arbitrary type secret which is destroyed
	badArbitSecret := func(smtc *secretManagerTestCase) {
		secret := &sm.ArbitrarySecret{
			SecretType: new(sm.Secret_SecretType_Arbitrary),
			Name:       new("testyname"),
			ID:         new(secretUUID),
		}
		smtc.name = "bad case: arbitrary type without property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUUID
		smtc.expectError = "key payload does not exist in secret " + secretUUID
	}

	// bad case: username_password type without property
	secretUserPass := "username_password/" + secretUUID
	badSecretUserPass := func(smtc *secretManagerTestCase) {
		secret := &sm.UsernamePasswordSecret{
			SecretType: new(sm.Secret_SecretType_UsernamePassword),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Username:   &secretUsername,
			Password:   &secretPassword,
		}
		smtc.name = "bad case: username_password type without property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUserPass
		smtc.expectError = "remoteRef.property required for secret type username_password"
	}

	// good case: username_password type with property
	funcSetUserPass := func(secretName, property, name string) func(smtc *secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.UsernamePasswordSecret{
				SecretType: new(sm.Secret_SecretType_UsernamePassword),
				Name:       new("testyname"),
				ID:         new(secretUUID),
				Username:   &secretUsername,
				Password:   &secretPassword,
			}
			smtc.name = name
			smtc.apiInput.ID = new(secretUUID)
			smtc.apiOutput = secret
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

	// good case: iam_credentials type
	funcSetSecretIam := func(secretName, name string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.IAMCredentialsSecret{
				SecretType: new(sm.Secret_SecretType_IamCredentials),
				Name:       new("testyname"),
				ID:         new(secretUUID),
				ApiKey:     new(secretAPIKey),
			}
			smtc.apiInput.ID = new(secretUUID)
			smtc.name = name
			smtc.apiOutput = secret
			smtc.ref.Key = iamCredentialsSecret + secretName
			smtc.expectedSecret = secretAPIKey
		}
	}

	setSecretIamByID := funcSetSecretIam(secretUUID, "good case: iam_credenatials type - get API Key by ID")

	// good case: iam_credentials type - get API Key by name, providing the secret group ID
	funcSetSecretIamNew := func(secretName, groupName, name string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.IAMCredentialsSecret{
				SecretType: new(sm.Secret_SecretType_IamCredentials),
				Name:       new("testyname"),
				ID:         new(secretUUID),
				ApiKey:     new(secretAPIKey),
			}
			smtc.getByNameInput.Name = &secretName
			smtc.getByNameInput.SecretGroupName = &groupName
			smtc.getByNameInput.SecretType = new(sm.Secret_SecretType_IamCredentials)

			smtc.name = name
			smtc.getByNameOutput = secret
			smtc.ref.Key = groupName + "/" + iamCredentialsSecret + secretName
			smtc.expectedSecret = secretAPIKey
		}
	}
	setSecretIamByNameNew := funcSetSecretIamNew("testyname", "testGroup", "good case: iam_credenatials type - get API Key by name - new mechanism")

	// good case: service_credentials type
	dummySrvCreds := &sm.ServiceCredentialsSecretCredentials{
		Apikey: &secretAPIKey,
	}

	funcSetSecretSrvCred := func(secretName, name string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			secret := &sm.ServiceCredentialsSecret{
				SecretType:  new(sm.Secret_SecretType_ServiceCredentials),
				Name:        new("testyname"),
				ID:          new(secretUUID),
				Credentials: dummySrvCreds,
			}
			smtc.apiInput.ID = new(secretUUID)
			smtc.name = name
			smtc.apiOutput = secret
			smtc.ref.Key = "service_credentials/" + secretName
			smtc.expectedSecret = "{\"apikey\":\"01234567890\"}"
		}
	}

	setSecretSrvCredByID := funcSetSecretSrvCred(secretUUID, "good case: service_credentials type - get creds by ID")

	funcSetCertSecretTest := func(secret sm.SecretIntf, name, certType string, good bool) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			smtc.name = name
			smtc.apiInput.ID = new(secretUUID)
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
		SecretType:   new(sm.Secret_SecretType_ImportedCert),
		Name:         new("testyname"),
		ID:           new(secretUUID),
		Certificate:  new(secretCertificate),
		Intermediate: new("intermediate"),
		PrivateKey:   new("private_key"),
	}
	setSecretCert := funcSetCertSecretTest(importedCert, "good case: imported_cert type with property", sm.Secret_SecretType_ImportedCert, true)

	// good case: imported_cert type without a private_key
	importedCertNoPvtKey := func(smtc *secretManagerTestCase) {
		secret := &sm.ImportedCertificate{
			SecretType:  new(sm.Secret_SecretType_ImportedCert),
			Name:        new("testyname"),
			ID:          new(secretUUID),
			Certificate: new(secretCertificate),
		}
		smtc.name = "good case: imported cert without private key"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "imported_cert/" + secretUUID
		smtc.ref.Property = "private_key"
		smtc.expectedSecret = ""
	}

	// bad case: imported_cert type without property
	badSecretCert := funcSetCertSecretTest(importedCert, "bad case: imported_cert type without property", sm.Secret_SecretType_ImportedCert, false)

	// good case: public_cert type with property
	publicCert := &sm.PublicCertificate{
		SecretType:   new(sm.Secret_SecretType_PublicCert),
		Name:         new("testyname"),
		ID:           new(secretUUID),
		Certificate:  new(secretCertificate),
		Intermediate: new("intermediate"),
		PrivateKey:   new("private_key"),
	}
	setSecretPublicCert := funcSetCertSecretTest(publicCert, "good case: public_cert type with property", sm.Secret_SecretType_PublicCert, true)

	// bad case: public_cert type without property
	badSecretPublicCert := funcSetCertSecretTest(publicCert, "bad case: public_cert type without property", sm.Secret_SecretType_PublicCert, false)

	// good case: private_cert type with property
	privateCert := &sm.PrivateCertificate{
		SecretType:  new(sm.Secret_SecretType_PublicCert),
		Name:        new("testyname"),
		ID:          new(secretUUID),
		Certificate: new(secretCertificate),
		PrivateKey:  new("private_key"),
	}
	setSecretPrivateCert := funcSetCertSecretTest(privateCert, "good case: private_cert type with property", sm.Secret_SecretType_PrivateCert, true)

	// bad case: private_cert type without property
	badSecretPrivateCert := funcSetCertSecretTest(privateCert, "bad case: private_cert type without property", sm.Secret_SecretType_PrivateCert, false)

	secretDataKV := make(map[string]any)
	secretDataKV["key1"] = firstValue

	secretDataKVComplex := make(map[string]any)
	secretKVComplex := `{"key1":"val1","key2":"val2","key3":"val3","keyC":{"keyC1":"valC1","keyC2":"valC2"},"special.log":"file-content"}`
	json.Unmarshal([]byte(secretKVComplex), &secretDataKVComplex)

	secretKV := "kv/" + secretUUID

	// bad case: kv type with key which is not in payload
	badSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKV,
		}
		smtc.name = "bad case: kv type with key which is not in payload"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "other-key"
		smtc.expectError = "key other-key does not exist in secret kv/" + secretUUID
	}

	// good case: kv type with property
	setSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKV,
		}
		smtc.name = "good case: kv type with property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key1"
		smtc.expectedSecret = firstValue
	}

	// good case: kv type with property, returns specific value
	setSecretKVWithKey := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "key2"
		smtc.expectedSecret = "val2"
	}

	// good case: kv type with property and path, returns specific value
	setSecretKVWithKeyPath := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property and path, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "keyC.keyC2"
		smtc.expectedSecret = "valC2"
	}

	// good case: kv type with property and dot, returns specific value
	setSecretKVWithKeyDot := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type with property and dot, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.ref.Property = "special.log"
		smtc.expectedSecret = "file-content"
	}

	// good case: kv type without property, returns all
	setSecretKVWithOutKey := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			SecretType: new(sm.Secret_SecretType_Kv),
			Name:       new("testyname"),
			ID:         new(secretUUID),
			Data:       secretDataKVComplex,
		}
		smtc.name = "good case: kv type without property, returns all"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKV
		smtc.expectedSecret = secretKVComplex
	}

	customCredentialsSecretCredentialsContent := make(map[string]any)
	customCredentialsSecretCredentialsContent["key1"] = firstValue

	customCredentialsSecretCredentialsContentComplex := make(map[string]any)
	customCredentialsSecretComplex := `{"key1":"val1","key2":"val2","key3":"val3","keyC":{"keyC1":"valC1","keyC2":"valC2"},"special.log":"file-content"}`
	json.Unmarshal([]byte(customCredentialsSecretComplex), &customCredentialsSecretCredentialsContentComplex)

	secretCustomCredentials := "custom_credentials/" + secretUUID

	// bad case: custom credentials type with key which is not in payload
	badSecretCustomCredentials := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContent,
		}
		smtc.name = "bad case: custom credentials type with key which is not in payload"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.ref.Property = "other-key"
		smtc.expectError = "key other-key does not exist in secret custom_credentials/" + secretUUID
	}

	// good case: custom credentials type with property
	setSecretCustomCredentials := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContent,
		}
		smtc.name = "good case: custom_credentials type with property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.ref.Property = "key1"
		smtc.expectedSecret = firstValue
	}

	// good case: custom_credentials type with property, returns specific value
	setSecretCustomCredentialsWithKey := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContentComplex,
		}
		smtc.name = "good case: custom_credentials type with property, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.ref.Property = "key2"
		smtc.expectedSecret = "val2"
	}

	// good case: custom_credentials type with property and path, returns specific value
	setSecretCustomCredentialsWithKeyPath := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContentComplex,
		}
		smtc.name = "good case: custom_credentials type with property and path, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.ref.Property = "keyC.keyC2"
		smtc.expectedSecret = "valC2"
	}

	// good case: custom_credentials type with property and dot, returns specific value
	setSecretCustomCredentialsWithKeyDot := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContentComplex,
		}
		smtc.name = "good case: custom_credentials type with property and dot, returns specific value"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.ref.Property = "special.log"
		smtc.expectedSecret = "file-content"
	}

	// good case: custom_credentials type without property, returns all
	setSecretCustomCredentialsWithOutKey := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			CredentialsContent: customCredentialsSecretCredentialsContentComplex,
		}
		smtc.name = "good case: custom_credentials type without property, returns all"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretCustomCredentials
		smtc.expectedSecret = customCredentialsSecretComplex
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setCustomKey),
		makeValidSecretManagerTestCaseCustom(badArbitSecret),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(badSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretUserPassByID),
		makeValidSecretManagerTestCaseCustom(setSecretIamByID),
		makeValidSecretManagerTestCaseCustom(setSecretCert),
		makeValidSecretManagerTestCaseCustom(setSecretKV),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKey),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyPath),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithKeyDot),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithOutKey),
		makeValidSecretManagerTestCaseCustom(badSecretKV),
		makeValidSecretManagerTestCaseCustom(badSecretCert),
		makeValidSecretManagerTestCaseCustom(importedCertNoPvtKey),
		makeValidSecretManagerTestCaseCustom(setSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(badSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(setSecretPrivateCert),
		makeValidSecretManagerTestCaseCustom(badSecretPrivateCert),
		makeValidSecretManagerTestCaseCustom(setSecretIamByNameNew),
		makeValidSecretManagerTestCaseCustom(setSecretSrvCredByID),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentials),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithKey),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithKeyPath),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithKeyDot),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithOutKey),
		makeValidSecretManagerTestCaseCustom(badSecretCustomCredentials),
	}

	sm := providerIBM{}
	for _, v := range successCases {
		t.Run(v.name, func(t *testing.T) {
			sm.IBMClient = v.mockClient
			out, err := sm.GetSecret(context.Background(), *v.ref)
			if !ErrorContains(err, v.expectError) {
				t.Errorf("unexpected error:\n%s, expected:\n'%s'", err.Error(), v.expectError)
			}
			if string(out) != v.expectedSecret {
				t.Errorf("unexpected secret: expected:\n%s\ngot:\n%s", v.expectedSecret, string(out))
			}
		})
	}
}

func TestGetSecretMap(t *testing.T) {
	secretUsername := "user1"
	secretPassword := "P@ssw0rd"
	secretAPIKey := "01234567890"
	nilValue := "<nil>"
	secretCertificate := "certificate_value"
	secretPrivateKey := "private_key_value"
	secretIntermediate := "intermediate_value"
	timeValue := "0001-01-01T00:00:00.000Z"

	secretComplex := map[string]any{
		"key1": "val1",
		"key2": "val2",
		"keyC": map[string]any{
			"keyC1": map[string]string{
				"keyA": "valA",
				"keyB": "valB",
			},
		},
	}

	dummySrvCreds := &sm.ServiceCredentialsSecretCredentials{
		Apikey: &secretAPIKey,
	}

	// good case: arbitrary
	setArbitrary := func(smtc *secretManagerTestCase) {
		payload := `{"foo":"bar"}`
		secret := &sm.ArbitrarySecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_Arbitrary),
			Payload:    &payload,
		}
		smtc.name = "good case: arbitrary"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUUID
		smtc.expectedData["arbitrary"] = []byte(payload)
	}

	// good case: username_password
	setSecretUserPass := func(smtc *secretManagerTestCase) {
		secret := &sm.UsernamePasswordSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_UsernamePassword),
			Username:   &secretUsername,
			Password:   &secretPassword,
		}
		smtc.name = "good case: username_password"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "username_password/" + secretUUID
		smtc.expectedData["username"] = []byte(secretUsername)
		smtc.expectedData["password"] = []byte(secretPassword)
	}

	// good case: iam_credentials
	setSecretIam := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_IamCredentials),
			ApiKey:     new(secretAPIKey),
		}
		smtc.name = "good case: iam_credentials"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = iamCredentialsSecret + secretUUID
		smtc.expectedData["apikey"] = []byte(secretAPIKey)
	}

	// good case: iam_credentials by name using new mechanism
	setSecretIamByName := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_IamCredentials),
			ApiKey:     new(secretAPIKey),
		}
		smtc.name = "good case: iam_credentials by name using new mechanism"
		smtc.getByNameInput.Name = new("testyname")
		smtc.getByNameInput.SecretGroupName = new("groupName")
		smtc.getByNameInput.SecretType = new(sm.Secret_SecretType_IamCredentials)

		smtc.getByNameOutput = secret
		smtc.apiOutput = secret
		smtc.ref.Key = "groupName/" + iamCredentialsSecret + "testyname"
		smtc.expectedData["apikey"] = []byte(secretAPIKey)
	}

	// bad case: iam_credentials of a destroyed secret
	badSecretIam := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_IamCredentials),
		}
		smtc.name = "bad case: iam_credentials of a destroyed secret"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = iamCredentialsSecret + secretUUID
		smtc.expectError = "key api_key does not exist in secret " + secretUUID
	}

	funcCertTest := func(secret sm.SecretIntf, name, certType string) func(*secretManagerTestCase) {
		return func(smtc *secretManagerTestCase) {
			smtc.name = name
			smtc.apiInput.ID = new(secretUUID)
			smtc.apiOutput = secret
			smtc.ref.Key = certType + "/" + secretUUID
			smtc.expectedData["certificate"] = []byte(secretCertificate)
			smtc.expectedData["private_key"] = []byte(secretPrivateKey)
			smtc.expectedData["intermediate"] = []byte(secretIntermediate)
		}
	}

	// good case: service_credentials
	setSecretSrvCreds := func(smtc *secretManagerTestCase) {
		secret := &sm.ServiceCredentialsSecret{
			Name:        new("testyname"),
			ID:          new(secretUUID),
			SecretType:  new(sm.Secret_SecretType_IamCredentials),
			Credentials: dummySrvCreds,
		}
		smtc.name = "good case: service_credentials"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "service_credentials/" + secretUUID
		smtc.expectedData["credentials"] = fmt.Appendf(nil, "%+v", map[string]string{"apikey": secretAPIKey})
	}

	// good case: imported_cert
	importedCert := &sm.ImportedCertificate{
		SecretType:   new(sm.Secret_SecretType_ImportedCert),
		Name:         new("testyname"),
		ID:           new(secretUUID),
		Certificate:  new(secretCertificate),
		Intermediate: new(secretIntermediate),
		PrivateKey:   new(secretPrivateKey),
	}
	setSecretCert := funcCertTest(importedCert, "good case: imported_cert", sm.Secret_SecretType_ImportedCert)

	// good case: public_cert
	publicCert := &sm.PublicCertificate{
		SecretType:   new(sm.Secret_SecretType_PublicCert),
		Name:         new("testyname"),
		ID:           new(secretUUID),
		Certificate:  new(secretCertificate),
		Intermediate: new(secretIntermediate),
		PrivateKey:   new(secretPrivateKey),
	}
	setSecretPublicCert := funcCertTest(publicCert, "good case: public_cert", sm.Secret_SecretType_PublicCert)

	// good case: private_cert
	setSecretPrivateCert := func(smtc *secretManagerTestCase) {
		secret := &sm.PrivateCertificate{
			Name:        new("testyname"),
			ID:          new(secretUUID),
			SecretType:  new(sm.Secret_SecretType_PrivateCert),
			Certificate: &secretCertificate,
			PrivateKey:  &secretPrivateKey,
		}
		smtc.name = "good case: private_cert"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "private_cert/" + secretUUID
		smtc.expectedData["certificate"] = []byte(secretCertificate)
		smtc.expectedData["private_key"] = []byte(secretPrivateKey)
	}

	// good case: arbitrary with metadata
	setArbitraryWithMetadata := func(smtc *secretManagerTestCase) {
		payload := `{"foo":"bar"}`
		secret := &sm.ArbitrarySecret{
			CreatedBy:  new("testCreatedBy"),
			CreatedAt:  &strfmt.DateTime{},
			Downloaded: new(false),
			Labels:     []string{"abc", "def", "xyz"},
			LocksTotal: new(int64(20)),
			Payload:    &payload,
		}
		smtc.name = "good case: arbitrary with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"arbitrary":       []byte(payload),
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"payload":         []byte(payload),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"versions_total":  []byte(nilValue),
		}
	}

	// good case: iam_credentials with metadata
	setSecretIamWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			CreatedBy:  new("testCreatedBy"),
			CreatedAt:  &strfmt.DateTime{},
			Downloaded: new(false),
			Labels:     []string{"abc", "def", "xyz"},
			LocksTotal: new(int64(20)),
			ApiKey:     new(secretAPIKey),
		}
		smtc.name = "good case: iam_credentials with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = iamCredentialsSecret + secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"api_key":         []byte(secretAPIKey),
			"apikey":          []byte(secretAPIKey),
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"reuse_api_key":   []byte(nilValue),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"ttl":             []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"versions_total":  []byte(nilValue),
		}
	}

	// "good case: username_password with metadata
	setSecretUserPassWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.UsernamePasswordSecret{
			CreatedBy:  new("testCreatedBy"),
			CreatedAt:  &strfmt.DateTime{},
			Downloaded: new(false),
			Labels:     []string{"abc", "def", "xyz"},
			LocksTotal: new(int64(20)),
			Username:   &secretUsername,
			Password:   &secretPassword,
		}
		smtc.name = "good case: username_password with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "username_password/" + secretUUID
		smtc.expectedData["username"] = []byte(secretUsername)
		smtc.expectedData["password"] = []byte(secretPassword)
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"password":        []byte(secretPassword),
			"rotation":        []byte(nilValue),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"username":        []byte(secretUsername),
			"versions_total":  []byte(nilValue),
		}
	}

	// good case: imported_cert with metadata
	setimportedCertWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.ImportedCertificate{
			CreatedBy:    new("testCreatedBy"),
			CreatedAt:    &strfmt.DateTime{},
			Downloaded:   new(false),
			Labels:       []string{"abc", "def", "xyz"},
			LocksTotal:   new(int64(20)),
			Certificate:  new(secretCertificate),
			Intermediate: new(secretIntermediate),
			PrivateKey:   new(secretPrivateKey),
		}
		smtc.name = "good case: imported_cert with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "imported_cert" + "/" + secretUUID

		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"certificate":     []byte(secretCertificate),
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"intermediate":    []byte(secretIntermediate),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"private_key":     []byte(secretPrivateKey),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"versions_total":  []byte(nilValue),
		}
	}

	// good case: imported_cert without a private_key
	setimportedCertWithNoPvtKey := func(smtc *secretManagerTestCase) {
		secret := &sm.ImportedCertificate{
			CreatedBy:    new("testCreatedBy"),
			CreatedAt:    &strfmt.DateTime{},
			Downloaded:   new(false),
			Labels:       []string{"abc", "def", "xyz"},
			LocksTotal:   new(int64(20)),
			Certificate:  new(secretCertificate),
			Intermediate: new(secretIntermediate),
		}
		smtc.name = "good case: imported_cert without private key"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "imported_cert/" + secretUUID

		smtc.expectedData = map[string][]byte{
			"certificate":  []byte(secretCertificate),
			"intermediate": []byte(secretIntermediate),
			"private_key":  []byte(""),
		}
	}

	// good case: public_cert with metadata
	setPublicCertWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.PublicCertificate{
			CreatedBy:    new("testCreatedBy"),
			CreatedAt:    &strfmt.DateTime{},
			Downloaded:   new(false),
			Labels:       []string{"abc", "def", "xyz"},
			LocksTotal:   new(int64(20)),
			Certificate:  new(secretCertificate),
			Intermediate: new(secretIntermediate),
			PrivateKey:   new(secretPrivateKey),
		}
		smtc.name = "good case: public_cert with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "public_cert" + "/" + secretUUID

		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"certificate":     []byte(secretCertificate),
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"intermediate":    []byte(secretIntermediate),
			"key_algorithm":   []byte(nilValue),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"private_key":     []byte(secretPrivateKey),
			"rotation":        []byte(nilValue),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"versions_total":  []byte(nilValue),
		}
	}

	// good case: private_cert with metadata
	setPrivateCertWithMetadata := func(smtc *secretManagerTestCase) {
		expirationDate := &strfmt.DateTime{}
		secret := &sm.PrivateCertificate{
			CreatedBy:      new("testCreatedBy"),
			CreatedAt:      &strfmt.DateTime{},
			Downloaded:     new(false),
			Labels:         []string{"abc", "def", "xyz"},
			LocksTotal:     new(int64(20)),
			Certificate:    new(secretCertificate),
			PrivateKey:     new(secretPrivateKey),
			ExpirationDate: expirationDate,
		}
		smtc.name = "good case: private_cert with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "private_cert" + "/" + secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"certificate":          []byte(secretCertificate),
			"certificate_template": []byte(nilValue),
			"common_name":          []byte(nilValue),
			"created_at":           []byte(timeValue),
			"created_by":           []byte(*secret.CreatedBy),
			"crn":                  []byte(nilValue),
			"downloaded":           []byte(strconv.FormatBool(*secret.Downloaded)),
			"expiration_date":      []byte(expirationDate.String()),
			"id":                   []byte(nilValue),
			"issuer":               []byte(nilValue),
			"labels":               []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":          []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"private_key":          []byte(secretPrivateKey),
			"secret_group_id":      []byte(nilValue),
			"secret_type":          []byte(nilValue),
			"serial_number":        []byte(nilValue),
			"signing_algorithm":    []byte(nilValue),
			"updated_at":           []byte(nilValue),
			"validity":             []byte(nilValue),
			"versions_total":       []byte(nilValue),
		}
	}

	// good case: kv with property and metadata
	setSecretKVWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			CreatedBy:  new("testCreatedBy"),
			CreatedAt:  &strfmt.DateTime{},
			Downloaded: new(false),
			Labels:     []string{"abc", "def", "xyz"},
			LocksTotal: new(int64(20)),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, with property and with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "kv/" + secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"created_at":      []byte(timeValue),
			"created_by":      []byte(*secret.CreatedBy),
			"crn":             []byte(nilValue),
			"data":            []byte("map[key1:val1 key2:val2 keyC:map[keyC1:map[keyA:valA keyB:valB]]]"),
			"downloaded":      []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":              []byte(nilValue),
			"key1":            []byte("val1"),
			"key2":            []byte("val2"),
			"keyC":            []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`),
			"labels":          []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":     []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"secret_group_id": []byte(nilValue),
			"secret_type":     []byte(nilValue),
			"updated_at":      []byte(nilValue),
			"versions_total":  []byte(nilValue),
		}
	}

	// good case: custom_credentials with property and metadata
	setSecretCustomCredentialsWithMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			CreatedBy:          new("testCreatedBy"),
			CreatedAt:          &strfmt.DateTime{},
			Downloaded:         new(false),
			Labels:             []string{"abc", "def", "xyz"},
			LocksTotal:         new(int64(20)),
			CredentialsContent: secretComplex,
		}
		smtc.name = "good case: custom_credentials, with property and with metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = "custom_credentials/" + secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyFetch
		smtc.expectedData = map[string][]byte{
			"created_at":          []byte(timeValue),
			"created_by":          []byte(*secret.CreatedBy),
			"crn":                 []byte(nilValue),
			"credentials_content": []byte("map[key1:val1 key2:val2 keyC:map[keyC1:map[keyA:valA keyB:valB]]]"),
			"downloaded":          []byte(strconv.FormatBool(*secret.Downloaded)),
			"id":                  []byte(nilValue),
			"key1":                []byte("val1"),
			"key2":                []byte("val2"),
			"keyC":                []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`),
			"labels":              []byte("[" + strings.Join(secret.Labels, " ") + "]"),
			"locks_total":         []byte(strconv.Itoa(int(*secret.LocksTotal))),
			"secret_group_id":     []byte(nilValue),
			"secret_type":         []byte(nilValue),
			"updated_at":          []byte(nilValue),
			"versions_total":      []byte(nilValue),
			"configuration":       []byte(nilValue),
		}
	}

	// good case: iam_credentials without metadata
	setSecretIamWithoutMetadata := func(smtc *secretManagerTestCase) {
		secret := &sm.IAMCredentialsSecret{
			CreatedBy:  new("testCreatedBy"),
			CreatedAt:  &strfmt.DateTime{},
			Downloaded: new(false),
			Labels:     []string{"abc", "def", "xyz"},
			LocksTotal: new(int64(20)),
			ApiKey:     new(secretAPIKey),
		}
		smtc.name = "good case: iam_credentials without metadata"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = iamCredentialsSecret + secretUUID
		smtc.ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicyNone
		smtc.expectedData = map[string][]byte{
			"apikey": []byte(secretAPIKey),
		}
	}

	secretKeyKV := "kv/" + secretUUID
	// good case: kv, no property, return entire payload as key:value pairs
	setSecretKV := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, no property, return entire payload as key:value pairs"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["key1"] = []byte("val1")
		smtc.expectedData["key2"] = []byte("val2")
		smtc.expectedData["keyC"] = []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`)
	}

	// good case: kv, with property
	setSecretKVWithProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       new("d5deb37a-7883-4fe2-a5e7-3c15420adc76"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, with property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "keyC"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["keyC1"] = []byte(`{"keyA":"valA","keyB":"valB"}`)
	}

	// good case: kv, with property and path
	setSecretKVWithPathAndProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       new(secretUUID),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "good case: kv, with property and path"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "keyC.keyC1"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectedData["keyA"] = []byte("valA")
		smtc.expectedData["keyB"] = []byte("valB")
	}

	// bad case: kv, with property and path
	badSecretKVWithUnknownProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.KVSecret{
			Name:       new("testyname"),
			ID:         new(secretUUID),
			SecretType: new(sm.Secret_SecretType_Kv),
			Data:       secretComplex,
		}
		smtc.name = "bad case: kv, with property and path"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "unknown.property"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyKV
		smtc.expectError = "key unknown.property does not exist in secret " + secretKeyKV
	}

	secretKeyCustomCredentials := "custom_credentials/" + secretUUID
	// good case: custom_credentials, no property, return entire payload as key:value pairs
	setSecretCustomCredentials := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			CredentialsContent: secretComplex,
		}
		smtc.name = "good case: custom_credentials, no property, return entire payload as key:value pairs"
		smtc.apiInput.ID = new(secretUUID)
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyCustomCredentials
		smtc.expectedData["key1"] = []byte("val1")
		smtc.expectedData["key2"] = []byte("val2")
		smtc.expectedData["keyC"] = []byte(`{"keyC1":{"keyA":"valA","keyB":"valB"}}`)
	}

	// good case: custom_credentials, with property
	setSecretCustomCredentialsWithProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			Name:               new("d5deb37a-7883-4fe2-a5e7-3c15420adc76"),
			ID:                 new(secretUUID),
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			CredentialsContent: secretComplex,
		}
		smtc.name = "good case: custom_credentials, with property"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "keyC"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyCustomCredentials
		smtc.expectedData["keyC1"] = []byte(`{"keyA":"valA","keyB":"valB"}`)
	}

	// good case: custom_credentials, with property and path
	setSecretCustomCredentialsWithPathAndProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			Name:               new(secretUUID),
			ID:                 new(secretUUID),
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			CredentialsContent: secretComplex,
		}
		smtc.name = "good case: custom_credentials, with property and path"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "keyC.keyC1"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyCustomCredentials
		smtc.expectedData["keyA"] = []byte("valA")
		smtc.expectedData["keyB"] = []byte("valB")
	}

	// bad case: custom_credentials, with property and path
	badSecretCustomCredentialsWithUnknownProperty := func(smtc *secretManagerTestCase) {
		secret := &sm.CustomCredentialsSecret{
			Name:               new("testyname"),
			ID:                 new(secretUUID),
			SecretType:         new(sm.Secret_SecretType_CustomCredentials),
			CredentialsContent: secretComplex,
		}
		smtc.name = "bad case: custom_credentials, with property and path"
		smtc.apiInput.ID = new(secretUUID)
		smtc.ref.Property = "unknown.property"
		smtc.apiOutput = secret
		smtc.ref.Key = secretKeyCustomCredentials
		smtc.expectError = "key unknown.property does not exist in secret " + secretKeyCustomCredentials
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(badSecretIam),
		makeValidSecretManagerTestCaseCustom(setSecretSrvCreds),
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
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentials),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithProperty),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithPathAndProperty),
		makeValidSecretManagerTestCaseCustom(badSecretCustomCredentialsWithUnknownProperty),
		makeValidSecretManagerTestCaseCustom(setSecretPublicCert),
		makeValidSecretManagerTestCaseCustom(setSecretPrivateCert),
		makeValidSecretManagerTestCaseCustom(setimportedCertWithNoPvtKey),
		makeValidSecretManagerTestCaseCustom(setSecretIamWithMetadata),
		makeValidSecretManagerTestCaseCustom(setArbitraryWithMetadata),
		makeValidSecretManagerTestCaseCustom(setSecretUserPassWithMetadata),
		makeValidSecretManagerTestCaseCustom(setimportedCertWithMetadata),
		makeValidSecretManagerTestCaseCustom(setPublicCertWithMetadata),
		makeValidSecretManagerTestCaseCustom(setPrivateCertWithMetadata),
		makeValidSecretManagerTestCaseCustom(setSecretKVWithMetadata),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentialsWithMetadata),
		makeValidSecretManagerTestCaseCustom(setSecretIamWithoutMetadata),
		makeValidSecretManagerTestCaseCustom(setSecretIamByName),
		makeValidSecretManagerTestCaseCustom(setSecretCustomCredentials),
	}

	sm := providerIBM{}
	for _, v := range successCases {
		t.Run(v.name, func(t *testing.T) {
			sm.IBMClient = v.mockClient
			out, err := sm.GetSecretMap(context.Background(), *v.ref)
			if !ErrorContains(err, v.expectError) {
				t.Errorf("unexpected error: %s, expected: '%s'", err.Error(), v.expectError)
			}
			if err == nil && !reflect.DeepEqual(out, v.expectedData) {
				// Add some debug logs because t.Errorf is hard to debug at a glance.
				for k, val := range out {
					if val2, ok := v.expectedData[k]; !ok {
						t.Logf("%s was not in expected data\n", k)
					} else if !bytes.Equal(val2, val) {
						fmt.Printf("%s did not equal expected value: %s\n", string(val), string(val2))
					}
				}

				for k := range v.expectedData {
					if _, ok := out[k]; !ok {
						t.Logf("%s expected key was not in out\n", k)
					}
				}
				t.Errorf("unexpected secret data: expected:\n%+v\ngot:\n%+v", v.expectedData, out)
			}
		})
	}
}

func TestValidRetryInput(t *testing.T) {
	sm := providerIBM{}

	invalid := "Invalid"
	serviceURL := "http://fake-service-url.cool"

	spec := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				IBM: &esv1.IBMProvider{
					Auth: esv1.IBMAuth{
						SecretRef: &esv1.IBMAuthSecretRef{
							SecretAPIKey: v1.SecretKeySelector{
								Name: "fake-secret",
								Key:  "fake-key",
							},
						},
					},
					ServiceURL: &serviceURL,
				},
			},
			RetrySettings: &esv1.SecretStoreRetrySettings{
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

func arbitraryMeta(id, name, group string) *sm.ArbitrarySecretMetadata {
	return &sm.ArbitrarySecretMetadata{
		ID:            new(id),
		Name:          new(name),
		SecretType:    new(sm.Secret_SecretType_Arbitrary),
		SecretGroupID: new(group),
	}
}

func arbitrarySecret(payload string) *sm.ArbitrarySecret {
	return &sm.ArbitrarySecret{
		SecretType: new(sm.Secret_SecretType_Arbitrary),
		Payload:    new(payload),
	}
}

func page(total int64, items ...sm.SecretMetadataIntf) *sm.SecretMetadataPaginatedCollection {
	return &sm.SecretMetadataPaginatedCollection{
		TotalCount: new(total),
		Secrets:    items,
	}
}

const (
	idAlpha  = "11111111-1111-1111-1111-111111111111"
	idBeta   = "22222222-2222-2222-2222-222222222222"
	idGone   = "33333333-3333-3333-3333-333333333333"
	idCert   = "44444444-4444-4444-4444-444444444444"
	idUUIDly = "55555555-5555-5555-5555-555555555555"
)

func TestGetAllSecrets(t *testing.T) {
	groups := []sm.SecretGroup{
		{ID: new("grp-1"), Name: new("prod")},
		{ID: new("grp-2"), Name: new("staging")},
	}
	prodPath := "prod"
	missingPath := "nope"

	for _, tc := range []struct {
		name        string
		ref         esv1.ExternalSecretFind
		pages       []*sm.SecretMetadataPaginatedCollection
		byID        map[string]sm.SecretIntf
		notFoundIDs map[string]bool
		want        map[string][]byte
		wantErr     string
		wantOptions []wantOption
	}{
		{
			name: "no criteria returns every secret, keyed group/type/name",
			ref:  esv1.ExternalSecretFind{},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(2, arbitraryMeta(idAlpha, "alpha", "grp-1"), arbitraryMeta(idBeta, "beta", "grp-2")),
			},
			byID: map[string]sm.SecretIntf{
				idAlpha: arbitrarySecret("alpha-value"),
				idBeta:  arbitrarySecret("beta-value"),
			},
			want: map[string][]byte{
				"prod/arbitrary/alpha":   []byte("alpha-value"),
				"staging/arbitrary/beta": []byte("beta-value"),
			},
			wantOptions: []wantOption{{offset: 0, limit: 200, groups: nil}},
		},
		{
			name: "regexp selects a subset",
			ref:  esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "^al"}},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(2, arbitraryMeta(idAlpha, "alpha", "grp-1"), arbitraryMeta(idBeta, "beta", "grp-2")),
			},
			byID:        map[string]sm.SecretIntf{idAlpha: arbitrarySecret("alpha-value")},
			want:        map[string][]byte{"prod/arbitrary/alpha": []byte("alpha-value")},
			wantOptions: []wantOption{{offset: 0, limit: 200, groups: nil}},
		},
		{
			name: "pages until TotalCount is covered, advancing the offset",
			ref:  esv1.ExternalSecretFind{},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(2, arbitraryMeta(idAlpha, "alpha", "grp-1")),
				page(2, arbitraryMeta(idBeta, "beta", "grp-1")),
			},
			byID: map[string]sm.SecretIntf{
				idAlpha: arbitrarySecret("alpha-value"),
				idBeta:  arbitrarySecret("beta-value"),
			},
			want: map[string][]byte{
				"prod/arbitrary/alpha": []byte("alpha-value"),
				"prod/arbitrary/beta":  []byte("beta-value"),
			},
			wantOptions: []wantOption{
				{offset: 0, limit: 200, groups: nil},
				{offset: 1, limit: 200, groups: nil},
			},
		},
		{
			name:        "find.path resolves to a group ID and is pushed to the API",
			ref:         esv1.ExternalSecretFind{Path: &prodPath},
			pages:       []*sm.SecretMetadataPaginatedCollection{page(1, arbitraryMeta(idAlpha, "alpha", "grp-1"))},
			byID:        map[string]sm.SecretIntf{idAlpha: arbitrarySecret("alpha-value")},
			want:        map[string][]byte{"prod/arbitrary/alpha": []byte("alpha-value")},
			wantOptions: []wantOption{{offset: 0, limit: 200, groups: []string{"grp-1"}}},
		},
		{
			name: "a UUID-shaped name is fetched by ID, not by name",
			ref:  esv1.ExternalSecretFind{},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(1, arbitraryMeta(idUUIDly, idBeta, "grp-1")),
			},
			byID: map[string]sm.SecretIntf{idUUIDly: arbitrarySecret("correct-value")},
			want: map[string][]byte{"prod/arbitrary/" + idBeta: []byte("correct-value")},
		},
		{
			name: "a secret deleted between listing and read is skipped",
			ref:  esv1.ExternalSecretFind{},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(2, arbitraryMeta(idAlpha, "alpha", "grp-1"), arbitraryMeta(idGone, "gone", "grp-1")),
			},
			byID:        map[string]sm.SecretIntf{idAlpha: arbitrarySecret("alpha-value")},
			notFoundIDs: map[string]bool{idGone: true},
			want:        map[string][]byte{"prod/arbitrary/alpha": []byte("alpha-value")},
		},
		{
			name:    "unknown find.path is an error",
			ref:     esv1.ExternalSecretFind{Path: &missingPath},
			wantErr: "does not match any secret group",
		},
		{
			name:    "tags are not supported",
			ref:     esv1.ExternalSecretFind{Tags: map[string]string{"env": "prod"}},
			wantErr: "find.tags is not supported",
		},
		{
			name:    "invalid regexp is an error",
			ref:     esv1.ExternalSecretFind{Name: &esv1.FindName{RegExp: "[unterminated"}},
			wantErr: "could not compile",
		},
		{
			name: "a type needing a property fails loudly and names the secret",
			ref:  esv1.ExternalSecretFind{},
			pages: []*sm.SecretMetadataPaginatedCollection{
				page(1, &sm.PublicCertificateMetadata{
					ID:            new(idCert),
					Name:          new("web-tls"),
					SecretType:    new(sm.Secret_SecretType_PublicCert),
					SecretGroupID: new("grp-1"),
				}),
			},
			wantErr: `secret "web-tls" has type public_cert`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mc := &fakesm.IBMMockClient{
				Groups:      groups,
				ListPages:   tc.pages,
				ByID:        tc.byID,
				NotFoundIDs: tc.notFoundIDs,
			}
			provider := &providerIBM{IBMClient: mc}

			got, err := provider.GetAllSecrets(context.Background(), tc.ref)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got result %v", tc.wantErr, got)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected an error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			if tc.wantOptions != nil {
				assertListOptions(t, mc.ListCalled, tc.wantOptions)
			}
		})
	}
}

type wantOption struct {
	offset int64
	limit  int64
	groups []string
}

// assertListOptions checks the paging arithmetic and the server-side group
// filter, which a call count alone would not catch.
func assertListOptions(t *testing.T, got []*sm.ListSecretsOptions, want []wantOption) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("made %d list calls, want %d", len(got), len(want))
	}
	for i, w := range want {
		if got[i].Offset == nil || *got[i].Offset != w.offset {
			t.Errorf("call %d: offset = %v, want %d", i, got[i].Offset, w.offset)
		}
		if got[i].Limit == nil || *got[i].Limit != w.limit {
			t.Errorf("call %d: limit = %v, want %d", i, got[i].Limit, w.limit)
		}
		if !reflect.DeepEqual(got[i].Groups, w.groups) {
			t.Errorf("call %d: groups = %v, want %v", i, got[i].Groups, w.groups)
		}
	}
}
