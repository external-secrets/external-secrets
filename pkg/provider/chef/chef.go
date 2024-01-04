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
package chef

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-chef/chef"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/metrics"
)

const (
	errChefStore                             = "received invalid Chef SecretStore resource: %w"
	errMissingStore                          = "missing store"
	errMisingStoreSpec                       = "missing store spec"
	errMissingProvider                       = "missing provider"
	errMissingChefProvider                   = "missing chef provider"
	errMissingUserName                       = "missing username"
	errMissingServerURL                      = "missing serverurl"
	errMissingAuth                           = "cannot initialize Chef Client: no valid authType was specified"
	errMissingSecretKey                      = "missing Secret Key"
	errInvalidClusterStoreMissingPKNamespace = "invalid ClusterSecretStore: missing privateKeySecretRef.Namespace"
	errFetchK8sSecret                        = "could not fetch SecretKey Secret: %w"
	errInvalidURL                            = "invalid serverurl: %w"
	errChefClient                            = "unable to create chef client: %w"
	errChefProvider                          = "missing or invalid spec: %w"
	errUninitalizedChefProvider              = "chef provider is not initialized"
	errNoDatabagItemFound                    = "data bag item %s not found in data bag %s"
	errNoDatabagItemPropertyFound            = "property %s not found in data bag item"
	errCannotListDataBagItems                = "unable to list items in data bag %s"
	errUnableToConvertToJSON                 = "unable to convert databagItem into JSON"
	errInvalidFormat                         = "invalid key format in data section. Expected value 'databagName/databagItemName'"
	errStoreValidateFailed                   = "unable to validate provided store. Check if username, serverUrl and privateKey are correct"
	errServerURLNoEndSlash                   = "serverurl does not end with slash(/)"
	errInvalidDataform                       = "invalid key format in dataForm section. Expected only 'databagName'"

	ProviderChef             = "Chef"
	CallChefGetDataBagItem   = "GetDataBagItem"
	CallChefListDataBagItems = "ListDataBagItems"
	CallChefGetUser          = "GetUser"
)

type DatabagFetcher interface {
	GetItem(databagName string, databagItem string) (item chef.DataBagItem, err error)
	ListItems(name string) (data *chef.DataBagListResult, err error)
}

type UserInterface interface {
	Get(name string) (user chef.User, err error)
}

type Providerchef struct {
	clientName     string
	databagService DatabagFetcher
	userService    UserInterface
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ v1beta1.SecretsClient = &Providerchef{}
var _ v1beta1.Provider = &Providerchef{}

var log = ctrl.Log.WithName("provider").WithName("chef").WithName("secretsmanager")

func init() {
	v1beta1.Register(&Providerchef{}, &v1beta1.SecretStoreProvider{
		Chef: &v1beta1.ChefProvider{},
	})
}

func (providerchef *Providerchef) NewClient(ctx context.Context, store v1beta1.GenericStore, kube kclient.Client, namespace string) (v1beta1.SecretsClient, error) {
	chefProvider, err := getChefProvider(store)
	if err != nil {
		return nil, fmt.Errorf(errChefProvider, err)
	}

	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      chefProvider.Auth.SecretRef.SecretKey.Name,
		Namespace: namespace,
	}

	if store.GetObjectKind().GroupVersionKind().Kind == v1beta1.ClusterSecretStoreKind {
		if chefProvider.Auth.SecretRef.SecretKey.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingPKNamespace)
		}
		objectKey.Namespace = *chefProvider.Auth.SecretRef.SecretKey.Namespace
	}

	err = kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchK8sSecret, err)
	}

	secretKey := credentialsSecret.Data[chefProvider.Auth.SecretRef.SecretKey.Key]
	if (secretKey == nil) || (len(secretKey) == 0) {
		return nil, fmt.Errorf(errMissingSecretKey)
	}

	client, err := chef.NewClient(&chef.Config{
		Name:    chefProvider.UserName,
		Key:     string(secretKey),
		BaseURL: chefProvider.ServerURL,
	})
	if err != nil {
		return nil, fmt.Errorf(errChefClient, err)
	}

	providerchef.clientName = chefProvider.UserName
	providerchef.databagService = client.DataBags
	providerchef.userService = client.Users
	return providerchef, nil
}

// getChefProvider validates the incoming store and return the chef provider.
func getChefProvider(store v1beta1.GenericStore) (*v1beta1.ChefProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errMissingStore)
	}
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errMisingStoreSpec)
	}
	provider := storeSpec.Provider
	if provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	chefProvider := storeSpec.Provider.Chef
	if chefProvider == nil {
		return nil, fmt.Errorf(errMissingChefProvider)
	}
	if chefProvider.UserName == "" {
		return chefProvider, fmt.Errorf(errMissingUserName)
	}
	if chefProvider.ServerURL == "" {
		return chefProvider, fmt.Errorf(errMissingServerURL)
	}
	if !strings.HasSuffix(chefProvider.ServerURL, "/") {
		return chefProvider, fmt.Errorf(errServerURLNoEndSlash)
	}
	// check valid URL
	if _, err := url.ParseRequestURI(chefProvider.ServerURL); err != nil {
		return chefProvider, fmt.Errorf(errInvalidURL, err)
	}
	if chefProvider.Auth == nil {
		return chefProvider, fmt.Errorf(errMissingAuth)
	}
	if chefProvider.Auth.SecretRef.SecretKey.Key == "" {
		return chefProvider, fmt.Errorf(errMissingSecretKey)
	}

	return chefProvider, nil
}

// Close closes the client connection.
func (providerchef *Providerchef) Close(ctx context.Context) error {
	return nil
}

// Validate checks if the client is configured correctly
// to be able to retrieve secrets from the provider.
func (providerchef *Providerchef) Validate() (v1beta1.ValidationResult, error) {
	_, err := providerchef.userService.Get(providerchef.clientName)
	metrics.ObserveAPICall(ProviderChef, CallChefGetUser, err)
	if err != nil {
		return v1beta1.ValidationResultError, fmt.Errorf(errStoreValidateFailed)
	}
	return v1beta1.ValidationResultReady, nil
}

// GetAllSecrets Retrieves a map[string][]byte with the Databag names as key and the Databag's Items as secrets.
// Retrives all DatabagItems of a Databag.
func (providerchef *Providerchef) GetAllSecrets(ctx context.Context, ref v1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("dataFrom.find not suppported")
}

// Not Implemented GetSecret
func (providerchef *Providerchef) GetSecret(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return nil, fmt.Errorf(errInvalidFormat)
}

// Not Implemented GetSecretMap.
func (providerchef *Providerchef) GetSecretMap(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Not Implemented ValidateStore.
func (providerchef *Providerchef) ValidateStore(store v1beta1.GenericStore) error {
	return fmt.Errorf("not implemented")
}

// Not Implemented DeleteSecret.
func (providerchef *Providerchef) DeleteSecret(_ context.Context, _ v1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf("not implemented")
}

// Not Implemented PushSecret.
func (providerchef *Providerchef) PushSecret(_ context.Context, _ *corev1.Secret, _ v1beta1.PushSecretData) error {
	return fmt.Errorf("not implemented")
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (providerchef *Providerchef) Capabilities() v1beta1.SecretStoreCapabilities {
	return v1beta1.SecretStoreReadOnly
}
