/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

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
	"testing"

	g "github.com/onsi/gomega"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
)

func TestValidateStore(t *testing.T) {
	p := &ProviderPassbolt{}
	g.RegisterTestingT(t)

	store := &esv1.SecretStore{
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Passbolt: &esv1.PassboltProvider{},
			},
		},
	}

	// missing auth
	_, err := p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errPassboltStoreMissingAuth)))

	// missing password
	store.Spec.Provider.Passbolt.Auth = &esv1.PassboltAuth{
		PrivateKeySecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "privatekey"},
	}
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errPassboltStoreMissingAuthPassword)))

	// missing privateKey
	store.Spec.Provider.Passbolt.Auth = &esv1.PassboltAuth{
		PasswordSecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "password"},
	}
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errPassboltStoreMissingAuthPrivateKey)))

	store.Spec.Provider.Passbolt.Auth = &esv1.PassboltAuth{
		PasswordSecretRef:   &esmeta.SecretKeySelector{Key: "some-secret", Name: "password"},
		PrivateKeySecretRef: &esmeta.SecretKeySelector{Key: "some-secret", Name: "privatekey"},
	}

	// missing host
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errPassboltStoreMissingHost)))

	// host not https
	store.Spec.Provider.Passbolt.Host = "http://passbolt.test"
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errPassboltStoreHostSchemeNotHTTPS)))

	// valid store
	store.Spec.Provider.Passbolt.Host = "https://passbolt.test"
	_, err = p.ValidateStore(store)
	g.Expect(err).To(g.BeNil())
}

func TestSecretGetProp(t *testing.T) {
	g.RegisterTestingT(t)

	secret := Secret{
		Name:        "test-name",
		Username:    "test-user",
		Password:    "test-pass",
		URI:         "https://test.com",
		Description: "test-desc",
	}

	// Test valid properties
	val, err := secret.GetProp("name")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("test-name"))

	val, err = secret.GetProp("username")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("test-user"))

	val, err = secret.GetProp("password")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("test-pass"))

	val, err = secret.GetProp("uri")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("https://test.com"))

	val, err = secret.GetProp("description")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("test-desc"))

	// Test invalid property
	_, err = secret.GetProp("invalid")
	g.Expect(err).To(g.MatchError(errPassboltSecretPropertyInvalid))
}

func TestCapabilities(t *testing.T) {
	g.RegisterTestingT(t)
	p := &ProviderPassbolt{}
	g.Expect(p.Capabilities()).To(g.Equal(esv1.SecretStoreReadOnly))
}

func TestSecretExists(t *testing.T) {
	p := &ProviderPassbolt{}
	g.RegisterTestingT(t)
	_, err := p.SecretExists(context.TODO(), nil)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errNotImplemented)))
}

func TestPushSecret(t *testing.T) {
	p := &ProviderPassbolt{}
	g.RegisterTestingT(t)
	err := p.PushSecret(context.TODO(), nil, nil)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errNotImplemented)))
}

func TestDeleteSecret(t *testing.T) {
	p := &ProviderPassbolt{}
	g.RegisterTestingT(t)
	err := p.DeleteSecret(context.TODO(), nil)
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errNotImplemented)))
}

func TestGetSecretMap(t *testing.T) {
	p := &ProviderPassbolt{}
	g.RegisterTestingT(t)
	_, err := p.GetSecretMap(context.TODO(), esv1.ExternalSecretDataRemoteRef{})
	g.Expect(err).To(g.BeEquivalentTo(errors.New(errNotImplemented)))
}
