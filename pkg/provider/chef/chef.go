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
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-chef/chef"
	"github.com/go-logr/logr"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errChefStore                             = "received invalid Chef SecretStore resource: %w"
	errMissingStore                          = "missing store"
	errMissingStoreSpec                      = "missing store spec"
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
	errCannotListDataBagItems                = "unable to list items in data bag %s, may be given data bag doesn't exists or it is empty"
	errUnableToConvertToJSON                 = "unable to convert databagItem into JSON"
	errInvalidFormat                         = "invalid key format in data section. Expected value 'databagName/databagItemName'"
	errStoreValidateFailed                   = "unable to validate provided store. Check if username, serverUrl and privateKey are correct"
	errServerURLNoEndSlash                   = "serverurl does not end with slash(/)"
	errInvalidDataform                       = "invalid key format in dataForm section. Expected only 'databagName'"
	errNotImplemented                        = "not implemented"

	ProviderChef             = "Chef"
	CallChefGetDataBagItem   = "GetDataBagItem"
	CallChefListDataBagItems = "ListDataBagItems"
	CallChefGetUser          = "GetUser"
)

var contextTimeout = time.Second * 25

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
	log            logr.Logger
}

var _ v1beta1.SecretsClient = &Providerchef{}
var _ v1beta1.Provider = &Providerchef{}

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

	if err := kube.Get(ctx, objectKey, credentialsSecret); err != nil {
		return nil, fmt.Errorf(errFetchK8sSecret, err)
	}

	secretKey := credentialsSecret.Data[chefProvider.Auth.SecretRef.SecretKey.Key]
	if len(secretKey) == 0 {
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
	providerchef.log = ctrl.Log.WithName("provider").WithName("chef").WithName("secretsmanager")
	return providerchef, nil
}

// Close closes the client connection.
func (providerchef *Providerchef) Close(_ context.Context) error {
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
func (providerchef *Providerchef) GetAllSecrets(_ context.Context, _ v1beta1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf("dataFrom.find not suppported")
}

// GetSecret returns a databagItem present in the databag. format example: databagName/databagItemName.
func (providerchef *Providerchef) GetSecret(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if utils.IsNil(providerchef.databagService) {
		return nil, fmt.Errorf(errUninitalizedChefProvider)
	}

	key := ref.Key
	databagName := ""
	databagItem := ""
	nameSplitted := strings.Split(key, "/")
	if len(nameSplitted) > 1 {
		databagName = nameSplitted[0]
		databagItem = nameSplitted[1]
	}
	providerchef.log.Info("fetching secret value", "databag Name:", databagName, "databag Item:", databagItem)
	if databagName != "" && databagItem != "" {
		return getSingleDatabagItemWithContext(ctx, providerchef, databagName, databagItem, ref.Property)
	}

	return nil, fmt.Errorf(errInvalidFormat)
}

func getSingleDatabagItemWithContext(ctx context.Context, providerchef *Providerchef, dataBagName, databagItemName, propertyName string) ([]byte, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, contextTimeout)
	defer cancel()
	type result = struct {
		values []byte
		err    error
	}
	getWithTimeout := func() chan result {
		resultChan := make(chan result, 1)
		go func() {
			defer close(resultChan)
			ditem, err := providerchef.databagService.GetItem(dataBagName, databagItemName)
			metrics.ObserveAPICall(ProviderChef, CallChefGetDataBagItem, err)
			if err != nil {
				resultChan <- result{err: fmt.Errorf(errNoDatabagItemFound, databagItemName, dataBagName)}
				return
			}
			jsonByte, err := json.Marshal(ditem)
			if err != nil {
				resultChan <- result{err: fmt.Errorf(errUnableToConvertToJSON)}
				return
			}
			if propertyName != "" {
				propertyValue, err := getPropertyFromDatabagItem(jsonByte, propertyName)
				if err != nil {
					resultChan <- result{err: err}
					return
				}
				resultChan <- result{values: propertyValue}
			} else {
				resultChan <- result{values: jsonByte}
			}
		}()
		return resultChan
	}
	select {
	case <-ctxWithTimeout.Done():
		return nil, ctxWithTimeout.Err()
	case r := <-getWithTimeout():
		if r.err != nil {
			return nil, r.err
		}
		return r.values, nil
	}
}

/*
A path is a series of keys separated by a dot.
A key may contain special wildcard characters '*' and '?'.
To access an array value use the index as the key.
To get the number of elements in an array or to access a child path, use the '#' character.
The dot and wildcard characters can be escaped with '\'.

refer https://github.com/tidwall/gjson#:~:text=JSON%20byte%20slices.-,Path%20Syntax,-Below%20is%20a
*/
func getPropertyFromDatabagItem(jsonByte []byte, propertyName string) ([]byte, error) {
	result := gjson.GetBytes(jsonByte, propertyName)

	if !result.Exists() {
		return nil, fmt.Errorf(errNoDatabagItemPropertyFound, propertyName)
	}
	return []byte(result.Str), nil
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.key
// dataFrom.extract.key only accepts dataBagName, example : dataFrom.extract.key: myDatabag
// databagItemName or Property not expected in key.
func (providerchef *Providerchef) GetSecretMap(ctx context.Context, ref v1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if utils.IsNil(providerchef.databagService) {
		return nil, fmt.Errorf(errUninitalizedChefProvider)
	}
	databagName := ref.Key

	if strings.Contains(databagName, "/") {
		return nil, fmt.Errorf(errInvalidDataform)
	}
	getAllSecrets := make(map[string][]byte)
	providerchef.log.Info("fetching all items from", "databag:", databagName)
	dataItems, err := providerchef.databagService.ListItems(databagName)
	metrics.ObserveAPICall(ProviderChef, CallChefListDataBagItems, err)
	if err != nil {
		return nil, fmt.Errorf(errCannotListDataBagItems, databagName)
	}

	for dataItem := range *dataItems {
		dItem, err := getSingleDatabagItemWithContext(ctx, providerchef, databagName, dataItem, "")
		if err != nil {
			return nil, fmt.Errorf(errNoDatabagItemFound, dataItem, databagName)
		}
		getAllSecrets[dataItem] = dItem
	}
	return getAllSecrets, nil
}

// ValidateStore checks if the provided store is valid.
func (providerchef *Providerchef) ValidateStore(store v1beta1.GenericStore) (admission.Warnings, error) {
	chefProvider, err := getChefProvider(store)
	if err != nil {
		return nil, fmt.Errorf(errChefStore, err)
	}
	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, chefProvider.Auth.SecretRef.SecretKey); err != nil {
		return nil, fmt.Errorf(errChefStore, err)
	}
	return nil, nil
}

// getChefProvider validates the incoming store and return the chef provider.
func getChefProvider(store v1beta1.GenericStore) (*v1beta1.ChefProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errMissingStore)
	}
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
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

// Not Implemented DeleteSecret.
func (providerchef *Providerchef) DeleteSecret(_ context.Context, _ v1beta1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented)
}

// Not Implemented PushSecret.
func (providerchef *Providerchef) PushSecret(_ context.Context, _ *corev1.Secret, _ v1beta1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented)
}

func (providerchef *Providerchef) SecretExists(_ context.Context, _ v1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented)
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (providerchef *Providerchef) Capabilities() v1beta1.SecretStoreCapabilities {
	return v1beta1.SecretStoreReadOnly
}
