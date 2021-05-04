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
	"github.com/stretchr/testify/assert"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	fakesm "github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager/fake"
	sess "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
)

func TestConstructor(t *testing.T) {
	s, err := sess.New("1111", "2222", "foo", "", nil)
	assert.Nil(t, err)
	c, err := New(s)
	assert.Nil(t, err)
	assert.NotNil(t, c.client)
}

type secretsManagerTestCase struct {
	fakeClient     *fakesm.Client
	apiInput       *awssm.GetSecretValueInput
	apiOutput      *awssm.GetSecretValueOutput
	ref            *esv1alpha1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
}

func makeValidSecretsManagerTestCase() *secretsManagerTestCase {
	smtc := secretsManagerTestCase{
		fakeClient:     &fakesm.Client{},
		apiInput:       makeValidAPIInput(),
		ref:            makeValidRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
	}
	smtc.fakeClient.WithValue(smtc.apiInput, smtc.apiOutput, smtc.apiErr)
	return &smtc
}

func makeValidRef() *esv1alpha1.ExternalSecretDataRemoteRef {
	return &esv1alpha1.ExternalSecretDataRemoteRef{
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
	setRefPropertyExistsInKey := func(smtc *secretsManagerTestCase) {
		smtc.ref.Property = "/shmoo"
		smtc.apiOutput.SecretString = aws.String(`{"/shmoo": "bang"}`)
		smtc.expectedSecret = "bang"
	}

	// bad case: missing property
	setRefMissingProperty := func(smtc *secretsManagerTestCase) {
		smtc.ref.Property = "INVALPROP"
		smtc.expectError = "key INVALPROP does not exist in secret"
	}

	// bad case: extract property failure due to invalid json
	setRefMissingPropertyInvalidJSON := func(smtc *secretsManagerTestCase) {
		smtc.ref.Property = "INVALPROP"
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
		smtc.ref.Property = "foobar.baz"
		smtc.expectedSecret = "nestedval"
	}

	// good case: custom version set
	setCustomVersion := func(smtc *secretsManagerTestCase) {
		smtc.apiInput.VersionStage = aws.String("1234")
		smtc.ref.Version = "1234"
		smtc.apiOutput.SecretString = aws.String("FOOBA!")
		smtc.expectedSecret = "FOOBA!"
	}
	// bad case: set apiErr
	setAPIErr := func(smtc *secretsManagerTestCase) {
		smtc.apiErr = fmt.Errorf("oh no")
		smtc.expectError = "oh no"
	}

	successCases := []*secretsManagerTestCase{
		makeValidSecretsManagerTestCase(),
		makeValidSecretsManagerTestCaseCustom(setSecretString),
		makeValidSecretsManagerTestCaseCustom(setRefPropertyExistsInKey),
		makeValidSecretsManagerTestCaseCustom(setRefMissingProperty),
		makeValidSecretsManagerTestCaseCustom(setRefMissingPropertyInvalidJSON),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryNotSecretString),
		makeValidSecretsManagerTestCaseCustom(setSecretBinaryAndSecretStringToNil),
		makeValidSecretsManagerTestCaseCustom(setNestedSecretValueJSONParsing),
		makeValidSecretsManagerTestCaseCustom(setCustomVersion),
		makeValidSecretsManagerTestCaseCustom(setAPIErr),
	}

	sm := SecretsManager{}
	for k, v := range successCases {
		sm.client = v.fakeClient
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
	fake := &fakesm.Client{}
	p := &SecretsManager{
		client: fake,
	}
	for i, row := range []struct {
		apiInput     *awssm.GetSecretValueInput
		apiOutput    *awssm.GetSecretValueOutput
		rr           esv1alpha1.ExternalSecretDataRemoteRef
		expectedData map[string]string
		apiErr       error
		expectError  string
	}{
		{
			// good case: default version & deserialization
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"foo":"bar"}`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{
				"foo": "bar",
			},
			apiErr:      nil,
			expectError: "",
		},
		{
			// bad case: api error returned
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`{"foo":"bar"}`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{
				"foo": "bar",
			},
			apiErr:      fmt.Errorf("some api err"),
			expectError: "some api err",
		},
		{
			// bad case: invalid json
			apiInput: &awssm.GetSecretValueInput{
				SecretId:     aws.String("/baz"),
				VersionStage: aws.String("AWSCURRENT"),
			},
			apiOutput: &awssm.GetSecretValueOutput{
				SecretString: aws.String(`-----------------`),
			},
			rr: esv1alpha1.ExternalSecretDataRemoteRef{
				Key: "/baz",
			},
			expectedData: map[string]string{},
			apiErr:       nil,
			expectError:  "unable to unmarshal secret",
		},
	} {
		fake.WithValue(row.apiInput, row.apiOutput, row.apiErr)
		out, err := p.GetSecretMap(context.Background(), row.rr)
		if !ErrorContains(err, row.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", i, err.Error(), row.expectError)
		}
		if cmp.Equal(out, row.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", i, row.expectedData, out)
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
