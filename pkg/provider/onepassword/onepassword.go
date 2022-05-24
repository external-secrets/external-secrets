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
package onepassword

import (
	"context"
	"fmt"
	"net/url"
	"sort"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	userAgent = "external-secrets"

	errOnePasswordStore                           = "received invalid 1Password SecretStore resource: %w"
	errOnePasswordStoreNilSpec                    = "nil spec"
	errOnePasswordStoreNilSpecProvider            = "nil spec.provider"
	errOnePasswordStoreNilSpecProviderOnePassword = "nil spec.provider.onepassword"
	errOnePasswordStoreMissingRefName             = "missing: spec.provider.onepassword.auth.secretRef.connectTokenSecretRef.name"
	errOnePasswordStoreMissingRefKey              = "missing: spec.provider.onepassword.auth.secretRef.connectTokenSecretRef.key"
	errOnePasswordStoreAtLeastOneVault            = "must be at least one vault: spec.provider.onepassword.vaults"
	errOnePasswordStoreInvalidConnectHost         = "unable to parse URL: spec.provider.onepassword.connectHost: %w"
	errOnePasswordStoreNonUniqueVaultNumbers      = "vault order numbers must be unique"
	errFetchK8sSecret                             = "could not fetch ConnectToken Secret: %w"
	errMissingToken                               = "missing Secret Token"
	errGetVault                                   = "error finding 1Password Vault: %w"
	errExpectedOneVault                           = "expected one 1Password Vault matching %w"
	errExpectedOneItem                            = "expected one 1Password Item matching %w"
	errGetItem                                    = "error finding 1Password Item: %w"
	errKeyNotFound                                = "key not found in 1Password Vaults: %w"
	errDocumentNotFound                           = "error finding 1Password Document: %w"
	errExpectedOneField                           = "expected one 1Password ItemField matching %w"
	errTagsNotImplemented                         = "'find.tags' is not implemented in the 1Password provider"
	errVersionNotImplemented                      = "'remoteRef.version' is not implemented in the 1Password provider"

	documentCategory      = "DOCUMENT"
	fieldsWithLabelFormat = "'%s' in '%s', got %d"
	incorrectCountFormat  = "'%s', got %d"
)

// ProviderOnePassword is a provider for 1Password.
type ProviderOnePassword struct {
	vaults map[string]int
	client connect.Client
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &ProviderOnePassword{}
var _ esv1beta1.Provider = &ProviderOnePassword{}

// NewClient constructs a 1Password Provider.
func (provider *ProviderOnePassword) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.OnePassword

	credentialsSecret := &corev1.Secret{}
	objectKey := types.NamespacedName{
		Name:      config.Auth.SecretRef.ConnectToken.Name,
		Namespace: namespace,
	}

	// only ClusterSecretStore is allowed to set namespace (and then it's required)
	if store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind {
		objectKey.Namespace = *config.Auth.SecretRef.ConnectToken.Namespace
	}

	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchK8sSecret, err)
	}
	token := credentialsSecret.Data[config.Auth.SecretRef.ConnectToken.Key]
	if (token == nil) || (len(token) == 0) {
		return nil, fmt.Errorf(errMissingToken)
	}
	provider.client = connect.NewClientWithUserAgent(config.ConnectHost, string(token), userAgent)
	provider.vaults = config.Vaults

	return provider, nil
}

// ValidateStore checks if the provided store is valid.
func (provider *ProviderOnePassword) ValidateStore(store esv1beta1.GenericStore) error {
	return validateStore(store)
}

func validateStore(store esv1beta1.GenericStore) error {
	// check nils
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNilSpec))
	}
	if storeSpec.Provider == nil {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNilSpecProvider))
	}
	if storeSpec.Provider.OnePassword == nil {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNilSpecProviderOnePassword))
	}

	// check mandatory fields
	config := storeSpec.Provider.OnePassword
	if config.Auth.SecretRef.ConnectToken.Name == "" {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreMissingRefName))
	}
	if config.Auth.SecretRef.ConnectToken.Key == "" {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreMissingRefKey))
	}

	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, config.Auth.SecretRef.ConnectToken); err != nil {
		return fmt.Errorf(errOnePasswordStore, err)
	}

	// check at least one vault
	if len(config.Vaults) == 0 {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreAtLeastOneVault))
	}

	// ensure vault numbers are unique
	if !hasUniqueVaultNumbers(config.Vaults) {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreNonUniqueVaultNumbers))
	}

	// check valid URL
	if _, err := url.Parse(config.ConnectHost); err != nil {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreInvalidConnectHost, err))
	}

	return nil
}

// GetSecret returns a single secret from the provider.
func (provider *ProviderOnePassword) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, esv1beta1.SecretsMetadata, error) {
	if ref.Version != "" {
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf(errVersionNotImplemented)
	}

	item, err := provider.findItem(ref.Key)
	if err != nil {
		return nil, esv1beta1.SecretsMetadata{}, err
	}

	// handle files
	if item.Category == documentCategory {
		// default to the first file when ref.Property is empty
		file, err := provider.getFile(item, ref.Property)
		return file, esv1beta1.SecretsMetadata{}, err
	}

	// handle fields
	field, err := provider.getField(item, ref.Property)
	return field, esv1beta1.SecretsMetadata{}, err
}

// Validate checks if the client is configured correctly
// to be able to retrieve secrets from the provider.
func (provider *ProviderOnePassword) Validate() (esv1beta1.ValidationResult, error) {
	for vaultName := range provider.vaults {
		_, err := provider.client.GetItems(vaultName)
		if err != nil {
			return esv1beta1.ValidationResultError, err
		}
	}

	return esv1beta1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (provider *ProviderOnePassword) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, esv1beta1.SecretsMetadata, error) {
	if ref.Version != "" {
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf(errVersionNotImplemented)
	}

	item, err := provider.findItem(ref.Key)
	if err != nil {
		return nil, esv1beta1.SecretsMetadata{}, err
	}

	// handle files
	if item.Category == documentCategory {
		// default to the first file when ref.Property is empty
		files, err := provider.getFiles(item, ref.Property)
		return files, esv1beta1.SecretsMetadata{}, err
	}

	// handle fields
	fields, err := provider.getFields(item, ref.Property)
	return fields, esv1beta1.SecretsMetadata{}, err
}

// GetAllSecrets syncs multiple 1Password Items into a single Kubernetes Secret, for dataFrom.find.
func (provider *ProviderOnePassword) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, esv1beta1.SecretsMetadata, error) {
	if ref.Tags != nil {
		return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf(errTagsNotImplemented)
	}

	secretData := make(map[string][]byte)
	sortedVaults := sortVaults(provider.vaults)
	for _, vaultName := range sortedVaults {
		vaults, err := provider.client.GetVaultsByTitle(vaultName)
		if err != nil {
			return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf(errGetVault, err)
		}
		if len(vaults) != 1 {
			return nil, esv1beta1.SecretsMetadata{}, fmt.Errorf(errExpectedOneVault, fmt.Errorf(incorrectCountFormat, vaultName, len(vaults)))
		}

		err = provider.getAllForVault(vaults[0].ID, ref, secretData)
		if err != nil {
			return nil, esv1beta1.SecretsMetadata{}, err
		}
	}

	return secretData, esv1beta1.SecretsMetadata{}, nil
}

// Close closes the client connection.
func (provider *ProviderOnePassword) Close(ctx context.Context) error {
	return nil
}

func (provider *ProviderOnePassword) findItem(name string) (*onepassword.Item, error) {
	sortedVaults := sortVaults(provider.vaults)
	for _, vaultName := range sortedVaults {
		vaults, err := provider.client.GetVaultsByTitle(vaultName)
		if err != nil {
			return nil, fmt.Errorf(errGetVault, err)
		}
		if len(vaults) != 1 {
			return nil, fmt.Errorf(errExpectedOneVault, fmt.Errorf(incorrectCountFormat, vaultName, len(vaults)))
		}

		// use GetItemsByTitle instead of GetItemByTitle in order to handle length cases
		items, err := provider.client.GetItemsByTitle(name, vaults[0].ID)
		if err != nil {
			return nil, fmt.Errorf(errGetItem, err)
		}
		switch {
		case len(items) == 1:
			return provider.client.GetItem(items[0].ID, items[0].Vault.ID)
		case len(items) > 1:
			return nil, fmt.Errorf(errExpectedOneItem, fmt.Errorf(incorrectCountFormat, name, len(items)))
		}
	}

	return nil, fmt.Errorf(errKeyNotFound, fmt.Errorf("%s in: %v", name, provider.vaults))
}

func (provider *ProviderOnePassword) getField(item *onepassword.Item, property string) ([]byte, error) {
	// default to a field labeled "password"
	fieldLabel := "password"
	if property != "" {
		fieldLabel = property
	}

	if length := countFieldsWithLabel(fieldLabel, item.Fields); length != 1 {
		return nil, fmt.Errorf(errExpectedOneField, fmt.Errorf(fieldsWithLabelFormat, fieldLabel, item.Title, length))
	}

	// caution: do not use client.GetValue here because it has undesirable behavior on keys with a dot in them
	value := ""
	for _, field := range item.Fields {
		if field.Label == fieldLabel {
			value = field.Value
			break
		}
	}

	return []byte(value), nil
}

func (provider *ProviderOnePassword) getFields(item *onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, field := range item.Fields {
		if property != "" && field.Label != property {
			continue
		}
		if length := countFieldsWithLabel(field.Label, item.Fields); length != 1 {
			return nil, fmt.Errorf(errExpectedOneField, fmt.Errorf(fieldsWithLabelFormat, field.Label, item.Title, length))
		}

		// caution: do not use client.GetValue here because it has undesirable behavior on keys with a dot in them
		secretData[field.Label] = []byte(field.Value)
	}

	return secretData, nil
}

func (provider *ProviderOnePassword) getAllFields(item onepassword.Item, ref esv1beta1.ExternalSecretFind, secretData map[string][]byte) error {
	i, err := provider.client.GetItem(item.ID, item.Vault.ID)
	if err != nil {
		return fmt.Errorf(errGetItem, err)
	}
	item = *i
	for _, field := range item.Fields {
		if length := countFieldsWithLabel(field.Label, item.Fields); length != 1 {
			return fmt.Errorf(errExpectedOneField, fmt.Errorf(fieldsWithLabelFormat, field.Label, item.Title, length))
		}
		if ref.Name != nil {
			matcher, err := find.New(*ref.Name)
			if err != nil {
				return err
			}
			if !matcher.MatchName(field.Label) {
				continue
			}
		}
		if _, ok := secretData[field.Label]; !ok {
			secretData[field.Label] = []byte(field.Value)
		}
	}

	return nil
}

func (provider *ProviderOnePassword) getFile(item *onepassword.Item, property string) ([]byte, error) {
	for _, file := range item.Files {
		// default to the first file when ref.Property is empty
		if file.Name == property || property == "" {
			contents, err := provider.client.GetFileContent(file)
			if err != nil {
				return nil, err
			}

			return contents, nil
		}
	}

	return nil, fmt.Errorf(errDocumentNotFound, fmt.Errorf("'%s', '%s'", item.Title, property))
}

func (provider *ProviderOnePassword) getFiles(item *onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, file := range item.Files {
		if property != "" && file.Name != property {
			continue
		}
		contents, err := provider.client.GetFileContent(file)
		if err != nil {
			return nil, err
		}
		secretData[file.Name] = contents
	}

	return secretData, nil
}

func (provider *ProviderOnePassword) getAllFiles(item onepassword.Item, ref esv1beta1.ExternalSecretFind, secretData map[string][]byte) error {
	for _, file := range item.Files {
		if ref.Name != nil {
			matcher, err := find.New(*ref.Name)
			if err != nil {
				return err
			}
			if !matcher.MatchName(file.Name) {
				continue
			}
		}
		if _, ok := secretData[file.Name]; !ok {
			contents, err := provider.client.GetFileContent(file)
			if err != nil {
				return err
			}
			secretData[file.Name] = contents
		}
	}

	return nil
}

func (provider *ProviderOnePassword) getAllForVault(vaultID string, ref esv1beta1.ExternalSecretFind, secretData map[string][]byte) error {
	items, err := provider.client.GetItems(vaultID)
	if err != nil {
		return fmt.Errorf(errGetItem, err)
	}
	for _, item := range items {
		if ref.Path != nil && *ref.Path != item.Title {
			continue
		}

		// handle files
		if item.Category == documentCategory {
			err = provider.getAllFiles(item, ref, secretData)
			if err != nil {
				return err
			}

			continue
		}

		// handle fields
		err = provider.getAllFields(item, ref, secretData)
		if err != nil {
			return err
		}
	}

	return nil
}

func countFieldsWithLabel(fieldLabel string, fields []*onepassword.ItemField) int {
	count := 0
	for _, field := range fields {
		if field.Label == fieldLabel {
			count++
		}
	}

	return count
}

type orderedVault struct {
	Name  string
	Order int
}

type orderedVaultList []orderedVault

func (list orderedVaultList) Len() int           { return len(list) }
func (list orderedVaultList) Swap(i, j int)      { list[i], list[j] = list[j], list[i] }
func (list orderedVaultList) Less(i, j int) bool { return list[i].Order < list[j].Order }

func sortVaults(vaults map[string]int) []string {
	list := make(orderedVaultList, len(vaults))
	index := 0
	for key, value := range vaults {
		list[index] = orderedVault{key, value}
		index++
	}
	sort.Sort(list)
	sortedVaults := []string{}
	for _, item := range list {
		sortedVaults = append(sortedVaults, item.Name)
	}

	return sortedVaults
}

func hasUniqueVaultNumbers(vaults map[string]int) bool {
	unique := make([]int, 0, len(vaults))
	tracker := make(map[int]bool)

	for _, number := range vaults {
		if _, ok := tracker[number]; !ok {
			tracker[number] = true
			unique = append(unique, number)
		}
	}

	return len(vaults) == len(unique)
}

func init() {
	esv1beta1.Register(&ProviderOnePassword{}, &esv1beta1.SecretStoreProvider{
		OnePassword: &esv1beta1.OnePasswordProvider{},
	})
}
