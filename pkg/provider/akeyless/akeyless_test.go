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
package akeyless

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakeakeyless "github.com/external-secrets/external-secrets/pkg/provider/akeyless/fake"
)

type akeylessTestCase struct {
	mockClient     *fakeakeyless.AkeylessMockClient
	apiInput       *fakeakeyless.Input
	apiOutput      *fakeakeyless.Output
	ref            *esv1beta1.ExternalSecretDataRemoteRef
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidAkeylessTestCase() *akeylessTestCase {
	smtc := akeylessTestCase{
		mockClient:     &fakeakeyless.AkeylessMockClient{},
		apiInput:       makeValidInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidOutput(),
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput)
	return &smtc
}

func makeValidRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "test-secret",
		Version: "1",
	}
}

func makeValidInput() *fakeakeyless.Input {
	return &fakeakeyless.Input{
		SecretName: "name",
		Version:    0,
		Token:      "token",
	}
}

func makeValidOutput() *fakeakeyless.Output {
	return &fakeakeyless.Output{
		Value: "secret-val",
		Err:   nil,
	}
}

func makeValidAkeylessTestCaseCustom(tweaks ...func(smtc *akeylessTestCase)) *akeylessTestCase {
	smtc := makeValidAkeylessTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(smtc.apiInput, smtc.apiOutput)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *akeylessTestCase) {
	smtc.apiOutput.Err = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

var setNilMockClient = func(smtc *akeylessTestCase) {
	smtc.mockClient = nil
	smtc.expectError = errUninitalizedAkeylessProvider
}

func TestAkeylessGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *akeylessTestCase) {
		smtc.apiOutput = &fakeakeyless.Output{
			Value: secretValue,
			Err:   nil,
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*akeylessTestCase{
		makeValidAkeylessTestCaseCustom(setAPIErr),
		makeValidAkeylessTestCaseCustom(setSecretString),
		makeValidAkeylessTestCaseCustom(setNilMockClient),
	}

	sm := Akeyless{}
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
	setDeserialization := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `{"foo":"bar"}`
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *akeylessTestCase) {
		smtc.apiOutput.Value = `-----------------`
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*akeylessTestCase{
		makeValidAkeylessTestCaseCustom(setDeserialization),
		makeValidAkeylessTestCaseCustom(setInvalidJSON),
		makeValidAkeylessTestCaseCustom(setAPIErr),
		makeValidAkeylessTestCaseCustom(setNilMockClient),
	}

	sm := Akeyless{}
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
