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

// Package adapter adapts v1 provider implementations to the v2 gRPC interface.
package adapter

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	genpb "github.com/external-secrets/external-secrets/proto/generator"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	"github.com/external-secrets/external-secrets/providers/v2/adapter/store"
)

// V1AdapterServer wraps v1 providers and generators and exposes them as v2 gRPC services.
// This allows existing v1 provider and generator implementations to be used in the v2 architecture.
type V1AdapterServer struct {
	pb.UnimplementedSecretStoreProviderServer
	genpb.UnimplementedGeneratorProviderServer
	kubeClient client.Client
	scheme     *runtime.Scheme

	// we support multiple v1 providers, so we need to map the v2 provider
	// with apiVersion+kind to the corresponding v1 provider
	resourceMapping ProviderMapping
	specMapper      SpecMapper

	// we support multiple v1 generators, so we need to map the v2 generator
	// with apiVersion+kind to the corresponding v1 generator
	generatorMapping GeneratorMapping
}

// ProviderMapping maps Kubernetes resources to their provider implementations.
type ProviderMapping map[schema.GroupVersionKind]esv1.ProviderInterface

// GeneratorMapping maps Kubernetes resources to their generator implementations.
type GeneratorMapping map[schema.GroupVersionKind]genv1alpha1.Generator

// SpecMapper maps a provider reference to a SecretStoreSpec.
// This is used to create a synthetic store for the v1 provider.
type SpecMapper func(ref *pb.ProviderReference) (*esv1.SecretStoreSpec, error)

// NewAdapterServer creates a new V1AdapterServer that wraps v1 providers and generators.
func NewAdapterServer(kubeClient client.Client, scheme *runtime.Scheme, resourceMapping ProviderMapping, specMapping SpecMapper, generatorMapping GeneratorMapping) *V1AdapterServer {
	return &V1AdapterServer{
		kubeClient:       kubeClient,
		scheme:           scheme,
		resourceMapping:  resourceMapping,
		specMapper:       specMapping,
		generatorMapping: generatorMapping,
	}
}

func (s *V1AdapterServer) resolveProvider(ref *pb.ProviderReference) (esv1.ProviderInterface, error) {
	if ref == nil {
		return nil, fmt.Errorf("provider reference is nil")
	}

	splitted := strings.Split(ref.ApiVersion, "/")
	if len(splitted) != 2 {
		return nil, fmt.Errorf("invalid api version: %s", ref.ApiVersion)
	}
	group := splitted[0]
	version := splitted[1]

	key := schema.GroupVersionKind{
		Group:   group,
		Version: version,
		Kind:    ref.Kind,
	}
	v1Provider, ok := s.resourceMapping[key]
	if !ok {
		return nil, fmt.Errorf("resource mapping not found for %q", key)
	}
	return v1Provider, nil
}

func (s *V1AdapterServer) getClient(ctx context.Context, ref *pb.ProviderReference, namespace string) (esv1.SecretsClient, error) {
	if ref == nil {
		return nil, fmt.Errorf("request or remote ref is nil")
	}

	spec, err := s.specMapper(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to map provider reference to spec: %w", err)
	}
	// TODO: support cluster scoped Provider
	syntheticStore, err := store.NewSyntheticStore(spec, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create synthetic store: %w", err)
	}
	provider, err := s.resolveProvider(ref)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}
	return provider.NewClient(ctx, syntheticStore, s.kubeClient, namespace)
}

// GetSecret retrieves a single secret from the provider.
func (s *V1AdapterServer) GetSecret(ctx context.Context, req *pb.GetSecretRequest) (*pb.GetSecretResponse, error) {
	if req == nil || req.RemoteRef == nil {
		return nil, fmt.Errorf("request or remote ref is nil")
	}
	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert protobuf remote ref to v1 remote ref
	ref := esv1.ExternalSecretDataRemoteRef{
		Key:      req.RemoteRef.Key,
		Version:  req.RemoteRef.Version,
		Property: req.RemoteRef.Property,
	}
	if req.RemoteRef.DecodingStrategy != "" {
		ref.DecodingStrategy = esv1.ExternalSecretDecodingStrategy(req.RemoteRef.DecodingStrategy)
	}
	if req.RemoteRef.MetadataPolicy != "" {
		ref.MetadataPolicy = esv1.ExternalSecretMetadataPolicy(req.RemoteRef.MetadataPolicy)
	}

	value, err := client.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return &pb.GetSecretResponse{
		Value: value,
	}, nil
}

// PushSecret writes a secret to the provider.
func (s *V1AdapterServer) PushSecret(ctx context.Context, req *pb.PushSecretRequest) (*pb.PushSecretResponse, error) {
	if req == nil || req.PushSecretData == nil {
		return nil, fmt.Errorf("request or push secret data is nil")
	}

	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert map[string][]byte to *corev1.Secret
	secret := &corev1.Secret{
		Data: req.SecretData,
		Type: corev1.SecretTypeOpaque,
	}

	// Convert protobuf PushSecretData to v1 PushSecretData
	pushData := &pushSecretData{
		property:  req.PushSecretData.Property,
		secretKey: req.PushSecretData.SecretKey,
		remoteKey: req.PushSecretData.RemoteKey,
		metadata:  req.PushSecretData.Metadata,
	}

	// Call v1 PushSecret
	if err := client.PushSecret(ctx, secret, pushData); err != nil {
		return nil, fmt.Errorf("failed to push secret: %w", err)
	}

	return &pb.PushSecretResponse{}, nil
}

// DeleteSecret deletes a secret from the provider.
func (s *V1AdapterServer) DeleteSecret(ctx context.Context, req *pb.DeleteSecretRequest) (*pb.DeleteSecretResponse, error) {
	if req == nil || req.RemoteRef == nil {
		return nil, fmt.Errorf("request or remote ref is nil")
	}

	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert protobuf remote ref to v1 PushSecretRemoteRef
	remoteRef := &pushSecretRemoteRef{
		remoteKey: req.RemoteRef.RemoteKey,
		property:  req.RemoteRef.Property,
	}

	// Call v1 DeleteSecret
	if err := client.DeleteSecret(ctx, remoteRef); err != nil {
		return nil, fmt.Errorf("failed to delete secret: %w", err)
	}

	return &pb.DeleteSecretResponse{}, nil
}

// SecretExists checks if a secret exists in the provider.
func (s *V1AdapterServer) SecretExists(ctx context.Context, req *pb.SecretExistsRequest) (*pb.SecretExistsResponse, error) {
	if req == nil || req.RemoteRef == nil {
		return nil, fmt.Errorf("request or remote ref is nil")
	}

	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert protobuf remote ref to v1 PushSecretRemoteRef
	remoteRef := &pushSecretRemoteRef{
		remoteKey: req.RemoteRef.RemoteKey,
		property:  req.RemoteRef.Property,
	}

	// Call v1 SecretExists
	exists, err := client.SecretExists(ctx, remoteRef)
	if err != nil {
		return nil, fmt.Errorf("failed to check if secret exists: %w", err)
	}

	return &pb.SecretExistsResponse{
		Exists: exists,
	}, nil
}

// GetAllSecrets retrieves multiple secrets from the provider.
func (s *V1AdapterServer) GetAllSecrets(ctx context.Context, req *pb.GetAllSecretsRequest) (*pb.GetAllSecretsResponse, error) {
	if req == nil || req.Find == nil {
		return nil, fmt.Errorf("request or find criteria is nil")
	}

	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert protobuf ExternalSecretFind to v1 ExternalSecretFind
	find := esv1.ExternalSecretFind{
		Tags:               req.Find.Tags,
		ConversionStrategy: esv1.ExternalSecretConversionStrategy(req.Find.ConversionStrategy),
		DecodingStrategy:   esv1.ExternalSecretDecodingStrategy(req.Find.DecodingStrategy),
	}

	// Convert Path from string to *string
	if req.Find.Path != "" {
		path := req.Find.Path
		find.Path = &path
	}

	if req.Find.Name != nil {
		find.Name = &esv1.FindName{
			RegExp: req.Find.Name.Regexp,
		}
	}

	// Call v1 GetAllSecrets
	secrets, err := client.GetAllSecrets(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get all secrets: %w", err)
	}

	return &pb.GetAllSecretsResponse{
		Secrets: secrets,
	}, nil
}

// Validate checks if the provider configuration is valid.
func (s *V1AdapterServer) Validate(ctx context.Context, req *pb.ValidateRequest) (*pb.ValidateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	client, err := s.getClient(ctx, req.ProviderRef, req.SourceNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	result, err := client.Validate()
	if err != nil {
		return &pb.ValidateResponse{
			Valid: false,
			Error: err.Error(),
		}, nil
	}

	var valid bool
	switch result {
	case esv1.ValidationResultReady:
		valid = true
	case esv1.ValidationResultUnknown:
		valid = true // Unknown is treated as valid but warns
	case esv1.ValidationResultError:
		valid = false
	}

	return &pb.ValidateResponse{
		Valid:    valid,
		Warnings: []string{},
	}, nil
}

// Capabilities returns the capabilities of the provider.
// TODO: remove / rewrite capabilities:
// the provider should advertise what providers/generators it supports.
func (s *V1AdapterServer) Capabilities(_ context.Context, req *pb.CapabilitiesRequest) (*pb.CapabilitiesResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	provider, err := s.resolveProvider(req.ProviderRef)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve provider: %w", err)
	}
	caps := provider.Capabilities()
	var pbCaps pb.SecretStoreCapabilities
	switch caps {
	case esv1.SecretStoreReadOnly:
		pbCaps = pb.SecretStoreCapabilities_READ_ONLY
	case esv1.SecretStoreWriteOnly:
		pbCaps = pb.SecretStoreCapabilities_WRITE_ONLY
	case esv1.SecretStoreReadWrite:
		pbCaps = pb.SecretStoreCapabilities_READ_WRITE
	default:
		pbCaps = pb.SecretStoreCapabilities_READ_ONLY
	}

	return &pb.CapabilitiesResponse{
		Capabilities: pbCaps,
	}, nil
}

// pushSecretData implements esv1.PushSecretData.
type pushSecretData struct {
	property  string
	secretKey string
	remoteKey string
	metadata  []byte
}

func (p *pushSecretData) GetProperty() string {
	return p.property
}

func (p *pushSecretData) GetSecretKey() string {
	return p.secretKey
}

func (p *pushSecretData) GetRemoteKey() string {
	return p.remoteKey
}

func (p *pushSecretData) GetMetadata() *apiextensionsv1.JSON {
	if len(p.metadata) == 0 {
		return nil
	}
	return &apiextensionsv1.JSON{
		Raw: p.metadata,
	}
}

// pushSecretRemoteRef implements esv1.PushSecretRemoteRef.
type pushSecretRemoteRef struct {
	remoteKey string
	property  string
}

func (p *pushSecretRemoteRef) GetRemoteKey() string {
	return p.remoteKey
}

func (p *pushSecretRemoteRef) GetProperty() string {
	return p.property
}
