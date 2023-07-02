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

package barbican

import (
	"context"
	"testing"

	"github.com/artashesbalabekyan/barbican-sdk-go/client"
	fake "github.com/artashesbalabekyan/barbican-sdk-go/fake"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/google/go-cmp/cmp"
)

const unexpectedErrorString = "[%d] unexpected error: %s, expected: '%s'"

type secretTest struct {
	name    string
	payload []byte
}

type secretsClientTestCase struct {
	fakeClient     client.Conn
	apiInput       secretTest
	apiOutput      secretTest
	remoteRef      *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string

	// for testing secretmap
	expectedData map[string][]byte
}

func makeValidSecretsClientTestCase() *secretsClientTestCase {
	client, _ := fake.New(context.Background(), nil)
	smtc := secretsClientTestCase{
		fakeClient:     client,
		apiInput:       makeValidAPIInput(),
		remoteRef:      makeValidRemoteRef(),
		apiOutput:      makeValidAPIOutput(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "my-value1",
		expectedData:   map[string][]byte{},
	}
	smtc.fakeClient.Create(context.Background(), smtc.apiInput.name, smtc.apiInput.payload)
	return &smtc
}

func makeValidSecretsManagerTestCaseCustom(tweaks ...func(smtc *secretsClientTestCase)) *secretsClientTestCase {
	smtc := makeValidSecretsClientTestCase()
	for _, fn := range tweaks {
		fn(smtc)
	}
	smtc.fakeClient.Create(context.Background(), smtc.apiInput.name, smtc.apiInput.payload)
	return smtc
}

func makeValidRemoteRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: "my-key1",
	}
}

func makeValidAPIInput() secretTest {
	return secretTest{
		name:    "my-key1",
		payload: []byte("my-value1"),
	}
}

func makeValidAPIOutput() secretTest {
	return secretTest{
		name:    "my-key1",
		payload: []byte("my-value1"),
	}
}

func TestSecretsManagerGetSecret(t *testing.T) {
	// good case: default version is set
	// key is passed in, output is sent back
	setSecretString := func(smtc *secretsClientTestCase) {
		smtc.apiOutput.name = "my-key1"
		smtc.expectedSecret = "my-value1"
	}

	// good case: custom key
	// key is passed in, output is sent back
	setSecretStringWithRefKey := func(smtc *secretsClientTestCase) {
		smtc.remoteRef.Key = "shmoo"
		smtc.apiInput.name = "shmoo"
		smtc.apiInput.payload = []byte("bang")
		smtc.expectedSecret = "bang"
	}

	successCases := []*secretsClientTestCase{
		makeValidSecretsClientTestCase(),
		makeValidSecretsManagerTestCaseCustom(setSecretString),
		makeValidSecretsManagerTestCaseCustom(setSecretStringWithRefKey),
	}

	for k, v := range successCases {
		sm := Client{
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

func TestGetSecretMap(t *testing.T) {
	setDeserialization := func(smtc *secretsClientTestCase) {
		smtc.remoteRef.Key = "foobar"
		smtc.apiInput.name = "foobar"
		smtc.apiInput.payload = []byte(`{"foo":"bar"}`)
		smtc.expectedData["foo"] = []byte("bar")
	}

	successCases := []*secretsClientTestCase{
		makeValidSecretsManagerTestCaseCustom(setDeserialization),
	}

	for k, v := range successCases {
		sm := Client{
			client: v.fakeClient,
		}
		out, err := sm.GetSecretMap(context.Background(), *v.remoteRef)
		if !ErrorContains(err, v.expectError) {
			t.Errorf(unexpectedErrorString, k, err.Error(), v.expectError)
		}
		if err == nil && !cmp.Equal(out, v.expectedData) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, v.expectedData, out)
		}
	}
}
