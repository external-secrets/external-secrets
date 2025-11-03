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

package certificatemanager

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	tassert "github.com/stretchr/testify/assert"
	"github.com/yandex-cloud/go-genproto/yandex/cloud/certificatemanager/v1"
	"github.com/yandex-cloud/go-sdk/iamkey"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/certificatemanager/client"
	ydxcommon "github.com/external-secrets/external-secrets/providers/v1/yandex/common"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/common/clock"
)

const (
	errMissingKey                    = "invalid Yandex Certificate Manager SecretStore resource: missing AuthorizedKey Name"
	errSecretPayloadPermissionDenied = "unable to request certificate content to get secret: permission denied"
	errSecretPayloadNotFound         = "unable to request certificate content to get secret: certificate not found"
	errSecretPayloadVersionNotFound  = "unable to request certificate content to get secret: version not found"
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
				YandexCertificateManager: &esv1.YandexCertificateManagerProvider{
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
	store.Spec.Provider.YandexCertificateManager.CAProvider = &esv1.YandexCAProvider{
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

func TestGetSecretWithoutProperty(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{certificate1, certificate2, privateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)
}

func TestGetSecretWithProperty(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	chainData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: chainProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(certificate1+"\n"+certificate2),
		strings.TrimSpace(string(chainData)),
	)

	privateKeyData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: privateKeyProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(privateKey),
		strings.TrimSpace(string(privateKeyData)),
	)

	chainAndPrivateKeyData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: chainAndPrivateKeyProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{certificate1, certificate2, privateKey}, "\n")),
		strings.TrimSpace(string(chainAndPrivateKeyData)),
	)
}

func TestGetSecretByVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	oldCertificate1 := uuid.NewString()
	oldCertificate2 := uuid.NewString()
	oldPrivateKey := uuid.NewString()
	certificateID, oldVersionID := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{oldCertificate1, oldCertificate2},
			PrivateKey:       oldPrivateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{oldCertificate1, oldCertificate2, oldPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)

	newCertificate1 := uuid.NewString()
	newCertificate2 := uuid.NewString()
	newPrivateKey := uuid.NewString()
	newVersionID := fakeCertificateManagerServer.AddVersion(certificateID,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{newCertificate1, newCertificate2},
			PrivateKey:       newPrivateKey,
		})

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{oldCertificate1, oldCertificate2, oldPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{newCertificate1, newCertificate2, newPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)
}

func TestGetSecretUnauthorized(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKeyA := newFakeAuthorizedKey()
	authorizedKeyB := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKeyA,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKeyB))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID})
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
}

func TestGetSecretNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "no-secret-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: "no-version-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadVersionNotFound)
}

func TestGetSecretWithTwoNamespaces(t *testing.T) {
	ctx := context.Background()
	namespace1 := uuid.NewString()
	namespace2 := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	privateKey1 := uuid.NewString()
	certificateID1, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey1,
		"folderId", "certificateName1",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1},
			PrivateKey:       privateKey1,
		})
	certificate2 := uuid.NewString()
	privateKey2 := uuid.NewString()
	certificateID2, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey2,
		"folderId", "certificateName2",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate2},
			PrivateKey:       privateKey2,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace1, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey1))
	tassert.Nil(t, err)
	err = createK8sSecret(ctx, t, k8sClient, namespace2, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey2))
	tassert.Nil(t, err)
	store1 := newYandexCertificateManagerSecretStore("", namespace1, authorizedKeySecretName, authorizedKeySecretKey)
	store2 := newYandexCertificateManagerSecretStore("", namespace2, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient1, err := provider.NewClient(ctx, store1, k8sClient, namespace1)
	tassert.Nil(t, err)
	secretsClient2, err := provider.NewClient(ctx, store2, k8sClient, namespace2)
	tassert.Nil(t, err)

	data, err := secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID1, Property: privateKeyProperty})
	tassert.Equal(t, privateKey1, strings.TrimSpace(string(data)))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID2, Property: privateKeyProperty})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)

	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID1, Property: privateKeyProperty})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID2, Property: privateKeyProperty})
	tassert.Equal(t, privateKey2, strings.TrimSpace(string(data)))
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
	fakeCertificateManagerServer1 := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	privateKey1 := uuid.NewString()
	certificateID1, _ := fakeCertificateManagerServer1.CreateCertificate(authorizedKey1,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1},
			PrivateKey:       privateKey1,
		})
	fakeCertificateManagerServer2 := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate2 := uuid.NewString()
	privateKey2 := uuid.NewString()
	certificateID2, _ := fakeCertificateManagerServer2.CreateCertificate(authorizedKey2,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate2},
			PrivateKey:       privateKey2,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName1 = "authorizedKeySecretName1"
	const authorizedKeySecretKey1 = "authorizedKeySecretKey1"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName1, authorizedKeySecretKey1, toJSON(t, authorizedKey1))
	tassert.Nil(t, err)
	const authorizedKeySecretName2 = "authorizedKeySecretName2"
	const authorizedKeySecretKey2 = "authorizedKeySecretKey2"
	err = createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName2, authorizedKeySecretKey2, toJSON(t, authorizedKey2))
	tassert.Nil(t, err)

	store1 := newYandexCertificateManagerSecretStore(apiEndpoint1, namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexCertificateManagerSecretStore(apiEndpoint2, namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider1 := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer1)
	provider2 := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer2)

	secretsClient1, err := provider1.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	secretsClient2, err := provider2.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)

	var data []byte

	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID1, Property: chainProperty})
	tassert.Equal(t, certificate1, strings.TrimSpace(string(data)))
	tassert.Nil(t, err)
	data, err = secretsClient1.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID2, Property: chainProperty})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID1, Property: chainProperty})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, errSecretPayloadNotFound)
	data, err = secretsClient2.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID2, Property: chainProperty})
	tassert.Equal(t, certificate2, strings.TrimSpace(string(data)))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenExpiration(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	tokenExpirationTime := time.Hour
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, tokenExpirationTime)
	certificate := uuid.NewString()
	privateKey := uuid.NewString()
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)

	var data []byte

	oldSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = oldSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: privateKeyProperty})
	tassert.Equal(t, privateKey, strings.TrimSpace(string(data)))
	tassert.Nil(t, err)

	fakeClock.AddDuration(2 * tokenExpirationTime)

	data, err = oldSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: privateKeyProperty})
	tassert.Nil(t, data)
	tassert.EqualError(t, err, "unable to request certificate content to get secret: iam token expired")

	newSecretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err = newSecretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Property: privateKeyProperty})
	tassert.Equal(t, privateKey, strings.TrimSpace(string(data)))
	tassert.Nil(t, err)
}

func TestGetSecretWithIamTokenCleanup(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey1 := newFakeAuthorizedKey()
	authorizedKey2 := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	tokenExpirationDuration := time.Hour
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, tokenExpirationDuration)
	certificateID1, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey1,
		"folderId", "certificateName1",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})
	certificateID2, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey2,
		"folderId", "certificateName2",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})

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

	store1 := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName1, authorizedKeySecretKey1)
	store2 := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName2, authorizedKeySecretKey2)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)

	tassert.False(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.False(t, provider.IsIamTokenCached(authorizedKey2))

	// Access secretID1 with authorizedKey1, IAM token for authorizedKey1 should be cached
	secretsClient, err := provider.NewClient(ctx, store1, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID1})
	tassert.Nil(t, err)

	tassert.True(t, provider.IsIamTokenCached(authorizedKey1))
	tassert.False(t, provider.IsIamTokenCached(authorizedKey2))

	fakeClock.AddDuration(tokenExpirationDuration * 2)

	// Access secretID2 with authorizedKey2, IAM token for authorizedKey2 should be cached
	secretsClient, err = provider.NewClient(ctx, store2, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID2})
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
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		map[string][]byte{
			chainProperty:      []byte(certificate1 + "\n" + certificate2),
			privateKeyProperty: []byte(privateKey),
		},
		data,
	)
}

func TestGetSecretMapByVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	oldCertificate := uuid.NewString()
	oldPrivateKey := uuid.NewString()
	certificateID, oldVersionID := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{oldCertificate},
			PrivateKey:       oldPrivateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStore("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		map[string][]byte{
			chainProperty:      []byte(oldCertificate),
			privateKeyProperty: []byte(oldPrivateKey),
		},
		data,
	)

	newCertificate := uuid.NewString()
	newPrivateKey := uuid.NewString()
	newVersionID := fakeCertificateManagerServer.AddVersion(certificateID,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{newCertificate},
			PrivateKey:       newPrivateKey,
		})

	data, err = secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		map[string][]byte{
			chainProperty:      []byte(oldCertificate),
			privateKeyProperty: []byte(oldPrivateKey),
		},
		data,
	)

	data, err = secretsClient.GetSecretMap(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		map[string][]byte{
			chainProperty:      []byte(newCertificate),
			privateKeyProperty: []byte(newPrivateKey),
		},
		data,
	)
}

func TestGetSecretWithByNameFetchingPolicyWithoutProperty(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	folderID := uuid.NewString()
	const certificateName = "certificateName"
	_, _ = fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		folderID, certificateName,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{certificate1, certificate2, privateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)
}

func TestGetSecretWithByNameFetchingPolicyWithProperty(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	folderID := uuid.NewString()
	const certificateName = "certificateName"
	_, _ = fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		folderID, certificateName,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)

	chainData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Property: chainProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(certificate1+"\n"+certificate2),
		strings.TrimSpace(string(chainData)),
	)

	privateKeyData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Property: privateKeyProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(privateKey),
		strings.TrimSpace(string(privateKeyData)),
	)

	chainAndPrivateKeyData, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Property: chainAndPrivateKeyProperty})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{certificate1, certificate2, privateKey}, "\n")),
		strings.TrimSpace(string(chainAndPrivateKeyData)),
	)
}

func TestGetSecretWithByNameFetchingPolicyAndVersionID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	oldCertificate1 := uuid.NewString()
	oldCertificate2 := uuid.NewString()
	oldPrivateKey := uuid.NewString()
	folderID := uuid.NewString()
	const certificateName = "certificateName"
	certificateID, oldVersionID := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		folderID, certificateName,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{oldCertificate1, oldCertificate2},
			PrivateKey:       oldPrivateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Version: oldVersionID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{oldCertificate1, oldCertificate2, oldPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)

	newCertificate1 := uuid.NewString()
	newCertificate2 := uuid.NewString()
	newPrivateKey := uuid.NewString()
	newVersionID := fakeCertificateManagerServer.AddVersion(certificateID,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{newCertificate1, newCertificate2},
			PrivateKey:       newPrivateKey,
		})

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Version: oldVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{oldCertificate1, oldCertificate2, oldPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)

	data, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Version: newVersionID})
	tassert.Nil(t, err)
	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{newCertificate1, newCertificate2, newPrivateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)
}

func TestGetSecretWithByNameFetchingPolicyUnauthorized(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKeyA := newFakeAuthorizedKey()
	authorizedKeyB := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	certificateName := "certificateName"
	_, _ = fakeCertificateManagerServer.CreateCertificate(authorizedKeyA,
		folderID, certificateName,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKeyB))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName})
	tassert.EqualError(t, err, errSecretPayloadPermissionDenied)
}

func TestGetSecretWithByNameFetchingPolicyNotFound(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	folderID := uuid.NewString()
	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: "no-secret-with-this-name"})
	tassert.EqualError(t, err, errSecretPayloadNotFound)

	certificateName := "certificateName"
	_, _ = fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		folderID, certificateName,
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{uuid.NewString()},
			PrivateKey:       uuid.NewString(),
		})
	_, err = secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateName, Version: "no-version-with-this-id"})
	tassert.EqualError(t, err, errSecretPayloadVersionNotFound)
}

func TestGetSecretWithByNameFetchingPolicyWithoutFolderID(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByName("", namespace, authorizedKeySecretName, authorizedKeySecretKey, "")
	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	_, err = provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.EqualError(t, err, "folderID is required when fetching policy is 'byName'")
}

func TestGetSecretWithByIDFetchingPolicyWithoutProperty(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)
	certificate1 := uuid.NewString()
	certificate2 := uuid.NewString()
	privateKey := uuid.NewString()
	certificateID, _ := fakeCertificateManagerServer.CreateCertificate(authorizedKey,
		"folderId", "certificateName",
		&certificatemanager.GetCertificateContentResponse{
			CertificateChain: []string{certificate1, certificate2},
			PrivateKey:       privateKey,
		})

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := newYandexCertificateManagerSecretStoreWithFetchByID("", namespace, authorizedKeySecretName, authorizedKeySecretKey)

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	secretsClient, err := provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.Nil(t, err)
	data, err := secretsClient.GetSecret(ctx, esv1.ExternalSecretDataRemoteRef{Key: certificateID})
	tassert.Nil(t, err)

	tassert.Equal(
		t,
		strings.TrimSpace(strings.Join([]string{certificate1, certificate2, privateKey}, "\n")),
		strings.TrimSpace(string(data)),
	)
}
func TestGetSecretWithInvalidFetchingPolicy(t *testing.T) {
	ctx := context.Background()
	namespace := uuid.NewString()
	authorizedKey := newFakeAuthorizedKey()

	fakeClock := clock.NewFakeClock()
	fakeCertificateManagerServer := client.NewFakeCertificateManagerServer(fakeClock, time.Hour)

	k8sClient := clientfake.NewClientBuilder().Build()
	const authorizedKeySecretName = "authorizedKeySecretName"
	const authorizedKeySecretKey = "authorizedKeySecretKey"
	err := createK8sSecret(ctx, t, k8sClient, namespace, authorizedKeySecretName, authorizedKeySecretKey, toJSON(t, authorizedKey))
	tassert.Nil(t, err)
	store := &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexCertificateManager: &esv1.YandexCertificateManagerProvider{
					APIEndpoint: "",
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

	provider := newCertificateManagerProvider(fakeClock, fakeCertificateManagerServer)
	_, err = provider.NewClient(ctx, store, k8sClient, namespace)
	tassert.EqualError(t, err, "invalid Yandex Certificate Manager SecretStore: requires either 'byName' or 'byID' policy")
}

// helper functions

func newCertificateManagerProvider(clock clock.Clock, fakeCertificateManagerServer *client.FakeCertificateManagerServer) *ydxcommon.YandexCloudProvider {
	return ydxcommon.InitYandexCloudProvider(
		ctrl.Log.WithName("provider").WithName("yandex").WithName("certificatemanager"),
		clock,
		adaptInput,
		func(_ context.Context, _ string, _ *iamkey.Key, _ []byte) (ydxcommon.SecretGetter, error) {
			return newCertificateManagerSecretGetter(client.NewFakeCertificateManagerClient(fakeCertificateManagerServer))
		},
		func(_ context.Context, _ string, authorizedKey *iamkey.Key, _ []byte) (*ydxcommon.IamToken, error) {
			return fakeCertificateManagerServer.NewIamToken(authorizedKey), nil
		},
		0,
	)
}

func newYandexCertificateManagerSecretStore(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexCertificateManager: &esv1.YandexCertificateManagerProvider{
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

func newYandexCertificateManagerSecretStoreWithFetchByName(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey, folderID string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexCertificateManager: &esv1.YandexCertificateManagerProvider{
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

func newYandexCertificateManagerSecretStoreWithFetchByID(apiEndpoint, namespace, authorizedKeySecretName, authorizedKeySecretKey string) esv1.GenericStore {
	return &esv1.SecretStore{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Spec: esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				YandexCertificateManager: &esv1.YandexCertificateManagerProvider{
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
