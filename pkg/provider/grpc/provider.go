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
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	pb "github.com/external-secrets/external-secrets/pkg/plugin/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &GRPCSecretClient{}
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

type GRPCSecretClient struct {
	kube      client.Client
	store     esv1beta1.GenericStore
	namespace string
	storeKind string

	conn     *grpc.ClientConn
	pbClient pb.SecretsClientClient
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		GRPC: &esv1beta1.GRPCProvider{},
	})
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	grpcClient := &GRPCSecretClient{
		kube:      kube,
		store:     store,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	// Set up a connection to the server.
	grpcClient.conn, err = grpc.Dial(provider.URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	grpcClient.pbClient = pb.NewSecretsClientClient(grpcClient.conn)

	return grpcClient, nil
}

func (p *Provider) ValidateStore(_ esv1beta1.GenericStore) error {
	return nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.GRPCProvider, error) {
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.GRPC == nil {
		return nil, fmt.Errorf("missing store provider webhook")
	}
	return spc.Provider.GRPC, nil
}

func (w *GRPCSecretClient) DeleteSecret(_ context.Context, _ esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Not Implemented PushSecret.
func (w *GRPCSecretClient) PushSecret(_ context.Context, _ []byte, _ esv1beta1.PushRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Empty GetAllSecrets.
func (w *GRPCSecretClient) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

func (w *GRPCSecretClient) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	store := esv1beta1.SecretStore{
		TypeMeta: v1.TypeMeta{
			APIVersion: esv1beta1.SchemeGroupVersion.String(),
			Kind:       esv1beta1.SecretStoreKind,
		},
		Spec: esv1beta1.SecretStoreSpec{
			Provider: &esv1beta1.SecretStoreProvider{
				Webhook: &esv1beta1.WebhookProvider{
					URL: "http://httpbin.org/anything",
				},
			},
		},
	}
	storeBytes, err := json.Marshal(store)
	if err != nil {
		return nil, err
	}

	res, err := w.pbClient.GetSecret(ctx, &pb.GetSecretRequest{
		Store:     storeBytes,
		Namespace: w.namespace,
		RemoteRef: &pb.RemoteRef{
			Key:                ref.Key,
			Version:            ref.Version,
			Property:           ref.Property,
			MetadataPolicy:     string(ref.MetadataPolicy),
			DecodingStrategy:   string(ref.DecodingStrategy),
			ConversionStrategy: string(ref.ConversionStrategy),
		},
	})
	if err != nil {
		return nil, err
	}
	log.Printf("secret=%s, err=%s", string(res.Secret), res.Error)
	if res.Error != "" {
		return nil, errors.New(res.Error)
	}
	return res.Secret, nil
}

func (w *GRPCSecretClient) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("GetSecretMap not implemented")
}

func (w *GRPCSecretClient) Close(_ context.Context) error {
	return w.conn.Close()
}

func (w *GRPCSecretClient) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}
