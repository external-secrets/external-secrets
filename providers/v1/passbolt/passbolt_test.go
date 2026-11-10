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
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	g "github.com/onsi/gomega"
	"github.com/passbolt/go-passbolt/api"
	"github.com/passbolt/go-passbolt/helper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
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

// The metadata/secret custom-field merge lives in the SDK
// (helper.ParseCustomFields). This is a contract test over the shapes this
// provider depends on, not a re-test of the library's full table.
func TestCustomFieldsFromResourceFieldMaps(t *testing.T) {
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
			name:         "a resource with no custom fields yields no entries",
			metaFields:   map[string]any{"name": "x"},
			secretFields: map[string]any{"password": "p"},
			want:         map[string]string{},
		},
		{
			name: "encrypted value: name from metadata_key, value from secret_value",
			metaFields: map[string]any{
				"custom_fields": []any{map[string]any{"id": idA, "metadata_key": "api-key"}},
			},
			secretFields: map[string]any{
				"custom_fields": []any{map[string]any{"id": idA, "secret_value": "secret-123"}},
			},
			want: map[string]string{"api-key": "secret-123"},
		},
		{
			// A cleartext field still carries an empty secret_value, so reading
			// it requires preferring the first non-empty of the two sides.
			name: "cleartext value comes from metadata_value",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "env", "metadata_value": "production"},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{map[string]any{"id": idA, "secret_value": ""}},
			},
			want: map[string]string{"env": "production"},
		},
		{
			name: "a field whose name is encrypted is keyed by its decrypted secret_key",
			metaFields: map[string]any{
				"custom_fields": []any{map[string]any{"id": idA, "metadata_key": ""}},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_key": "hidden-name", "secret_value": "hidden-val"},
				},
			},
			want: map[string]string{"hidden-name": "hidden-val"},
		},
		{
			name: "non-string values are stringified for the map[string]string payload",
			metaFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "metadata_key": "port"},
					map[string]any{"id": idB, "metadata_key": "enabled", "metadata_value": true},
				},
			},
			secretFields: map[string]any{
				"custom_fields": []any{
					map[string]any{"id": idA, "secret_value": float64(8080)},
					map[string]any{"id": idB, "secret_value": ""},
				},
			},
			want: map[string]string{"port": "8080", "enabled": "true"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := helper.ParseCustomFields(tt.metaFields, tt.secretFields).Map()
			g.Expect(got).To(g.Equal(tt.want))
		})
	}
}

// A resource without custom fields must not grow a "custom_fields" key in the
// JSON payload, whether the map comes back empty or nil.
func TestSecretJSONOmitsEmptyCustomFields(t *testing.T) {
	g.RegisterTestingT(t)

	for _, cf := range []map[string]string{nil, {}} {
		marshaled, err := esutils.JSONMarshal(Secret{Name: "x", CustomFields: cf})
		g.Expect(err).ToNot(g.HaveOccurred())
		g.Expect(string(marshaled)).ToNot(g.ContainSubstring("custom_fields"))
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
			Name: "passbolt-ca", Namespace: "ns",
			Data: map[string][]byte{"ca.crt": caPEM},
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

// testUserKey is a throwaway PGP key without a passphrase, generated for these tests only.
const testUserKey = `-----BEGIN PGP PRIVATE KEY BLOCK-----

xVgEavMlaxYJKwYBBAHaRw8BAQdAWtdhwTBScc8ad0p5/sZGqG5PGRprqq00hRw7
DfwaYKkAAP91yvKmkQ3mfbyyegQCeFvWoURPkNkUesRVtuoMdYsWexDvzR9lc28t
dGVzdCA8ZXNvLXRlc3RAZXhhbXBsZS5jb20+wr8EExYIAHEFgmrzJWsDCwkHCRBF
jfmODfSNeTUUAAAAAAAcABBzYWx0QG5vdGF0aW9ucy5vcGVucGdwanMub3JnhAtt
8J2qaj5LaaYfg4186AIVCAMWAAICGQECmwMCHgEWIQSpzvX15mv7P/VErdtFjfmO
DfSNeQAA8bIA/jSVESaJNAPu3zn5XZZp9l6WDFD2ZZ7PtVpU6/cyYTOeAP9lwDwv
aYp2GJdUJPYzwxtoNhxfbKKULZDdFV2b6EQgCMddBGrzJWsSCisGAQQBl1UBBQEB
B0CMON3mdjLgin6+sbGDikZTsxL/DXTg+APmMkF0YccQdwMBCgkAAP9fKLsDTvk3
YCva5TxcTPHkXq/3Skhmcxvp8vnmdNp9oBJ0wq4EGBYIAGAFgmrzJWsJEEWN+Y4N
9I15NRQAAAAAABwAEHNhbHRAbm90YXRpb25zLm9wZW5wZ3Bqcy5vcmfX/0OUHGpX
AzZF4YX1oWBhApsMFiEEqc719eZr+z/1RK3bRY35jg30jXkAALaYAQCCb0Xinodo
X6TsWFlYymjnEBEFyhSbQ4PiPxfy+RPihAD/Th0QjiOie1KzAqZqJKTdUmvrHBK8
mEd6Nw+2Y+8I+QQ=
=kGfk
-----END PGP PRIVATE KEY BLOCK-----`

// fakePassbolt serves what NewClient, GetSecret and Close need from a Passbolt server:
// the two-stage GPGAuth login, the session check and the logout. It knows no resources.
type fakePassbolt struct {
	*httptest.Server

	mu         sync.Mutex
	user       *api.Client     // encrypts the login challenge to the test user
	challenges map[string]bool // tokens handed out in stage 1
	sessions   map[string]bool // values of the session cookie that are logged in
	logins     int
}

func newFakePassbolt(t *testing.T) *fakePassbolt {
	t.Helper()
	user, err := api.NewClient(nil, "", "http://passbolt.test", testUserKey, "")
	if err != nil {
		t.Fatalf("load test key: %v", err)
	}
	f := &fakePassbolt{user: user, challenges: map[string]bool{}, sessions: map[string]bool{}}
	f.Server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.Close)
	return f
}

func (f *fakePassbolt) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	session := ""
	if c, err := r.Cookie("passbolt_session"); err == nil && f.sessions[c.Value] {
		session = c.Value
	}

	switch r.Method + " " + r.URL.Path {
	case "POST /auth/login.json":
		f.login(w, r)
	case "POST /auth/logout.json":
		if session == "" {
			f.respond(w, http.StatusForbidden, nil)
			return
		}
		delete(f.sessions, session)
		f.respond(w, http.StatusOK, nil)
	case "GET /auth/is-authenticated.json", "GET /users/me.json", "GET /settings.json":
		if session == "" {
			f.respond(w, http.StatusUnauthorized, nil)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "csrfToken", Value: "csrf"})
		f.respond(w, http.StatusOK, map[string]any{"id": "user", "passbolt": map[string]any{"plugins": map[string]any{}}})
	default:
		f.respond(w, http.StatusNotFound, nil)
	}
}

func (f *fakePassbolt) login(w http.ResponseWriter, r *http.Request) {
	var req api.Login
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Auth == nil {
		f.respond(w, http.StatusBadRequest, nil)
		return
	}

	if req.Auth.Token == "" {
		challenge := fmt.Sprintf("gpgauthv1.3.0|36|00000000-0000-0000-0000-%012d|gpgauthv1.3.0", len(f.challenges))
		encrypted, err := f.user.EncryptMessage(challenge)
		if err != nil {
			f.respond(w, http.StatusInternalServerError, nil)
			return
		}
		f.challenges[challenge] = true
		w.Header().Set("X-GPGAuth-User-Auth-Token", url.QueryEscape(encrypted))
		f.respond(w, http.StatusOK, nil)
		return
	}

	if !f.challenges[req.Auth.Token] {
		f.respond(w, http.StatusUnauthorized, nil)
		return
	}
	delete(f.challenges, req.Auth.Token)
	f.logins++
	session := fmt.Sprintf("session-%d", f.logins)
	f.sessions[session] = true
	http.SetCookie(w, &http.Cookie{Name: "passbolt_session", Value: session})
	f.respond(w, http.StatusOK, nil)
}

// respond writes the envelope every Passbolt API response comes in.
func (f *fakePassbolt) respond(w http.ResponseWriter, code int, body any) {
	status := "success"
	if code != http.StatusOK {
		status = "error"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"header": map[string]any{"status": status, "code": code, "message": http.StatusText(code)},
		"body":   body,
	})
}

func (f *fakePassbolt) state() (logins, openSessions int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.logins, len(f.sessions)
}

// The registered provider is one instance for the whole process. The SecretStore
// controller and the ExternalSecret reconciles call NewClient on it side by side, so
// closing one client must not log out the session of another.
func TestNewClientReturnsIndependentClients(t *testing.T) {
	g.RegisterTestingT(t)
	ctx := context.Background()
	server := newFakePassbolt(t)

	scheme := runtime.NewScheme()
	g.Expect(corev1.AddToScheme(scheme)).To(g.Succeed())
	kube := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(&corev1.Secret{
		Name: "passbolt", Namespace: "ns",
		Data: map[string][]byte{"privateKey": []byte(testUserKey), "password": []byte("unused")},
	}).Build()
	store := &esv1.SecretStore{
		Name: "passbolt", Namespace: "ns",
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Passbolt: &esv1.PassboltProvider{
					Host: server.URL,
					Auth: &esv1.PassboltAuth{
						PasswordSecretRef:   &esmeta.SecretKeySelector{Name: "passbolt", Key: "password"},
						PrivateKeySecretRef: &esmeta.SecretKeySelector{Name: "passbolt", Key: "privateKey"},
					},
				},
			},
		},
	}

	provider := NewProvider()
	first, err := provider.NewClient(ctx, store, kube, "ns")
	g.Expect(err).ToNot(g.HaveOccurred())
	second, err := provider.NewClient(ctx, store, kube, "ns")
	g.Expect(err).ToNot(g.HaveOccurred())

	g.Expect(first.Close(ctx)).To(g.Succeed())

	// The second client still uses the session it logged in with: its request reaches
	// the server, which answers that it has no such resource, and no new login happens.
	_, err = second.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "5d9f4f6f-0b9e-4f3a-9a52-8f0a3b6f0c11"})
	var apiErr *api.APIError
	g.Expect(errors.As(err, &apiErr)).To(g.BeTrue(), "unexpected error: %v", err)
	g.Expect(apiErr.StatusCode).To(g.Equal(http.StatusNotFound))
	logins, openSessions := server.state()
	g.Expect(logins).To(g.Equal(2))
	g.Expect(openSessions).To(g.Equal(1))

	// Each client logs out its own session, so none is left behind.
	g.Expect(second.Close(ctx)).To(g.Succeed())
	_, openSessions = server.state()
	g.Expect(openSessions).To(g.BeZero())
}
