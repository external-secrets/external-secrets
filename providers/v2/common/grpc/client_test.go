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

package grpc

import (
	"bytes"
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	v2 "github.com/external-secrets/external-secrets/providers/v2/common"
)

const (
	bufSize             = 1024 * 1024
	testSourceNamespace = "tenant-a"
)

type mockServer struct {
	pb.UnimplementedSecretStoreProviderServer

	getSecretResponse *pb.GetSecretResponse
	getSecretRequest  *pb.GetSecretRequest

	getSecretMapResponse *pb.GetSecretMapResponse
	getSecretMapRequest  *pb.GetSecretMapRequest

	getAllSecretsResponse *pb.GetAllSecretsResponse
	getAllSecretsRequest  *pb.GetAllSecretsRequest

	pushSecretRequest *pb.PushSecretRequest
	deleteRequest     *pb.DeleteSecretRequest
	existsRequest     *pb.SecretExistsRequest
	existsResponse    *pb.SecretExistsResponse

	validateResponse *pb.ValidateResponse
	validateRequest  *pb.ValidateRequest

	capabilitiesResponse *pb.CapabilitiesResponse
	capabilitiesRequest  *pb.CapabilitiesRequest
}

func (m *mockServer) GetSecret(_ context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	m.getSecretRequest = req
	if m.getSecretResponse != nil {
		return m.getSecretResponse, nil
	}
	return &pb.GetSecretResponse{Value: []byte("test-secret-value")}, nil
}

func (m *mockServer) GetSecretMap(_ context.Context, req *pb.GetSecretMapRequest) (*pb.GetSecretMapResponse, error) {
	m.getSecretMapRequest = req
	if m.getSecretMapResponse != nil {
		return m.getSecretMapResponse, nil
	}
	return &pb.GetSecretMapResponse{
		Secrets: map[string][]byte{"foo": []byte("bar")},
	}, nil
}

func (m *mockServer) GetAllSecrets(_ context.Context, req *pb.GetAllSecretsRequest) (*pb.GetAllSecretsResponse, error) {
	m.getAllSecretsRequest = req
	if m.getAllSecretsResponse != nil {
		return m.getAllSecretsResponse, nil
	}
	return &pb.GetAllSecretsResponse{
		Secrets: map[string][]byte{"db-password": []byte("value")},
	}, nil
}

func (m *mockServer) PushSecret(_ context.Context, req *pb.PushSecretRequest) (*pb.PushSecretResponse, error) {
	m.pushSecretRequest = req
	return &pb.PushSecretResponse{}, nil
}

func (m *mockServer) DeleteSecret(_ context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	m.deleteRequest = req
	return &pb.DeleteSecretResponse{}, nil
}

func (m *mockServer) SecretExists(_ context.Context, req *pb.SecretExistsRequest) (*pb.SecretExistsResponse, error) {
	m.existsRequest = req
	if m.existsResponse != nil {
		return m.existsResponse, nil
	}
	return &pb.SecretExistsResponse{Exists: true}, nil
}

func (m *mockServer) Validate(_ context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	m.validateRequest = req
	if m.validateResponse != nil {
		return m.validateResponse, nil
	}
	return &pb.ValidateResponse{Valid: true}, nil
}

func (m *mockServer) Capabilities(_ context.Context, req *pb.CapabilitiesRequest) (*pb.CapabilitiesResponse, error) {
	m.capabilitiesRequest = req
	if m.capabilitiesResponse != nil {
		return m.capabilitiesResponse, nil
	}
	return &pb.CapabilitiesResponse{Capabilities: pb.SecretStoreCapabilities_READ_WRITE}, nil
}

func setupTestServer(t *testing.T, mock *mockServer) (*grpc.ClientConn, func()) {
	t.Helper()

	lis := bufconn.Listen(bufSize)
	baseServer := grpc.NewServer()
	pb.RegisterSecretStoreProviderServer(baseServer, mock)
	go func() {
		if err := baseServer.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(context.Background(), "",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	cleanup := func() {
		_ = conn.Close()
		baseServer.Stop()
		_ = lis.Close()
	}

	return conn, cleanup
}

func TestClientGetSecretSendsProviderReferenceAndNamespace(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ProviderStoreKindStr}
	ref := esv1.ExternalSecretDataRemoteRef{
		Key:              "test-key",
		Version:          "v1",
		Property:         "password",
		DecodingStrategy: esv1.ExternalSecretDecodeBase64,
		MetadataPolicy:   esv1.ExternalSecretMetadataPolicyFetch,
	}

	value, err := client.GetSecret(context.Background(), ref, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if string(value) != "test-secret-value" {
		t.Fatalf("expected test-secret-value, got %q", string(value))
	}
	if mock.getSecretRequest == nil {
		t.Fatal("expected get secret request to be recorded")
	}
	assertProviderRefEqual(t, mock.getSecretRequest.ProviderRef, providerRef)
	if mock.getSecretRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected source namespace: %q", mock.getSecretRequest.SourceNamespace)
	}
	if mock.getSecretRequest.RemoteRef.Key != "test-key" || mock.getSecretRequest.RemoteRef.Version != "v1" || mock.getSecretRequest.RemoteRef.Property != "password" {
		t.Fatalf("unexpected remote ref: %#v", mock.getSecretRequest.RemoteRef)
	}
	if mock.getSecretRequest.RemoteRef.DecodingStrategy != string(esv1.ExternalSecretDecodeBase64) {
		t.Fatalf("unexpected decoding strategy: %q", mock.getSecretRequest.RemoteRef.DecodingStrategy)
	}
	if mock.getSecretRequest.RemoteRef.MetadataPolicy != string(esv1.ExternalSecretMetadataPolicyFetch) {
		t.Fatalf("unexpected metadata policy: %q", mock.getSecretRequest.RemoteRef.MetadataPolicy)
	}
}

func TestClientGetSecretMapSendsProviderReferenceAndNamespace(t *testing.T) {
	mock := &mockServer{
		getSecretMapResponse: &pb.GetSecretMapResponse{
			Secrets: map[string][]byte{"a": []byte("b")},
		},
	}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ProviderStoreKindStr}

	value, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "test-key"}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetSecretMap failed: %v", err)
	}

	if string(value["a"]) != "b" {
		t.Fatalf("expected map[a]=b, got %#v", value)
	}
	if mock.getSecretMapRequest == nil {
		t.Fatal("expected get secret map request to be recorded")
	}
	if mock.getSecretMapRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected request: %#v", mock.getSecretMapRequest)
	}
	assertProviderRefEqual(t, mock.getSecretMapRequest.ProviderRef, providerRef)
}

func TestClientGetAllSecretsSendsFindCriteria(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ProviderStoreKindStr}
	path := "/team-a"

	secrets, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Tags: map[string]string{"team": "a"},
		Path: &path,
		Name: &esv1.FindName{RegExp: "db-.*"},
	}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetAllSecrets failed: %v", err)
	}

	if string(secrets["db-password"]) != "value" {
		t.Fatalf("unexpected secrets: %#v", secrets)
	}
	if mock.getAllSecretsRequest == nil {
		t.Fatal("expected get all secrets request to be recorded")
	}
	if mock.getAllSecretsRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected request: %#v", mock.getAllSecretsRequest)
	}
	assertProviderRefEqual(t, mock.getAllSecretsRequest.ProviderRef, providerRef)
	if mock.getAllSecretsRequest.Find.Path != "/team-a" {
		t.Fatalf("unexpected path: %q", mock.getAllSecretsRequest.Find.Path)
	}
	if mock.getAllSecretsRequest.Find.Name == nil || mock.getAllSecretsRequest.Find.Name.Regexp != "db-.*" {
		t.Fatalf("unexpected name matcher: %#v", mock.getAllSecretsRequest.Find.Name)
	}
}

func TestClientGetSecretSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	value, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "test-key"}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if string(value) != "test-secret-value" {
		t.Fatalf("expected test-secret-value, got %q", string(value))
	}
	if mock.getSecretRequest == nil {
		t.Fatal("expected get secret request to be recorded")
	}
	if mock.getSecretRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.getSecretRequest.ProviderRef)
	}
	if mock.getSecretRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
	if mock.getSecretRequest.CompatibilityStore.StoreName != "compat-store" {
		t.Fatalf("unexpected compatibility store: %#v", mock.getSecretRequest.CompatibilityStore)
	}
}

func TestClientGetSecretMapSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	_, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "test-key"}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetSecretMap failed: %v", err)
	}

	if mock.getSecretMapRequest == nil {
		t.Fatal("expected get secret map request to be recorded")
	}
	if mock.getSecretMapRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.getSecretMapRequest.ProviderRef)
	}
	if mock.getSecretMapRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
	if mock.getSecretMapRequest.CompatibilityStore.StoreName != "compat-store" {
		t.Fatalf("unexpected compatibility store: %#v", mock.getSecretMapRequest.CompatibilityStore)
	}
}

func TestClientGetAllSecretsSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	_, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("GetAllSecrets failed: %v", err)
	}

	if mock.getAllSecretsRequest == nil {
		t.Fatal("expected get all secrets request to be recorded")
	}
	if mock.getAllSecretsRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.getAllSecretsRequest.ProviderRef)
	}
	if mock.getAllSecretsRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
	if mock.getAllSecretsRequest.CompatibilityStore.StoreName != "compat-store" {
		t.Fatalf("unexpected compatibility store: %#v", mock.getAllSecretsRequest.CompatibilityStore)
	}
}

func TestClientValidateSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	err := client.Validate(context.Background(), nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if mock.validateRequest == nil {
		t.Fatal("expected validate request to be recorded")
	}
	if mock.validateRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.validateRequest.ProviderRef)
	}
	if mock.validateRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
}

func TestClientPushSecretSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	err := client.PushSecret(context.Background(), &corev1.Secret{
		Data: map[string][]byte{"token": []byte("value")},
	}, &pb.PushSecretData{RemoteKey: "remote", SecretKey: "token"}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("PushSecret failed: %v", err)
	}

	if mock.pushSecretRequest == nil {
		t.Fatal("expected push secret request to be recorded")
	}
	if mock.pushSecretRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.pushSecretRequest.ProviderRef)
	}
	if mock.pushSecretRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
}

func TestClientDeleteSecretSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	err := client.DeleteSecret(context.Background(), &pb.PushSecretRemoteRef{RemoteKey: "remote"}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}

	if mock.deleteRequest == nil {
		t.Fatal("expected delete secret request to be recorded")
	}
	if mock.deleteRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.deleteRequest.ProviderRef)
	}
	if mock.deleteRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
}

func TestClientSecretExistsSendsCompatibilityStore(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	compatibilityStore := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "config-ns",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   []byte(`{"provider":{"fake":{"data":[{"key":"test-key","value":"test-secret-value"}]}}}`),
	}

	exists, err := client.SecretExists(context.Background(), &pb.PushSecretRemoteRef{RemoteKey: "remote"}, nil, compatibilityStore, testSourceNamespace)
	if err != nil {
		t.Fatalf("SecretExists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected secret to exist")
	}

	if mock.existsRequest == nil {
		t.Fatal("expected secret exists request to be recorded")
	}
	if mock.existsRequest.ProviderRef != nil {
		t.Fatalf("expected provider ref to be omitted, got %#v", mock.existsRequest.ProviderRef)
	}
	if mock.existsRequest.CompatibilityStore == nil {
		t.Fatal("expected compatibility store to be recorded")
	}
}

func TestReadRequestIdentityValidationRejectsMissingReadIdentity(t *testing.T) {
	testCases := []struct {
		name string
		call func(client v2.Provider) error
	}{
		{
			name: "get secret",
			call: func(client v2.Provider) error {
				_, err := client.GetSecret(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"}, nil, nil, testSourceNamespace)
				return err
			},
		},
		{
			name: "get secret map",
			call: func(client v2.Provider) error {
				_, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "sample"}, nil, nil, testSourceNamespace)
				return err
			},
		},
		{
			name: "get all secrets",
			call: func(client v2.Provider) error {
				_, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{}, nil, nil, testSourceNamespace)
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mock := &mockServer{}
			conn, cleanup := setupTestServer(t, mock)
			defer cleanup()

			err := tc.call(NewClientWithConn(conn))
			if err == nil || err.Error() != "provider reference or compatibility store is required for read operations" {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestCompatibilityStoreLogFieldsRedactSpecPayload(t *testing.T) {
	specPayload := []byte(`{"provider":{"fake":{"data":[{"key":"db","value":"secret-value"}]}}}`)
	store := &pb.CompatibilityStore{
		StoreName:       "compat-store",
		StoreNamespace:  "team-a",
		StoreKind:       esv1.SecretStoreKind,
		StoreUid:        "uid-1",
		StoreGeneration: 7,
		StoreSpecJson:   specPayload,
	}

	fields := compatibilityStoreLogFields(store)
	if len(fields) == 0 {
		t.Fatal("expected compatibility store log fields")
	}

	found := map[string]bool{}
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			t.Fatalf("expected string log key, got %T", fields[i])
		}
		found[key] = true
		if bytes.Equal([]byte(key), specPayload) {
			t.Fatalf("unexpected spec payload key: %q", key)
		}
		value := fields[i+1]
		if value == store {
			t.Fatalf("unexpected raw compatibility store in log fields")
		}
		if payload, ok := value.([]byte); ok && bytes.Equal(payload, specPayload) {
			t.Fatalf("unexpected raw spec payload in log fields")
		}
		if text, ok := value.(string); ok && text == string(specPayload) {
			t.Fatalf("unexpected raw spec payload string in log fields")
		}
	}

	for _, key := range []string{
		"compatibilityStoreKind",
		"compatibilityStoreName",
		"compatibilityStoreNamespace",
		"compatibilityStoreUID",
		"compatibilityStoreGeneration",
		"compatibilityStoreSpecBytes",
	} {
		if !found[key] {
			t.Fatalf("expected log field %q, got %#v", key, fields)
		}
	}
}

func TestClientPushDeleteExistsAndCapabilitiesSendProviderReferenceAndNamespace(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ProviderStoreKindStr}

	err := client.PushSecret(context.Background(), &corev1.Secret{
		Data: map[string][]byte{"token": []byte("value")},
	}, &pb.PushSecretData{
		RemoteKey: "remote/path",
		SecretKey: "token",
		Property:  "property",
		Metadata:  []byte(`{"mergePolicy":"replace"}`),
	}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("PushSecret failed: %v", err)
	}
	if mock.pushSecretRequest == nil {
		t.Fatal("expected push secret request to be recorded")
	}
	if mock.pushSecretRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected push request: %#v", mock.pushSecretRequest)
	}
	assertProviderRefEqual(t, mock.pushSecretRequest.ProviderRef, providerRef)
	if string(mock.pushSecretRequest.SecretData["token"]) != "value" {
		t.Fatalf("unexpected pushed secret data: %#v", mock.pushSecretRequest.SecretData)
	}

	err = client.DeleteSecret(context.Background(), &pb.PushSecretRemoteRef{
		RemoteKey: "remote/path",
		Property:  "property",
	}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}
	if mock.deleteRequest == nil || mock.deleteRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected delete request: %#v", mock.deleteRequest)
	}
	assertProviderRefEqual(t, mock.deleteRequest.ProviderRef, providerRef)

	exists, err := client.SecretExists(context.Background(), &pb.PushSecretRemoteRef{
		RemoteKey: "remote/path",
		Property:  "property",
	}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("SecretExists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}
	if mock.existsRequest == nil || mock.existsRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected exists request: %#v", mock.existsRequest)
	}
	assertProviderRefEqual(t, mock.existsRequest.ProviderRef, providerRef)

	caps, err := client.Capabilities(context.Background(), providerRef, testSourceNamespace)
	if err != nil {
		t.Fatalf("Capabilities failed: %v", err)
	}
	if caps != pb.SecretStoreCapabilities_READ_WRITE {
		t.Fatalf("expected READ_WRITE, got %v", caps)
	}
	if mock.capabilitiesRequest == nil || mock.capabilitiesRequest.SourceNamespace != testSourceNamespace {
		t.Fatalf("unexpected capabilities request: %#v", mock.capabilitiesRequest)
	}
	assertProviderRefEqual(t, mock.capabilitiesRequest.ProviderRef, providerRef)
}

func TestClientPushSecretSendsExpandedKubernetesSecretFields(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ClusterProviderStoreKindStr}

	err := client.PushSecret(context.Background(), &corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		ObjectMeta: metav1.ObjectMeta{
			Labels:      map[string]string{"team": "platform"},
			Annotations: map[string]string{"owner": "app-team"},
		},
		Data: map[string][]byte{
			".dockerconfigjson": []byte("payload"),
		},
	}, &pb.PushSecretData{
		RemoteKey: "remote/path",
		SecretKey: ".dockerconfigjson",
		Property:  "property",
		Metadata:  []byte(`{"mergePolicy":"replace"}`),
	}, providerRef, nil, testSourceNamespace)
	if err != nil {
		t.Fatalf("PushSecret failed: %v", err)
	}
	if mock.pushSecretRequest == nil {
		t.Fatal("expected push secret request to be recorded")
	}
	if got, want := string(mock.pushSecretRequest.SecretData[".dockerconfigjson"]), "payload"; got != want {
		t.Errorf("expected request secret data %q, got %q", want, got)
	}
	assertProviderRefEqual(t, mock.pushSecretRequest.ProviderRef, providerRef)
	if got, want := mock.pushSecretRequest.SourceNamespace, testSourceNamespace; got != want {
		t.Errorf("expected source namespace %q, got %q", want, got)
	}
	if got, want := mock.pushSecretRequest.SecretType, string(corev1.SecretTypeDockerConfigJson); got != want {
		t.Errorf("expected secret_type=%q, got %q", want, got)
	}
	if got, want := mock.pushSecretRequest.SecretLabels["team"], "platform"; got != want {
		t.Errorf("expected secret_labels.team=%q, got %q", want, got)
	}
	if got, want := mock.pushSecretRequest.SecretAnnotations["owner"], "app-team"; got != want {
		t.Errorf("expected secret_annotations.owner=%q, got %q", want, got)
	}
	if got, want := string(mock.pushSecretRequest.PushSecretData.Metadata), `{"mergePolicy":"replace"}`; got != want {
		t.Errorf("expected metadata=%q, got %q", want, got)
	}
}

func TestClientValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockServer{}
		conn, cleanup := setupTestServer(t, mock)
		defer cleanup()

		client := NewClientWithConn(conn)
		providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns", StoreRefKind: esv1.ProviderStoreKindStr}

		err := client.Validate(context.Background(), providerRef, nil, testSourceNamespace)
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if mock.validateRequest == nil {
			t.Fatal("expected validate request to be recorded")
		}
		if mock.validateRequest.SourceNamespace != testSourceNamespace {
			t.Fatalf("unexpected validate request: %#v", mock.validateRequest)
		}
		assertProviderRefEqual(t, mock.validateRequest.ProviderRef, providerRef)
	})

	t.Run("validation_error", func(t *testing.T) {
		mock := &mockServer{
			validateResponse: &pb.ValidateResponse{
				Valid: false,
				Error: "invalid credentials",
			},
		}
		conn, cleanup := setupTestServer(t, mock)
		defer cleanup()

		client := NewClientWithConn(conn)

		err := client.Validate(context.Background(), &pb.ProviderReference{Name: "provider"}, nil, testSourceNamespace)
		if err == nil {
			t.Fatal("Expected validation to fail, but it succeeded")
		}
		if err.Error() != "provider validation failed: invalid credentials" {
			t.Fatalf("unexpected error message: %v", err)
		}
	})
}

func TestClientClose(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)

	if err := client.Close(context.Background()); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestNewClientInvalidAddress(t *testing.T) {
	_, err := NewClient("", nil)
	if err == nil {
		t.Fatal("expected error for empty address, got nil")
	}
}

func assertProviderRefEqual(t *testing.T, got, want *pb.ProviderReference) {
	t.Helper()

	if got == nil || want == nil {
		t.Fatalf("provider refs must not be nil: got=%#v want=%#v", got, want)
	}
	if got.ApiVersion != want.ApiVersion || got.Kind != want.Kind || got.Name != want.Name || got.Namespace != want.Namespace || got.StoreRefKind != want.StoreRefKind {
		t.Fatalf("unexpected provider ref: got=%#v want=%#v", got, want)
	}
}

func TestProtoCompatibilityRequestsExposeCompatibilityStoreField(t *testing.T) {
	cases := []struct {
		name string
		msg  protoreflect.ProtoMessage
	}{
		{name: "GetSecretRequest", msg: &pb.GetSecretRequest{}},
		{name: "GetSecretMapRequest", msg: &pb.GetSecretMapRequest{}},
		{name: "GetAllSecretsRequest", msg: &pb.GetAllSecretsRequest{}},
		{name: "ValidateRequest", msg: &pb.ValidateRequest{}},
		{name: "PushSecretRequest", msg: &pb.PushSecretRequest{}},
		{name: "DeleteSecretRequest", msg: &pb.DeleteSecretRequest{}},
		{name: "SecretExistsRequest", msg: &pb.SecretExistsRequest{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fields := tc.msg.ProtoReflect().Descriptor().Fields()
			if fields.ByName("compatibility_store") == nil {
				t.Fatalf("expected %s to have field compatibility_store", tc.name)
			}
		})
	}
}
