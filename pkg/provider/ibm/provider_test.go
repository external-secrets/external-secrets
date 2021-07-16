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
	utilpointer "k8s.io/utils/pointer"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/ibm/fake"
)

type secretManagerTestCase struct {
	mockClient     *fakesm.IBMMockClient
	apiInput       *sm.GetSecretOptions
	apiOutput      *sm.GetSecret
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
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

func makeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
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

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestIBMSecretManagerGetSecret(t *testing.T) {
	secretData := make(map[string]interface{})
	secretValue := "changedvalue"
	secretData["payload"] = secretValue
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
		smtc.expectedSecret = secretValue
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
		smtc.expectedSecret = secretValue
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCase(),
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setCustomKey),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
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
	// good case: default version & deserialization
	setDeserialization := func(smtc *secretManagerTestCase) {
		secretData := make(map[string]interface{})
		secretValue := `{"foo":"bar"}`
		secretData["payload"] = secretValue
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

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
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

func ErrorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}
