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

package lockbox

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	ydxcommon "github.com/external-secrets/external-secrets/providers/v1/yandex/common"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/common/clock"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/lockbox/client"
)

const (
	errMissingKey                    = "invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name"
	errSecretPayloadPermissionDenied = "unable to request secret payload to get secret: permission denied"
	errSecretPayloadNotFound         = "unable to request secret payload to get secret: secret not found"
	errSecretPayloadVersionNotFound  = "unable to request secret payload to get secret: version not found"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"

	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexLockbox: &esv1.YandexLockboxProvider{
					Auth: esv1.YandexAuth{
						AuthorizedKey: esmeta.SecretKeySelector{
							Key:  authorizedKeySecretKey,
							Name: authorizedKeySecretName,
						},
					},
				},
			},
		},
	}
	esv1.Register(NewProvider(), ProviderSpec(), MaintenanceStatus())
	provider, err := esv1.GetProvider(store)
	tassert.Nil(t, err)

	k8sClient := clientfake.NewClientBuilder().Build()
	secretClient, err := provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "cannot get Kubernetes secret \"authorizedKeySecretName\" from namespace \"namespace\": secrets \"authorizedKeySecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, newFakeAuthorizedKey()))
	tassert.Nil(t, err)

	const caCertificateSecretName = "caCertificateSecretName"
	const caCertificateSecretKey = "caCertificateSecretKey"
	store.Spec.Provider.YandexLockbox.CAProvider = &esv1.YandexCAProvider{
		Certificate: esmeta.SecretKeySelector{
			Key:  caCertificateSecretKey,
			Name: caCertificateSecretName,
		},
	}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "cannot get Kubernetes secret \"caCertificateSecretName\" from namespace \"namespace\": secrets \"caCertificateSecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, t, k8sClient, namespace, caCertificateSecretName, caCertificateSecretKey, []byte("it-is-not-a-certificate"))
	tassert.Nil(t, err)
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "failed to create Yandex.Cloud client: unable to read trusted CA certificates")
	tassert.Nil(t, secretClient)
}

func TestGetSecretForAllEntries(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID})
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

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, err)

	tassert.Equal(t, v1, string(data))
}

func TestGetSecretForBinaryEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k2})
	tassert.Nil(t, err)

	tassert.Equal(t, v2, data)
}

func TestGetSecretByVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	const oldKey, oldVal = "oldKey", "oldVal"
	secretID, oldVersionID := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	const newKey, newVal = "newKey", "newVal"
	newVersionID := fakeLockboxServer.AddVersion(secretID, textEntry(newKey, newVal))

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{newKey: newVal}, unmarshalStringMap(t, data))
}

func TestGetSecretUnauthorized(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKeyA := newFakeAuthorizedKey()
	authorizedKeyB := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKeyA,
		"folderId", "secretName",
		textEntry("k1", "v1"),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKeyB))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
}

func TestGetSecretNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "no-secret-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry("k1", "v1"),
	)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: "no-version-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadVersionNotFound)
}

func TestGetSecretWithTwoNamespaces(t *testing.T) {
	ctx := context.Background()
	namespace1 := uuid.NewString()
	namespace2 := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	secretID1, _ := fakeLockboxServer.CreateSecret(authorizedKey1,
		"folderId", "secretName1",
		textEntry(k1, v1),
	)
	k2, v2 := "k2", "v2"
	secretID2, _ := fakeLockboxServer.CreateSecret(authorizedKey2,
		"folderId", "secretName2",
		textEntry(k2, v2),
		textEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace1, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey1))
	tassert.Nil(t, err)
	err = createK8sSecret(ctx, t, k8sClient, namespace2, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey2))
	tassert.Nil(t, err)
	store1 := newYandexLockboxSecretStore("", namespace1, authorizedKeySecretName, authorizedKeySecretKey)
	store2 := newYandexLockboxSecretStore("", namespace2, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient1, err := provider.NewClient(ctx, store1, k8sClient, namespace1)
	tassert.Nil(t, err)
	secretsClient2, err := provider.NewClient(ctx, store2, k8sClient, namespace2)
	tassert.Nil(t, err)

	data, err := secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)

	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
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

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer1 := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	secretID1, _ := fakeLockboxServer1.CreateSecret(authorizedKey1,
		"folderId", "secretName",
		textEntry(k1, v1),
	)
	fakeLockboxServer2 := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k2, v2 := "k2", "v2"
	secretID2, _ := fakeLockboxServer2.CreateSecret(authorizedKey2,
		"folderId", "secretName",
		textEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName1 = "authorizedKeySecretName1"
	const authorizedKeySecretKey1 = "authorizedKeySecretKey1"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName1, authorizedKeySecretKey1, toJSON(t, authorizedKey1))
	tassert.Nil(t, err)
	const authorizedKeySecretName2 = "authorizedKeySecretName2"
	const authorizedKeySecretKey2 = "authorizedKeySecretKey2"
	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName2, authorizedKeySecretKey2, toJSON(t, authorizedKey2))
	tassert.Nil(t, err)

	store1 := newYandexLockboxSecretStore(apiEndpoint1, namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexLockboxSecretStore(apiEndpoint2, namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider1 := newLockboxProvider(fakeClock, fakeLockboxServer1)
	provider2 := newLockboxProvider(fakeClock, fakeLockboxServer2)

	secretsClient1, err := provider1.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	secretsClient2, err := provider2.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)

	var data []byte

	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID1, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)
	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID2, Property: k2})
	tassert.Equal(t, v2, string(data))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenExpiration(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	tokenExpirationTime := time.Hour
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, tokenExpirationTime)
	k1, v1 := "k1", "v1"
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(k1, v1),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)

	var data []byte

	oldSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = oldSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)

	fakeClock.AddDuration(2 * tokenExpirationTime)

	data, err = oldSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, "unable to request secret payload to get secret: iam token expired")

	newSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = newSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Equal(t, v1, string(data))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenCleanup(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	tokenExpirationDuration := time.Hour
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, tokenExpirationDuration)
	secretID1, _ := fakeLockboxServer.CreateSecret(authorizedKey1,
		"folderId", "secretName1",
		textEntry("k1", "v1"),
	)
	secretID2, _ := fakeLockboxServer.CreateSecret(authorizedKey2,
		"folderId", "secretName2",
		textEntry("k2", "v2"),
	)

	var err error

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName1 = "authorizedKeySecretName1"
	const authorizedKeySecretKey1 = "authorizedKeySecretKey1"
	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName1, authorizedKeySecretKey1, toJSON(t, authorizedKey1))
	tassert.Nil(t, err)
	const authorizedKeySecretName2 = "authorizedKeySecretName2"
	const authorizedKeySecretKey2 = "authorizedKeySecretKey2"
	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName2, authorizedKeySecretKey2, toJSON(t, authorizedKey2))
	tassert.Nil(t, err)

	store1 := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)

	tassert.False(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.False(t, provider.IsIamTokenCached(authorizedKey2))

	// Access secretID1 with authorizedKey1, IAM token for authorizedKey1 should be cached
	secretsClient, err := provider.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID1})
	tassert.Nil(t, err)

	tassert.True(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.False(t, provider.IsIamTokenCached(authorizedKey2))

	fakeClock.AddDuration(tokenExpirationDuration * 2)

	// Access secretID2 with authorizedKey2, IAM token for authorizedKey2 should be cached
	secretsClient, err = provider.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID2})
	tassert.Nil(t, err)

	tassert.True(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.True(t, provider.IsIamTokenCached(authorizedKey2))

	fakeClock.AddDuration(tokenExpirationDuration)

	tassert.True(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.True(t, provider.IsIamTokenCached(authorizedKey2))

	provider.CleanUpIamTokenMap()

	tassert.False(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.True(t, provider.IsIamTokenCached(authorizedKey2))

	fakeClock.AddDuration(tokenExpirationDuration)

	tassert.False(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.True(t, provider.IsIamTokenCached(authorizedKey2))

	provider.CleanUpIamTokenMap()

	tassert.False(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.False(t, provider.IsIamTokenCached(authorizedKey2))
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID})
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

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secretName",
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	newKey, newVal := "newKey", "newVal"
	newVersionID := fakeLockboxServer.AddVersion(secretID, textEntry(newKey, newVal))

	data, err = secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	data, err = secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{newKey: []byte(newVal)}, data)
}

func TestGetSecretWithByNameFetchingPolicyForAllEntries(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	const secretName = "secretName"
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	_, _ = fakeLockboxServer.CreateSecret(authorizedKey, folderID, secretName, textEntry(k1, v1), binaryEntry(k2, v2))

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName})
	tassert.Nil(t, err)
	expected := map[string]string{
		k1: base64([]byte(v1)),
		k2: base64(v2),
	}
	tassert.Equal(t, expected, unmarshalStringMap(t, data))
}

func TestGetSecretWithByNameFetchingPolicyAndVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	const secretName = "secretName"
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := fakeLockboxServer.CreateSecret(authorizedKey, folderID, secretName, textEntry(oldKey, oldVal))

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string]string{oldKey: base64([]byte(oldVal))}, unmarshalStringMap(t, data))

	newKey, newVal := "newKey", "newVal"
	newVersionID := fakeLockboxServer.AddVersion(secretID, textEntry(newKey, newVal))

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{oldKey: base64([]byte(oldVal))}, unmarshalStringMap(t, data))

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{newKey: base64([]byte(newVal))}, unmarshalStringMap(t, data))
}

func TestGetSecretWithByNameFetchingPolicyForTextEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	const secretName = "secretName"
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	_, _ = fakeLockboxServer.CreateSecret(authorizedKey, folderID, secretName, textEntry(k1, v1), binaryEntry(k2, v2))

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Property: k1})
	tassert.Nil(t, err)
	tassert.Equal(t, v1, string(data))
}

func TestGetSecretWithByNameFetchingPolicyForBinaryEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	const secretName = "secretName"
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	_, _ = fakeLockboxServer.CreateSecret(authorizedKey, folderID, secretName, textEntry(k1, v1), binaryEntry(k2, v2))

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Property: k2})
	tassert.Nil(t, err)
	tassert.Equal(t, v2, data)
}

func TestGetSecretWithByNameFetchingPolicyNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "no-secret-with-such-name"})
	tassert.EqualError(t, err, errSecretPayloadNotFound)
	secretName := "secretName"
	_, _ = fakeLockboxServer.CreateSecret(authorizedKey, folderID, secretName, textEntry("k1", "v1"))
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName, Version: "no-version-with-such-id"})
	tassert.EqualError(t, err, errSecretPayloadVersionNotFound)
}

func TestGetSecretWithByNameFetchingPolicyUnauthorized(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKeyA := newFakeAuthorizedKey()
	authorizedKeyB := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	secretName := "secretName"
	_, _ = fakeLockboxServer.CreateSecret(authorizedKeyA, folderID, secretName, textEntry("k1", "v1"))

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKeyB))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretName})
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
}

func TestGetSecretWithByNameFetchingPolicyWithoutFolderID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, "")

	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	_, err = provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.EqualError(t, err, "folderID is required when fetching policy is 'byName'")
}

func TesGetSecretWithByIDFetchingPolicyForAllEntries(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secret",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByID("", namespace, authorizedKeySecretName, authorizedKeySecretKey)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.Nil(t, err)
	expected := map[string]string{
		k1: v1,
		k2: base64(v2),
	}
	tassert.Equal(t, expected, unmarshalStringMap(t, data))
}

func TestGetSecretWithByIDFetchingPolicyForTextEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secret",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByID("", namespace, authorizedKeySecretName, authorizedKeySecretKey)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, err)
	tassert.Equal(t, v1, string(data))
}

func TestGetSecretWithByIDFetchingPolicyForBinaryEntry(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k1, v1 := "k1", "v1"
	k2, v2 := "k2", []byte("v2")
	secretID, _ := fakeLockboxServer.CreateSecret(authorizedKey,
		"folderId", "secret",
		textEntry(k1, v1),
		binaryEntry(k2, v2),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStoreWithFetchByID("", namespace, authorizedKeySecretName, authorizedKeySecretKey)
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: secretID, Property: k2})
	tassert.Nil(t, err)
	tassert.Equal(t, v2, data)
}

func TestGetSecretWithInvalidFetchingPolicy(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeLockboxServer := client.NewFakeLockboxServer(fakeClock, time.Hour)
	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(
		ctx, t, k8sClient, namespace,
		authorizedKeySecretName, authorizedKeySecretKey,
		toJSON(t, authorizedKey),
	)
	tassert.Nil(t, err)
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexLockbox: &esv1.YandexLockboxProvider{
					Auth: esv1.YandexAuth{
						AuthorizedKey: esmeta.SecretKeySelector{
							Name: authorizedKeySecretName,
							Key:  authorizedKeySecretKey,
						},
					},
					FetchingPolicy: &esv1.FetchingPolicy{
						ByID:   nil,
						ByName: nil,
					},
				},
			},
		},
	}
	provider := newLockboxProvider(fakeClock, fakeLockboxServer)
	_, err = provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.EqualError(
		t,
		err,
		"invalid Yandex Lockbox SecretStore: requires either 'byName' or 'byID' policy",
	)
}

// helper fuxnctions

func newLockboxProvider(clock clock.Clock, fakeLockboxServer *client.FakeLockboxServer) *ydxcommon.YandexCloudProvider {
	return ydxcommon.InitYandexCloudProvider(
		ctrl.Log.WithName("provider").WithName("yandex").WithName("lockbox"),
		clock,
		adaptInput,
		func(context.Context, string, *iamkey.Key, []byte) (ydxcommon.SecretGetter, error) {
			return newLockboxSecretGetter(client.NewFakeLockboxClient(fakeLockboxServer))
		},
		func(_ context.Context, _ string, authorizedKey *iamkey.Key, _ []byte) (*ydxcommon.IamToken, error) {
			return fakeLockboxServer.NewIamToken(authorizedKey), nil
		},
		0,
	)
}

func newYandexLockboxSecretStore(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexLockbox: &esv1.YandexLockboxProvider{
					APIEndpoint: apiEndpoint,
					Auth: esv1.YandexAuth{
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

func newYandexLockboxSecretStoreWithFetchByName(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexLockbox: &esv1.YandexLockboxProvider{
					APIEndpoint: apiEndpoint,
					Auth: esv1.YandexAuth{
						AuthorizedKey: esmeta.SecretKeySelector{
							Name: authorizedKeySecretName,
							Key:  authorizedKeySecretKey,
						},
					},
					FetchingPolicy: &esv1.FetchingPolicy{
						ByName: &esv1.ByName{
							FolderID: folderID,
						},
					},
				},
			},
		},
	}
}

func newYandexLockboxSecretStoreWithFetchByID(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexLockbox: &esv1.YandexLockboxProvider{
					APIEndpoint: apiEndpoint,
					Auth: esv1.YandexAuth{
						AuthorizedKey: esmeta.SecretKeySelector{
							Name: authorizedKeySecretName,
							Key:  authorizedKeySecretKey,
						},
					},
					FetchingPolicy: &esv1.FetchingPolicy{
						ByID: &esv1.ByID{},
					},
				},
			},
		},
	}
}

func toJSON(t *testing.T, v any) []byte {
	jsonBytes, err := json.Marshal(v)
	tassert.Nil(t, err)
	return jsonBytes
}

func createK8sSecret(ctx context.Context, t *testing.T, k8sClient k8sclient.Client, namespace, secretName, secretKey string, secretValue []byte) error {
	err := k8sClient.Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Data: map[string][]byte{secretKey: secretValue},
	})
	tassert.Nil(t, err)
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
