package remote

import (
	"context"
	"errors"
	"fmt"
	"log"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct{}

var provider = &Provider{}

func GetProvider(store esapi.GenericStore) (esapi.Provider, error) {
	return provider, nil
}

// NewClient constructs a SecretsManager Provider
func (p *Provider) NewClient(ctx context.Context, store esapi.GenericStore, kube client.Client, namespace string) (esapi.SecretsClient, error) {
	spec := store.GetSpec()
	if spec == nil {
		return nil, errors.New("store spec is nil")
	}
	if spec.Provider == nil {
		return nil, errors.New("store provider is nil")
	}

	providerName, err := esapi.GetProviderName(spec.Provider)
	if err != nil {
		return nil, errors.New("could not get provider name")
	}

	log.Printf("remote provider found providerName=%s\n", providerName)

	addr := fmt.Sprintf("unix:///tmp/eso-%s.sock", providerName)

	// Set up a connection to the server.
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("unable to connect: %w", err)
	}
	grpcClient := pb.NewSecretsClientClient(conn)

	return &Client{
		store:      store,
		namespace:  namespace,
		conn:       conn,
		grpcClient: grpcClient,
	}, nil
}

// ValidateStore checks if the provided store is valid
func (p *Provider) ValidateStore(store esapi.GenericStore) error {
	return nil
}

// Capabilities returns the provider Capabilities (Read, Write, ReadWrite)
func (p *Provider) Capabilities() esapi.SecretStoreCapabilities {
	return esapi.SecretStoreReadWrite
}
