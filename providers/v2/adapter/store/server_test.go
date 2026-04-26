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
	"context"
	"errors"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

const (
	serverTestRemoteKey       = "remote/path"
	serverTestProperty        = "property"
	serverTestSourceNamespace = "tenant-a"
	serverTestValue           = "value"
)

type fakeProviderInterface struct {
	newClient func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error)
	caps      esv1.SecretStoreCapabilities
}

func (f *fakeProviderInterface) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return f.newClient(ctx, store, kube, namespace)
}

func (f *fakeProviderInterface) Capabilities() esv1.SecretStoreCapabilities {
	return f.caps
}

func (f *fakeProviderInterface) ValidateStore(esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

type fakeSecretsClient struct {
	getSecretResponse []byte
	getSecretErr      error
	getSecretRef      esv1.ExternalSecretDataRemoteRef

	getSecretMapResponse map[string][]byte
	getSecretMapErr      error
	getSecretMapRef      esv1.ExternalSecretDataRemoteRef

	getAllSecretsResponse map[string][]byte
	getAllSecretsErr      error
	getAllSecretsFind     esv1.ExternalSecretFind

	pushSecretErr    error
	pushSecretSecret *corev1.Secret
	pushSecretData   esv1.PushSecretData
	deleteSecretErr  error
	deleteSecretRef  esv1.PushSecretRemoteRef
	secretExistsResp bool
	secretExistsErr  error
	secretExistsRef  esv1.PushSecretRemoteRef
	validateResult   esv1.ValidationResult
	validateErr      error
	closeCalled      bool
}

func (f *fakeSecretsClient) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	f.getSecretRef = ref
	return f.getSecretResponse, f.getSecretErr
}

func (f *fakeSecretsClient) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	f.getSecretMapRef = ref
	return f.getSecretMapResponse, f.getSecretMapErr
}

func (f *fakeSecretsClient) GetAllSecrets(_ context.Context, find esv1.ExternalSecretFind) (map[string][]byte, error) {
	f.getAllSecretsFind = find
	return f.getAllSecretsResponse, f.getAllSecretsErr
}

func (f *fakeSecretsClient) PushSecret(_ context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	f.pushSecretSecret = secret
	f.pushSecretData = data
	return f.pushSecretErr
}

func (f *fakeSecretsClient) DeleteSecret(_ context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	f.deleteSecretRef = remoteRef
	return f.deleteSecretErr
}

func (f *fakeSecretsClient) SecretExists(_ context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	f.secretExistsRef = remoteRef
	return f.secretExistsResp, f.secretExistsErr
}

func (f *fakeSecretsClient) Validate() (esv1.ValidationResult, error) {
	return f.validateResult, f.validateErr
}

func (f *fakeSecretsClient) Close(context.Context) error {
	f.closeCalled = true
	return nil
}

type specMapperRecorder struct {
	ref             *pb.ProviderReference
	sourceNamespace string
	spec            *esv1.SecretStoreSpec
	err             error
}

func (r *specMapperRecorder) mapRef(ref *pb.ProviderReference, sourceNamespace string) (*esv1.SecretStoreSpec, error) {
	r.ref = ref
	r.sourceNamespace = sourceNamespace
	return r.spec, r.err
}

func TestServerGetSecretMapsRemoteRefAndSyntheticStoreNamespace(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{},
			},
		},
	}
	fakeClient := &fakeSecretsClient{getSecretResponse: []byte("secret-value")}

	var receivedStore esv1.GenericStore
	var receivedNamespace string

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(_ context.Context, store esv1.GenericStore, _ client.Client, namespace string) (esv1.SecretsClient, error) {
				receivedStore = store
				receivedNamespace = namespace
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	req := &pb.GetSecretRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion:   "provider.external-secrets.io/v2alpha1",
			Kind:         "Fake",
			Name:         "backend",
			Namespace:    "provider-config-ns",
			StoreRefKind: esv1.SecretStoreKind,
		},
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef: &pb.ExternalSecretDataRemoteRef{
			Key:              "sample",
			Version:          "v1",
			Property:         "password",
			DecodingStrategy: string(esv1.ExternalSecretDecodeBase64),
			MetadataPolicy:   string(esv1.ExternalSecretMetadataPolicyFetch),
		},
	}

	resp, err := server.GetSecret(context.Background(), req)
	if err != nil {
		t.Fatalf("GetSecret() error = %v", err)
	}

	if string(resp.Value) != "secret-value" {
		t.Fatalf("expected secret-value, got %q", string(resp.Value))
	}
	if mapper.ref != req.ProviderRef || mapper.sourceNamespace != serverTestSourceNamespace {
		t.Fatalf("unexpected spec mapper input: ref=%#v namespace=%q", mapper.ref, mapper.sourceNamespace)
	}
	if receivedNamespace != "provider-config-ns" {
		t.Fatalf("unexpected new client namespace: %q", receivedNamespace)
	}
	syntheticStore, ok := receivedStore.(*SyntheticStore)
	if !ok {
		t.Fatalf("expected SyntheticStore, got %T", receivedStore)
	}
	if syntheticStore.Namespace != "provider-config-ns" {
		t.Fatalf("unexpected synthetic store namespace: %q", syntheticStore.Namespace)
	}
	if syntheticStore.Kind != esv1.SecretStoreKind {
		t.Fatalf("unexpected synthetic store kind: %q", syntheticStore.Kind)
	}
	if syntheticStore.GetSpec() != mapper.spec {
		t.Fatalf("unexpected synthetic spec: %#v", syntheticStore.GetSpec())
	}
	if fakeClient.getSecretRef.Key != "sample" || fakeClient.getSecretRef.Version != "v1" || fakeClient.getSecretRef.Property != "password" {
		t.Fatalf("unexpected remote ref: %#v", fakeClient.getSecretRef)
	}
	if fakeClient.getSecretRef.DecodingStrategy != esv1.ExternalSecretDecodeBase64 {
		t.Fatalf("unexpected decoding strategy: %q", fakeClient.getSecretRef.DecodingStrategy)
	}
	if fakeClient.getSecretRef.MetadataPolicy != esv1.ExternalSecretMetadataPolicyFetch {
		t.Fatalf("unexpected metadata policy: %q", fakeClient.getSecretRef.MetadataPolicy)
	}
	if !fakeClient.closeCalled {
		t.Fatal("expected secrets client to be closed")
	}
}

func TestServerPushSecretUsesClusterSecretStoreKind(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{
			Provider: &esv1.SecretStoreProvider{
				Fake: &esv1.FakeProvider{},
			},
		},
	}
	fakeClient := &fakeSecretsClient{}

	var receivedStore esv1.GenericStore

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(_ context.Context, store esv1.GenericStore, _ client.Client, _ string) (esv1.SecretsClient, error) {
				receivedStore = store
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	_, err := server.PushSecret(context.Background(), &pb.PushSecretRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion:   "provider.external-secrets.io/v2alpha1",
			Kind:         "Fake",
			Name:         "backend",
			StoreRefKind: esv1.ClusterSecretStoreKind,
		},
		SourceNamespace: serverTestSourceNamespace,
		SecretData: map[string][]byte{
			serverTestValue: []byte("secret-value"),
		},
		PushSecretData: &pb.PushSecretData{
			RemoteKey: "remote-secret",
			SecretKey: serverTestValue,
		},
	})
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	syntheticStore, ok := receivedStore.(*SyntheticStore)
	if !ok {
		t.Fatalf("expected SyntheticStore, got %T", receivedStore)
	}
	if syntheticStore.Kind != esv1.ClusterSecretStoreKind {
		t.Fatalf("unexpected synthetic store kind: %q", syntheticStore.Kind)
	}
}

func TestServerGetSecretMapDelegates(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}
	fakeClient := &fakeSecretsClient{
		getSecretMapResponse: map[string][]byte{"foo": []byte("bar")},
	}

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	resp, err := server.GetSecretMap(context.Background(), &pb.GetSecretMapRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef:       &pb.ExternalSecretDataRemoteRef{Key: "sample"},
	})
	if err != nil {
		t.Fatalf("GetSecretMap() error = %v", err)
	}

	if string(resp.Secrets["foo"]) != "bar" {
		t.Fatalf("unexpected response: %#v", resp.Secrets)
	}
	if fakeClient.getSecretMapRef.Key != "sample" {
		t.Fatalf("unexpected ref: %#v", fakeClient.getSecretMapRef)
	}
}

func TestServerGetSecretRejectsMissingProviderReference(t *testing.T) {
	server := NewServer(nil, ProviderMapping{}, (&specMapperRecorder{}).mapRef)

	_, err := server.GetSecret(context.Background(), &pb.GetSecretRequest{
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef:       &pb.ExternalSecretDataRemoteRef{Key: "sample"},
	})
	if !errors.Is(err, errProviderReferenceRequired) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerGetSecretMapRejectsMissingProviderReference(t *testing.T) {
	server := NewServer(nil, ProviderMapping{}, (&specMapperRecorder{}).mapRef)

	_, err := server.GetSecretMap(context.Background(), &pb.GetSecretMapRequest{
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef:       &pb.ExternalSecretDataRemoteRef{Key: "sample"},
	})
	if !errors.Is(err, errProviderReferenceRequired) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerGetAllSecretsRejectsMissingProviderReference(t *testing.T) {
	server := NewServer(nil, ProviderMapping{}, (&specMapperRecorder{}).mapRef)

	_, err := server.GetAllSecrets(context.Background(), &pb.GetAllSecretsRequest{
		SourceNamespace: serverTestSourceNamespace,
		Find:            &pb.ExternalSecretFind{},
	})
	if !errors.Is(err, errProviderReferenceRequired) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestServerGetAllSecretsMapsFindCriteria(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}
	fakeClient := &fakeSecretsClient{
		getAllSecretsResponse: map[string][]byte{"db-password": []byte(serverTestValue)},
	}

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	resp, err := server.GetAllSecrets(context.Background(), &pb.GetAllSecretsRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		Find: &pb.ExternalSecretFind{
			Tags:               map[string]string{"team": "a"},
			Path:               "/team-a",
			ConversionStrategy: string(esv1.ExternalSecretConversionDefault),
			DecodingStrategy:   string(esv1.ExternalSecretDecodeBase64),
			Name:               &pb.FindName{Regexp: "db-.*"},
		},
	})
	if err != nil {
		t.Fatalf("GetAllSecrets() error = %v", err)
	}

	if string(resp.Secrets["db-password"]) != serverTestValue {
		t.Fatalf("unexpected response: %#v", resp.Secrets)
	}
	if fakeClient.getAllSecretsFind.Tags["team"] != "a" {
		t.Fatalf("unexpected find tags: %#v", fakeClient.getAllSecretsFind)
	}
	if fakeClient.getAllSecretsFind.Path == nil || *fakeClient.getAllSecretsFind.Path != "/team-a" {
		t.Fatalf("unexpected find path: %#v", fakeClient.getAllSecretsFind.Path)
	}
	if fakeClient.getAllSecretsFind.Name == nil || fakeClient.getAllSecretsFind.Name.RegExp != "db-.*" {
		t.Fatalf("unexpected find name: %#v", fakeClient.getAllSecretsFind.Name)
	}
}

func TestServerPushDeleteAndExistsMapWriteRequests(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}
	fakeClient := &fakeSecretsClient{secretExistsResp: true}

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	_, err := server.PushSecret(context.Background(), &pb.PushSecretRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		SecretData: map[string][]byte{
			"token": []byte(serverTestValue),
		},
		PushSecretData: &pb.PushSecretData{
			RemoteKey: serverTestRemoteKey,
			SecretKey: "token",
			Property:  serverTestProperty,
			Metadata:  []byte(`{"owner":"eso"}`),
		},
	})
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	if fakeClient.pushSecretSecret == nil || string(fakeClient.pushSecretSecret.Data["token"]) != serverTestValue {
		t.Fatalf("unexpected pushed secret: %#v", fakeClient.pushSecretSecret)
	}
	if fakeClient.pushSecretSecret.Type != "" {
		t.Fatalf("unexpected secret type: %q", fakeClient.pushSecretSecret.Type)
	}
	if fakeClient.pushSecretData.GetRemoteKey() != serverTestRemoteKey || fakeClient.pushSecretData.GetSecretKey() != "token" || fakeClient.pushSecretData.GetProperty() != serverTestProperty {
		t.Fatalf("unexpected push data: %#v", fakeClient.pushSecretData)
	}
	if got := fakeClient.pushSecretData.GetMetadata(); got == nil || string(got.Raw) != `{"owner":"eso"}` {
		t.Fatalf("unexpected metadata: %#v", got)
	}

	_, err = server.DeleteSecret(context.Background(), &pb.DeleteSecretRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef: &pb.PushSecretRemoteRef{
			RemoteKey: serverTestRemoteKey,
			Property:  serverTestProperty,
		},
	})
	if err != nil {
		t.Fatalf("DeleteSecret() error = %v", err)
	}

	if fakeClient.deleteSecretRef.GetRemoteKey() != serverTestRemoteKey || fakeClient.deleteSecretRef.GetProperty() != serverTestProperty {
		t.Fatalf("unexpected delete ref: %#v", fakeClient.deleteSecretRef)
	}

	resp, err := server.SecretExists(context.Background(), &pb.SecretExistsRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		RemoteRef: &pb.PushSecretRemoteRef{
			RemoteKey: serverTestRemoteKey,
			Property:  serverTestProperty,
		},
	})
	if err != nil {
		t.Fatalf("SecretExists() error = %v", err)
	}

	if !resp.Exists {
		t.Fatal("expected exists response to be true")
	}
	if fakeClient.secretExistsRef.GetRemoteKey() != serverTestRemoteKey || fakeClient.secretExistsRef.GetProperty() != serverTestProperty {
		t.Fatalf("unexpected exists ref: %#v", fakeClient.secretExistsRef)
	}
}

func TestServerPushSecretForwardsKubernetesSecretMetadata(t *testing.T) {
	mapper := &specMapperRecorder{
		spec: &esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}
	fakeClient := &fakeSecretsClient{}

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
				return fakeClient, nil
			},
		},
	}, mapper.mapRef)

	req := &pb.PushSecretRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: serverTestSourceNamespace,
		SecretData: map[string][]byte{
			".dockerconfigjson": []byte("payload"),
		},
		SecretType:        string(corev1.SecretTypeDockerConfigJson),
		SecretLabels:      map[string]string{"team": "platform"},
		SecretAnnotations: map[string]string{"owner": "app-team"},
		PushSecretData: &pb.PushSecretData{
			RemoteKey: serverTestRemoteKey,
			SecretKey: ".dockerconfigjson",
			Property:  serverTestProperty,
			Metadata:  []byte(`{"mergePolicy":"replace"}`),
		},
	}

	_, err := server.PushSecret(context.Background(), req)
	if err != nil {
		t.Fatalf("PushSecret() error = %v", err)
	}

	if fakeClient.pushSecretSecret == nil {
		t.Fatal("expected pushed secret to be recorded")
	}
	if got, want := string(fakeClient.pushSecretSecret.Data[".dockerconfigjson"]), "payload"; got != want {
		t.Errorf("expected payload %q, got %q", want, got)
	}
	if got, want := fakeClient.pushSecretSecret.Type, corev1.SecretTypeDockerConfigJson; got != want {
		t.Errorf("expected secret type %q, got %q", want, got)
	}
	if got, want := fakeClient.pushSecretSecret.Labels["team"], "platform"; got != want {
		t.Errorf("expected secret label team=%q, got %q", want, got)
	}
	if got, want := fakeClient.pushSecretSecret.Annotations["owner"], "app-team"; got != want {
		t.Errorf("expected secret annotation owner=%q, got %q", want, got)
	}
}

func TestServerValidateMapsReadyUnknownAndErrorResults(t *testing.T) {
	t.Run("ready", func(t *testing.T) {
		resp := runValidateTest(t, esv1.ValidationResultReady, nil)
		if !resp.Valid {
			t.Fatalf("expected valid response, got %#v", resp)
		}
	})

	t.Run("unknown", func(t *testing.T) {
		resp := runValidateTest(t, esv1.ValidationResultUnknown, nil)
		if !resp.Valid {
			t.Fatalf("expected unknown to be treated as valid, got %#v", resp)
		}
	})

	t.Run("error_result", func(t *testing.T) {
		resp := runValidateTest(t, esv1.ValidationResultError, nil)
		if resp.Valid {
			t.Fatalf("expected invalid response, got %#v", resp)
		}
	})

	t.Run("error", func(t *testing.T) {
		validateErr := errors.New("invalid credentials")
		resp := runValidateTest(t, esv1.ValidationResultError, validateErr)
		if resp.Valid || resp.Error != "invalid credentials" {
			t.Fatalf("unexpected response: %#v", resp)
		}
	})
}

func TestServerCapabilitiesMapsProviderCapabilities(t *testing.T) {
	testCases := []struct {
		name       string
		caps       esv1.SecretStoreCapabilities
		expectedPB pb.SecretStoreCapabilities
	}{
		{name: "read_only", caps: esv1.SecretStoreReadOnly, expectedPB: pb.SecretStoreCapabilities_READ_ONLY},
		{name: "write_only", caps: esv1.SecretStoreWriteOnly, expectedPB: pb.SecretStoreCapabilities_WRITE_ONLY},
		{name: "read_write", caps: esv1.SecretStoreReadWrite, expectedPB: pb.SecretStoreCapabilities_READ_WRITE},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := NewServer(nil, ProviderMapping{
				schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
					caps: tc.caps,
					newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
						return &fakeSecretsClient{}, nil
					},
				},
			}, (&specMapperRecorder{}).mapRef)

			resp, err := server.Capabilities(context.Background(), &pb.CapabilitiesRequest{
				ProviderRef: &pb.ProviderReference{
					ApiVersion: "provider.external-secrets.io/v2alpha1",
					Kind:       "Fake",
					Name:       "backend",
				},
			})
			if err != nil {
				t.Fatalf("Capabilities() error = %v", err)
			}
			if resp.Capabilities != tc.expectedPB {
				t.Fatalf("expected %v, got %v", tc.expectedPB, resp.Capabilities)
			}
		})
	}
}

func TestServerRejectsInvalidRequests(t *testing.T) {
	server := NewServer(nil, ProviderMapping{}, (&specMapperRecorder{}).mapRef)

	testCases := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "get_secret_nil_request",
			call: func() error {
				_, err := server.GetSecret(context.Background(), nil)
				return err
			},
			want: "request or remote ref is nil",
		},
		{
			name: "get_secret_empty_source_namespace",
			call: func() error {
				_, err := server.GetSecret(context.Background(), &pb.GetSecretRequest{
					RemoteRef: &pb.ExternalSecretDataRemoteRef{Key: "sample"},
				})
				return err
			},
			want: "source namespace is required",
		},
		{
			name: "push_secret_nil_payload",
			call: func() error {
				_, err := server.PushSecret(context.Background(), &pb.PushSecretRequest{
					SourceNamespace: serverTestSourceNamespace,
				})
				return err
			},
			want: "request or push secret data is nil",
		},
		{
			name: "validate_nil_request",
			call: func() error {
				_, err := server.Validate(context.Background(), nil)
				return err
			},
			want: "request is nil",
		},
		{
			name: "capabilities_nil_request",
			call: func() error {
				_, err := server.Capabilities(context.Background(), nil)
				return err
			},
			want: "request is nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if err == nil || err.Error() != tc.want {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
	}
}

func runValidateTest(t *testing.T, result esv1.ValidationResult, validateErr error) *pb.ValidateResponse {
	t.Helper()

	server := NewServer(nil, ProviderMapping{
		schema.GroupVersionKind{Group: "provider.external-secrets.io", Version: "v2alpha1", Kind: "Fake"}: &fakeProviderInterface{
			caps: esv1.SecretStoreReadWrite,
			newClient: func(context.Context, esv1.GenericStore, client.Client, string) (esv1.SecretsClient, error) {
				return &fakeSecretsClient{
					validateResult: result,
					validateErr:    validateErr,
				}, nil
			},
		},
	}, (&specMapperRecorder{
		spec: &esv1.SecretStoreSpec{Provider: &esv1.SecretStoreProvider{Fake: &esv1.FakeProvider{}}},
	}).mapRef)

	resp, err := server.Validate(context.Background(), &pb.ValidateRequest{
		ProviderRef: &pb.ProviderReference{
			ApiVersion: "provider.external-secrets.io/v2alpha1",
			Kind:       "Fake",
			Name:       "backend",
		},
		SourceNamespace: "tenant-a",
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	return resp
}
