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
package secretmanager

import (
	"context"
	"fmt"
	"log"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

const (
	cloudPlatformRole = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion    = "latest"
)

type GoogleSecretManagerClient interface {
	AccessSecretVersion(ctx context.Context, req *secretmanagerpb.AccessSecretVersionRequest, opts ...gax.CallOption) (*secretmanagerpb.AccessSecretVersionResponse, error)
	Close() error
}

// ProviderGCP is a provider for GCP Secret Manager.
type ProviderGCP struct {
	projectID           string
	SecretManagerClient GoogleSecretManagerClient
}

// NewClient constructs a GCP Provider.
func (sm *ProviderGCP) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	// Fetch credential Secret
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := store.GetSpec().Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Name
	objectKey := types.NamespacedName{Name: credentialsSecretName, Namespace: store.GetNamespace()}
	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		log.Panicf(err.Error(), "Failed to get credentials Secret")
		return nil, err
	}

	credentials := credentialsSecret.Data[store.GetSpec().Provider.GCPSM.Auth.SecretRef.SecretAccessKey.Key]
	if len(credentials) == 0 {
		return nil, fmt.Errorf("credentials GCP invalid/not provided")
	}

	projectID := store.GetSpec().Provider.GCPSM.ProjectID

	sm.projectID = projectID

	config, err := google.JWTConfigFromJSON(credentials, cloudPlatformRole)
	if err != nil {
		log.Panicf(err.Error(), "Failed to process the provided JSON credentials")
		return nil, err
	}

	ts := config.TokenSource(ctx)

	client, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP secretmanager client: %w", err)
	}
	sm.SecretManagerClient = client
	return sm, nil
}

// GetSecret returns a single secret from the provider.
func (sm *ProviderGCP) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if sm.SecretManagerClient == nil || sm.projectID == "" {
		return nil, fmt.Errorf("provider GCP is not initialized")
	}

	version := ref.Version
	if version == "" {
		version = defaultVersion
	}

	resourceName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", sm.projectID, ref.Key, version)

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: resourceName,
	}

	result, err := sm.SecretManagerClient.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, err
	}
	err = sm.SecretManagerClient.Close()
	if err != nil {
		return nil, err
	}
	return []byte(string(result.Payload.Data)), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *ProviderGCP) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if sm.SecretManagerClient == nil || sm.projectID == "" {
		return nil, fmt.Errorf("provider GCP is not initialized")
	}
	return map[string][]byte{
		"noop": []byte("NOOP"),
	}, nil
}

func init() {
	schema.Register(&ProviderGCP{}, &esv1alpha1.SecretStoreProvider{
		GCPSM: &esv1alpha1.GCPSMProvider{},
	})
}
