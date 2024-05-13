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

package infisical

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esv1meta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/fake"
)

type storeModifier func(*esv1beta1.SecretStore) *esv1beta1.SecretStore

func TestGetSecret(t *testing.T) {
	p := &Provider{
		apiClient: &fake.MockInfisicalClient{},
		apiScope: &InfisicalClientScope{
			SecretPath:      "/",
			ProjectSlug:     "first-project",
			EnvironmentSlug: "dev",
		},
	}

	secret, err := p.GetSecret(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
		Key: "key",
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err == nil && !reflect.DeepEqual(string(secret), "value") {
		t.Errorf("unexpected secret data: expected %#v, got %#v", "value", string(secret))
	}
}

func TestGetSecretMap(t *testing.T) {
	p := &Provider{
		apiClient: &fake.MockInfisicalClient{},
		apiScope: &InfisicalClientScope{
			SecretPath:      "/",
			ProjectSlug:     "first-project",
			EnvironmentSlug: "dev",
		},
	}

	secret, err := p.GetSecretMap(context.Background(), esv1beta1.ExternalSecretDataRemoteRef{
		Key: "key",
	})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if err == nil && !reflect.DeepEqual(secret, map[string][]byte{
		"key": []byte("value"),
	}) {
		t.Errorf("unexpected secret data map: expected %#v, got %#v", map[string][]byte{"key": []byte("value")}, secret)
	}
}

func makeSecretStore(projectSlug, environment, secretPath string, fn ...storeModifier) *esv1beta1.SecretStore {
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Infisical: &esv1beta1.InfisicalProvider{
					Auth: esv1beta1.InfisicalAuth{},
					SecretsScope: esv1beta1.MachineIdentityScopeInWorkspace{
						SecretsPath:     secretPath,
						EnvironmentSlug: environment,
						ProjectSlug:     projectSlug,
					},
				},
			},
		},
	}
	for _, f := range fn {
		store = f(store)
	}
	return store
}

func withClientID(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientID = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

func withClientSecret(name, key string, namespace *string) storeModifier {
	return func(store *esv1beta1.SecretStore) *esv1beta1.SecretStore {
		store.Spec.Provider.Infisical.Auth.UniversalAuthCredentials.ClientSecret = esv1meta.SecretKeySelector{
			Name:      name,
			Key:       key,
			Namespace: namespace,
		}
		return store
	}
}

type ValidateStoreTestCase struct {
	store *esv1beta1.SecretStore
	err   error
}

func TestValidateStore(t *testing.T) {
	testCases := []ValidateStoreTestCase{
		{
			store: makeSecretStore("", "", ""),
			err:   fmt.Errorf("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty"),
		},
		{
			store: makeSecretStore("first-project", "dev", "/", withClientID("universal-auth", "some-random-id", nil)),
			err:   fmt.Errorf("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty"),
		},
		{
			store: makeSecretStore("first-project", "dev", "/", withClientSecret("universal-auth", "some-random-id", nil)),
			err:   fmt.Errorf("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty"),
		},
		{
			store: makeSecretStore("first-project", "dev", "/", withClientID("universal-auth", "some-random-id", nil), withClientSecret("universal-auth", "some-random-id", nil)),
			err:   nil,
		},
	}
	p := Provider{}
	for _, tc := range testCases {
		_, err := p.ValidateStore(tc.store)
		if tc.err != nil && err != nil && err.Error() != tc.err.Error() {
			t.Errorf("test failed! want %v, got %v", tc.err, err)
		} else if tc.err == nil && err != nil {
			t.Errorf("want nil got err %v", err)
		} else if tc.err != nil && err == nil {
			t.Errorf("want err %v got nil", tc.err)
		}
	}
}
