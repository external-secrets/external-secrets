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
package lockbox

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	b64 "encoding/base64"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client/fake"
)

const (
	errMissingKey                    = "invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name"
	errSecretPayloadPermissionDenied = "unable to request secret payload to get secret: permission denied"
	errSecretPayloadNotFound         = "unable to request secret payload to get secret: secret not found"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"

	store := &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				YandexLockbox: &esv1beta1.YandexLockboxProvider{},
			},
		},
	}
	provider, err := esv1beta1.GetProvider(store)
	tassert.Nil(t, err)

	k8sClient := clientfake.NewClientBuilder().Build()
	secretClient, err := provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingKey)
	tassert.Nil(t, secretClient)

	store.Spec.Provider.YandexLockbox.Auth = esv1beta1.YandexLockboxAuth{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingKey)
	tassert.Nil(t, secretClient)

	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey = esmeta.SecretKeySelector{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, errMissingKey)
	tassert.Nil(t, secretClient)

	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey.Name = authorizedKeySecretName
	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey.Key = authorizedKeySecretKey
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "could not fetch AuthorizedKey secret: secrets \"authorizedKeySecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, newFakeAuthorizedKey())
	tassert.Nil(t, err)

	const caCertificateSecretName = "caCertificateSecretName"
	const caCertificateSecretKey = "caCertificateSecretKey"
	store.Spec.Provider.YandexLockbox.CAProvider = &esv1beta1.YandexLockboxCAProvider{
		Certificate: esmeta.SecretKeySelector{
			Key:  caCertificateSecretKey,
			Name: caCertificateSecretName,
		},
	}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "could not fetch CA certificate secret: secrets \"caCertificateSecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, k8sClient, namespace, caCertificateSecretName, caCertificateSecretKey, newFakeCACertificate())
	tassert.Nil(t, err)
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "failed to create Yandex Lockbox client: private key parsing failed: invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key")
	tassert.Nil(t, secretClient)
}

func TestGetSecretForAllEntries(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		map[string]string{
			k1: v1,
			k2: base64(v2),
		},
		unmarshalStringMap(t, data),
	)
}

func TestGetSecretForTextEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, err)

	tassert.Equal(t, v1, string(data))
}

func TestGetSecretForBinaryEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Property: k2})
	tassert.Nil(t, err)

	tassert.Equal(t, v2, data)
}

func TestGetSecretByVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	newKey, newVal := "newKey", "newVal"
	newVersionID := lockboxBackend.AddVersion(secretID,
		textEntry(newKey, newVal),
	)

	data, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	data, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{newKey: newVal}, unmarshalStringMap(t, data))
}

func TestGetSecretUnauthorized(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKeyA := newFakeAuthorizedKey()
	authorizedKeyB := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	secretID, _ := lockboxBackend.CreateSecret(authorizedKeyA,
		textEntry("k1", "v1"),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKeyB)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
}

func TestGetSecretNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: "no-secret-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry("k1", "v1"),
	)
	_, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: "no-version-with-this-id"})
	tassert.EqualError(t, err, "unable to request secret payload to get secret: version not found")
}

func TestGetSecretWithTwoNamespaces(t *testing.T) {
	ctx := context.Background()
	namespace1 := uuid.NewString()
	namespace2 := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	secretID1, _ := lockboxBackend.CreateSecret(authorizedKey1,
		textEntry(k1, v1),
	)
	k2, v2 := "k2", "v2"
	secretID2, _ := lockboxBackend.CreateSecret(authorizedKey2,
		textEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace1, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey1)
	tassert.Nil(t, err)
	err = createK8sSecret(ctx, k8sClient, namespace2, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey2)
	tassert.Nil(t, err)
	store1 := newYandexLockboxSecretStore("", namespace1, authorizedKeySecretName, authorizedKeySecretKey)
	store2 := newYandexLockboxSecretStore("", namespace2, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient1, err := provider.NewClient(ctx, store1, k8sClient, namespace1)
	tassert.Nil(t, err)
	secretsClient2, err := provider.NewClient(ctx, store2, k8sClient, namespace2)
	tassert.Nil(t, err)

	data, err := secretsClient1.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)

	data, err = secretsClient2.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
	data, err = secretsClient2.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Equal(t, v2, string(data))
	tassert.Nil(t, err)
}

func TestGetSecretWithTwoApiEndpoints(t *testing.T) {
	ctx := context.Background()
	apiEndpoint1 := uuid.NewString()
	apiEndpoint2 := uuid.NewString()
	namespace := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	lockboxBackend1 := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	secretID1, _ := lockboxBackend1.CreateSecret(authorizedKey1,
		textEntry(k1, v1),
	)
	lockboxBackend2 := fake.NewLockboxBackend(time.Hour)
	k2, v2 := "k2", "v2"
	secretID2, _ := lockboxBackend2.CreateSecret(authorizedKey2,
		textEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName1 = "authorizedKeySecretName1"
	const authorizedKeySecretKey1 = "authorizedKeySecretKey1"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName1, authorizedKeySecretKey1, authorizedKey1)
	tassert.Nil(t, err)
	const authorizedKeySecretName2 = "authorizedKeySecretName2"
	const authorizedKeySecretKey2 = "authorizedKeySecretKey2"
	err = createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName2, authorizedKeySecretKey2, authorizedKey2)
	tassert.Nil(t, err)

	store1 := newYandexLockboxSecretStore(apiEndpoint1, namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexLockboxSecretStore(apiEndpoint2, namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider1 := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend1,
	})
	provider2 := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend2,
	})

	secretsClient1, err := provider1.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	secretsClient2, err := provider2.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)

	var data []byte

	data, err = secretsClient1.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	data, err = secretsClient2.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)
	data, err = secretsClient2.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Equal(t, v2, string(data))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenExpiration(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	tokenExpirationTime := time.Hour
	lockboxBackend := fake.NewLockboxBackend(tokenExpirationTime)
	k1, v1 := "k1", "v1"
	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(k1, v1),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})

	var data []byte

	oldSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = oldSecretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)

	lockboxBackend.AdvanceClock(2 * tokenExpirationTime)

	data, err = oldSecretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, "unable to request secret payload to get secret: iam token expired")

	newSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = newSecretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenCleanup(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	tokenExpirationDuration := time.Hour
	lockboxBackend := fake.NewLockboxBackend(tokenExpirationDuration)
	secretID1, _ := lockboxBackend.CreateSecret(authorizedKey1,
		textEntry("k1", "v1"),
	)
	secretID2, _ := lockboxBackend.CreateSecret(authorizedKey2,
		textEntry("k2", "v2"),
	)

	var err error

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName1 = "authorizedKeySecretName1"
	const authorizedKeySecretKey1 = "authorizedKeySecretKey1"
	err = createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName1, authorizedKeySecretKey1, authorizedKey1)
	tassert.Nil(t, err)
	const authorizedKeySecretName2 = "authorizedKeySecretName2"
	const authorizedKeySecretKey2 = "authorizedKeySecretKey2"
	err = createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName2, authorizedKeySecretKey2, authorizedKey2)
	tassert.Nil(t, err)

	store1 := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})

	tassert.False(t, provider.isIamTokenCached(authorizedKey1))
	tassert.False(t, provider.isIamTokenCached(authorizedKey2))

	// Access secretID1 with authorizedKey1, IAM token for authorizedKey1 should be cached
	secretsClient, err := provider.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID1})
	tassert.Nil(t, err)

	tassert.True(t, provider.isIamTokenCached(authorizedKey1))
	tassert.False(t, provider.isIamTokenCached(authorizedKey2))

	lockboxBackend.AdvanceClock(tokenExpirationDuration * 2)

	// Access secretID2 with authorizedKey2, IAM token for authorizedKey2 should be cached
	secretsClient, err = provider.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID2})
	tassert.Nil(t, err)

	tassert.True(t, provider.isIamTokenCached(authorizedKey1))
	tassert.True(t, provider.isIamTokenCached(authorizedKey2))

	lockboxBackend.AdvanceClock(tokenExpirationDuration)

	tassert.True(t, provider.isIamTokenCached(authorizedKey1))
	tassert.True(t, provider.isIamTokenCached(authorizedKey2))

	provider.cleanUpIamTokenMap()

	tassert.False(t, provider.isIamTokenCached(authorizedKey1))
	tassert.True(t, provider.isIamTokenCached(authorizedKey2))

	lockboxBackend.AdvanceClock(tokenExpirationDuration)

	tassert.False(t, provider.isIamTokenCached(authorizedKey1))
	tassert.True(t, provider.isIamTokenCached(authorizedKey2))

	provider.cleanUpIamTokenMap()

	tassert.False(t, provider.isIamTokenCached(authorizedKey1))
	tassert.False(t, provider.isIamTokenCached(authorizedKey2))
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		map[string][]byte{
			k1: []byte(v1),
			k2: v2,
		},
		data,
	)
}

func TestGetSecretMapByVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	lockboxBackend := fake.NewLockboxBackend(time.Hour)
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(&fake.YandexCloudCreator{
		Backend: lockboxBackend,
	})
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	newKey, newVal := "newKey", "newVal"
	newVersionID := lockboxBackend.AddVersion(secretID,
		textEntry(newKey, newVal),
	)

	data, err = secretsClient.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	data, err = secretsClient.GetSecretMap(ctx, esv1beta1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{newKey: []byte(newVal)}, data)
}

// helper functions

func newYandexLockboxSecretStore(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1beta1.GenericStore {
	return &esv1beta1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				YandexLockbox: &esv1beta1.YandexLockboxProvider{
					APIEndpoint: apiEndpoint,
					Auth: esv1beta1.YandexLockboxAuth{
						AuthorizedKey: esmeta.SecretKeySelector{
							Name: authorizedKeySecretName,
							Key:  authorizedKeySecretKey,
						},
					},
				},
			},
		},
	}
}

func createK8sSecret(ctx context.Context, k8sClient client.Client, namespace, secretName, secretKey string, secretContent interface{}) error {
	data, err := json.Marshal(secretContent)
	if err != nil {
		return err
	}

	err = k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Data: map[string][]byte{secretKey: data},
	})
	if err != nil {
		return err
	}

	return nil
}

func newFakeAuthorizedKey() *iamkey.Key {
	uniqueLabel := uuid.NewString()
	return &iamkey.Key{
		Id: uniqueLabel,
		Subject: &iamkey.Key_ServiceAccountId{
			ServiceAccountId: uniqueLabel,
		},
		PrivateKey: uniqueLabel,
	}
}

func newFakeCACertificate() []byte {
	cert := x509.Certificate{
		SerialNumber: big.NewInt(2019),
		Subject: pkix.Name{
			Organization:  []string{"Company, INC."},
			Country:       []string{"US"},
			Locality:      []string{"San Francisco"},
			StreetAddress: []string{"Golden Gate Bridge"},
			PostalCode:    []string{"94016"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}
	return cert.Raw
}

func textEntry(key, value string) *lockbox.Payload_Entry {
	return &lockbox.Payload_Entry{
		Key: key,
		Value: &lockbox.Payload_Entry_TextValue{
			TextValue: value,
		},
	}
}

func binaryEntry(key string, value []byte) *lockbox.Payload_Entry {
	return &lockbox.Payload_Entry{
		Key: key,
		Value: &lockbox.Payload_Entry_BinaryValue{
			BinaryValue: value,
		},
	}
}

func unmarshalStringMap(t *testing.T, data []byte) map[string]string {
	stringMap := make(map[string]string)
	err := json.Unmarshal(data, &stringMap)
	tassert.Nil(t, err)
	return stringMap
}

func base64(data []byte) string {
	return b64.StdEncoding.EncodeToString(data)
}
