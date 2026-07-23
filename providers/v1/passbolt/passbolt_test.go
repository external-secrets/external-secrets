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
	"fmt"
	"math/big"
	"net/http"
	"testing"
	"time"

	g "github.com/onsi/gomega"
	"github.com/passbolt/go-passbolt/api"
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
	g.Expect(err).To(g.MatchError(g.ContainSubstring(errPassboltCustomFieldNotFound)))
	g.Expect(err).To(g.MatchError(g.ContainSubstring("missing")))

	// Custom fields present but the requested key does not exist.
	secret.CustomFields = map[string]string{"other-key": "v"}
	_, err = secret.GetProp("custom_fields.nonexistent")
	g.Expect(err).To(g.MatchError(g.ContainSubstring(errPassboltCustomFieldNotFound)))
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
			name: "non-secret field: metadata_key with metadata_value, no secret entry",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "env", "metadata_value": "production"},
				},
			},
			secretFields: map[string]any{},
			want:         map[string]string{"env": "production"},
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
					// idB has no entry: its value lives in metadata_value.
				},
			},
			want: map[string]string{
				"token":  "tok-abc123",
				"region": "us-east-1",
			},
		},
		{
			name: "nil secretFields falls back to metadata_value",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "env", "metadata_value": "staging"},
				},
			},
			secretFields: nil,
			want:         map[string]string{"env": "staging"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildCustomFields(tt.metaFields, tt.secretFields)
			g.Expect(got).To(g.Equal(tt.want))
		})
	}
}

func TestIndexSecretValues(t *testing.T) {
	g.RegisterTestingT(t)

	const idA = "11111111-1111-1111-1111-111111111111"
	const idB = "22222222-2222-2222-2222-222222222222"

	tests := []struct {
		name         string
		secretFields map[string]any
		want         map[string]string
	}{
		{
			name:         "nil secretFields returns empty map",
			secretFields: nil,
			want:         map[string]string{},
		},
		{
			name:         "no custom_fields key returns empty map",
			secretFields: map[string]any{"password": "p"},
			want:         map[string]string{},
		},
		{
			name:         "custom_fields not a slice returns empty map",
			secretFields: map[string]any{"custom_fields": "not-a-slice"},
			want:         map[string]string{},
		},
		{
			name: "standard entry is indexed by id",
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "tok-abc"},
				},
			},
			want: map[string]string{idA: "tok-abc"},
		},
		{
			name: "empty string secret_value is preserved in the index",
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": ""},
				},
			},
			want: map[string]string{idA: ""},
		},
		{
			name: "entry with empty id is skipped",
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": "", "secret_value": "v"},
				},
			},
			want: map[string]string{},
		},
		{
			name: "non-map item in slice is skipped",
			secretFields: map[string]any{
				"custom_fields": []any{"not-a-map"},
			},
			want: map[string]string{},
		},
		{
			name: "multiple entries are all indexed",
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": "val-a"},
					map[string]any{"id": idB, "secret_value": "val-b"},
				},
			},
			want: map[string]string{idA: "val-a", idB: "val-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := indexSecretValues(tt.secretFields, 0)
			g.Expect(got).To(g.Equal(tt.want))
		})
	}
}

func TestResolveCustomField(t *testing.T) {
	g.RegisterTestingT(t)

	const idA = "11111111-1111-1111-1111-111111111111"

	tests := []struct {
		name          string
		item          any
		secretValByID map[string]string
		wantName      string
		wantValue     string
		wantOK        bool
	}{
		{
			name:   "non-map item returns false",
			item:   "not-a-map",
			wantOK: false,
		},
		{
			name:   "no metadata_key (secret_key field) returns false",
			item:   map[string]any{"id": idA},
			wantOK: false,
		},
		{
			name:   "empty id returns false",
			item:   map[string]any{"metadata_key": "k", "id": ""},
			wantOK: false,
		},
		{
			name:          "secret_value takes precedence over metadata_value",
			item:          map[string]any{"id": idA, "metadata_key": "field", "metadata_value": "meta-val"},
			secretValByID: map[string]string{idA: "secret-val"},
			wantName:      "field",
			wantValue:     "secret-val",
			wantOK:        true,
		},
		{
			name:          "falls back to metadata_value when id absent from secret index",
			item:          map[string]any{"id": idA, "metadata_key": "env", "metadata_value": "production"},
			secretValByID: map[string]string{},
			wantName:      "env",
			wantValue:     "production",
			wantOK:        true,
		},
		{
			// An empty-string secret_value must be returned as-is, not treated as
			// "missing" and silently replaced by metadata_value.
			name:          "empty string secret_value is preserved, not treated as missing",
			item:          map[string]any{"id": idA, "metadata_key": "field", "metadata_value": "should-not-appear"},
			secretValByID: map[string]string{idA: ""},
			wantName:      "field",
			wantValue:     "",
			wantOK:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, value, ok := resolveCustomField(tt.item, tt.secretValByID)
			g.Expect(ok).To(g.Equal(tt.wantOK))
			if tt.wantOK {
				g.Expect(name).To(g.Equal(tt.wantName))
				g.Expect(value).To(g.Equal(tt.wantValue))
			}
		})
	}
}

func TestIsPassboltNotFound(t *testing.T) {
	g.RegisterTestingT(t)

	g.Expect(isPassboltNotFound(&api.APIError{StatusCode: http.StatusNotFound})).To(g.BeTrue())

	// Other API status codes are not not-found.
	g.Expect(isPassboltNotFound(&api.APIError{StatusCode: http.StatusForbidden})).To(g.BeFalse())
	g.Expect(isPassboltNotFound(&api.APIError{StatusCode: http.StatusInternalServerError})).To(g.BeFalse())

	// Non-API errors and nil are never not-found.
	g.Expect(isPassboltNotFound(errors.New("some other error"))).To(g.BeFalse())
	g.Expect(isPassboltNotFound(nil)).To(g.BeFalse())

	// errors.As unwraps, so a wrapped APIError is still detected.
	g.Expect(isPassboltNotFound(fmt.Errorf("wrapped: %w", &api.APIError{StatusCode: http.StatusNotFound}))).To(g.BeTrue())
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
