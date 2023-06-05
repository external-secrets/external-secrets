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

package conjur

import (
	"context"
	"fmt"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	fakeconjur "github.com/external-secrets/external-secrets/pkg/provider/conjur/fake"
)

var (
	svcURL     = "https://example.com"
	svcUser    = "user"
	svcApikey  = "apikey"
	svcAccount = "account1"
)

type secretManagerTestCase struct {
	err    error
	refKey string
}

func TestConjurGetSecret(t *testing.T) {
	p := Provider{}
	p.ConjurClient = &fakeconjur.ConjurMockClient{}

	testCases := []*secretManagerTestCase{
		{
			err:    nil,
			refKey: "secret",
		},
		{
			err:    fmt.Errorf("error"),
			refKey: "error",
		},
	}

	for _, tc := range testCases {
		ref := makeValidRef(tc.refKey)
		_, err := p.GetSecret(context.Background(), *ref)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}

func makeValidRef(k string) *esv1beta1.ExternalSecretDataRemoteRef {
	return &esv1beta1.ExternalSecretDataRemoteRef{
		Key:     k,
		Version: "default",
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore(svcURL, svcUser, svcApikey, svcAccount),
			err:   nil,
		},
		{
			store: makeSecretStore("", svcUser, svcApikey, svcAccount),
			err:   fmt.Errorf("ServiceURL cannot be empty"),
		},
		{
			store: makeSecretStore(svcURL, "", svcApikey, svcAccount),
			err:   fmt.Errorf("ServiceUser cannot be empty"),
		},
		{
			store: makeSecretStore(svcURL, svcUser, "", svcAccount),
			err:   fmt.Errorf("ServiceAPIKey cannot be empty"),
		},
		{
			store: makeSecretStore(svcURL, svcUser, svcApikey, ""),
			err:   fmt.Errorf("SeviceAccount cannot be empty"),
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}

func makeSecretStore(svcURL, svcUser, svcApikey, svcAccount string) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Conjur: &esv1beta1.ConjurProvider{
					ServiceURL:     &svcURL,
					ServiceUser:    &svcUser,
					ServiceAPIKey:  &svcApikey,
					ServiceAccount: &svcAccount,
				},
			},
		},
	}
	return store
}
