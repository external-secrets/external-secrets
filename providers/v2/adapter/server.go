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

// Package adapter provides a unified server that wraps v1 providers and generators for v2 gRPC services.
package adapter

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genpb "github.com/external-secrets/external-secrets/proto/generator"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	"github.com/external-secrets/external-secrets/providers/v2/adapter/generator"
	"github.com/external-secrets/external-secrets/providers/v2/adapter/store"
)

// Server is a unified gRPC server that implements both SecretStoreProvider and GeneratorProvider.
// It embeds both the store and generator servers to provide a single implementation.
type Server struct {
	pb.UnimplementedSecretStoreProviderServer
	genpb.UnimplementedGeneratorProviderServer
	storeServer     *store.Server
	generatorServer *generator.Server
}

// NewServer creates a new unified adapter server that wraps v1 providers and generators.
// It combines both store and generator functionality into a single gRPC server.
func NewServer(
	kubeClient client.Client,
	scheme *runtime.Scheme,
	providerMapping store.ProviderMapping,
	specMapper store.SpecMapper,
	generatorMapping generator.GeneratorMapping,
) *Server {
	return &Server{
		storeServer:     store.NewServer(kubeClient, providerMapping, specMapper),
		generatorServer: generator.NewServer(kubeClient, scheme, generatorMapping),
	}
}

// Ensure Server implements both interfaces.
var _ pb.SecretStoreProviderServer = (*Server)(nil)
var _ genpb.GeneratorProviderServer = (*Server)(nil)

// Store methods - delegated to store.Server

// GetSecret retrieves a single secret from the provider.
func (s *Server) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	return s.storeServer.GetSecret(ctx, req)
}

// PushSecret pushes a secret to the provider.
func (s *Server) PushSecret(ctx context.Context, req *pb.PushSecretRequest) (*pb.PushSecretResponse, error) {
	return s.storeServer.PushSecret(ctx, req)
}

// DeleteSecret deletes a secret from the provider.
func (s *Server) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	return s.storeServer.DeleteSecret(ctx, req)
}

// SecretExists checks if a secret exists in the provider.
func (s *Server) SecretExists(ctx context.Context, req *pb.SecretExistsRequest) (*pb.SecretExistsResponse, error) {
	return s.storeServer.SecretExists(ctx, req)
}

// GetAllSecrets retrieves multiple secrets from the provider.
func (s *Server) GetAllSecrets(ctx context.Context, req *pb.GetAllSecretsRequest) (*pb.GetAllSecretsResponse, error) {
	return s.storeServer.GetAllSecrets(ctx, req)
}

// Validate validates the provider configuration.
func (s *Server) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	return s.storeServer.Validate(ctx, req)
}

// Capabilities returns the capabilities of the provider.
func (s *Server) Capabilities(ctx context.Context, req *pb.CapabilitiesRequest) (*pb.CapabilitiesResponse, error) {
	return s.storeServer.Capabilities(ctx, req)
}

// Generator methods - delegated to generator.Server

// Generate generates a new secret value.
func (s *Server) Generate(ctx context.Context, req *genpb.GenerateRequest) (*genpb.GenerateResponse, error) {
	return s.generatorServer.Generate(ctx, req)
}

// Cleanup performs cleanup operations for the generator.
func (s *Server) Cleanup(ctx context.Context, req *genpb.CleanupRequest) (*genpb.CleanupResponse, error) {
	return s.generatorServer.Cleanup(ctx, req)
}
