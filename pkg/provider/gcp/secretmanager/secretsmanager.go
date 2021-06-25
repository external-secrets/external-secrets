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
	"encoding/json"
	"fmt"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

const (
	CloudPlatformRole = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion    = "latest"

	errGCPSMStore                             = "received invalid GCPSM SecretStore resource"
	errGCPSMCredSecretName                    = "invalid GCPSM SecretStore resource: missing GCP Secret Access Key"
	errInvalidClusterStoreMissingSAKNamespace = "invalid ClusterSecretStore: missing GCP SecretAccessKey Namespace"
	errFetchSAKSecret                         = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                             = "missing SecretAccessKey"
	errUnableProcessJSONCredentials           = "failed to process the provided JSON credentials: %v"
	errUnableProcessDefaultCredentials        = "failed to process the default credentials: %w"
	errUnableCreateGCPSMClient                = "failed to create GCP secretmanager client: %w"
	errUninitalizedGCPProvider                = "provider GCP is not initialized"
	errClientGetSecretAccess                  = "unable to access Secret from SecretManager Client: %w"
	errClientClose                            = "unable to close SecretManager client: %w"
	errJSONSecretUnmarshal                    = "unable to unmarshal secret: %w"
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

type gClient struct {
	kube        kclient.Client
	store       *esv1alpha1.GCPSMProvider
	namespace   string
	storeKind   string
	credentials []byte
}

func (c *gClient) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.SecretAccessKey.Name
	/*if credentialsSecretName == "" {
		creds, err := google.FindDefaultCredentials(ctx, CloudPlatformRole)
		if err != nil {
			return fmt.Errorf(errUnableProcessDefaultCredentials, err)
		}
		c.credentials = creds.JSON
	}*/
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.SecretAccessKey.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.SecretAccessKey.Namespace
	}
	if credentialsSecretName == "" {
		c.credentials = nil
		return nil
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchSAKSecret, err)
	}

	c.credentials = credentialsSecret.Data[c.store.Auth.SecretRef.SecretAccessKey.Key]
	if (c.credentials == nil) || (len(c.credentials) == 0) {
		return fmt.Errorf(errMissingSAK)
	}
	return nil
}

// NewClient constructs a GCP Provider.
func (sm *ProviderGCP) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.GCPSM == nil {
		return nil, fmt.Errorf(errGCPSMStore)
	}
	storeSpecGCPSM := storeSpec.Provider.GCPSM

	cliStore := gClient{
		kube:      kube,
		store:     storeSpecGCPSM,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	sm.projectID = cliStore.store.ProjectID

	if err := cliStore.setAuth(ctx); err != nil {
		return nil, err
	}

	if cliStore.credentials != nil {
		config, err := google.JWTConfigFromJSON(cliStore.credentials, CloudPlatformRole)
		if err != nil {
			return nil, fmt.Errorf(errUnableProcessJSONCredentials, cliStore.credentials)
		}
		ts := config.TokenSource(ctx)
		clientGCPSM, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
		if err != nil {
			return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
		}
		sm.SecretManagerClient = clientGCPSM
		return sm, nil
	} else {
		ts, err := google.DefaultTokenSource(ctx, CloudPlatformRole)
		if err != nil {
			return nil, fmt.Errorf(errUnableProcessDefaultCredentials, err)
		}
		clientGCPSM, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
		if err != nil {
			return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
		}
		sm.SecretManagerClient = clientGCPSM
		return sm, nil
	}
}

// GetSecret returns a single secret from the provider.
func (sm *ProviderGCP) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if sm.SecretManagerClient == nil || sm.projectID == "" {
		return nil, fmt.Errorf(errUninitalizedGCPProvider)
	}

	version := ref.Version
	if version == "" {
		version = defaultVersion
	}

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", sm.projectID, ref.Key, version),
	}
	result, err := sm.SecretManagerClient.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf(errClientGetSecretAccess, err)
	}

	err = sm.SecretManagerClient.Close()
	if err != nil {
		return nil, fmt.Errorf(errClientClose, err)
	}

	return result.Payload.Data, nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *ProviderGCP) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if sm.SecretManagerClient == nil || sm.projectID == "" {
		return nil, fmt.Errorf(errUninitalizedGCPProvider)
	}

	data, err := sm.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}

	return secretData, nil
}

func init() {
	schema.Register(&ProviderGCP{}, &esv1alpha1.SecretStoreProvider{
		GCPSM: &esv1alpha1.GCPSMProvider{},
	})
}
