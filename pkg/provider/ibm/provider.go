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
package ibm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/IBM/go-sdk-core/v5/core"
	sm "github.com/IBM/secrets-manager-go-sdk/secretsmanagerv1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	SecretsManagerEndpointEnv = "IBM_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "IBM_STS_ENDPOINT"
	SSMEndpointEnv            = "IBM_SSM_ENDPOINT"

	errIBMClient                             = "cannot setup new ibm client: %w"
	errIBMCredSecretName                     = "invalid IBM SecretStore resource: missing IBM APIKey"
	errUninitalizedIBMProvider               = "provider IBM is not initialized"
	errInvalidClusterStoreMissingSKNamespace = "invalid ClusterStore, missing namespace"
	errFetchSAKSecret                        = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                            = "missing SecretAccessKey"
	errJSONSecretUnmarshal                   = "unable to unmarshal secret: %w"
)

type SecretManagerClient interface {
	GetSecret(getSecretOptions *sm.GetSecretOptions) (result *sm.GetSecret, response *core.DetailedResponse, err error)
}

type providerIBM struct {
	IBMClient SecretManagerClient
}

type client struct {
	kube        kclient.Client
	store       *esv1alpha1.IBMProvider
	namespace   string
	storeKind   string
	credentials []byte
}

func (c *client) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.SecretAPIKey.Name
	if credentialsSecretName == "" {
		return fmt.Errorf(errIBMCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1alpha1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.SecretAPIKey.Namespace == nil {
			return fmt.Errorf(errInvalidClusterStoreMissingSKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.SecretAPIKey.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchSAKSecret, err)
	}

	c.credentials = credentialsSecret.Data[c.store.Auth.SecretRef.SecretAPIKey.Key]
	if (c.credentials == nil) || (len(c.credentials) == 0) {
		return fmt.Errorf(errMissingSAK)
	}
	return nil
}

func (ibm *providerIBM) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, fmt.Errorf(errUninitalizedIBMProvider)
	}
	response, _, err := ibm.IBMClient.GetSecret(
		&sm.GetSecretOptions{
			SecretType: core.StringPtr(sm.GetSecretOptionsSecretTypeArbitraryConst),
			ID:         &ref.Key,
		})

	if err != nil {
		return nil, err
	}

	secret := response.Resources[0].(*sm.SecretResource)
	secretData := secret.SecretData.(map[string]interface{})
	arbitrarySecretPayload := secretData["payload"].(string)
	return []byte(arbitrarySecretPayload), nil
}

func (ibm *providerIBM) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(ibm.IBMClient) {
		return nil, fmt.Errorf(errUninitalizedIBMProvider)
	}
	response, _, err := ibm.IBMClient.GetSecret(
		&sm.GetSecretOptions{
			SecretType: core.StringPtr(sm.GetSecretOptionsSecretTypeArbitraryConst),
			ID:         &ref.Key,
		})
	if err != nil {
		return nil, err
	}

	secret := response.Resources[0].(*sm.SecretResource)
	secretData := secret.SecretData.(map[string]interface{})
	arbitrarySecretPayload := secretData["payload"].(string)

	kv := make(map[string]string)
	err = json.Unmarshal([]byte(arbitrarySecretPayload), &kv)
	if err != nil {
		return nil, fmt.Errorf(errJSONSecretUnmarshal, err)
	}

	secretMap := make(map[string][]byte)
	for k, v := range kv {
		secretMap[k] = []byte(v)
	}

	return secretMap, nil
}

func (ibm *providerIBM) Close() error {
	return nil
}

func (ibm *providerIBM) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube kclient.Client, namespace string) (provider.SecretsClient, error) {
	storeSpec := store.GetSpec()
	ibmSpec := storeSpec.Provider.IBM

	iStore := &client{
		kube:      kube,
		store:     ibmSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if err := iStore.setAuth(ctx); err != nil {
		return nil, err
	}

	secretsManager, err := sm.NewSecretsManagerV1(&sm.SecretsManagerV1Options{
		URL: *storeSpec.Provider.IBM.ServiceURL,
		Authenticator: &core.IamAuthenticator{
			ApiKey: string(iStore.credentials),
		},
	})

	if err != nil {
		return nil, fmt.Errorf(errIBMClient, err)
	}

	ibm.IBMClient = secretsManager
	return ibm, nil
}

func init() {
	schema.Register(&providerIBM{}, &esv1alpha1.SecretStoreProvider{
		IBM: &esv1alpha1.IBMProvider{},
	})
}
