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

package doppler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
	"github.com/external-secrets/external-secrets/pkg/provider/doppler/fake"
)

const (
	validSecretName   = "API_KEY"
	validSecretValue  = "3a3ea4f5"
	dopplerProject    = "DOPPLER_PROJECT"
	dopplerProjectVal = "auth-api"
	missingSecret     = "INVALID_NAME"
	invalidSecret     = "doppler_project"
	missingSecretErr  = "could not get secret"
)

type dopplerTestCase struct {
	label          string
	fakeClient     *fake.DopplerClient
	request        client.SecretRequest
	response       *client.SecretResponse
	remoteRef      *esv1beta1.ExternalSecretDataRemoteRef
	apiErr         error
	expectError    string
	expectedSecret string
	expectedData   map[string][]byte
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

func makeValidDopplerTestCase() *dopplerTestCase {
	return &dopplerTestCase{
		fakeClient:     &fake.DopplerClient{},
		request:        makeValidAPIRequest(),
		response:       makeValidAPIOutput(),
		remoteRef:      makeValidRemoteRef(),
		apiErr:         nil,
		expectError:    "",
		expectedSecret: "",
		expectedData:   make(map[string][]byte),
	}
}

func makeValidDopplerTestCaseCustom(tweaks ...func(pstc *dopplerTestCase)) *dopplerTestCase {
	pstc := makeValidDopplerTestCase()
	for _, fn := range tweaks {
		fn(pstc)
	}
	pstc.fakeClient.WithValue(pstc.request, pstc.response, pstc.apiErr)
	return pstc
}

func TestGetSecret(t *testing.T) {
	setSecret := func(pstc *dopplerTestCase) {
		pstc.label = "set secret"
		pstc.request.Name = dopplerProject
		pstc.response.Name = dopplerProject
		pstc.response.Value = dopplerProjectVal
		pstc.expectedSecret = dopplerProjectVal
		pstc.remoteRef.Key = dopplerProject
	}

	setMissingSecret := func(pstc *dopplerTestCase) {
		pstc.label = "invalid missing secret"
		pstc.remoteRef.Key = missingSecret
		pstc.request.Name = missingSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	setInvalidSecret := func(pstc *dopplerTestCase) {
		pstc.label = "invalid secret name format"
		pstc.remoteRef.Key = invalidSecret
		pstc.request.Name = invalidSecret
		pstc.response = nil
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	setClientError := func(pstc *dopplerTestCase) {
		pstc.label = "invalid client error"
		pstc.response = &client.SecretResponse{}
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	testCases := []*dopplerTestCase{
		makeValidDopplerTestCaseCustom(setSecret),
		makeValidDopplerTestCaseCustom(setMissingSecret),
		makeValidDopplerTestCaseCustom(setInvalidSecret),
		makeValidDopplerTestCaseCustom(setClientError),
	}

	c := Client{}
	for k, tc := range testCases {
		c.doppler = tc.fakeClient
		out, err := c.GetSecret(context.Background(), *tc.remoteRef)
		if !ErrorContains(err, tc.expectError) {
			t.Errorf("[%d] unexpected error: %s, expected: '%s'", k, err.Error(), tc.expectError)
		}
		if err == nil && !cmp.Equal(string(out), tc.expectedSecret) {
			t.Errorf("[%d] unexpected secret data: expected %#v, got %#v", k, tc.expectedSecret, string(out))
		}
	}
}

func TestGetSecretMap(t *testing.T) {
	simpleJSON := func(pstc *dopplerTestCase) {
		pstc.label = "valid unmarshalling"
		pstc.response.Value = `{"API_KEY":"3a3ea4f5"}`
		pstc.expectedData["API_KEY"] = []byte("3a3ea4f5")
	}

	complexJSON := func(pstc *dopplerTestCase) {
		pstc.label = "valid unmarshalling for nested json"
		pstc.response.Value = `{"API_KEY": "3a3ea4f5", "AUTH_SA": {"appID": "a1ea-48bd-8749-b6f5ec3c5a1f"}}`
		pstc.expectedData["API_KEY"] = []byte("3a3ea4f5")
		pstc.expectedData["AUTH_SA"] = []byte(`{"appID": "a1ea-48bd-8749-b6f5ec3c5a1f"}`)
	}

	setInvalidJSON := func(pstc *dopplerTestCase) {
		pstc.label = "invalid json"
		pstc.response.Value = `{"API_KEY": "3a3ea4f`
		pstc.expectError = "unable to unmarshal secret"
	}

	setAPIError := func(pstc *dopplerTestCase) {
		pstc.label = "client error"
		pstc.response = &client.SecretResponse{}
		pstc.expectError = missingSecretErr
		pstc.apiErr = fmt.Errorf("")
	}

	testCases := []*dopplerTestCase{
		makeValidDopplerTestCaseCustom(simpleJSON),
		makeValidDopplerTestCaseCustom(complexJSON),
		makeValidDopplerTestCaseCustom(setInvalidJSON),
		makeValidDopplerTestCaseCustom(setAPIError),
	}

	d := Client{}
	for k, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			d.doppler = tc.fakeClient
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
				Doppler: &esv1beta1.DopplerProvider{
					Auth: &esv1beta1.DopplerAuth{},
				},
			},
		},
	}
	for _, f := range fn {
		store = f(store)
	}
	return store
}

func withAuth(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Doppler.Auth.SecretRef.DopplerToken = v1.SecretKeySelector{
			Name:      name,
			Key:       key,
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
	secretName := "doppler-token-secret"
	testCases := []ValidateStoreTestCase{
		{
			label: "invalid store missing dopplerToken.name",
			store: makeSecretStore(withAuth("", "", nil)),
			err:   fmt.Errorf("invalid store: dopplerToken.name cannot be empty"),
		},
		{
			label: "invalid store namespace not allowed",
			store: makeSecretStore(withAuth(secretName, "", &namespace)),
			err:   fmt.Errorf("invalid store: namespace not allowed with namespaced SecretStore"),
		},
		{
			label: "valid provide optional dopplerToken.key",
			store: makeSecretStore(withAuth(secretName, "customSecretKey", nil)),
			err:   nil,
		},
		{
			label: "valid namespace not set",
			store: makeSecretStore(withAuth(secretName, "", nil)),
			err:   nil,
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		t.Run(tc.label, func(t *testing.T) {
			err := p.ValidateStore(tc.store)
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
