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
	"fmt"
	"reflect"
	"strings"
	"testing"

	

	vault "github.com/oracle/oci-go-sdk/v45/vault"
	utilpointer "k8s.io/utils/pointer"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakeoracle "github.com/external-secrets/external-secrets/pkg/provider/oracle/fake"
)

type secretManagerTestCase struct {
	mockClient     *fakeoracle.OracleMockClient
	apiInput       *vault.GetSecretRequest
	apiOutput      *vault.GetSecretResponse
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockClient:     &fakeoracle.OracleMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.WithValue(*smtc.apiInput)
	return &smtc
}

func makeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "default",
	}
}

func makeValidAPIInput() *vault.GetSecretRequest {
	return &vault.GetSecretRequest{
		SecretId: utilpointer.StringPtr("i have no idea what should go here TEST CHECK"),
	}
}

func makeValidAPIOutput() *vault.GetSecretResponse {
	return &vault.GetSecretResponse{
		Etag:   utilpointer.StringPtr("i have no idea if I need this TEST CHECK"),
		Secret: vault.Secret{},
	}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(*smtc.apiInput) //HAVING ONLY 1 VARIABLE SEEMS WRONG TEST CHECK
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
	smtc.expectError = errUninitalizedOracleProvider
}

func TestOracleSecretManagerGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back

	setSecretString := func(smtc *secretManagerTestCase) {
		smtc.apiOutput = &vault.GetSecretResponse{
			Etag: utilpointer.StringPtr("i have no idea if I need this TEST CHECK"),
			Secret: vault.Secret{
				CompartmentId:  utilpointer.StringPtr("i have no idea if I need this TEST CHECK"),
				Id:             utilpointer.StringPtr("i have no idea if I need this TEST CHECK"),
				LifecycleState: vault.SecretLifecycleStateEnum("Unsure TEST CHECK"),
				SecretName:     utilpointer.StringPtr("changedvalue"),
			},
			// Key:   "testkey",
			// Value: "changedvalue",
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := KeyManagementService{}
	for k, v := range successCases {
		sm.Client = v.mockClient
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
		smtc.apiOutput.SecretName = utilpointer.StringPtr(`{"foo":"bar"}`)
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.SecretName = utilpointer.StringPtr(`-----------------`)
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
	}

	sm := KeyManagementService{}
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

// func TestCreateOracleClient(t *testing.T) {
// 	//credentials := OracleCredentials{Token: ORACLETOKEN}
// 	oracle := NewOracleProvider()
// 	fakeMain(oracle)
// 	request, err := http.NewRequest("GET", "https://cloud.oracle.com/security/kms/vaults/ocid1.vault.oc1.uk-london-1.cbqqsukvaacas.abwgiljrme75gqca4lh5uaykjk2ot2uwdae3p4l536h3mnrbkezqaphp5vdq/secrets/ocid1.vaultsecret.oc1.uk-london-1.amaaaaaa5op3n4qaqzzdsi3pn2kwcyyoypi5xg3nhzwpr7balrgjwy6uepsq?region=uk-london-1", nil)
// 	///20180608/secrets/ocid1.vaultsecret.oc1.uk-london-1.amaaaaaa5op3n4qaqzzdsi3pn2kwcyyoypi5xg3nhzwpr7balrgjwy6uepsq
// 	oracle.client.Call(context.Background(), request)
// 	//user := oracle.client.UserAgent
// 	fmt.Printf("Created client for username: %v", err)
// }
