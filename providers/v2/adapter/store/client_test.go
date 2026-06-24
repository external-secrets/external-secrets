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
	getSecretResponse    []byte
	getSecretErr         error
	getSecretRef         esv1.ExternalSecretDataRemoteRef
	getSecretProviderRef *pb.ProviderReference
	getSecretNamespace   string

	getSecretMapResponse map[string][]byte
	getSecretMapErr      error
	getSecretMapRef      esv1.ExternalSecretDataRemoteRef

	getAllSecretsResponse map[string][]byte
	getAllSecretsErr      error
	getAllSecretsFind     esv1.ExternalSecretFind

	pushSecretErr         error
	pushSecretData        map[string][]byte
	pushSecretSecret      *corev1.Secret
	pushSecretPayload     *pb.PushSecretData
	pushSecretProviderRef *pb.ProviderReference
	pushSecretNamespace   string

	deleteSecretErr         error
	deleteSecretRemoteRef   *pb.PushSecretRemoteRef
	deleteSecretProviderRef *pb.ProviderReference
	deleteSecretNamespace   string

	secretExistsResponse    bool
	secretExistsErr         error
	secretExistsRemoteRef   *pb.PushSecretRemoteRef
	secretExistsProviderRef *pb.ProviderReference
	secretExistsNamespace   string

	validateErr         error
	validateProviderRef *pb.ProviderReference
	validateNamespace   string
	validateCalled      bool

	capabilitiesResponse pb.SecretStoreCapabilities
	capabilitiesErr      error
	capabilitiesSet      bool

	closeErr    error
	closeCalled bool
}

func (f *fakeV2Provider) GetSecret(
	_ context.Context,
	ref esv1.ExternalSecretDataRemoteRef,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) ([]byte, error) {
	f.getSecretRef = ref
	f.getSecretProviderRef = providerRef
	f.getSecretNamespace = sourceNamespace
	return f.getSecretResponse, f.getSecretErr
}

func (f *fakeV2Provider) GetSecretMap(
	_ context.Context,
	ref esv1.ExternalSecretDataRemoteRef,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) (map[string][]byte, error) {
	f.getSecretMapRef = ref
	f.getSecretProviderRef = providerRef
	f.getSecretNamespace = sourceNamespace
	return f.getSecretMapResponse, f.getSecretMapErr
}

func (f *fakeV2Provider) GetAllSecrets(
	_ context.Context,
	find esv1.ExternalSecretFind,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) (map[string][]byte, error) {
	f.getAllSecretsFind = find
	f.getSecretProviderRef = providerRef
	f.getSecretNamespace = sourceNamespace
	return f.getAllSecretsResponse, f.getAllSecretsErr
}

func (f *fakeV2Provider) PushSecret(
	_ context.Context,
	secret *corev1.Secret,
	pushSecretData *pb.PushSecretData,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) error {
	f.pushSecretData = secret.Data
	f.pushSecretSecret = secret.DeepCopy()
	f.pushSecretPayload = pushSecretData
	f.pushSecretProviderRef = providerRef
	f.pushSecretNamespace = sourceNamespace
	return f.pushSecretErr
}

func (f *fakeV2Provider) DeleteSecret(
	_ context.Context,
	remoteRef *pb.PushSecretRemoteRef,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) error {
	f.deleteSecretRemoteRef = remoteRef
	f.deleteSecretProviderRef = providerRef
	f.deleteSecretNamespace = sourceNamespace
	return f.deleteSecretErr
}

func (f *fakeV2Provider) SecretExists(
	_ context.Context,
	remoteRef *pb.PushSecretRemoteRef,
	providerRef *pb.ProviderReference,
	sourceNamespace string,
) (bool, error) {
	f.secretExistsRemoteRef = remoteRef
	f.secretExistsProviderRef = providerRef
	f.secretExistsNamespace = sourceNamespace
	return f.secretExistsResponse, f.secretExistsErr
}

func (f *fakeV2Provider) Validate(_ context.Context, providerRef *pb.ProviderReference, sourceNamespace string) error {
	f.validateCalled = true
	f.validateProviderRef = providerRef
	f.validateNamespace = sourceNamespace
	return f.validateErr
}

func (f *fakeV2Provider) Capabilities(context.Context, *pb.ProviderReference, string) (pb.SecretStoreCapabilities, error) {
	if !f.capabilitiesSet && f.capabilitiesErr == nil {
		return pb.SecretStoreCapabilities_READ_WRITE, nil
	}
	return f.capabilitiesResponse, f.capabilitiesErr
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

func TestClientCapabilitiesMapsProviderCapabilities(t *testing.T) {
	tests := []struct {
		name     string
		caps     pb.SecretStoreCapabilities
		expected esv1.SecretStoreCapabilities
	}{
		{
			name:     "read only",
			caps:     pb.SecretStoreCapabilities_READ_ONLY,
			expected: esv1.SecretStoreReadOnly,
		},
		{
			name:     "write only",
			caps:     pb.SecretStoreCapabilities_WRITE_ONLY,
			expected: esv1.SecretStoreWriteOnly,
		},
		{
			name:     "read write",
			caps:     pb.SecretStoreCapabilities_READ_WRITE,
			expected: esv1.SecretStoreReadWrite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &fakeV2Provider{capabilitiesResponse: tt.caps, capabilitiesSet: true}
			client := NewClient(provider, &pb.ProviderReference{Name: "provider"}, testSourceNamespace)
			capabilityClient, ok := client.(CapabilityAwareClient)
			if !ok {
				t.Fatalf("expected client to implement CapabilityAwareClient")
			}

			got, err := capabilityClient.Capabilities(context.Background())
			if err != nil {
				t.Fatalf("Capabilities() error = %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %v, got %v", tt.expected, got)
			}
		})
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
