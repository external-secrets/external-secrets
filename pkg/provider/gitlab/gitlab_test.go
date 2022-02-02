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
package gitlab

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	gitlab "github.com/xanzy/go-gitlab"

	esv1alpha2 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha2"
	fakegitlab "github.com/external-secrets/external-secrets/pkg/provider/gitlab/fake"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

type secretManagerTestCase struct {
	mockClient        *fakegitlab.GitlabMockClient
	apiInputProjectID string
	apiInputKey       string
	apiOutput         *gitlab.ProjectVariable
	ref               *esv1alpha2.ExternalSecretDataRemoteRef
	refFrom           *esv1alpha2.ExternalSecretDataFromRemoteRef
	projectID         *string
	apiErr            error
	expectError       string
	expectedSecret    string
	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretManagerTestCase() *secretManagerTestCase {
	smtc := secretManagerTestCase{
		mockClient:        &fakegitlab.GitlabMockClient{},
		apiInputProjectID: makeValidAPIInputProjectID(),
		apiInputKey:       makeValidAPIInputKey(),
		ref:               utils.MakeValidRef(),
		refFrom:           utils.MakeValidRefFrom(),
		projectID:         nil,
		apiOutput:         makeValidAPIOutput(),
		apiErr:            nil,
		expectError:       "",
		expectedSecret:    "",
		expectedData:      map[string][]byte{},
	}
	smtc.mockClient.WithValue(smtc.apiInputProjectID, smtc.apiInputKey, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidAPIInputProjectID() string {
	return "testID"
}

func makeValidAPIInputKey() string {
	return "testKey"
}

func makeValidAPIOutput() *gitlab.ProjectVariable {
	return &gitlab.ProjectVariable{
		Key:   "testKey",
		Value: "",
	}
}

func makeValidSecretManagerTestCaseCustom(tweaks ...func(smtc *secretManagerTestCase)) *secretManagerTestCase {
	smtc := makeValidSecretManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.mockClient.WithValue(smtc.apiInputProjectID, smtc.apiInputKey, smtc.apiOutput, smtc.apiErr)
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
	smtc.expectError = errUninitalizedGitlabProvider
}

// test the sm<->gcp interface
// make sure correct values are passed and errors are handled accordingly.
func TestGitlabSecretManagerGetSecret(t *testing.T) {
	secretValue := "changedvalue"
	// good case: default version is set
	// key is passed in, output is sent back

	setSecretString := func(smtc *secretManagerTestCase) {
		smtc.apiOutput = &gitlab.ProjectVariable{
			Key:   "testkey",
			Value: "changedvalue",
		}
		smtc.expectedSecret = secretValue
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setSecretString),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
	}

	sm := Gitlab{}
	for k, v := range successCases {
		sm.client = v.mockClient
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
		smtc.apiOutput.Value = `{"foo":"bar"}`
		smtc.expectedData["foo"] = []byte("bar")
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretManagerTestCase) {
		smtc.apiOutput.Value = `-----------------`
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*secretManagerTestCase{
		makeValidSecretManagerTestCaseCustom(setDeserialization),
		makeValidSecretManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretManagerTestCaseCustom(setNilMockClient),
		makeValidSecretManagerTestCaseCustom(setAPIErr),
	}

	sm := Gitlab{}
	for k, v := range successCases {
		sm.client = v.mockClient
		out, err := sm.GetSecretMap(context.Background(), *v.refFrom)
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
