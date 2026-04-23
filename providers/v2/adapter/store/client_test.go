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

package store

import (
	"bytes"
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

const (
	testProperty        = "property"
	testSecretValue     = "secret-value"
	testSourceNamespace = "tenant-a"
	testBarValue        = "bar"
	testValue           = "value"
)

type fakeV2Provider struct {
	getSecretResponse           []byte
	getSecretErr                error
	getSecretRef                esv1.ExternalSecretDataRemoteRef
	getSecretProviderRef        *pb.ProviderReference
	getSecretCompatibilityStore *pb.CompatibilityStore
	getSecretNamespace          string

	getSecretMapResponse           map[string][]byte
	getSecretMapErr                error
	getSecretMapRef                esv1.ExternalSecretDataRemoteRef
	getSecretMapCompatibilityStore *pb.CompatibilityStore

	getAllSecretsResponse           map[string][]byte
	getAllSecretsErr                error
	getAllSecretsFind               esv1.ExternalSecretFind
	getAllSecretsCompatibilityStore *pb.CompatibilityStore

	pushSecretErr                error
	pushSecretData               map[string][]byte
	pushSecretSecret             *corev1.Secret
	pushSecretPayload            *pb.PushSecretData
	pushSecretProviderRef        *pb.ProviderReference
	pushSecretCompatibilityStore *pb.CompatibilityStore
	pushSecretNamespace          string

	deleteSecretErr                error
	deleteSecretRemoteRef          *pb.PushSecretRemoteRef
	deleteSecretProviderRef        *pb.ProviderReference
	deleteSecretCompatibilityStore *pb.CompatibilityStore
	deleteSecretNamespace          string

	secretExistsResponse           bool
	secretExistsErr                error
	secretExistsRemoteRef          *pb.PushSecretRemoteRef
	secretExistsProviderRef        *pb.ProviderReference
	secretExistsCompatibilityStore *pb.CompatibilityStore
	secretExistsNamespace          string

	validateErr                error
	validateProviderRef        *pb.ProviderReference
	validateCompatibilityStore *pb.CompatibilityStore
	validateNamespace          string
	validateCalled             bool

	closeErr    error
	closeCalled bool
}

func (f *fakeV2Provider) GetSecret(
	_ context.Context,
	ref esv1.ExternalSecretDataRemoteRef,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) ([]byte, error) {
	f.getSecretRef = ref
	f.getSecretProviderRef = providerRef
	f.getSecretCompatibilityStore = compatibilityStore
	f.getSecretNamespace = sourceNamespace
	return f.getSecretResponse, f.getSecretErr
}

func (f *fakeV2Provider) GetSecretMap(
	_ context.Context,
	ref esv1.ExternalSecretDataRemoteRef,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) (map[string][]byte, error) {
	f.getSecretMapRef = ref
	f.getSecretProviderRef = providerRef
	f.getSecretMapCompatibilityStore = compatibilityStore
	f.getSecretNamespace = sourceNamespace
	return f.getSecretMapResponse, f.getSecretMapErr
}

func (f *fakeV2Provider) GetAllSecrets(
	_ context.Context,
	find esv1.ExternalSecretFind,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) (map[string][]byte, error) {
	f.getAllSecretsFind = find
	f.getSecretProviderRef = providerRef
	f.getAllSecretsCompatibilityStore = compatibilityStore
	f.getSecretNamespace = sourceNamespace
	return f.getAllSecretsResponse, f.getAllSecretsErr
}

func (f *fakeV2Provider) PushSecret(
	_ context.Context,
	secret *corev1.Secret,
	pushSecretData *pb.PushSecretData,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) error {
	f.pushSecretData = secret.Data
	f.pushSecretSecret = secret.DeepCopy()
	f.pushSecretPayload = pushSecretData
	f.pushSecretProviderRef = providerRef
	f.pushSecretCompatibilityStore = compatibilityStore
	f.pushSecretNamespace = sourceNamespace
	return f.pushSecretErr
}

func (f *fakeV2Provider) DeleteSecret(
	_ context.Context,
	remoteRef *pb.PushSecretRemoteRef,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) error {
	f.deleteSecretRemoteRef = remoteRef
	f.deleteSecretProviderRef = providerRef
	f.deleteSecretCompatibilityStore = compatibilityStore
	f.deleteSecretNamespace = sourceNamespace
	return f.deleteSecretErr
}

func (f *fakeV2Provider) SecretExists(
	_ context.Context,
	remoteRef *pb.PushSecretRemoteRef,
	providerRef *pb.ProviderReference,
	compatibilityStore *pb.CompatibilityStore,
	sourceNamespace string,
) (bool, error) {
	f.secretExistsRemoteRef = remoteRef
	f.secretExistsProviderRef = providerRef
	f.secretExistsCompatibilityStore = compatibilityStore
	f.secretExistsNamespace = sourceNamespace
	return f.secretExistsResponse, f.secretExistsErr
}

func (f *fakeV2Provider) Validate(_ context.Context, providerRef *pb.ProviderReference, compatibilityStore *pb.CompatibilityStore, sourceNamespace string) error {
	f.validateCalled = true
	f.validateProviderRef = providerRef
	f.validateCompatibilityStore = compatibilityStore
	f.validateNamespace = sourceNamespace
	return f.validateErr
}

func (f *fakeV2Provider) Capabilities(context.Context, *pb.ProviderReference, string) (pb.SecretStoreCapabilities, error) {
	return pb.SecretStoreCapabilities_READ_WRITE, nil
}

func (f *fakeV2Provider) Close(context.Context) error {
	f.closeCalled = true
	return f.closeErr
}

type fakePushSecretData struct {
	property  string
	secretKey string
	remoteKey string
	metadata  *apiextensionsv1.JSON
}

func (f fakePushSecretData) GetProperty() string {
	return f.property
}

func (f fakePushSecretData) GetSecretKey() string {
	return f.secretKey
}

func (f fakePushSecretData) GetRemoteKey() string {
	return f.remoteKey
}

func (f fakePushSecretData) GetMetadata() *apiextensionsv1.JSON {
	return f.metadata
}

type fakePushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (f fakePushSecretRemoteRef) GetRemoteKey() string {
	return f.remoteKey
}

func (f fakePushSecretRemoteRef) GetProperty() string {
	return f.property
}

func TestClientGetSecretDelegatesProviderReferenceAndNamespace(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	provider := &fakeV2Provider{getSecretResponse: []byte(testSecretValue)}
	client := NewClient(provider, providerRef, testSourceNamespace)

	ref := esv1.ExternalSecretDataRemoteRef{Key: "sample", Version: "v1", Property: "password"}
	value, err := client.GetSecret(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if string(value) != testSecretValue {
		t.Fatalf("expected %s, got %q", testSecretValue, string(value))
	}
	if provider.getSecretRef != ref {
		t.Fatalf("unexpected ref: %#v", provider.getSecretRef)
	}
	if provider.getSecretProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.getSecretProviderRef)
	}
	if provider.getSecretNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.getSecretNamespace)
	}
}

func TestClientGetSecretMapDelegatesProviderReferenceAndNamespace(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	expected := map[string][]byte{
		"foo": []byte(testBarValue),
		"baz": []byte("qux"),
	}
	provider := &fakeV2Provider{getSecretMapResponse: expected}
	client := NewClient(provider, providerRef, testSourceNamespace)

	ref := esv1.ExternalSecretDataRemoteRef{Key: "sample"}
	secretMap, err := client.GetSecretMap(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetSecretMap() error = %v", err)
	}

	if string(secretMap["foo"]) != testBarValue || string(secretMap["baz"]) != "qux" {
		t.Fatalf("unexpected secret map: %#v", secretMap)
	}
	if provider.getSecretMapRef != ref {
		t.Fatalf("unexpected ref: %#v", provider.getSecretMapRef)
	}
	if provider.getSecretProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.getSecretProviderRef)
	}
	if provider.getSecretNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.getSecretNamespace)
	}
}

func TestClientGetAllSecretsDelegatesFindCriteria(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	path := "/team-a"
	expected := map[string][]byte{"db-password": []byte(testValue)}
	provider := &fakeV2Provider{getAllSecretsResponse: expected}
	client := NewClient(provider, providerRef, testSourceNamespace)

	find := esv1.ExternalSecretFind{
		Tags: map[string]string{
			"team": "a",
		},
		Path: &path,
		Name: &esv1.FindName{RegExp: "db-.*"},
	}

	secrets, err := client.GetAllSecrets(context.Background(), find)
	if err != nil {
		t.Fatalf("GetAllSecrets() error = %v", err)
	}

	if string(secrets["db-password"]) != testValue {
		t.Fatalf("unexpected secret value: %#v", secrets)
	}
	if provider.getAllSecretsFind.Tags["team"] != "a" {
		t.Fatalf("unexpected find tags: %#v", provider.getAllSecretsFind)
	}
	if provider.getAllSecretsFind.Path == nil || *provider.getAllSecretsFind.Path != path {
		t.Fatalf("unexpected find path: %#v", provider.getAllSecretsFind.Path)
	}
	if provider.getAllSecretsFind.Name == nil || provider.getAllSecretsFind.Name.RegExp != "db-.*" {
		t.Fatalf("unexpected find name: %#v", provider.getAllSecretsFind.Name)
	}
	if provider.getSecretProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.getSecretProviderRef)
	}
	if provider.getSecretNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.getSecretNamespace)
	}
}

func TestCompatibilityClientGetSecretDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{getSecretResponse: []byte(testSecretValue)}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	value, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"})
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if string(value) != testSecretValue {
		t.Fatalf("expected %s, got %q", testSecretValue, string(value))
	}
	if provider.getSecretProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.getSecretProviderRef)
	}
	if provider.getSecretCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.getSecretCompatibilityStore)
	}
}

func TestCompatibilityClientGetSecretMapDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{getSecretMapResponse: map[string][]byte{"foo": []byte(testBarValue)}}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	secretMap, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"})
	if err != nil {
		t.Fatalf("GetSecretMap() error = %v", err)
	}

	if string(secretMap["foo"]) != testBarValue {
		t.Fatalf("unexpected secret map: %#v", secretMap)
	}
	if provider.getSecretProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.getSecretProviderRef)
	}
	if provider.getSecretMapCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.getSecretMapCompatibilityStore)
	}
}

func TestCompatibilityClientGetAllSecretsDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{getAllSecretsResponse: map[string][]byte{"db-password": []byte(testValue)}}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	secrets, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{})
	if err != nil {
		t.Fatalf("GetAllSecrets() error = %v", err)
	}

	if string(secrets["db-password"]) != testValue {
		t.Fatalf("unexpected secret value: %#v", secrets)
	}
	if provider.getSecretProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.getSecretProviderRef)
	}
	if provider.getAllSecretsCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.getAllSecretsCompatibilityStore)
	}
}

func TestCompatibilityClientPushSecretDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	err := client.PushSecret(context.Background(), &corev1.Secret{
		Data: map[string][]byte{"token": []byte(testValue)},
	}, fakePushSecretData{remoteKey: serverTestRemoteKey, secretKey: "token"})
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	if provider.pushSecretProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.pushSecretProviderRef)
	}
	if provider.pushSecretCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.pushSecretCompatibilityStore)
	}
}

func TestCompatibilityClientDeleteSecretDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	err := client.DeleteSecret(context.Background(), fakePushSecretRemoteRef{
		remoteKey: serverTestRemoteKey,
		property:  testProperty,
	})
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	if provider.deleteSecretProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.deleteSecretProviderRef)
	}
	if provider.deleteSecretCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.deleteSecretCompatibilityStore)
	}
}

func TestCompatibilityClientSecretExistsDelegatesCompatibilityStore(t *testing.T) {
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"sample","value":"secret-value"}]}}}`),
	}
	provider := &fakeV2Provider{secretExistsResponse: true}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	exists, err := client.SecretExists(context.Background(), fakePushSecretRemoteRef{
		remoteKey: serverTestRemoteKey,
		property:  testProperty,
	})
	if err != nil {
		t.Fatalf("SecretExists() error = %v", err)
	}
	if !exists {
		t.Fatal("expected secret to exist")
	}
	if provider.secretExistsProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.secretExistsProviderRef)
	}
	if provider.secretExistsCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.secretExistsCompatibilityStore)
	}
}

func TestClientPushSecretConvertsPayloadAndMetadata(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	provider := &fakeV2Provider{}
	client := NewClient(provider, providerRef, testSourceNamespace)

	metadata := []byte(`{"owner":"eso"}`)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"token": []byte(testValue),
		},
	}
	pushData := fakePushSecretData{
		property:  testProperty,
		secretKey: "token",
		remoteKey: serverTestRemoteKey,
		metadata:  &apiextensionsv1.JSON{Raw: metadata},
	}

	err := client.PushSecret(context.Background(), secret, pushData)
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	if string(provider.pushSecretData["token"]) != testValue {
		t.Fatalf("unexpected secret data: %#v", provider.pushSecretData)
	}
	if provider.pushSecretPayload == nil {
		t.Fatal("expected push payload to be recorded")
	}
	if provider.pushSecretPayload.RemoteKey != serverTestRemoteKey || provider.pushSecretPayload.SecretKey != "token" || provider.pushSecretPayload.Property != testProperty {
		t.Fatalf("unexpected push payload: %#v", provider.pushSecretPayload)
	}
	if !bytes.Equal(provider.pushSecretPayload.Metadata, metadata) {
		t.Fatalf("unexpected metadata: %q", string(provider.pushSecretPayload.Metadata))
	}
	if provider.pushSecretProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.pushSecretProviderRef)
	}
	if provider.pushSecretNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.pushSecretNamespace)
	}
}

func TestClientPushSecretForwardsKubernetesSecretShape(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	provider := &fakeV2Provider{}
	client := NewClient(provider, providerRef, testSourceNamespace)

	metadata := []byte(`{"mergePolicy":"replace"}`)
	secret := &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"team": "platform"},
			Annotations: map[string]string{"owner": "app-team"},
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("payload"),
		},
	}
	pushData := fakePushSecretData{
		property:  testProperty,
		secretKey: ".dockerconfigjson",
		remoteKey: serverTestRemoteKey,
		metadata:  &apiextensionsv1.JSON{Raw: metadata},
	}

	err := client.PushSecret(context.Background(), secret, pushData)
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	if provider.pushSecretSecret == nil {
		t.Fatal("expected pushed secret to be recorded")
	}
	if provider.pushSecretSecret.Type != corev1.SecretTypeDockerConfigJson {
		t.Errorf("expected secret type %q, got %q", corev1.SecretTypeDockerConfigJson, provider.pushSecretSecret.Type)
	}
	if got, want := provider.pushSecretSecret.Labels["team"], "platform"; got != want {
		t.Errorf("expected secret label team=%q, got %q", want, got)
	}
	if got, want := provider.pushSecretSecret.Annotations["owner"], "app-team"; got != want {
		t.Errorf("expected secret annotation owner=%q, got %q", want, got)
	}
	if got, want := string(provider.pushSecretSecret.Data[".dockerconfigjson"]), "payload"; got != want {
		t.Errorf("expected secret payload %q, got %q", want, got)
	}
	if provider.pushSecretPayload == nil {
		t.Fatal("expected push payload to be recorded")
	}
	if !bytes.Equal(provider.pushSecretPayload.Metadata, metadata) {
		t.Fatalf("unexpected metadata: %q", string(provider.pushSecretPayload.Metadata))
	}
}

func TestClientDeleteSecretConvertsRemoteRef(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	provider := &fakeV2Provider{}
	client := NewClient(provider, providerRef, testSourceNamespace)

	err := client.DeleteSecret(context.Background(), fakePushSecretRemoteRef{
		remoteKey: serverTestRemoteKey,
		property:  testProperty,
	})
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	if provider.deleteSecretRemoteRef == nil {
		t.Fatal("expected delete remote ref to be recorded")
	}
	if provider.deleteSecretRemoteRef.RemoteKey != serverTestRemoteKey || provider.deleteSecretRemoteRef.Property != testProperty {
		t.Fatalf("unexpected remote ref: %#v", provider.deleteSecretRemoteRef)
	}
	if provider.deleteSecretProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.deleteSecretProviderRef)
	}
	if provider.deleteSecretNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.deleteSecretNamespace)
	}
}

func TestClientSecretExistsConvertsRemoteRef(t *testing.T) {
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	provider := &fakeV2Provider{secretExistsResponse: true}
	client := NewClient(provider, providerRef, testSourceNamespace)

	exists, err := client.SecretExists(context.Background(), fakePushSecretRemoteRef{
		remoteKey: serverTestRemoteKey,
		property:  testProperty,
	})
	if err != nil {
		t.Fatalf("SecretExists() error = %v", err)
	}

	if !exists {
		t.Fatal("expected secret to exist")
	}
	if provider.secretExistsRemoteRef == nil {
		t.Fatal("expected exists remote ref to be recorded")
	}
	if provider.secretExistsRemoteRef.RemoteKey != serverTestRemoteKey || provider.secretExistsRemoteRef.Property != testProperty {
		t.Fatalf("unexpected remote ref: %#v", provider.secretExistsRemoteRef)
	}
	if provider.secretExistsProviderRef != providerRef {
		t.Fatalf("unexpected provider ref: %#v", provider.secretExistsProviderRef)
	}
	if provider.secretExistsNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", provider.secretExistsNamespace)
	}
}

func TestClientValidateMapsProviderErrors(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		providerRef := &pb.ProviderReference{
			Name:         "provider",
			Namespace:    "config-ns",
			StoreRefKind: esv1.SecretStoreKind,
		}
		provider := &fakeV2Provider{}
		client := NewClient(provider, providerRef, testSourceNamespace)

		result, err := client.Validate()
		if err != nil {
			t.Fatalf("Validate() error = %v", err)
		}
		if result != esv1.ValidationResultReady {
			t.Fatalf("expected ValidationResultReady, got %q", result)
		}
		if provider.validateProviderRef != providerRef {
			t.Fatalf("unexpected provider ref: %#v", provider.validateProviderRef)
		}
		if provider.validateProviderRef.StoreRefKind != esv1.SecretStoreKind {
			t.Fatalf("unexpected store_ref_kind: %q", provider.validateProviderRef.StoreRefKind)
		}
		if provider.validateNamespace != testSourceNamespace {
			t.Fatalf("unexpected source namespace: %q", provider.validateNamespace)
		}
	})

	t.Run("error", func(t *testing.T) {
		validateErr := errors.New("invalid credentials")
		provider := &fakeV2Provider{validateErr: validateErr}
		client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, testSourceNamespace)

		result, err := client.Validate()
		if !errors.Is(err, validateErr) {
			t.Fatalf("expected %v, got %v", validateErr, err)
		}
		if result != esv1.ValidationResultError {
			t.Fatalf("expected ValidationResultError, got %q", result)
		}
	})
}

func TestCompatibilityClientValidateUsesCompatibilityStore(t *testing.T) {
	provider := &fakeV2Provider{}
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "runtime-store",
		StoreNamespace:  "tenant-a",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"db","value":"secret"}]}}}`),
	}
	client := NewCompatibilityClient(provider, compatibilityStore, testSourceNamespace)

	result, err := client.Validate()
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if result != esv1.ValidationResultReady {
		t.Fatalf("expected ValidationResultReady, got %q", result)
	}
	if !provider.validateCalled {
		t.Fatal("expected provider Validate to be called")
	}
	if provider.validateProviderRef != nil {
		t.Fatalf("expected provider ref to be nil, got %#v", provider.validateProviderRef)
	}
	if provider.validateCompatibilityStore != compatibilityStore {
		t.Fatalf("unexpected compatibility store: %#v", provider.validateCompatibilityStore)
	}
}

func TestCompatibilityClientValidateReturnsErrorWhenValidationHookFails(t *testing.T) {
	validateErr := errors.New("runtime not serving")
	client := NewCompatibilityClient(&fakeV2Provider{validateErr: validateErr}, &pb.CompatibilityStore{
		StoreName:       "runtime-store",
		StoreNamespace:  "tenant-a",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"db","value":"secret"}]}}}`),
	}, testSourceNamespace)

	result, err := client.Validate()
	if !errors.Is(err, validateErr) {
		t.Fatalf("expected %v, got %v", validateErr, err)
	}
	if result != esv1.ValidationResultError {
		t.Fatalf("expected ValidationResultError, got %q", result)
	}
}

func TestClientCloseDelegates(t *testing.T) {
	closeErr := errors.New("close failed")
	provider := &fakeV2Provider{closeErr: closeErr}
	client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, testSourceNamespace)

	err := client.Close(context.Background())
	if !errors.Is(err, closeErr) {
		t.Fatalf("expected %v, got %v", closeErr, err)
	}
	if !provider.closeCalled {
		t.Fatal("expected provider close to be called")
	}
}
