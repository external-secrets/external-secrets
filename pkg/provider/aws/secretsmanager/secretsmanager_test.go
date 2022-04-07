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

package secretsmanager

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	awssm "github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/google/go-cmp/cmp"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
)

type secretsManagerTestCase struct {
	fakeClient     *fakesm.Client
	apiInput       *awssm.GetSecretValueInput
	apiOutput      *awssm.GetSecretValueOutput
	remoteRef      *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	// for testing secretmap
	expectedData map[string][]byte
	// for testing caching
	expectedCounter *int
}

const unexpectedErrorString = "[%d] unexpected error: %s, expected: '%s'"

func makeValidSecretsManagerTestCase() *secretsManagerTestCase {
	smtc := secretsManagerTestCase{
		fakeClient:     fakesm.NewClient(),
		apiInput:       makeValidAPIInput(),
		remoteRef:      makeValidRemoteRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   map[string][]byte{},
	}
	smtc.fakeClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRemoteRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     "/baz",
		Version: "AWSCURRENT",
	}
}

func makeValidAPIInput() *awssm.GetSecretValueInput {
	return &awssm.GetSecretValueInput{
		SecretId:     aws.String("/baz"),
		VersionStage: aws.String("AWSCURRENT"),
	}
}

func makeValidAPIOutput() *awssm.GetSecretValueOutput {
	return &awssm.GetSecretValueOutput{
		SecretString: aws.String(""),
	}
}

func makeValidSecretsManagerTestCaseCustom(tweaks ...func(smtc *secretsManagerTestCase)) *secretsManagerTestCase {
	smtc := makeValidSecretsManagerTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.fakeClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return smtc
}

// This case can be shared by both GetSecret and GetSecretMap tests.
// bad case: set apiErr.
var setAPIErr = func(smtc *secretsManagerTestCase) {
	smtc.apiErr = fmt.Errorf("oh no")
	smtc.expectError = "oh no"
}

// test the sm<->aws interface
// make sure correct values are passed and errors are handled accordingly.
func TestSecretsManagerGetSecret(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String("testtesttest")
		smtc.expectedSecret = "testtesttest"
	}

	// good case: extract property
	// Testing that the property exists in the SecretString
	setRemoteRefPropertyExistsInKey := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "/shmoo"
		smtc.apiOutput.SecretString = aws.String(`{"/shmoo": "bang"}`)
		smtc.expectedSecret = "bang"
	}

	// bad case: missing property
	setRemoteRefMissingProperty := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "INVALPROP"
		smtc.expectError = "key INVALPROP does not exist in secret"
	}

	// bad case: extract property failure due to invalid json
	setRemoteRefMissingPropertyInvalidJSON := func(smtc *secretsManagerTestCase) {
		smtc.remoteRef.Property = "INVALPROP"
		smtc.apiOutput.SecretString = aws.String(`------`)
		smtc.expectError = "key INVALPROP does not exist in secret"
	}

	// good case: set .SecretString to nil but set binary with value
	setSecretBinaryNotSecretString := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretBinary = []byte("yesplease")
		// needs to be set as nil, empty quotes ("") is considered existing
		smtc.apiOutput.SecretString = nil
		smtc.expectedSecret = "yesplease"
	}

	// bad case: both .SecretString and .SecretBinary are nil
	setSecretBinaryAndSecretStringToNil := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretBinary = nil
		smtc.apiOutput.SecretString = nil
		smtc.expectError = "no secret string nor binary for key"
	}
	// good case: secretOut.SecretBinary JSON parsing
	setNestedSecretValueJSONParsing := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = nil
		smtc.apiOutput.SecretBinary = []byte(`{"foobar":{"baz":"nestedval"}}`)
		smtc.remoteRef.Property = "foobar.baz"
		smtc.expectedSecret = "nestedval"
	}
	// good case: secretOut.SecretBinary no JSON parsing if name on key
	setSecretValueWithDot := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = nil
		smtc.apiOutput.SecretBinary = []byte(`{"foobar.baz":"nestedval"}`)
		smtc.remoteRef.Property = "foobar.baz"
		smtc.expectedSecret = "nestedval"
	}

	// good case: custom version set
	setCustomVersion := func(smtc *secretsManagerTestCase) {
		smtc.apiInput.VersionStage = aws.String("1234")
		smtc.remoteRef.Version = "1234"
		smtc.apiOutput.SecretString = aws.String("FOOBA!")
		smtc.expectedSecret = "FOOBA!"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCase(),
		makeValidSecretsManagerTestCaseCustom(setSecretString),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefPropertyExistsInKey),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefMissingProperty),
		makeValidSecretsManagerTestCaseCustom(setRemoteRefMissingPropertyInvalidJSON),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryNotSecretString),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryAndSecretStringToNil),
		makeValidSecretsManagerTestCaseCustom(setNestedSecretValueJSONParsing),
		makeValidSecretsManagerTestCaseCustom(setSecretValueWithDot),
		makeValidSecretsManagerTestCaseCustom(setCustomVersion),
		makeValidSecretsManagerTestCaseCustom(setAPIErr),
	}

	for k, v := range successCases {
		sm := SecretsManager{
			cache:  make(map[string]*awssm.GetSecretValueOutput),
			client: v.fakeClient,
		}
		out, err := sm.GetSecret(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
	}
}
func TestCaching(t *testing.T) {
	fakeClient := fakesm.NewClient()

	// good case: first call, since we are using the same key, results should be cached and the counter should not go
	// over 1
	firstCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "bar":"vodka"}`)
		smtc.remoteRef.Property = "foo"
		smtc.expectedSecret = "bar"
		smtc.expectedCounter = aws.Int(1)
		smtc.fakeClient = fakeClient
	}
	secondCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "bar":"vodka"}`)
		smtc.remoteRef.Property = "bar"
		smtc.expectedSecret = "vodka"
		smtc.expectedCounter = aws.Int(1)
		smtc.fakeClient = fakeClient
	}
	notCachedCall := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"sheldon":"bazinga", "bar":"foo"}`)
		smtc.remoteRef.Property = "sheldon"
		smtc.expectedSecret = "bazinga"
		smtc.expectedCounter = aws.Int(2)
		smtc.fakeClient = fakeClient
		smtc.apiInput.SecretId = aws.String("xyz")
		smtc.remoteRef.Key = "xyz" // it should reset the cache since the key is different
	}

	cachedCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCaseCustom(firstCall),
		makeValidSecretsManagerTestCaseCustom(firstCall),
		makeValidSecretsManagerTestCaseCustom(secondCall),
		makeValidSecretsManagerTestCaseCustom(notCachedCall),
	}
	sm := SecretsManager{
		cache: make(map[string]*awssm.GetSecretValueOutput),
	}
	for k, v := range cachedCases {
		sm.client = v.fakeClient
		out, err := sm.GetSecret(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && string(out) != v.expectedSecret {
			t.Errorf("[%d] unexpected secret: expected %s, got %s", k, v.expectedSecret, string(out))
		}
		if v.expectedCounter != nil && v.fakeClient.ExecutionCounter != *v.expectedCounter {
			t.Errorf("[%d] unexpected counter value: expected %d, got %d", k, v.expectedCounter, v.fakeClient.ExecutionCounter)
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	// good case: default version & deserialization
	setDeserialization := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar"}`)
		smtc.expectedData["foo"] = []byte("bar")
	}

	// good case: nested json
	setNestedJSON := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foobar":{"baz":"nestedval"}}`)
		smtc.expectedData["foobar"] = []byte("{\"baz\":\"nestedval\"}")
	}

	// good case: caching
	cachedMap := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`{"foo":"bar", "plus": "one"}`)
		smtc.expectedData["foo"] = []byte("bar")
		smtc.expectedData["plus"] = []byte("one")
		smtc.expectedCounter = aws.Int(1)
	}

	// bad case: invalid json
	setInvalidJSON := func(smtc *secretsManagerTestCase) {
		smtc.apiOutput.SecretString = aws.String(`-----------------`)
		smtc.expectError = "unable to unmarshal secret"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCaseCustom(setDeserialization),
		makeValidSecretsManagerTestCaseCustom(setNestedJSON),
		makeValidSecretsManagerTestCaseCustom(setAPIErr),
		makeValidSecretsManagerTestCaseCustom(setInvalidJSON),
		makeValidSecretsManagerTestCaseCustom(cachedMap),
	}

	for k, v := range successCases {
		sm := SecretsManager{
			cache:  make(map[string]*awssm.GetSecretValueOutput),
			client: v.fakeClient,
		}
		out, err := sm.GetSecretMap(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && !cmp.Equal(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
		if v.expectedCounter != nil && v.fakeClient.ExecutionCounter != *v.expectedCounter {
			t.Errorf("[%d] unexpected counter value: expected %d, got %d", k, v.expectedCounter, v.fakeClient.ExecutionCounter)
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
