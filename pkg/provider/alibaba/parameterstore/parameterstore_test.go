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

package parameterstore

import (
	"context"
	"fmt"
	oossdk "github.com/alibabacloud-go/oos-20190601/v3/client"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/auth"
	"reflect"
	"strings"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	secretName  = "test-example"
	secretValue = "value"
)

type parameterStoreTestCase struct {
	mockClient     *AlibabaMockClient
	apiInput       *oossdk.GetSecretParameterRequest
	apiOutput      *oossdk.GetSecretParameterResponseBody
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidPSTestCase() *parameterStoreTestCase {
	pstc := parameterStoreTestCase{
		mockClient:     &AlibabaMockClient{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   make(map[string][]byte),
	}
	pstc.mockClient.WithValue(pstc.apiInput, pstc.apiOutput, pstc.apiErr)
	return &pstc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: secretName,
	}
}

func makeValidAPIInput() *oossdk.GetSecretParameterRequest {
	return &oossdk.GetSecretParameterRequest{
		Name: utils.Ptr(secretName),
	}
}

func makeValidAPIOutput() *oossdk.GetSecretParameterResponseBody {
	responseParameter := &oossdk.GetSecretParameterResponseBodyParameter{
		Name:  utils.Ptr(secretName),
		Value: utils.Ptr(secretValue),
	}
	response := &oossdk.GetSecretParameterResponseBody{
		Parameter: responseParameter,
	}
	return response
}

func makeValidPSTestCaseCustom(tweaks ...func(pstc *parameterStoreTestCase)) *parameterStoreTestCase {
	pstc := makeValidPSTestCase()
	for _, fn := range tweaks {
		fn(pstc)
	}
	pstc.mockClient.WithValue(pstc.apiInput, pstc.apiOutput, pstc.apiErr)
	return pstc
}

var setAPIErr = func(pstc *parameterStoreTestCase) {
	pstc.apiErr = fmt.Errorf("oh no")
	pstc.expectError = "oh no"
}

var setNilMockClient = func(pstc *parameterStoreTestCase) {
	pstc.mockClient = nil
	pstc.expectError = auth.ErrUninitializedAlibabaProvider
}

func TestAlibabaPSGetSecret(t *testing.T) {
	secretData := make(map[string]interface{})
	secretValue := "changedvalue"
	secretData["payload"] = secretValue

	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(pstc *parameterStoreTestCase) {
		pstc.apiOutput.Parameter.Name = utils.Ptr(secretName)
		pstc.apiOutput.Parameter.Value = utils.Ptr(secretValue)
		pstc.expectedSecret = secretValue
	}

	// good case: custom version set
	setCustomKey := func(pstc *parameterStoreTestCase) {
		pstc.apiOutput.Parameter.Name = utils.Ptr("test-example-other")
		pstc.ref.Key = "test-example-other"
		pstc.apiOutput.Parameter.Value = utils.Ptr(secretValue)
		pstc.expectedSecret = secretValue
	}

	successCases := []*parameterStoreTestCase{
		makeValidPSTestCaseCustom(setSecretString),
		makeValidPSTestCaseCustom(setCustomKey),
		makeValidPSTestCaseCustom(setAPIErr),
		makeValidPSTestCaseCustom(setNilMockClient),
	}

	sm := ParameterStore{}
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
	setDeserialization := func(pstc *parameterStoreTestCase) {
		pstc.apiOutput.Parameter.Name = utils.Ptr("foo")
		pstc.expectedData["foo"] = []byte("bar")
		pstc.apiOutput.Parameter.Value = utils.Ptr(`{"foo":"bar"}`)
	}

	// bad case: invalid json
	setInvalidJSON := func(pstc *parameterStoreTestCase) {
		pstc.apiOutput.Parameter.Value = utils.Ptr("-----------------")
		pstc.expectError = "unable to unmarshal secret"
	}

	successCases := []*parameterStoreTestCase{
		makeValidPSTestCaseCustom(setDeserialization),
		makeValidPSTestCaseCustom(setInvalidJSON),
		makeValidPSTestCaseCustom(setNilMockClient),
		makeValidPSTestCaseCustom(setAPIErr),
	}

	sm := ParameterStore{}
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
