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
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru"
	"github.com/nebius/gosdk/auth"
	tassert "github.com/stretchr/testify/assert"
	trequire "github.com/stretchr/testify/require"
	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	clocktesting "k8s.io/utils/clock/testing"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	nebiusauth "github.com/external-secrets/external-secrets/providers/v1/nebius/common/auth"
	nebiusiam "github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/iam"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox"
	"github.com/external-secrets/external-secrets/providers/v1/nebius/common/sdk/mysterybox/fake"
	utilfake "github.com/external-secrets/external-secrets/runtime/util/fake"
)

const (
	tokenSecretName       = "tokenSecretName"
	tokenSecretKey        = "tokenSecretKey"
	saCredsSecretName     = "saCredsSecretName"
	saCredsSecretKey      = "saCredsSecretKey"
	authRefName           = "authRefSecretName"
	authRefKey            = "authRefSecretKey"
	apiDomain             = "api.public"
	tokenToBeIssued       = "token-to-be-issued"
	serviceAccountName    = "testServiceAccountName"
	iamServiceAccount     = "serviceaccount-e00test"
	workloadIdentityToken = "test-service-account-jwt"
	correctPrivateKey     = "-----BEGIN PRIVATE KEY-----\nTEST-KEY\n-----END PRIVATE KEY-----"
)

var (
	logger           = ctrl.Log.WithName("provider").WithName("nebius").WithName("mysterybox")
	audiences        = []string{"test-nebius-audience"}
	testSubjectCreds = &auth.ServiceAccountCredentials{
		SubjectCredentials: auth.SubjectCredentials{
			PrivateKey: correctPrivateKey,
			KeyID:      "keyId",
			Subject:    "subjectId",
			Issuer:     "subjectId",
		},
	}
)

func setupClientWithTokenAuth(t *testing.T, entries []mysterybox.Entry, tokenGetter TokenGetter) (context.Context, *SecretsClient, *fake.Secret, k8sclient.Client, *fake.MysteryboxService) {
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
		tokenGetter,
	)
	createK8sSecret(ctx, t, k8sClient, namespace, tokenSecretName, tokenSecretKey, []byte("token"))
	store := newNebiusMysteryboxSecretStoreWithAuthTokenKey(apiDomain, namespace, tokenSecretName, tokenSecretKey)
	client, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.NoError(t, err)

	mysteryboxSecretsClient, ok := client.(*SecretsClient)
	tassert.True(t, ok, "expected *SecretsClient, got %T", client)
	return ctx, mysteryboxSecretsClient, secret, k8sClient, mysteryboxService
}

func TestNewClient_GetTokenError(t *testing.T) {
	t.Parallel()
	tokenGetter := faketokenGetter{returnError: true}

	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()
	subjectCreds, err := json.Marshal(testSubjectCreds)
	if err != nil {
		t.Fatal(err)
	}
	createK8sSecret(ctx, t, k8sClient, namespace, saCredsSecretName, saCredsSecretKey, subjectCreds)

	provider := newProvider(
		t,
		func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		&tokenGetter,
	)

	_, err = provider.NewClient(ctx, newNebiusMysteryboxSecretStoreWithServiceAccountAndPrivateKeyCreds(apiDomain, namespace, saCredsSecretName, saCredsSecretKey), k8sClient, namespace)
	tassert.Error(t, err)
	tassert.ErrorContains(t, err, "failed to retrieve iam token by credentials")
}

func TestGetSecret(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			ctx, client, secret, _, svc := setupClientWithTokenAuth(t, entries, nil)
			result, err := tt.prepare(ctx, client, secret, svc)
			tassert.NoError(t, err)
			if tt.expectJSON != nil {
				tassert.Equal(t, tt.expectJSON, unmarshalStringMap(t, result))
			} else {
				tassert.Equal(t, tt.expectRaw, result)
			}
		})
	}
}

func TestGetSecret_ByVersionId(t *testing.T) {
	t.Parallel()
	ctx, client, secret, _, mboxService := setupClientWithTokenAuth(t, []mysterybox.Entry{{Key: "key", StringValue: "string_value"}}, nil)
	_, err := mboxService.CreateNewSecretVersion(secret.Id, []mysterybox.Entry{
		{Key: "new_key", StringValue: "updated_string_value"},
		{Key: "new", StringValue: "new"},
	})
	tassert.NoError(t, err)

	result, err := client.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "key", Version: secret.VersionId})
	tassert.NoError(t, err)
	tassert.Equal(t, []byte("string_value"), result)
}

func TestGetSecretMap(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			ctx, client, secret, _, svc := setupClientWithTokenAuth(t, tt.entries, nil)
			result, err := tt.prepare(ctx, client, secret, svc)
			tassert.NoError(t, err)
			tassert.Equal(t, tt.expectMap, result)
		})
	}
}

func TestNewClient_ValidationErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()
	createK8sSecret(ctx, t, k8sClient, namespace, tokenSecretName, tokenSecretKey, []byte("token"))

	tokenToIssue := tokenToBeIssued
	notExistingSecretName := "not-existing-secret"
	notExistingSecretKey := "not-existing-secret-key"

	cache, err := lru.New(10)
	tassert.NoError(t, err)
	tokenGetter := &faketokenGetter{tokenToIssue: tokenToIssue}

	newProvider := func() *Provider {
		return &Provider{
			Logger: logger,
			NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
				return fake.NewFakeMysteryboxClient(mysteryboxService), nil
			},
			TokenGetter:            tokenGetter,
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
			name: "specified service account creds secret does not exist in kubernetes secrets",
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
			expectErr: fmt.Sprintf("error read service account secret creds %s/%s", namespace, notExistingSecretName),
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
			t.Parallel()
			p := newProvider()
			store := tt.storeSpec()
			_, err := p.NewClient(ctx, store, k8sClient, namespace)
			tassert.Error(t, err)
			tassert.ErrorContains(t, err, tt.expectErr)
		})
	}
}

func TestNewClient_AuthWithSecretAccountCreds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mysteryboxService.CreateSecret([]mysterybox.Entry{{Key: "k", StringValue: "v"}})

	providedCreds, _ := json.Marshal(testSubjectCreds)

	tokenToIssue := tokenToBeIssued

	tokenExchanger := &nebiusiam.FakeTokenExchanger{
		TokenToIssue: tokenToIssue,
	}
	tokenGetter := newTestCachedTokenGetter(t, tokenExchanger)

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		mysteryboxClientsCache: cache,
		TokenGetter:            tokenGetter,
	}

	createK8sSecret(ctx, t, k8sClient, namespace, authRefName, authRefKey, providedCreds)
	store := newNebiusMysteryboxSecretStoreWithServiceAccountAndPrivateKeyCreds(apiDomain, namespace, authRefName, authRefKey)

	client, err := p.NewClient(ctx, store, k8sClient, namespace)
	tassert.NoError(t, err)

	msc, ok := client.(*SecretsClient)
	tassert.True(t, ok, "expected *MysteryboxSecretsClient, got %T", client)
	tassert.Equal(t, tokenToIssue, msc.token, fmt.Sprintf("token mismatch: got %q want %q (issued by TokenGetter)", msc.token, tokenToIssue))

	// also ensure TokenExchanger was exercised with the domain and creds we expect
	tassert.Equal(t, int64(1), tokenExchanger.Calls.Load(), "expected TokenExchanger to be called once")
	tassert.Equal(t, apiDomain, tokenExchanger.DomainRequest, "expected TokenExchanger to be called with the correct domain")
	tassert.Equal(t, testSubjectCreds.SubjectCredentials.KeyID, tokenExchanger.KeyIDRequest, "expected TokenExchanger to be called with the correct creds")
	tassert.Equal(t, testSubjectCreds.SubjectCredentials.PrivateKey, tokenExchanger.PrivateKeyRequest, "expected TokenExchanger to be called with the correct creds")
	tassert.Equal(t, testSubjectCreds.SubjectCredentials.Subject, tokenExchanger.SubjectRequest, "expected TokenExchanger to be called with the correct creds")
	tassert.Nil(t, tokenExchanger.CaRequest, "expected TokenExchanger to be called without CA cert")

	got, err := msc.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "k"})
	tassert.NoError(t, err)
	tassert.Equal(t, []byte("v"), got)
}

func TestNewClient_AuthWithWorkloadIdentityCreds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mysteryboxService.CreateSecret([]mysterybox.Entry{{Key: "k1", StringValue: "v1"}})

	tokenToIssue := tokenToBeIssued

	tokenExchanger := &nebiusiam.FakeTokenExchanger{
		TokenToIssue: tokenToIssue,
	}
	tokenGetter := newTestCachedTokenGetter(t, tokenExchanger)

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	coreV1Client := &recordingCoreV1{
		CoreV1Interface: utilfake.NewCreateTokenMock().
			WithToken(workloadIdentityToken),
		onTokenRequest: func(request *authenticationv1.TokenRequest) {
			expectedAudiences := append([]string{nebiusauth.NebiusIamAudience}, audiences...)
			tassert.Equal(
				t,
				expectedAudiences,
				request.Spec.Audiences,
			)
		},
	}

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		coreV1Client:           coreV1Client,
		mysteryboxClientsCache: cache,
		TokenGetter:            tokenGetter,
	}

	createK8sServiceAccount(ctx, t, k8sClient, namespace, serviceAccountName)
	store := newNebiusMysteryboxSecretStoreWithK8sServiceAccountCreds(apiDomain, namespace, serviceAccountName, iamServiceAccount, audiences)

	client, err := p.NewClient(ctx, store, k8sClient, namespace)
	tassert.NoError(t, err)

	msc, ok := client.(*SecretsClient)
	tassert.True(t, ok, "expected *MysteryboxSecretsClient, got %T", client)
	tassert.Equal(t, tokenToIssue, msc.token, fmt.Sprintf("token mismatch: got %q want %q (issued by TokenGetter)", msc.token, tokenToIssue))

	// also ensure TokenExchanger was exercised with creds we expect
	tassert.Equal(t, int64(1), tokenExchanger.Calls.Load(), "expected TokenExchanger to be called once")
	tassert.Equal(t, apiDomain, tokenExchanger.DomainRequest, "expected TokenExchanger to be called with the correct domain")
	tassert.Equal(t, workloadIdentityToken, tokenExchanger.TokenRequest, "expected jwt token to be received")
	tassert.Equal(t, iamServiceAccount, tokenExchanger.ServiceAccountIDRequest, "expected serviceAccount to be correct ")
	tassert.Nil(t, tokenExchanger.CaRequest, "expected TokenExchanger to be called without CA cert")

	got, err := msc.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "k1"})
	tassert.NoError(t, err)
	tassert.Equal(t, []byte("v1"), got)
}

func TestNewClient_AuthWithWorkloadIdentityCredsWithoutAudiences(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	namespace := uuid.NewString()
	mysteryboxService := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mysteryboxService.CreateSecret([]mysterybox.Entry{{Key: "k1", StringValue: "v1"}})

	tokenToIssue := tokenToBeIssued

	tokenExchanger := &nebiusiam.FakeTokenExchanger{
		TokenToIssue: tokenToIssue,
	}
	tokenGetter := newTestCachedTokenGetter(t, tokenExchanger)

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	coreV1Client := &recordingCoreV1{
		CoreV1Interface: utilfake.NewCreateTokenMock().
			WithToken(workloadIdentityToken),
		onTokenRequest: func(request *authenticationv1.TokenRequest) {
			tassert.Equal(
				t,
				[]string{nebiusauth.NebiusIamAudience},
				request.Spec.Audiences,
			)
		},
	}

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			return fake.NewFakeMysteryboxClient(mysteryboxService), nil
		},
		coreV1Client:           coreV1Client,
		mysteryboxClientsCache: cache,
		TokenGetter:            tokenGetter,
	}

	createK8sServiceAccount(ctx, t, k8sClient, namespace, serviceAccountName)
	store := newNebiusMysteryboxSecretStoreWithK8sServiceAccountCreds(apiDomain, namespace, serviceAccountName, iamServiceAccount, nil)

	client, err := p.NewClient(ctx, store, k8sClient, namespace)
	tassert.NoError(t, err)

	msc, ok := client.(*SecretsClient)
	tassert.True(t, ok, "expected *MysteryboxSecretsClient, got %T", client)
	tassert.Equal(t, tokenToIssue, msc.token, fmt.Sprintf("token mismatch: got %q want %q (issued by TokenGetter)", msc.token, tokenToIssue))

	// also ensure TokenExchanger was exercised with creds we expect
	tassert.Equal(t, int64(1), tokenExchanger.Calls.Load(), "expected TokenExchanger to be called once")
	tassert.Equal(t, apiDomain, tokenExchanger.DomainRequest, "expected TokenExchanger to be called with the correct domain")
	tassert.Equal(t, workloadIdentityToken, tokenExchanger.TokenRequest, "expected jwt token to be received")
	tassert.Equal(t, iamServiceAccount, tokenExchanger.ServiceAccountIDRequest, "expected serviceAccount to be correct ")
	tassert.Nil(t, tokenExchanger.CaRequest, "expected TokenExchanger to be called without CA cert")

	got, err := msc.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "k1"})
	tassert.NoError(t, err)
	tassert.Equal(t, []byte("v1"), got)
}

func TestGetSecret_NotFound(t *testing.T) {
	t.Parallel()
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
			t.Parallel()
			ctx, client, secret, _, _ := setupClientWithTokenAuth(t, tc.entries, nil)
			_, err := tc.prepare(ctx, client, secret)
			tc.expectErrEqual(t, err, secret)
		})
	}
}

func TestGetSecretMap_NotFound(t *testing.T) {
	t.Parallel()

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
			t.Parallel()
			ctx, client, secret, _, _ := setupClientWithTokenAuth(t, tc.entries, nil)
			_, err := tc.prepare(ctx, client, secret)
			tc.expectErrEqual(t, err, secret)
		})
	}
}

func TestCreateOrGetMysteryboxClient_CachesByKey(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cache, err := lru.New(10)
	tassert.NoError(t, err)
	var factoryCalls int32
	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	// same domain + same CA
	_, err = p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA1"))
	tassert.NoError(t, err)
	_, err = p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA1"))
	tassert.NoError(t, err)

	// different CA
	_, err = p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte("CA2"))
	tassert.NoError(t, err)

	_, err = p.createOrGetMysteryboxClient(ctx, "other.nebius.example", []byte("CA1"))
	tassert.NoError(t, err)

	tassert.Equal(t, int32(3), atomic.LoadInt32(&factoryCalls), fmt.Sprintf("factory called %d times, want %d", atomic.LoadInt32(&factoryCalls), 3))
}

func TestCreateOrGetMysteryboxClient_EmptyCA_EqualsNil(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cache, err := lru.New(10)
	tassert.NoError(t, err)

	var factoryCalls int32
	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	_, err = p.createOrGetMysteryboxClient(ctx, "api.nebius.example", nil)
	tassert.NoError(t, err)
	_, err = p.createOrGetMysteryboxClient(ctx, "api.nebius.example", []byte{})
	tassert.NoError(t, err)

	tassert.Equal(t, int32(1), atomic.LoadInt32(&factoryCalls), fmt.Sprintf("factory called %d times, want %d when CA=nil and CA=empty should map to same key", factoryCalls, 1))
}

func TestMysteryboxClientsCache_EvictionClosesClient(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	var created []*fake.FakeMysteryboxClient
	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			c := &fake.FakeMysteryboxClient{}
			created = append(created, c)
			return c, nil
		},
	}

	setCacheSizeWorkaround(t, 1, p)

	_, err := p.createOrGetMysteryboxClient(ctx, "domain-a", nil)
	tassert.NoError(t, err)

	tassert.Len(t, created, 1, "expected 1 client created, got %d", len(created))

	_, err = p.createOrGetMysteryboxClient(ctx, "domain-b", nil)
	tassert.NoError(t, err)

	tassert.Len(t, created, 2, "expected 2 clients created, got %d", len(created))

	// clients are not closed because of the eviction policy for provider's mysterybox clients connections cache
	tassert.Equal(t, int32(0), atomic.LoadInt32(&created[0].Closed), "expected second client not to be closed")
	tassert.Equal(t, int32(0), atomic.LoadInt32(&created[1].Closed), "expected second client not to be closed")
}

// concurrent tests

func TestCreateOrGetMysteryboxClient_Concurrent_SingleClient(t *testing.T) {
	t.Parallel()
	clientData := ClientData{domain: "api.nebius.example", ca: []byte("CA1")}

	ctx := context.Background()
	cache, err := lru.New(10)
	tassert.NoError(t, err)

	var factoryCalls int32
	p := &Provider{
		Logger: logger,
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

	errs := make([]error, goroutines)
	for i := range goroutines {
		go func(ix int) {
			defer wg.Done()
			<-start
			_, err := p.createOrGetMysteryboxClient(ctx, clientData.domain, clientData.ca)
			errs[ix] = err
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			tassert.NoError(t, err, "goroutine %d", i)
		}
	}

	tassert.Equal(t, int32(1), atomic.LoadInt32(&factoryCalls), fmt.Sprintf("factory called %d times, want %d for concurrent same-key requests", factoryCalls, 1))
}

func TestCreateOrGetMysteryboxClient_Concurrent_MultipleClients(t *testing.T) {
	clientRequests := []ClientData{
		{domain: "api.nebius.example1", ca: []byte("CA1")},
		{domain: "api.nebius.example1", ca: []byte("CA1")}, // duplicate
		{domain: "api.nebius.example1", ca: []byte("CA2")}, // the same domain, different CA
		{domain: "api.nebius.example2", ca: []byte("CA2")}, // different domain, the same CA
		{domain: "api.nebius.example1", ca: []byte{}},      // the same domain, empty CA
	}

	ctx := context.Background()
	cache, err := lru.New(10)
	tassert.NoError(t, err)
	var factoryCalls int32
	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(nil), nil
		},
		mysteryboxClientsCache: cache,
	}

	var wg sync.WaitGroup
	wg.Add(len(clientRequests))
	start := make(chan struct{})

	errs := make([]error, len(clientRequests))
	for i, r := range clientRequests {
		go func(ix int, req ClientData) {
			defer wg.Done()
			<-start
			_, err := p.createOrGetMysteryboxClient(ctx, req.domain, req.ca)
			errs[ix] = err
		}(i, r)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			tassert.NoError(t, err, "goroutine %d", i)
		}
	}

	tassert.Equal(t, int32(4), atomic.LoadInt32(&factoryCalls), fmt.Sprintf("factory called %d times, want %d", atomic.LoadInt32(&factoryCalls), 4))
}

func TestMysteryboxClientsCache_ConcurrentEviction_CloseOnce(t *testing.T) {
	ctx := context.Background()

	var created []*fake.FakeMysteryboxClient
	var mu sync.Mutex

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			c := &fake.FakeMysteryboxClient{}
			mu.Lock()
			created = append(created, c)
			mu.Unlock()
			return c, nil
		},
	}
	setCacheSizeWorkaround(t, 1, p)

	var wg sync.WaitGroup
	wg.Add(3)
	start := make(chan struct{})

	go func() {
		defer wg.Done()
		<-start
		_, _ = p.createOrGetMysteryboxClient(ctx, "domain-a", nil)
	}()

	go func() {
		defer wg.Done()
		<-start
		_, _ = p.createOrGetMysteryboxClient(ctx, "domain-b", nil)
	}()
	go func() {
		defer wg.Done()
		<-start
		_, _ = p.createOrGetMysteryboxClient(ctx, "domain-c", nil)
	}()

	close(start)
	wg.Wait()

	tassert.Len(t, created, 3, "expected 3 clients created, got %d", len(created))
	// clients are not closed because of the eviction policy for provider's mysterybox clients connections cache
	tassert.Equal(t, int32(0), atomic.LoadInt32(&created[0].Closed), "expected first client not to be closed on eviction")
	tassert.Equal(t, int32(0), atomic.LoadInt32(&created[1].Closed), "expected first client not to be closed on eviction")
	tassert.Equal(t, int32(0), atomic.LoadInt32(&created[2].Closed), "expected first client not to be closed on eviction")
}

func TestNewClient_Concurrent_SameConfig_SingleClient_DifferentTokens(t *testing.T) {
	ctx := context.Background()

	namespace := uuid.NewString()
	mboxSvc := fake.InitMysteryboxService()
	k8sClient := clientfake.NewClientBuilder().Build()

	secret := mboxSvc.CreateSecret([]mysterybox.Entry{{Key: "k", StringValue: "v"}})

	var factoryCalls int32
	tokenToIssue := tokenToBeIssued

	tokenGetter := &faketokenGetter{tokenToIssue: tokenToIssue}

	p := &Provider{
		Logger: logger,
		NewMysteryboxClient: func(ctx context.Context, apiDomain string, caCertificate []byte) (mysterybox.Client, error) {
			atomic.AddInt32(&factoryCalls, 1)
			return fake.NewFakeMysteryboxClient(mboxSvc), nil
		},
	}
	setTokenGetterWorkaround(tokenGetter, p)

	creds, err := json.Marshal(testSubjectCreds)
	trequire.NoError(t, err)
	createK8sSecret(ctx, t, k8sClient, namespace, authRefName, authRefKey, creds)

	store := newNebiusMysteryboxSecretStoreWithServiceAccountAndPrivateKeyCreds(apiDomain, namespace, authRefName, authRefKey)

	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	start := make(chan struct{})

	clients := make([]esv1.SecretsClient, goroutines)
	errs := make([]error, goroutines)

	for i := range goroutines {
		go func(ix int) {
			defer wg.Done()
			<-start
			c, err := p.NewClient(ctx, store, k8sClient, namespace)
			clients[ix], errs[ix] = c, err
		}(i)
	}
	close(start)
	wg.Wait()

	for i := range goroutines {
		tassert.NoError(t, errs[i], "NewClient error: %w", errs[i])
		msc := clients[i].(*SecretsClient)
		got, err := msc.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secret.Id, Property: "k"})
		tassert.NoError(t, err)
		tassert.Equal(t, []byte("v"), got)
	}

	tassert.Equal(t, int32(goroutines), atomic.LoadInt32(&tokenGetter.calls), fmt.Sprintf("TokenGetter.GetToken called %d times, want %d", factoryCalls, goroutines))
	tassert.Equal(t, int32(1), atomic.LoadInt32(&factoryCalls), fmt.Sprintf("NewMysteryboxClient called %d times, want 1", factoryCalls))
}

// helpers

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

func newNebiusMysteryboxSecretStoreWithServiceAccountAndPrivateKeyCreds(apiDomain, namespace, keySecretName, keySecretKey string) esv1.GenericStore {
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
func newNebiusMysteryboxSecretStoreWithK8sServiceAccountCreds(apiDomain, namespace, serviceAccountName, iamServiceAccount string, audiences []string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			UID:       types.UID(uuid.New().String()),
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				NebiusMysterybox: &esv1.NebiusMysteryboxProvider{
					APIDomain: apiDomain,
					Auth: esv1.NebiusAuth{
						WorkloadIdentity: &esv1.NebiusWorkloadIdentity{
							ServiceAccountRef: &esmeta.ServiceAccountSelector{
								Name:      serviceAccountName,
								Namespace: &namespace,
								Audiences: audiences,
							},
							IAMServiceAccountID: iamServiceAccount,
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
	tassert.NoError(t, err)
}
func createK8sServiceAccount(ctx context.Context, t *testing.T, k8sClient k8sclient.Client, namespace, serviceAccountName string) {
	err := k8sClient.Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      serviceAccountName,
		},
	})
	tassert.NoError(t, err)
}

func unmarshalStringMap(t *testing.T, data []byte) map[string]string {
	stringMap := make(map[string]string)
	err := json.Unmarshal(data, &stringMap)
	tassert.NoError(t, err)
	return stringMap
}

func newProvider(t *testing.T, newMysteryboxClientFunc NewMysteryboxClient, tokenGetter TokenGetter) *Provider {
	t.Helper()
	cache, err := lru.New(10)
	tassert.NoError(t, err)
	return &Provider{
		Logger:                 logger,
		NewMysteryboxClient:    newMysteryboxClientFunc,
		mysteryboxClientsCache: cache,
		TokenGetter:            tokenGetter,
	}
}

func newTestCachedTokenGetter(t *testing.T, tokenExchanger nebiusiam.TokenExchanger) *CachedTokenGetter {
	t.Helper()
	tokenGetter, err := NewCachedTokenGetter(
		mysteryboxTokensCacheSize,
		tokenExchanger,
		clocktesting.NewFakeClock(time.Unix(0, 0)),
	)
	trequire.NoError(t, err)
	return tokenGetter
}

type faketokenGetter struct {
	calls        int32
	tokenToIssue string
	returnError  bool

	mu sync.Mutex
}

func (f *faketokenGetter) GetToken(_ context.Context, _ string, _ *nebiusauth.CredentialRequest, _ []byte) (string, error) {
	atomic.AddInt32(&f.calls, 1)

	if f.returnError {
		return "", errors.New("internal error")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	return f.tokenToIssue, nil
}

func setCacheSizeWorkaround(t *testing.T, size int, p *Provider) {
	t.Helper()
	err := p.initMysteryboxClientsCache()
	tassert.NoError(t, err)
	p.mysteryboxClientsCache.Resize(size)
}

func setTokenGetterWorkaround(tokenGetter TokenGetter, p *Provider) {
	p.tokenInitMutex.Lock()
	defer p.tokenInitMutex.Unlock()

	p.TokenGetter = tokenGetter
}

type ClientData struct {
	domain string
	ca     []byte
}

type recordingCoreV1 struct {
	typedcorev1.CoreV1Interface
	onTokenRequest func(*authenticationv1.TokenRequest)
}

func (c *recordingCoreV1) ServiceAccounts(namespace string) typedcorev1.ServiceAccountInterface {
	return &recordingServiceAccounts{
		ServiceAccountInterface: c.CoreV1Interface.ServiceAccounts(namespace),
		onTokenRequest:          c.onTokenRequest,
	}
}

type recordingServiceAccounts struct {
	typedcorev1.ServiceAccountInterface
	onTokenRequest func(*authenticationv1.TokenRequest)
}

func (s *recordingServiceAccounts) CreateToken(
	ctx context.Context,
	name string,
	request *authenticationv1.TokenRequest,
	opts metav1.CreateOptions,
) (*authenticationv1.TokenRequest, error) {
	s.onTokenRequest(request.DeepCopy())

	return s.ServiceAccountInterface.CreateToken(
		ctx,
		name,
		request,
		opts,
	)
}
