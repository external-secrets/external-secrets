// /*
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

package mysterybox

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/nebius/gosdk/auth"
	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox/fake"
)

const (
	tokenSecretName   = "tokenSecretName"
	tokenSecretKey    = "tokenSecretKey"
	saCredsSecretName = "saCredsSecretName"
	saCredsSecretKey  = "saCredsSecretKey"
	authRefName       = "authRefSecretName"
	authRefKey        = "authRefSecretKey"
	apiDomain         = "api.public"
)

var (
	logger = ctrl.Log.WithName("provider").WithName("nebius").WithName("mysterybox")
)

func setupClientWithTokenAuth(t *testing.T, entries []mysterybox.Entry, tokenService TokenService) (context.Context, *SecretsClient, *fake.Secret, k8sclient.Client, *fake.MysteryboxService) {
	t.Helper()
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mysteryboxService.CreateSecret(entries)

	provider := newProvider(
		t,
		func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		tokenService,
	)
	createK8sSecret(ctx, t, k8sClient, namespace, tokenSecretName, tokenSecretKey, []byte("token"))
	store := newNebiusMysteryboxSecretStoreWithAuthTokenKey(apiDomain, namespace, tokenSecretName, tokenSecretKey)
	client, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	mysteryboxSecretsClient, ok := client.(*SecretsClient)
	if !ok {
		t.Fatalf("expected *SecretsClient, got %T", client)
	}
	return ctx, mysteryboxSecretsClient, secret, k8sClient, mysteryboxService
}

func TestNewClient_GetTokenError(t *testing.T) {
	tokenService := fakeTokenService{returnError: true}

	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()
	createK8sSecret(ctx, t, k8sClient, namespace, saCredsSecretName, saCredsSecretKey, []byte("PRIVATE KEY"))

	provider := newProvider(
		t,
		func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		&tokenService,
	)

	_, err := provider.NewClient(ctx, newNebiusMysteryboxSecretStoreWithServiceAccountCreds(apiDomain, namespace, saCredsSecretName, saCredsSecretKey), k8sClient, namespace)
	tassert.Error(t, err)
	tassert.ErrorContains(t, err, "failed to retrieve iam token by credentials")
}

func TestGetSecret(t *testing.T) {
	entries := []mysterybox.Entry{
		{Key: "key1", StringValue: "string"},
		{Key: "key2", StringValue: "string2"},
		{Key: "key3", BinaryValue: []byte("binaryValue")},
	}

	tests := []struct {
		name       string
		prepare    func(ctx context.Context, client *SecretsClient, secret *fake.Secret, svc *fake.MysteryboxService) ([]byte, error)
		expectJSON map[string]string
		expectRaw  []byte
	}{
		{
			name: "get all entries as JSON",
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret, _ *fake.MysteryboxService) ([]byte, error) {
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id})
			},
			expectJSON: map[string]string{
				"key1": "string",
				"key2": "string2",
				"key3": b64.StdEncoding.EncodeToString([]byte("binaryValue")),
			},
		},
		{
			name: "string entry by key",
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret, _ *fake.MysteryboxService) ([]byte, error) {
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "key1"})
			},
			expectRaw: []byte("string"),
		},
		{
			name: "binary entry by key",
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret, _ *fake.MysteryboxService) ([]byte, error) {
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "key3"})
			},
			expectRaw: []byte("binaryValue"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, secret, _, svc := setupClientWithTokenAuth(t, entries, nil)
			result, err := tt.prepare(ctx, client, secret, svc)
			tassert.Nil(t, err)
			if tt.expectJSON != nil {
				tassert.Equal(t, tt.expectJSON, unmarshalStringMap(t, result))
			} else {
				tassert.Equal(t, tt.expectRaw, result)
			}
		})
	}
}

func TestGetSecret_ByVersionId(t *testing.T) {
	ctx, client, secret, _, mboxService := setupClientWithTokenAuth(t, []mysterybox.Entry{{Key: "key", StringValue: "string_value"}}, nil)
	_, err := mboxService.CreateNewSecretVersion(secret.Id, []mysterybox.Entry{
		{Key: "new_key", StringValue: "updated_string_value"},
		{Key: "new", StringValue: "new"},
	})
	tassert.Nil(t, err)

	result, err := client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "key", Version: secret.VersionId})
	tassert.Nil(t, err)
	tassert.Equal(t, []byte("string_value"), result)
}

func TestGetSecretMap(t *testing.T) {
	allEntries := []mysterybox.Entry{
		{Key: "key1", StringValue: "string"},
		{Key: "key2", StringValue: "string2"},
		{Key: "key3", BinaryValue: []byte("binaryValue")},
	}

	tests := []struct {
		name      string
		entries   []mysterybox.Entry
		prepare   func(ctx context.Context, client *SecretsClient, secret *fake.Secret, svc *fake.MysteryboxService) (map[string][]byte, error)
		expectMap map[string][]byte
	}{
		{
			name:    "all entries",
			entries: allEntries,
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret, _ *fake.MysteryboxService) (map[string][]byte, error) {
				return client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id})
			},
			expectMap: map[string][]byte{
				"key1": []byte("string"),
				"key2": []byte("string2"),
				"key3": []byte("binaryValue"),
			},
		},
		{
			name:    "by version id",
			entries: []mysterybox.Entry{{Key: "key", StringValue: "string_value"}},
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret, svc *fake.MysteryboxService) (map[string][]byte, error) {
				_, err := svc.CreateNewSecretVersion(secret.Id, []mysterybox.Entry{{Key: "new_key", StringValue: "updated_string_value"}, {Key: "new", StringValue: "new"}})
				if err != nil {
					return nil, err
				}
				return client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Version: secret.VersionId})
			},
			expectMap: map[string][]byte{
				"key": []byte("string_value"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, client, secret, _, svc := setupClientWithTokenAuth(t, tt.entries, nil)
			result, err := tt.prepare(ctx, client, secret, svc)
			tassert.Nil(t, err)
			tassert.Equal(t, tt.expectMap, result)
		})
	}
}

func TestNewClient_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()
	createK8sSecret(ctx, t, k8sClient, namespace, tokenSecretName, tokenSecretKey, []byte("token"))

	tokenToIssue := "token-to-be-issued"
	notExistingSecretName := "not-existing-secret"
	notExistingSecretKey := "not-existing-secret-key"

	cache, err := lru.New(10)
	tassert.NoError(t, err)
	tokenService := &fakeTokenService{tokenToIssue: tokenToIssue}

	newProvider := func() *Provider {
		return &Provider{
			Logger: logger,
			NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
				return fake.NewFakeMysteryboxClient(mysteryboxService), nil
			},
			TokenService:           tokenService,
			mysteryboxClientsCache: cache,
		}
	}

	tests := []struct {
		name      string
		storeSpec func() *esv1.SecretStore
		expectErr string
	}{
		{
			name: "missing api domain",
			storeSpec: func() *esv1.SecretStore {
				return &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							NebiusMysterybox: &esv1.NebiusMysteryboxProvider{},
						},
					},
				}
			},
			expectErr: errMissingAPIDomain,
		},
		{
			name: "missing auth options",
			storeSpec: func() *esv1.SecretStore {
				return &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							NebiusMysterybox: &esv1.NebiusMysteryboxProvider{APIDomain: apiDomain},
						},
					},
				}
			},
			expectErr: errMissingAuthOptions,
		},
		{
			name: "specified token secret does not exist in kubernetes secrets",
			storeSpec: func() *esv1.SecretStore {
				return &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
								APIDomain: apiDomain,
								Auth: esv1.NebiusAuth{
									Token: esmeta.SecretKeySelector{
										Namespace: &namespace,
										Name:      notExistingSecretName,
										Key:       tokenSecretKey,
									},
								},
							},
						},
					},
				}
			},
			expectErr: fmt.Sprintf("read token secret %s/%s: cannot get Kubernetes secret", namespace, notExistingSecretName),
		},
		{
			name: "specified service account " +
				"creds secret does not exist in kubernetes secrets",
			storeSpec: func() *esv1.SecretStore {
				return &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
								APIDomain: apiDomain,
								Auth: esv1.NebiusAuth{
									ServiceAccountCreds: esmeta.SecretKeySelector{
										Namespace: &namespace,
										Name:      notExistingSecretName,
										Key:       "secretKey",
									},
								},
							},
						},
					},
				}
			},
			expectErr: fmt.Sprintf("read service account creds %s/%s: cannot get Kubernetes secret", namespace, notExistingSecretName),
		},
		{
			name: "specified token secret's key does not exist in the secret",
			storeSpec: func() *esv1.SecretStore {
				return &esv1.SecretStore{
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
					Spec: esv1.SecretStoreSpec{
						Provider: &esv1.SecretStoreProvider{
							NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
								APIDomain: apiDomain,
								Auth: esv1.NebiusAuth{
									Token: esmeta.SecretKeySelector{
										Namespace: &namespace,
										Name:      tokenSecretName,
										Key:       notExistingSecretKey,
									},
								},
							},
						},
					},
				}
			},
			expectErr: fmt.Sprintf("cannot find secret data for key: %q", notExistingSecretKey),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newProvider()
			store := tt.storeSpec()
			_, err := p.NewClient(ctx, store, k8sClient, namespace)
			tassert.Error(t, err)
			tassert.ErrorContains(t, err, tt.expectErr)
		})
	}
}

func TestNewClient_AuthWithSecretAccountCreds(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mysteryboxService.CreateSecret([]mysterybox.Entry{{Key: "k", StringValue: "v"}})

	providedCreds, _ := json.Marshal(&auth.ServiceAccountCredentials{
		SubjectCredentials: auth.SubjectCredentials{
			PrivateKey: "-----BEGIN PRIVATE KEY-----\nTEST-KEY\n-----END PRIVATE KEY-----",
			KeyID:      "keyId",
			Subject:    "subjectId",
			Issuer:     "subjectId",
		},
	})

	tokenToIssue := "token-to-be-issued"

	tokenService := &fakeTokenService{
		tokenToIssue: tokenToIssue,
	}

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		TokenService:           tokenService,
		mysteryboxClientsCache: cache,
	}

	createK8sSecret(ctx, t, k8sClient, namespace, authRefName, authRefKey, providedCreds)
	store := newNebiusMysteryboxSecretStoreWithServiceAccountCreds(apiDomain, namespace, authRefName, authRefKey)

	client, err := p.NewClient(ctx, store, k8sClient, namespace)
	tassert.NoError(t, err)

	msc, ok := client.(*SecretsClient)
	if !ok {
		t.Fatalf("expected *SecretsClient, got %T", client)
	}
	if got := msc.token; got != tokenToIssue {
		t.Fatalf("token mismatch: got %q want %q (issued by TokenService)", got, tokenToIssue)
	}

	// also ensure TokenService was exercised with the domain and creds we expect
	if tokenService.calls != 1 {
		t.Fatalf("expected TokenService to be called once, got %d", tokenService.calls)
	}
	if tokenService.gotDomain != apiDomain {
		t.Fatalf("TokenService called with wrong domain: got %q want %q", tokenService.gotDomain, apiDomain)
	}
	if tokenService.gotCreds != string(providedCreds) {
		t.Fatalf("TokenService called with wrong creds; got %q", tokenService.gotCreds)
	}
	if tokenService.gotCACert != nil {
		t.Fatalf("expected nil CA cert to be passed to TokenService, got non-nil")
	}

	got, err := msc.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "k"})
	tassert.NoError(t, err)
	tassert.Equal(t, []byte("v"), got)
}

func TestGetSecret_NotFound(t *testing.T) {
	// Use table-driven tests to cover all NotFound scenarios in one place
	cases := []struct {
		name           string
		entries        []mysterybox.Entry
		prepare        func(ctx context.Context, client *SecretsClient, secret *fake.Secret) ([]byte, error)
		expectErrEqual func(t *testing.T, err error, secret *fake.Secret)
	}{
		{
			name:    "SecretID not found",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, _ *fake.Secret) ([]byte, error) {
				desiredSecretId := "notexists"
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: desiredSecretId})
			},
			expectErrEqual: func(t *testing.T, err error, _ *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretNotFound, "notexists", esv1.NoSecretErr).Error())
			},
		},
		{
			name:    "Version not found",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret) ([]byte, error) {
				desiredVersion := "notexistversion"
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Version: desiredVersion})
			},
			expectErrEqual: func(t *testing.T, err error, secret *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretVersionNotFound, "notexistversion", secret.Id, esv1.NoSecretErr).Error())
			},
		},
		{
			name:    "Property key not found in version",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret) ([]byte, error) {
				desiredKey := "key"
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Version: secret.VersionId, Property: desiredKey})
			},
			expectErrEqual: func(t *testing.T, err error, secret *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretVersionByKeyNotFound, secret.VersionId, secret.Id, "key", esv1.NoSecretErr).Error())
			},
		},
		{
			name:    "Property key not found",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret) ([]byte, error) {
				desiredKey := "key"
				return client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: desiredKey})
			},
			expectErrEqual: func(t *testing.T, err error, secret *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretByKeyNotFound, "key", secret.Id, esv1.NoSecretErr).Error())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, client, secret, _, _ := setupClientWithTokenAuth(t, tc.entries, nil)
			_, err := tc.prepare(ctx, client, secret)
			tc.expectErrEqual(t, err, secret)
		})
	}
}

func TestGetSecretMap_NotFound(t *testing.T) {
	// Use table-driven tests to cover all NotFound scenarios in one place
	cases := []struct {
		name           string
		entries        []mysterybox.Entry
		prepare        func(ctx context.Context, client *SecretsClient, secret *fake.Secret) (map[string][]byte, error)
		expectErrEqual func(t *testing.T, err error, secret *fake.Secret)
	}{
		{
			name:    "SecretID not found",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, _ *fake.Secret) (map[string][]byte, error) {
				desiredSecretId := "notexists"
				return client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: desiredSecretId})
			},
			expectErrEqual: func(t *testing.T, err error, _ *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretNotFound, "notexists", esv1.NoSecretErr).Error())
			},
		},
		{
			name:    "Version not found",
			entries: []mysterybox.Entry{{Key: "key1", StringValue: "string"}},
			prepare: func(ctx context.Context, client *SecretsClient, secret *fake.Secret) (map[string][]byte, error) {
				desiredVersion := "notexistversion"
				return client.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Version: desiredVersion})
			},
			expectErrEqual: func(t *testing.T, err error, secret *fake.Secret) {
				tassert.Error(t, err)
				tassert.ErrorIs(t, err, esv1.NoSecretErr)
				tassert.EqualError(t, err, fmt.Errorf(errSecretVersionNotFound, "notexistversion", secret.Id, esv1.NoSecretErr).Error())
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, client, secret, _, _ := setupClientWithTokenAuth(t, tc.entries, nil)
			_, err := tc.prepare(ctx, client, secret)
			tc.expectErrEqual(t, err, secret)
		})
	}
}

func TestCreateOrGetMysteryboxClient_CachesByKey(t *testing.T) {
	ctx := context.Background()

	cache, err := lru.New(10)
	tassert.NoError(t, err)
	var factoryCalls int32
	p := &Provider{
		Logger: log,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	// same domain + same CA
	if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// different CA
	if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA2")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// different domain
	if _, err := p.createOrGetMysteryboxClient(ctx, "other.nebius.example", []byte("CA1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := atomic.LoadInt32(&factoryCalls), int32(3); got != want {
		t.Fatalf("factory called %d times, want %d (3 distinct keys)", got, want)
	}
}

func TestCreateOrGetMysteryboxClient_EmptyCA_EqualsNil(t *testing.T) {
	ctx := context.Background()

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	var factoryCalls int32
	p := &Provider{
		Logger: log,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got, want := atomic.LoadInt32(&factoryCalls), int32(1); got != want {
		t.Fatalf("factory called %d times, want %d when CA=nil and CA=empty should map to same key", got, want)
	}
}

func TestCreateOrGetMysteryboxClient_ConcurrentSingleCreation(t *testing.T) {
	ctx := context.Background()

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	var factoryCalls int32
	p := &Provider{
		Logger: log,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	const goroutines = 8
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			<-start
			if _, err := p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA1")); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if got, want := atomic.LoadInt32(&factoryCalls), int32(1); got != want {
		t.Fatalf("factory called %d times, want %d for concurrent same-key requests", got, want)
	}
}

func newNebiusMysteryboxSecretStoreWithAuthTokenKey(apiDomain, namespace, tokenSecretName, tokenSecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
					APIDomain: apiDomain,
					Auth: esv1.NebiusAuth{
						Token: esmeta.SecretKeySelector{
							Name: tokenSecretName,
							Key:  tokenSecretKey,
						},
					},
				},
			},
		},
	}
}

func newNebiusMysteryboxSecretStoreWithServiceAccountCreds(apiDomain, namespace, keySecretName, keySecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
					APIDomain: apiDomain,
					Auth: esv1.NebiusAuth{
						ServiceAccountCreds: esmeta.SecretKeySelector{
							Name: keySecretName,
							Key:  keySecretKey,
						},
					},
				},
			},
		},
	}
}

func createK8sSecret(ctx context.Context, t *testing.T, k8sClient k8sclient.Client, namespace, secretName, secretKey string, secretValue []byte) {
	err := k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Data: map[string][]byte{secretKey: secretValue},
	})
	tassert.Nil(t, err)
}

func unmarshalStringMap(t *testing.T, data []byte) map[string]string {
	stringMap := make(map[string]string)
	err := json.Unmarshal(data, &stringMap)
	tassert.Nil(t, err)
	return stringMap
}

func newProvider(t *testing.T, newMysteryboxClientFunc NewMysteryboxClient, tokenService TokenService) *Provider {
	t.Helper()
	cache, err := lru.New(10)
	tassert.NoError(t, err)
	return &Provider{
		Logger:                 logger,
		NewMysteryboxClient:    newMysteryboxClientFunc,
		mysteryboxClientsCache: cache,
		TokenService:           tokenService,
	}
}

func TestMysteryboxClientsCache_EvictionClosesClient(t *testing.T) {
	ctx := context.Background()

	mysteryboxClientsCacheSize = 1

	var created []*fake.FakeMysteryboxClient
	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			c := &fake.FakeMysteryboxClient{}
			created = append(created, c)
			return c, nil
		},
	}

	_, err := p.createOrGetMysteryboxClient(ctx, "domain-a", nil)
	tassert.Nil(t, err)

	tassert.Equal(t, 1, len(created), "expected 1 client created, got %d", len(created))

	_, err = p.createOrGetMysteryboxClient(ctx, "domain-b", nil)
	tassert.Nil(t, err)
	if len(created) != 2 {
		t.Fatalf("expected 2 clients created, got %d", len(created))
	}

	if got := atomic.LoadInt32(&created[0].Closed); got != 1 {
		t.Fatalf("expected first client Close() to be called once on eviction, got %d", got)
	}
	if got := atomic.LoadInt32(&created[1].Closed); got != 0 {
		t.Fatalf("expected second client not to be closed, got %d", got)
	}
}

type fakeTokenService struct {
	calls        int32
	returnError  bool
	gotDomain    string
	gotCreds     string
	gotCACert    []byte
	tokenToIssue string
}

func (f *fakeTokenService) GetToken(_ context.Context, apiDomain, subjectCreds string, caCert []byte) (string, error) {
	f.calls++
	f.gotDomain = apiDomain
	f.gotCreds = subjectCreds
	f.gotCACert = caCert
	if f.returnError {
		return "", errors.New("internal error")
	}
	return f.tokenToIssue, nil
}
