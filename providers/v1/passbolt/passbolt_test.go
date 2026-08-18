/*
Copyright © The ESO Authors

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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"testing"
	"time"

	g "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
		Name:         "test-name",
		Username:     "test-user",
		Password:     "test-pass",
		URI:          "https://test.com",
		Description:  "test-desc",
		CustomFields: map[string]string{"my-field": "my-value"},
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

	// Test custom field
	val, err = secret.GetProp("custom_fields.my-field")
	g.Expect(err).To(g.BeNil())
	g.Expect(string(val)).To(g.Equal("my-value"))

	// Test invalid property
	_, err = secret.GetProp("invalid")
	g.Expect(err).To(g.MatchError(errPassboltSecretPropertyInvalid))
}

func TestSecretGetPropCustomFieldNotFound(t *testing.T) {
	g.RegisterTestingT(t)

	// No custom fields set at all.
	secret := Secret{Name: "test-name"}
	_, err := secret.GetProp("custom_fields.missing")
	g.Expect(err).To(g.MatchError(errPassboltCustomFieldNotFound))
	g.Expect(err).To(g.MatchError(g.ContainSubstring("missing")))

	// Custom fields present but the requested key does not exist.
	secret.CustomFields = map[string]string{"other-key": "v"}
	_, err = secret.GetProp("custom_fields.nonexistent")
	g.Expect(err).To(g.MatchError(errPassboltCustomFieldNotFound))
	g.Expect(err).To(g.MatchError(g.ContainSubstring("nonexistent")))
}

func TestBuildCustomFields(t *testing.T) {
	g.RegisterTestingT(t)

	const idA = "11111111-1111-1111-1111-111111111111"
	const idB = "22222222-2222-2222-2222-222222222222"

	tests := []struct {
		name         string
		metaFields   map[string]any
		secretFields map[string]any
		want         map[string]string
	}{
		{
			name:         "no custom_fields in metadata returns nil",
			metaFields:   map[string]any{"name": "x"},
			secretFields: map[string]any{"password": "p"},
			want:         nil,
		},
		{
			name:         "empty custom_fields array returns nil",
			metaFields:   map[string]any{"custom_fields": []any{}},
			secretFields: map[string]any{},
			want:         nil,
		},
		{
			name: "standard case: metadata_key with secret_value",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "api-key"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "secret-123"},
				},
			},
			want: map[string]string{"api-key": "secret-123"},
		},
		{
			name: "non-secret field: metadata_key with metadata_value and secret-side stub",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "env", "metadata_value": "production"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "type": "text"},
				},
			},
			want: map[string]string{"env": "production"},
		},
		{
			name: "secret_key field (no metadata_key) is silently skipped",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA /* no metadata_key */},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_key": "hidden-name", "secret_value": "hidden-val"},
				},
			},
			want: nil,
		},
		{
			name: "multiple fields: encrypted value and non-secret value",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "token"},
					map[string]any{"id": idB, "metadata_key": "region", "metadata_value": "us-east-1"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "tok-abc123"},
					map[string]any{"id": idB, "type": "text"},
				},
			},
			want: map[string]string{
				"token":  "tok-abc123",
				"region": "us-east-1",
			},
		},
		{
			name: "secret_value takes precedence over metadata_value",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "field", "metadata_value": "meta-val"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "secret-val"},
				},
			},
			want: map[string]string{"field": "secret-val"},
		},
		{
			name: "secret_value of empty string is preserved",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "empty-field"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": ""},
				},
			},
			want: map[string]string{"empty-field": ""},
		},
		{
			name: "numeric and boolean values are stringified",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "port"},
					map[string]any{"id": idB, "metadata_key": "enabled", "metadata_value": true},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": float64(8080)},
					map[string]any{"id": idB, "type": "boolean"},
				},
			},
			want: map[string]string{
				"port":    "8080",
				"enabled": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCustomFields(tt.metaFields, tt.secretFields)
			g.Expect(got).To(g.Equal(tt.want))
		})
	}
}

func TestExtractCustomFields(t *testing.T) {
	g.RegisterTestingT(t)

	const idA = "11111111-1111-1111-1111-111111111111"
	const idB = "22222222-2222-2222-2222-222222222222"

	tests := []struct {
		name   string
		fields map[string]any
		want   []map[string]any
		wantOK bool
	}{
		{
			name:   "no custom_fields key",
			fields: map[string]any{"password": "p"},
			want:   nil,
			wantOK: false,
		},
		{
			name:   "custom_fields not a slice",
			fields: map[string]any{"custom_fields": "not-a-slice"},
			want:   nil,
			wantOK: false,
		},
		{
			name: "already typed as []map[string]any",
			fields: map[string]any{
				"custom_fields": []map[string]any{
					{"id": idA, "secret_value": "tok-abc"},
				},
			},
			want:   []map[string]any{{"id": idA, "secret_value": "tok-abc"}},
			wantOK: true,
		},
		{
			name: "entries are extracted, non-map items skipped",
			fields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "tok-abc"},
					"not-a-map",
					map[string]any{"id": idB, "type": "text"},
				},
			},
			want: []map[string]any{
				{"id": idA, "secret_value": "tok-abc"},
				{"id": idB, "type": "text"},
			},
			wantOK: true,
		},
		{
			name: "empty custom_fields slice returns false",
			fields: map[string]any{
				"custom_fields": []any{},
			},
			want:   []map[string]any{},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractCustomFields(tt.fields)
			g.Expect(ok).To(g.Equal(tt.wantOK))
			g.Expect(got).To(g.Equal(tt.want))
		})
	}
}

func TestHasNonEmptyString(t *testing.T) {
	g.RegisterTestingT(t)

	tests := []struct {
		name string
		m    map[string]any
		key  string
		want bool
	}{
		{name: "missing key", m: map[string]any{}, key: "k", want: false},
		{name: "non-empty string", m: map[string]any{"k": "v"}, key: "k", want: true},
		{name: "empty string", m: map[string]any{"k": ""}, key: "k", want: false},
		{name: "non-string value", m: map[string]any{"k": 42}, key: "k", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g.Expect(hasNonEmptyString(tt.m, tt.key)).To(g.Equal(tt.want))
		})
	}
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

// generateCABundlePEM creates a self-signed CA certificate in PEM format,
// for exercising buildHTTPClient without any real PKI dependencies.
func generateCABundlePEM(t *testing.T) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}

func TestBuildHTTPClient(t *testing.T) {
	g.RegisterTestingT(t)
	caPEM := generateCABundlePEM(t)

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(g.Succeed())

	kubeObjs := []runtime.Object{
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "passbolt-ca", Namespace: "ns"},
			Data:       map[string][]byte{"ca.crt": caPEM},
		},
	}
	kube := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(kubeObjs...).Build()

	tests := []struct {
		name     string
		provider *esv1.PassboltProvider
		wantNil  bool
	}{
		{
			name:     "no CA configured returns nil client (SDK default + system roots)",
			provider: &esv1.PassboltProvider{},
			wantNil:  true,
		},
		{
			name: "inline caBundle populates the client's RootCAs and leaves default transport settings intact",
			provider: &esv1.PassboltProvider{
				CABundle: caPEM,
			},
		},
		{
			name: "caProvider with a Secret is dereferenced into RootCAs",
			provider: &esv1.PassboltProvider{
				CAProvider: &esv1.CAProvider{
					Type: esv1.CAProviderTypeSecret,
					Name: "passbolt-ca",
					Key:  "ca.crt",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := buildHTTPClient(t.Context(), tt.provider, kube, esv1.SecretStoreKind, "ns")
			g.Expect(err).ToNot(g.HaveOccurred())
			if tt.wantNil {
				g.Expect(client).To(g.BeNil())
				return
			}

			g.Expect(client).ToNot(g.BeNil())
			transport, ok := client.Transport.(*http.Transport)
			g.Expect(ok).To(g.BeTrue())
			g.Expect(transport.TLSClientConfig).ToNot(g.BeNil())
			g.Expect(transport.TLSClientConfig.MinVersion).To(g.Equal(uint16(tls.VersionTLS12)))

			// Verify the configured pool actually contains *our* CA, not just any
			// non-nil pool — guards against silently loading a different bundle.
			expectedPool := x509.NewCertPool()
			g.Expect(expectedPool.AppendCertsFromPEM(caPEM)).To(g.BeTrue())
			g.Expect(transport.TLSClientConfig.RootCAs).ToNot(g.BeNil())
			g.Expect(transport.TLSClientConfig.RootCAs.Equal(expectedPool)).To(g.BeTrue())

			// Confirm we cloned the default transport and didn't end up with a
			// bare http.Transport{} (which would drop proxy/dialer defaults).
			defaultTransport := http.DefaultTransport.(*http.Transport)
			g.Expect(transport.Proxy).ToNot(g.BeNil(), "cloned transport should keep DefaultTransport.Proxy")
			g.Expect(transport.MaxIdleConns).To(g.Equal(defaultTransport.MaxIdleConns))
		})
	}
}
