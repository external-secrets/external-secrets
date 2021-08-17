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
	b64 "encoding/base64"
	"encoding/json"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/lockbox/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client/fake"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"

	store := &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				YandexLockbox: &esv1alpha1.YandexLockboxProvider{},
			},
		},
	}
	provider, err := schema.GetProvider(store)
	tassert.Nil(t, err)

	k8sClient := clientfake.NewClientBuilder().Build()
	secretClient, err := provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	tassert.Nil(t, secretClient)

	store.Spec.Provider.YandexLockbox.Auth = esv1alpha1.YandexLockboxAuth{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	tassert.Nil(t, secretClient)

	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey = esmeta.SecretKeySelector{}
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	tassert.Nil(t, secretClient)

	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey.Name = authorizedKeySecretName
	store.Spec.Provider.YandexLockbox.Auth.AuthorizedKey.Key = authorizedKeySecretKey
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "could not fetch AuthorizedKey secret: secrets \"authorizedKeySecretName\" not found")
	tassert.Nil(t, secretClient)

	err = createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, newFakeAuthorizedKey("0"))
	tassert.Nil(t, err)
	secretClient, err = provider.NewClient(context.Background(), store, k8sClient, namespace)
	tassert.EqualError(t, err, "failed to create Yandex.Cloud SDK: private key parsing failed: Invalid Key: Key must be PEM encoded PKCS1 or PKCS8 private key")
	tassert.Nil(t, secretClient)
}

func TestGetSecretForAllEntries(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
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
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID})
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
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
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
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Property: k1})
	tassert.Nil(t, err)

	tassert.Equal(t, v1, string(data))
}

func TestGetSecretForBinaryEntry(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
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
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Property: k2})
	tassert.Nil(t, err)

	tassert.Equal(t, v2, data)
}

func TestGetSecretByVersionID(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	newKey, newVal := "newKey", "newVal"
	newVersionID := lockboxBackend.AddVersion(secretID,
		textEntry(newKey, newVal),
	)

	data, err = secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{oldKey: oldVal}, unmarshalStringMap(t, data))

	data, err = secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string]string{newKey: newVal}, unmarshalStringMap(t, data))
}

func TestGetSecretUnauthorized(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKeyA := newFakeAuthorizedKey("A")
	authorizedKeyB := newFakeAuthorizedKey("B")

	lockboxBackend := fake.NewLockboxBackend()
	secretID, _ := lockboxBackend.CreateSecret(authorizedKeyA,
		textEntry("k1", "v1"),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKeyB)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID})
	tassert.EqualError(t, err, "unable to request secret payload to get secret: permission denied")
}

func TestGetSecretNotFound(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: "no-secret-with-this-id"})
	tassert.EqualError(t, err, "unable to request secret payload to get secret: secret not found")

	secretID, _ := lockboxBackend.CreateSecret(authorizedKey,
		textEntry("k1", "v1"),
	)
	_, err = secretsClient.GetSecret(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: "no-version-with-this-id"})
	tassert.EqualError(t, err, "unable to request secret payload to get secret: version not found")
}

func TestGetSecretMap(t *testing.T) {
	ctx := context.Background()
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
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
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID})
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
	const namespace = "namespace"
	authorizedKey := newFakeAuthorizedKey("0")

	lockboxBackend := fake.NewLockboxBackend()
	oldKey, oldVal := "oldKey", "oldVal"
	secretID, oldVersionID := lockboxBackend.CreateSecret(authorizedKey,
		textEntry(oldKey, oldVal),
	)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, authorizedKey)
	tassert.Nil(t, err)
	store := newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := &lockboxProvider{&fake.LockboxClientCreator{
		Backend: lockboxBackend,
	}}
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	newKey, newVal := "newKey", "newVal"
	newVersionID := lockboxBackend.AddVersion(secretID,
		textEntry(newKey, newVal),
	)

	data, err = secretsClient.GetSecretMap(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{oldKey: []byte(oldVal)}, data)

	data, err = secretsClient.GetSecretMap(ctx, esv1alpha1.ExternalSecretDataRemoteRef{Key: secretID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(t, map[string][]byte{newKey: []byte(newVal)}, data)
}

// helper functions

func newYandexLockboxSecretStore(namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1alpha1.GenericStore {
	return &esv1alpha1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1alpha1.SecretStoreSpec{
			Provider: &esv1alpha1.SecretStoreProvider{
				YandexLockbox: &esv1alpha1.YandexLockboxProvider{
					Auth: esv1alpha1.YandexLockboxAuth{
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

func newFakeAuthorizedKey(uniqueLabel string) *iamkey.Key {
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
