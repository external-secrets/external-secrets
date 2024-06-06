/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package onboardbase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/onboardbase/client"
	"github.com/external-secrets/external-secrets/pkg/provider/onboardbase/fake"
)

const (
	validSecretName        = "API_KEY"
	validSecretValue       = "3a3ea4f5"
	onboardbaseProject     = "ONBOARDBASE_PROJECT"
	onboardbaseEnvironment = "development"
	onboardbaseProjectVal  = "payments-service"
	missingSecret          = "INVALID_NAME"
	invalidSecret          = "unknown_project"
	missingSecretErr       = "could not get secret"
)

type onboardbaseTestCase struct {
	label               string
	fakeClient          *fake.OnboardbaseClient
	request             client.SecretRequest
	response            *client.SecretResponse
	remoteRef           *esv1beta1.ExternalSecretDataRemoteRef
	PushSecretRemoteRef esv1beta1.PushSecretRemoteRef
	apiErr              error
	expectError         string
	expectedSecret      string
	expectedData        map[string][]byte
}

func makeValidAPIRequest() client.SecretRequest {
	return client.SecretRequest{
		Name: validSecretName,
	}
}

func makeValidAPIOutput() *client.SecretResponse {
	return &client.SecretResponse{
		Name:  validSecretName,
		Value: validSecretValue,
	}
}

func makeValidRemoteRef() *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key: validSecretName,
	}
}

type pushRemoteRef struct {
	secretKey string
}

func (pRef pushRemoteRef) GetProperty() string {
	return ""
}

func (pRef pushRemoteRef) GetRemoteKey() string {
	return pRef.secretKey
}

func makeValidPushRemoteRef(key string) esv1beta1.PushSecretRemoteRef {
	return pushRemoteRef{
		secretKey: key,
	}
}

func makeValidOnboardbaseTestCase() *onboardbaseTestCase {
	return &onboardbaseTestCase{
		fakeClient:          &fake.OnboardbaseClient{},
		request:             makeValidAPIRequest(),
		response:            makeValidAPIOutput(),
		remoteRef:           makeValidRemoteRef(),
		PushSecretRemoteRef: makeValidPushRemoteRef(validSecretName),
		apiErr:              nil,
		expectError:         "",
		expectedSecret:      "",
		expectedData:        make(map[string][]byte),
	}
}

func makeValidOnboardbaseTestCaseCustom(tweaks ...func(pstc *onboardbaseTestCase)) *onboardbaseTestCase {
	pstc := makeValidOnboardbaseTestCase()
	for _, fn := range tweaks {
		fn(pstc)
	}
	pstc.fakeClient.WithValue(pstc.request, pstc.response, pstc.apiErr)
	return pstc
}

func TestGetSecret(t *testing.T) {
	setSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "set secret"
		pstc.request.Name = onboardbaseProject
		pstc.response.Name = onboardbaseProject
		pstc.response.Value = onboardbaseProjectVal
		pstc.expectedSecret = onboardbaseProjectVal
		pstc.remoteRef.Key = onboardbaseProject
	}

	setMissingSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid missing secret"
		pstc.remoteRef.Key = missingSecret
		pstc.request.Name = missingSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	setInvalidSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid secret name format"
		pstc.remoteRef.Key = invalidSecret
		pstc.request.Name = invalidSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	setClientError := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid client error"
		pstc.response = &client.SecretResponse{}
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	testCases := []*onboardbaseTestCase{
		makeValidOnboardbaseTestCaseCustom(setSecret),
		makeValidOnboardbaseTestCaseCustom(setMissingSecret),
		makeValidOnboardbaseTestCaseCustom(setInvalidSecret),
		makeValidOnboardbaseTestCaseCustom(setClientError),
	}

	c := Client{}
	for k, tc := range testCases {
		c.onboardbase = tc.fakeClient
		out, err := c.GetSecret(context.Background(), *tc.remoteRef)
		if !ErrorContains(err, tc.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), tc.expectError)
		}
		if err == nil && !cmp.Equal(string(out), tc.expectedSecret) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, tc.expectedSecret, string(out))
		}
	}
}

func TestDeleteSecret(t *testing.T) {
	setMissingSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid missing secret"
		pstc.remoteRef.Key = missingSecret
		pstc.PushSecretRemoteRef = makeValidPushRemoteRef(missingSecret)
		pstc.request.Name = missingSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	setInvalidSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid secret name format"
		pstc.remoteRef.Key = invalidSecret
		pstc.PushSecretRemoteRef = makeValidPushRemoteRef(invalidSecret)
		pstc.request.Name = invalidSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	deleteSecret := func(pstc *onboardbaseTestCase) {
		pstc.label = "delete secret successfully"
		pstc.remoteRef.Key = validSecretName
		pstc.PushSecretRemoteRef = makeValidPushRemoteRef(validSecretName)
		pstc.request.Name = validSecretName
		pstc.response = nil
		pstc.apiErr = nil
	}

	testCases := []*onboardbaseTestCase{
		makeValidOnboardbaseTestCaseCustom(setMissingSecret),
		makeValidOnboardbaseTestCaseCustom(setInvalidSecret),
		makeValidOnboardbaseTestCaseCustom(deleteSecret),
	}

	c := Client{}
	for k, tc := range testCases {
		c.onboardbase = tc.fakeClient
		err := c.DeleteSecret(context.Background(), tc.PushSecretRemoteRef)
		if err != nil && !ErrorContains(err, tc.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), tc.expectError)
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	simpleJSON := func(pstc *onboardbaseTestCase) {
		pstc.label = "valid unmarshalling"
		pstc.response.Value = `{"API_KEY":"3a3ea4f5"}`
		pstc.expectedData["API_KEY"] = []byte("3a3ea4f5")
	}

	complexJSON := func(pstc *onboardbaseTestCase) {
		pstc.label = "valid unmarshalling for nested json"
		pstc.response.Value = `{"API_KEY": "3a3ea4fs5", "AUTH_SA": {"appID": "a1ea-48bd-8749-b6f5ec3c5a1f"}}`
		pstc.expectedData["API_KEY"] = []byte("3a3ea4fs5")
		pstc.expectedData["AUTH_SA"] = []byte(`{"appID": "a1ea-48bd-8749-b6f5ec3c5a1f"}`)
	}

	setInvalidJSON := func(pstc *onboardbaseTestCase) {
		pstc.label = "invalid json"
		pstc.response.Value = `{"API_KEY": "3a3ea4f`
		pstc.expectError = "unable to unmarshal secret"
	}

	setAPIError := func(pstc *onboardbaseTestCase) {
		pstc.label = "client error"
		pstc.response = &client.SecretResponse{}
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	testCases := []*onboardbaseTestCase{
		makeValidOnboardbaseTestCaseCustom(simpleJSON),
		makeValidOnboardbaseTestCaseCustom(complexJSON),
		makeValidOnboardbaseTestCaseCustom(setInvalidJSON),
		makeValidOnboardbaseTestCaseCustom(setAPIError),
	}

	d := Client{}
	for k, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			d.onboardbase = tc.fakeClient
			out, err := d.GetSecretMap(context.Background(), *tc.remoteRef)
			if !ErrorContains(err, tc.expectError) {
				t.Errorf("[%d] unexpected error: %q, expected: %q", k, err.Error(), tc.expectError)
			}
			if err == nil && !cmp.Equal(out, tc.expectedData) {
				t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, tc.expectedData, out)
			}
		})
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

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func makeSecretStore(fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Onboardbase: &esv1beta1.OnboardbaseProvider{
					Auth: &esv1beta1.OnboardbaseAuthSecretRef{},
				},
			},
		},
	}
	for _, f := range fn {
		store = f(store)
	}
	return store
}

func withAuth(name, key string, namespace *string, passcode string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Onboardbase.Auth.OnboardbaseAPIKeyRef = v1.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		store.Spec.Provider.Onboardbase.Auth.OnboardbasePasscodeRef = v1.SecretKeySelector{
			Name:      passcode,
			Key:       passcode,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	label string
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	namespace := "ns"
	secretName := "onboardbase-api-key-secret"
	testCases := []ValidateStoreTestCase{
		{
			label: "invalid store missing onboardbaseAPIKey.name",
			store: makeSecretStore(withAuth("", "", nil, "")),
			err:   fmt.Errorf("invalid store: onboardbaseAPIKey.name cannot be empty"),
		},
		{
			label: "invalid store missing onboardbasePasscode.name",
			store: makeSecretStore(withAuth(secretName, "", nil, "")),
			err:   fmt.Errorf("invalid store: onboardbasePasscode.name cannot be empty"),
		},
		{
			label: "invalid store namespace not allowed",
			store: makeSecretStore(withAuth(secretName, "", &namespace, "passcode")),
			err:   fmt.Errorf("invalid store: namespace should either be empty or match the namespace of the SecretStore for a namespaced SecretStore"),
		},
		{
			label: "valid provide optional onboardbaseAPIKey.key",
			store: makeSecretStore(withAuth(secretName, "customSecretKey", nil, "passcode")),
			err:   nil,
		},
		{
			label: "valid namespace not set",
			store: makeSecretStore(withAuth(secretName, "", nil, "passcode")),
			err:   nil,
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			_, err := p.ValidateStore(tc.store)
			if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
				t.Errorf("test failed! want %v, got %v", tc.err, err)
			} else if tc.err == nil && err != nil {
				t.Errorf("want nil got err %v", err)
			} else if tc.err != nil && err == nil {
				t.Errorf("want err %v got nil", tc.err)
			}
		})
	}
}
