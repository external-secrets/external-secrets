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

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
)

const bufSize = 1024 * 1024

// mockServer is a simple mock implementation of the SecretStoreProvider service
type mockServer struct {
	pb.UnimplementedSecretStoreProviderServer
	getSecretResponse *pb.GetSecretResponse
	validateResponse  *pb.ValidateResponse
}

func (m *mockServer) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	if m.getSecretResponse != nil {
		return m.getSecretResponse, nil
	}
	return &pb.GetSecretResponse{
		Value: []byte("test-secret-value"),
		Metadata: map[string]string{
			"version": "1",
		},
	}, nil
}

func (m *mockServer) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	if m.validateResponse != nil {
		return m.validateResponse, nil
	}
	return &pb.ValidateResponse{
		Valid: true,
	}, nil
}

// setupTestServer creates an in-memory gRPC server for testing
func setupTestServer(t *testing.T, mock *mockServer) (*grpc.ClientConn, func()) {
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
		conn.Close()
		baseServer.Stop()
		lis.Close()
	}

	return conn, cleanup
}

func TestClient_GetSecret(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)

	ref := esv1.ExternalSecretDataRemoteRef{
		Key:      "test-key",
		Version:  "v1",
		Property: "password",
	}

	value, err := client.GetSecret(context.Background(), ref)
	if err != nil {
		t.Fatalf("GetSecret failed: %v", err)
	}

	if string(value) != "test-secret-value" {
		t.Errorf("Expected 'test-secret-value', got '%s'", string(value))
	}
}

func TestClient_Validate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mock := &mockServer{}
		conn, cleanup := setupTestServer(t, mock)
		defer cleanup()

		client := NewClientWithConn(conn)

		err := client.Validate(context.Background())
		if err != nil {
			t.Fatalf("Validate failed: %v", err)
		}
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

		err := client.Validate(context.Background())
		if err == nil {
			t.Fatal("Expected validation to fail, but it succeeded")
		}

		if err.Error() != "provider validation failed: invalid credentials" {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestClient_Close(t *testing.T) {
	mock := &mockServer{}
	conn, cleanup := setupTestServer(t, mock)
	defer cleanup()

	client := NewClientWithConn(conn)

	err := client.Close(context.Background())
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestNewClient_InvalidAddress(t *testing.T) {
	_, err := NewClient("", nil)
	if err == nil {
		t.Fatal("Expected error for empty address, got nil")
	}
}
