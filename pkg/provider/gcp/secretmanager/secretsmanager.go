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
	"github.com/googleapis/gax-go/v2"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	CloudPlatformRole = "https://www.googleapis.com/auth/cloud-platform"
	defaultVersion    = "latest"

	errGCPSMStore                             = "received invalid GCPSM SecretStore resource"
	errUnableGetCredentials                   = "unable to get credentials: %w"
	errClientClose                            = "unable to close SecretManager client: %w"
	errMissingStoreSpec                       = "invalid: missing store spec"
	errInvalidClusterStoreMissingSAKNamespace = "invalid ClusterSecretStore: missing GCP SecretAccessKey Namespace"
	errInvalidClusterStoreMissingSANamespace  = "invalid ClusterSecretStore: missing GCP Service Account Namespace"
	errFetchSAKSecret                         = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                             = "missing SecretAccessKey"
	errUnableProcessJSONCredentials           = "failed to process the provided JSON credentials: %w"
	errUnableCreateGCPSMClient                = "failed to create GCP secretmanager client: %w"
	errUninitalizedGCPProvider                = "provider GCP is not initialized"
	errClientGetSecretAccess                  = "unable to access Secret from SecretManager Client: %w"
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
	gClient             *gClient
}

type gClient struct {
	kube             kclient.Client
	store            *esv1beta1.GCPSMProvider
	namespace        string
	storeKind        string
	workloadIdentity *workloadIdentity
}

func (c *gClient) getTokenSource(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	ts, err := serviceAccountTokenSource(ctx, store, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}
	ts, err = c.workloadIdentity.TokenSource(ctx, store, kube, namespace)
	if ts != nil || err != nil {
		return ts, err
	}

	return google.DefaultTokenSource(ctx, CloudPlatformRole)
}

func (c *gClient) Close() error {
	return c.workloadIdentity.Close()
}

func serviceAccountTokenSource(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider.GCPSM == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}
	sr := spec.Provider.GCPSM.Auth.SecretRef
	if sr == nil {
		return nil, nil
	}
	storeKind := store.GetObjectKind().GroupVersionKind().Kind
	credentialsSecret := &v1.Secret{}
	credentialsSecretName := sr.SecretAccessKey.Name
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if storeKind == esv1beta1.ClusterSecretStoreKind {
		if credentialsSecretName != "" && sr.SecretAccessKey.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		} else if credentialsSecretName != "" {
			objectKey.Namespace = *sr.SecretAccessKey.Namespace
		}
	}
	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchSAKSecret, err)
	}
	credentials := credentialsSecret.Data[sr.SecretAccessKey.Key]
	if (credentials == nil) || (len(credentials) == 0) {
		return nil, fmt.Errorf(errMissingSAK)
	}
	config, err := google.JWTConfigFromJSON(credentials, CloudPlatformRole)
	if err != nil {
		return nil, fmt.Errorf(errUnableProcessJSONCredentials, err)
	}
	return config.TokenSource(ctx), nil
}

// NewClient constructs a GCP Provider.
func (sm *ProviderGCP) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.GCPSM == nil {
		return nil, fmt.Errorf(errGCPSMStore)
	}
	storeSpecGCPSM := storeSpec.Provider.GCPSM

	wi, err := newWorkloadIdentity(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to initialize workload identity")
	}

	cliStore := gClient{
		kube:             kube,
		store:            storeSpecGCPSM,
		namespace:        namespace,
		storeKind:        store.GetObjectKind().GroupVersionKind().Kind,
		workloadIdentity: wi,
	}
	sm.gClient = &cliStore
	defer func() {
		// closes IAMClient to prevent gRPC connection leak in case of an error.
		if sm.SecretManagerClient == nil {
			_ = sm.gClient.Close()
		}
	}()

	sm.projectID = cliStore.store.ProjectID

	ts, err := cliStore.getTokenSource(ctx, store, kube, namespace)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}

	// check if we can get credentials
	_, err = ts.Token()
	if err != nil {
		return nil, fmt.Errorf(errUnableGetCredentials, err)
	}

	clientGCPSM, err := secretmanager.NewClient(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateGCPSMClient, err)
	}
	sm.SecretManagerClient = clientGCPSM
	return sm, nil
}

// Empty GetAllSecrets.
func (sm *ProviderGCP) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (sm *ProviderGCP) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(sm.SecretManagerClient) || sm.projectID == "" {
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

	if ref.Property == "" {
		if result.Payload.Data != nil {
			return result.Payload.Data, nil
		}
		return nil, fmt.Errorf("invalid secret received. no secret string for key: %s", ref.Key)
	}

	var payload string
	if result.Payload.Data != nil {
		payload = string(result.Payload.Data)
	}

	val := gjson.Get(payload, ref.Property)
	if !val.Exists() {
		return nil, fmt.Errorf("key %s does not exist in secret %s", ref.Property, ref.Key)
	}
	return []byte(val.String()), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *ProviderGCP) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if sm.SecretManagerClient == nil || sm.projectID == "" {
		return nil, fmt.Errorf(errUninitalizedGCPProvider)
	}

	data, err := sm.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}

	return secretData, nil
}

func (sm *ProviderGCP) Close(ctx context.Context) error {
	err := sm.SecretManagerClient.Close()
	if sm.gClient != nil {
		err = sm.gClient.Close()
	}
	if err != nil {
		return fmt.Errorf(errClientClose, err)
	}
	return nil
}

func (sm *ProviderGCP) Validate() error {
	return nil
}

func init() {
	schema.Register(&ProviderGCP{}, &esv1beta1.SecretStoreProvider{
		GCPSM: &esv1beta1.GCPSMProvider{},
	})
}
