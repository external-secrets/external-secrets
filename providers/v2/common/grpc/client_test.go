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

package grpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/reflect/protoreflect"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

const bufSize = 1024 * 1024

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
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	ref := esv1.ExternalSecretDataRemoteRef{
		Key:              "test-key",
		Version:          "v1",
		Property:         "password",
		DecodingStrategy: esv1.ExternalSecretDecodeBase64,
		MetadataPolicy:   esv1.ExternalSecretMetadataPolicyFetch,
	}

	value, err := client.GetSecret(context.Background(), ref, providerRef, "tenant-a")
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
	if mock.getSecretRequest.SourceNamespace != "tenant-a" {
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
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}

	value, err := client.GetSecretMap(context.Background(), esv1.ExternalSecretDataRemoteRef{Key: "test-key"}, providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("GetSecretMap failed: %v", err)
	}

	if string(value["a"]) != "b" {
		t.Fatalf("expected map[a]=b, got %#v", value)
	}
	if mock.getSecretMapRequest == nil {
		t.Fatal("expected get secret map request to be recorded")
	}
	if mock.getSecretMapRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected request: %#v", mock.getSecretMapRequest)
	}
	assertProviderRefEqual(t, mock.getSecretMapRequest.ProviderRef, providerRef)
}

func TestClientGetAllSecretsSendsFindCriteria(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}
	path := "/team-a"

	secrets, err := client.GetAllSecrets(context.Background(), esv1.ExternalSecretFind{
		Tags: map[string]string{"team": "a"},
		Path: &path,
		Name: &esv1.FindName{RegExp: "db-.*"},
	}, providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("GetAllSecrets failed: %v", err)
	}

	if string(secrets["db-password"]) != "value" {
		t.Fatalf("unexpected secrets: %#v", secrets)
	}
	if mock.getAllSecretsRequest == nil {
		t.Fatal("expected get all secrets request to be recorded")
	}
	if mock.getAllSecretsRequest.SourceNamespace != "tenant-a" {
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

func TestClientPushDeleteExistsAndCapabilitiesSendProviderReferenceAndNamespace(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}

	err := client.PushSecret(context.Background(), map[string][]byte{"token": []byte("value")}, &pb.PushSecretData{
		RemoteKey: "remote/path",
		SecretKey: "token",
		Property:  "property",
		Metadata:  []byte(`{"owner":"eso"}`),
	}, providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("PushSecret failed: %v", err)
	}
	if mock.pushSecretRequest == nil {
		t.Fatal("expected push secret request to be recorded")
	}
	if mock.pushSecretRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected push request: %#v", mock.pushSecretRequest)
	}
	assertProviderRefEqual(t, mock.pushSecretRequest.ProviderRef, providerRef)
	if string(mock.pushSecretRequest.SecretData["token"]) != "value" {
		t.Fatalf("unexpected pushed secret data: %#v", mock.pushSecretRequest.SecretData)
	}

	err = client.DeleteSecret(context.Background(), &pb.PushSecretRemoteRef{
		RemoteKey: "remote/path",
		Property:  "property",
	}, providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("DeleteSecret failed: %v", err)
	}
	if mock.deleteRequest == nil || mock.deleteRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected delete request: %#v", mock.deleteRequest)
	}
	assertProviderRefEqual(t, mock.deleteRequest.ProviderRef, providerRef)

	exists, err := client.SecretExists(context.Background(), &pb.PushSecretRemoteRef{
		RemoteKey: "remote/path",
		Property:  "property",
	}, providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("SecretExists failed: %v", err)
	}
	if !exists {
		t.Fatal("expected exists to be true")
	}
	if mock.existsRequest == nil || mock.existsRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected exists request: %#v", mock.existsRequest)
	}
	assertProviderRefEqual(t, mock.existsRequest.ProviderRef, providerRef)

	caps, err := client.Capabilities(context.Background(), providerRef, "tenant-a")
	if err != nil {
		t.Fatalf("Capabilities failed: %v", err)
	}
	if caps != pb.SecretStoreCapabilities_READ_WRITE {
		t.Fatalf("expected READ_WRITE, got %v", caps)
	}
	if mock.capabilitiesRequest == nil || mock.capabilitiesRequest.SourceNamespace != "tenant-a" {
		t.Fatalf("unexpected capabilities request: %#v", mock.capabilitiesRequest)
	}
	assertProviderRefEqual(t, mock.capabilitiesRequest.ProviderRef, providerRef)
}

func TestClientPushSecretSendsExpandedKubernetesSecretFields(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)
	providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}

	err := client.PushSecret(context.Background(), map[string][]byte{
		".dockerconfigjson": []byte("payload"),
	}, &pb.PushSecretData{
		RemoteKey: "remote/path",
		SecretKey: ".dockerconfigjson",
		Property:  "property",
		Metadata:  []byte(`{"owner":"eso"}`),
	}, providerRef, "tenant-a")
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
	if got, want := mock.pushSecretRequest.SourceNamespace, "tenant-a"; got != want {
		t.Errorf("expected source namespace %q, got %q", want, got)
	}

	secretType, ok := getPushSecretRequestStringField(mock.pushSecretRequest, "secret_type")
	if !ok {
		t.Errorf("push request is missing secret_type field")
	} else if want := "kubernetes.io/dockerconfigjson"; secretType != want {
		t.Errorf("expected secret_type=%q, got %q", want, secretType)
	}

	labels, ok := getPushSecretRequestStringMapField(mock.pushSecretRequest, "secret_labels")
	if !ok {
		t.Errorf("push request is missing secret_labels field")
	} else if got, want := labels["team"], "platform"; got != want {
		t.Errorf("expected secret_labels.team=%q, got %q", want, got)
	}

	annotations, ok := getPushSecretRequestStringMapField(mock.pushSecretRequest, "secret_annotations")
	if !ok {
		t.Errorf("push request is missing secret_annotations field")
	} else if got, want := annotations["owner"], "eso"; got != want {
		t.Errorf("expected secret_annotations.owner=%q, got %q", want, got)
	}
}

func TestClientValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockServer{}
		conn, cleanup := setupTestServer(t, mock)
		defer cleanup()

		client := NewClientWithConn(conn)
		providerRef := &pb.ProviderReference{Name: "provider", Namespace: "config-ns"}

		err := client.Validate(context.Background(), providerRef, "tenant-a")
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
		if mock.validateRequest == nil {
			t.Fatal("expected validate request to be recorded")
		}
		if mock.validateRequest.SourceNamespace != "tenant-a" {
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

		err := client.Validate(context.Background(), &pb.ProviderReference{Name: "provider"}, "tenant-a")
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
	if got.ApiVersion != want.ApiVersion || got.Kind != want.Kind || got.Name != want.Name || got.Namespace != want.Namespace {
		t.Fatalf("unexpected provider ref: got=%#v want=%#v", got, want)
	}
}

func getPushSecretRequestStringField(req *pb.PushSecretRequest, fieldName protoreflect.Name) (string, bool) {
	msg := req.ProtoReflect()
	field := msg.Descriptor().Fields().ByName(fieldName)
	if field == nil {
		return "", false
	}
	return msg.Get(field).String(), true
}

func getPushSecretRequestStringMapField(req *pb.PushSecretRequest, fieldName protoreflect.Name) (map[string]string, bool) {
	msg := req.ProtoReflect()
	field := msg.Descriptor().Fields().ByName(fieldName)
	if field == nil {
		return nil, false
	}

	values := map[string]string{}
	msg.Get(field).Map().Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
		values[key.String()] = value.String()
		return true
	})
	return values, true
}
