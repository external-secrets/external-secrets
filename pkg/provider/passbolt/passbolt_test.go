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

package passbolt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	g "github.com/onsi/gomega"
	"github.com/passbolt/go-passbolt/api"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

type PassboltClientMock struct {
}

func (p *PassboltClientMock) CheckSession(_ context.Context) bool {
	return true
}
func (p *PassboltClientMock) Login(_ context.Context) error {
	return nil
}
func (p *PassboltClientMock) Logout(_ context.Context) error {
	return nil
}
func (p *PassboltClientMock) GetResource(_ context.Context, resourceID string) (*api.Resource, error) {
	resmap := map[string]api.Resource{
		"some-key1": {ID: "some-key1", Name: "some-name1", URI: "some-uri1"},
		"some-key2": {ID: "some-key2", Name: "some-name2", URI: "some-uri2"},
	}

	if res, ok := resmap[resourceID]; ok {
		return &res, nil
	}

	return nil, errors.New("ID not found")
}

func (p *PassboltClientMock) GetResources(_ context.Context, _ *api.GetResourcesOptions) ([]api.Resource, error) {
	res := []api.Resource{
		{ID: "some-key1", Name: "some-name1", URI: "some-uri1"},
		{ID: "some-key2", Name: "some-name2", URI: "some-uri2"},
	}
	return res, nil
}

func (p *PassboltClientMock) GetResourceType(_ context.Context, _ string) (*api.ResourceType, error) {
	res := &api.ResourceType{Slug: "password-and-description"}
	return res, nil
}

func (p *PassboltClientMock) DecryptMessage(message string) (string, error) {
	return message, nil
}

func (p *PassboltClientMock) GetSecret(_ context.Context, resourceID string) (*api.Secret, error) {
	resmap := map[string]api.Secret{
		"some-key1": {Data: `{"password": "some-password1", "description": "some-description1"}`},
		"some-key2": {Data: `{"password": "some-password2", "description": "some-description2"}`},
	}

	if res, ok := resmap[resourceID]; ok {
		return &res, nil
	}

	return nil, errors.New("ID not found")
}

var clientMock = &PassboltClientMock{}

func TestValidateStore(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}

	g.RegisterTestingT(t)
	store := &esv1beta1.SecretStore{
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Passbolt: &esv1beta1.PassboltProvider{},
			},
		},
	}

	// missing auth
	_, err := p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errPassboltStoreMissingAuth)))

	// missing password
	store.Spec.Provider.Passbolt.Auth = &esv1beta1.PassboltAuth{
		PrivateKeySecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "privatekey"},
	}
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errPassboltStoreMissingAuthPassword)))

	// missing privateKey
	store.Spec.Provider.Passbolt.Auth = &esv1beta1.PassboltAuth{
		PasswordSecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "password"},
	}
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errPassboltStoreMissingAuthPrivateKey)))

	store.Spec.Provider.Passbolt.Auth = &esv1beta1.PassboltAuth{

		PasswordSecretRef:   &esmeta.SecretKeySelector{Key: "some-secret", Name: "password"},
		PrivateKeySecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "privatekey"},
	}

	// missing host
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errPassboltStoreMissingHost)))

	// not https
	store.Spec.Provider.Passbolt.Host = "http://passbolt.test"
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errPassboltStoreHostSchemeNotHTTPS)))

	// spec ok
	store.Spec.Provider.Passbolt.Host = "https://passbolt.test"
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeNil())
}

func TestClose(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}
	g.RegisterTestingT(t)
	err := p.Close(context.TODO())
	g.Expect(err).To(g.BeNil())
}

func TestGetAllSecrets(t *testing.T) {
	cases := []struct {
		desc        string
		ref         esv1beta1.ExternalSecretFind
		expected    map[string][]byte
		expectedErr string
	}{
		{
			desc: "no matches",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "nonexistant",
				},
			},
			expected: map[string][]byte{},
		},
		{
			desc: "matches",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "some-name.*",
				},
			},
			expected: map[string][]byte{
				"some-key1": []byte(`{"name":"some-name1","username":"","password":"some-password1","uri":"some-uri1","description":"some-description1"}`),
				"some-key2": []byte(`{"name":"some-name2","username":"","password":"some-password2","uri":"some-uri2","description":"some-description2"}`),
			},
		},
		{
			desc:        "missing find.name",
			ref:         esv1beta1.ExternalSecretFind{},
			expectedErr: errPassboltExternalSecretMissingFindNameRegExp,
		},
		{
			desc: "empty find.name.regexp",
			ref: esv1beta1.ExternalSecretFind{
				Name: &esv1beta1.FindName{
					RegExp: "",
				},
			},
			expectedErr: errPassboltExternalSecretMissingFindNameRegExp,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			ctx := context.Background()
			p := ProviderPassbolt{client: clientMock}

			got, err := p.GetAllSecrets(ctx, tc.ref)
			if err != nil {
				if tc.expectedErr == "" {
					t.Fatalf("failed to call GetAllSecrets: %v", err)
				}

				if !strings.Contains(err.Error(), tc.expectedErr) {
					t.Fatalf("%q expected to contain substring %q", err.Error(), tc.expectedErr)
				}

				return
			}

			if tc.expectedErr != "" {
				t.Fatal("expected to receive an error but got nil")
			}

			if diff := cmp.Diff(tc.expected, got); diff != "" {
				t.Fatalf("(-got, +want)\n%s", diff)
			}
		})
	}
}

func TestGetSecret(t *testing.T) {
	g.RegisterTestingT(t)
	tbl := []struct {
		name     string
		request  esv1beta1.ExternalSecretDataRemoteRef
		expValue string
		expErr   string
	}{
		{
			name: "return err when not found",
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "nonexistent",
			},
			expErr: "ID not found",
		},
		{
			name: "get property from secret",
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "some-key1",
				Property: "password",
			},
			expValue: "some-password1",
		},
		{
			name: "get full secret",
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key: "some-key1",
			},
			expValue: `{"name":"some-name1","username":"","password":"some-password1","uri":"some-uri1","description":"some-description1"}`,
		},
		{
			name: "return err when using invalid property",
			request: esv1beta1.ExternalSecretDataRemoteRef{
				Key:      "some-key1",
				Property: "invalid",
			},
			expErr: errPassboltSecretPropertyInvalid,
		},
	}

	for _, row := range tbl {
		t.Run(row.name, func(_ *testing.T) {
			p := &ProviderPassbolt{client: clientMock}

			out, err := p.GetSecret(context.Background(), row.request)
			if row.expErr != "" {
				g.Expect(err).To(g.MatchError(row.expErr))
			} else {
				g.Expect(err).ToNot(g.HaveOccurred())
			}
			g.Expect(string(out)).To(g.Equal(row.expValue))
		})
	}
}

func TestSecretExists(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}
	g.RegisterTestingT(t)
	_, err := p.SecretExists(context.TODO(), nil)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errNotImplemented)))
}
func TestPushSecret(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}
	g.RegisterTestingT(t)
	err := p.PushSecret(context.TODO(), nil, nil)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errNotImplemented)))
}
func TestDeleteSecret(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}
	g.RegisterTestingT(t)
	err := p.DeleteSecret(context.TODO(), nil)
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errNotImplemented)))
}
func TestGetSecretMap(t *testing.T) {
	p := &ProviderPassbolt{client: clientMock}
	g.RegisterTestingT(t)
	_, err := p.GetSecretMap(context.TODO(), esv1beta1.ExternalSecretDataRemoteRef{})
	g.Expect(err).To(g.BeEquivalentTo(fmt.Errorf(errNotImplemented)))
}
