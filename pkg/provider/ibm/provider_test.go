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

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/ibm/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type secretManagerTestCase struct {
	mockClient     *fakesm.IBMMockClient
	apiInput       *sm.GetSecretOptions
	apiOutput      *sm.GetSecret
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
	refFrom        *esv1alpha1.ExternalSecretDataFromRemoteRef
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
		ref:            utils.MakeValidRef(),
		refFrom:        utils.MakeValidRefFrom(),
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
		smtc.expectError = "remoteref.Property required for secret type imported_cert"
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
	secretUsername := "user1"
	secretPassword := "P@ssw0rd"
	secretAPIKey := "01234567890"
	secretCertificate := "certificate_value"
	secretPrivateKey := "private_key_value"
	secretIntermediate := "intermediate_value"

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
		smtc.refFrom.Extract.Key = "username_password/test-secret"
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
		smtc.refFrom.Extract.Key = "iam_credentials/test-secret"
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
		smtc.refFrom.Extract.Key = "imported_cert/test-secret"
		smtc.expectedData["certificate"] = []byte(secretCertificate)
		smtc.expectedData["private_key"] = []byte(secretPrivateKey)
		smtc.expectedData["intermediate"] = []byte(secretIntermediate)
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setSecretUserPass),
		makeValidSecretManagerTestCaseCustom(setSecretIam),
		makeValidSecretManagerTestCaseCustom(setSecretCert),
	}

	sm := providerIBM{}
	for k, v := range successCases {
		sm.IBMClient = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.refFrom)
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

	spec := &esv1alpha1.SecretStore{
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				IBM: &esv1alpha1.IBMProvider{
					Auth: esv1alpha1.IBMAuth{
						SecretRef: esv1alpha1.IBMAuthSecretRef{
							SecretAPIKey: v1.SecretKeySelector{
								Name: "fake-secret",
								Key:  "fake-key",
							},
						},
					},
					ServiceURL: &serviceURL,
				},
			},
			RetrySettings: &esv1alpha1.SecretStoreRetrySettings{
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
