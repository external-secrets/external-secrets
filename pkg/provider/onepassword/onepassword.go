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
	"errors"
	"fmt"
	"net/url"
	"sort"
	"time"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
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
	errGetVault                                   = "error finding 1Password Vault: %w"

	errGetItem               = "error finding 1Password Item: %w"
	errUpdateItem            = "error updating 1Password Item: %w"
	errDocumentNotFound      = "error finding 1Password Document: %w"
	errTagsNotImplemented    = "'find.tags' is not implemented in the 1Password provider"
	errVersionNotImplemented = "'remoteRef.version' is not implemented in the 1Password provider"
	errCreateItem            = "error creating 1Password Item: %w"
	errDeleteItem            = "error deleting 1Password Item: %w"
	// custom error messages.
	errKeyNotFoundMsg       = "key not found in 1Password Vaults"
	errNoVaultsMsg          = "no vaults found"
	errExpectedOneItemMsg   = "expected one 1Password Item matching"
	errExpectedOneFieldMsg  = "expected one 1Password ItemField matching"
	errExpectedOneFieldMsgF = "%w: '%s' in '%s', got %d"

	documentCategory = "DOCUMENT"
)

// Custom Errors //.
var (
	// ErrKeyNotFound is returned when a key is not found in the 1Password Vaults.
	ErrKeyNotFound = errors.New(errKeyNotFoundMsg)
	// ErrNoVaults is returned when no vaults are found in the 1Password provider.
	ErrNoVaults = errors.New(errNoVaultsMsg)
	// ErrExpectedOneField is returned when more than 1 field is found in the 1Password Vaults.
	ErrExpectedOneField = errors.New(errExpectedOneFieldMsg)
	// ErrExpectedOneItem is returned when more than 1 item is found in the 1Password Vaults.
	ErrExpectedOneItem = errors.New(errExpectedOneItemMsg)
)

// ProviderOnePassword is a provider for 1Password.
type ProviderOnePassword struct {
	vaults map[string]int
	client connect.Client
}

// https://github.com/external-secrets/external-secrets/issues/644
var (
	_ esv1beta1.SecretsClient = &ProviderOnePassword{}
	_ esv1beta1.Provider      = &ProviderOnePassword{}
)

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (provider *ProviderOnePassword) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (provider *ProviderOnePassword) ApplyReferent(spec kclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kclient.Object, error) {
	return spec, nil
}

func (provider *ProviderOnePassword) NewClientFromObj(_ context.Context, _ kclient.Object, _ kclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

func (provider *ProviderOnePassword) Convert(_ esv1beta1.GenericStore) (kclient.Object, error) {
	return nil, nil
}

// NewClient constructs a 1Password Provider.
func (provider *ProviderOnePassword) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.OnePassword
	token, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&config.Auth.SecretRef.ConnectToken,
	)
	if err != nil {
		return nil, err
	}
	provider.client = connect.NewClientWithUserAgent(config.ConnectHost, token, userAgent)
	provider.vaults = config.Vaults
	return provider, nil
}

// ValidateStore checks if the provided store is valid.
func (provider *ProviderOnePassword) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, validateStore(store)
}

func validateStore(store esv1beta1.GenericStore) error {
	// check nils
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreNilSpec))
	}
	if storeSpec.Provider == nil {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreNilSpecProvider))
	}
	if storeSpec.Provider.OnePassword == nil {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreNilSpecProviderOnePassword))
	}

	// check mandatory fields
	config := storeSpec.Provider.OnePassword
	if config.Auth.SecretRef.ConnectToken.Name == "" {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreMissingRefName))
	}
	if config.Auth.SecretRef.ConnectToken.Key == "" {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreMissingRefKey))
	}

	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, config.Auth.SecretRef.ConnectToken); err != nil {
		return fmt.Errorf(errOnePasswordStore, err)
	}

	// check at least one vault
	if len(config.Vaults) == 0 {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreAtLeastOneVault))
	}

	// ensure vault numbers are unique
	if !hasUniqueVaultNumbers(config.Vaults) {
		return fmt.Errorf(errOnePasswordStore, errors.New(errOnePasswordStoreNonUniqueVaultNumbers))
	}

	// check valid URL
	if _, err := url.Parse(config.ConnectHost); err != nil {
		return fmt.Errorf(errOnePasswordStore, fmt.Errorf(errOnePasswordStoreInvalidConnectHost, err))
	}

	return nil
}

func deleteField(fields []*onepassword.ItemField, label string) ([]*onepassword.ItemField, error) {
	// This will always iterate over all items
	// but its done to ensure that two fields with the same label
	// exist resulting in undefined behavior
	var (
		found   bool
		fieldsF = make([]*onepassword.ItemField, 0, len(fields))
	)
	for _, item := range fields {
		if item.Label == label {
			if found {
				return nil, ErrExpectedOneField
			}
			found = true
			continue
		}
		fieldsF = append(fieldsF, item)
	}
	return fieldsF, nil
}

func (provider *ProviderOnePassword) DeleteSecret(_ context.Context, ref esv1beta1.PushSecretRemoteRef) error {
	providerItem, err := provider.findItem(ref.GetRemoteKey())
	if err != nil {
		return err
	}

	providerItem.Fields, err = deleteField(providerItem.Fields, ref.GetProperty())
	if err != nil {
		return fmt.Errorf(errUpdateItem, err)
	}

	if len(providerItem.Fields) == 0 && len(providerItem.Files) == 0 && len(providerItem.Sections) == 0 {
		// Delete the item if there are no fields, files or sections
		if err = provider.client.DeleteItem(providerItem, providerItem.Vault.ID); err != nil {
			return fmt.Errorf(errDeleteItem, err)
		}
		return nil
	}

	if _, err = provider.client.UpdateItem(providerItem, providerItem.Vault.ID); err != nil {
		return fmt.Errorf(errDeleteItem, err)
	}
	return nil
}

func (provider *ProviderOnePassword) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

const (
	passwordLabel = "password"
)

// createItem creates a new item in the first vault. If no vaults exist, it returns an error.
func (provider *ProviderOnePassword) createItem(val []byte, ref esv1beta1.PushSecretData) error {
	// Get the first vault
	sortedVaults := sortVaults(provider.vaults)
	if len(sortedVaults) == 0 {
		return ErrNoVaults
	}
	vaultID := sortedVaults[0]
	// Get the label
	label := ref.GetProperty()
	if label == "" {
		label = passwordLabel
	}

	// Create the item
	item := &onepassword.Item{
		Title:    ref.GetRemoteKey(),
		Category: onepassword.Server,
		Vault: onepassword.ItemVault{
			ID: vaultID,
		},
		Fields: []*onepassword.ItemField{
			generateNewItemField(label, string(val)),
		},
	}

	_, err := provider.client.CreateItem(item, vaultID)
	return err
}

// updateFieldValue updates the fields value of an item with the given label.
// If the label does not exist, a new field is created. If the label exists but
// the value is different, the value is updated. If the label exists and the
// value is the same, nothing is done.
func updateFieldValue(fields []*onepassword.ItemField, label, newVal string) ([]*onepassword.ItemField, error) {
	// This will always iterate over all items.
	// This is done to ensure that two fields with the same label
	// exist resulting in undefined behavior.
	var (
		found bool
		index int
	)
	for i, item := range fields {
		if item.Label == label {
			if found {
				return nil, ErrExpectedOneField
			}
			found = true
			index = i
		}
	}
	if !found {
		return append(fields, generateNewItemField(label, newVal)), nil
	}

	if fields[index].Value != newVal {
		fields[index].Value = newVal
	}

	return fields, nil
}

// generateNewItemField generates a new item field with the given label and value.
func generateNewItemField(label, newVal string) *onepassword.ItemField {
	field := &onepassword.ItemField{
		Label: label,
		Value: newVal,
		Type:  onepassword.FieldTypeConcealed,
	}

	return field
}

func (provider *ProviderOnePassword) PushSecret(ctx context.Context, secret *corev1.Secret, ref esv1beta1.PushSecretData) error {
	val, ok := secret.Data[ref.GetSecretKey()]
	if !ok {
		return ErrKeyNotFound
	}

	title := ref.GetRemoteKey()
	providerItem, err := provider.findItem(title)
	if errors.Is(err, ErrKeyNotFound) {
		if err = provider.createItem(val, ref); err != nil {
			return fmt.Errorf(errCreateItem, err)
		}

		err = provider.waitForFunc(ctx, provider.waitForItemToExist(title))
		return err
	} else if err != nil {
		return err
	}

	label := ref.GetProperty()
	if label == "" {
		label = passwordLabel
	}

	providerItem.Fields, err = updateFieldValue(providerItem.Fields, label, string(val))
	if err != nil {
		return fmt.Errorf(errUpdateItem, err)
	}

	if _, err = provider.client.UpdateItem(providerItem, providerItem.Vault.ID); err != nil {
		return fmt.Errorf(errUpdateItem, err)
	}

	if err := provider.waitForFunc(ctx, provider.waitForLabelToBeUpdated(title, label, val)); err != nil {
		return fmt.Errorf("failed waiting for label update: %w", err)
	}

	return nil
}

// GetSecret returns a single secret from the provider.
func (provider *ProviderOnePassword) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}

	item, err := provider.findItem(ref.Key)
	if err != nil {
		return nil, err
	}

	// handle files
	if item.Category == documentCategory {
		// default to the first file when ref.Property is empty
		return provider.getFile(item, ref.Property)
	}

	// handle fields
	return provider.getField(item, ref.Property)
}

// Validate checks if the client is configured correctly
// to be able to retrieve secrets from the provider.
func (provider *ProviderOnePassword) Validate() (esv1beta1.ValidationResult, error) {
	for vaultName := range provider.vaults {
		_, err := provider.client.GetVaultByTitle(vaultName)
		if err != nil {
			return esv1beta1.ValidationResultError, err
		}
	}

	return esv1beta1.ValidationResultReady, nil
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (provider *ProviderOnePassword) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}

	item, err := provider.findItem(ref.Key)
	if err != nil {
		return nil, err
	}

	// handle files
	if item.Category == documentCategory {
		return provider.getFiles(item, ref.Property)
	}

	// handle fields
	return provider.getFields(item, ref.Property)
}

// GetAllSecrets syncs multiple 1Password Items into a single Kubernetes Secret, for dataFrom.find.
func (provider *ProviderOnePassword) GetAllSecrets(_ context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errors.New(errTagsNotImplemented)
	}

	secretData := make(map[string][]byte)
	sortedVaults := sortVaults(provider.vaults)
	for _, vaultName := range sortedVaults {
		vault, err := provider.client.GetVaultByTitle(vaultName)
		if err != nil {
			return nil, fmt.Errorf(errGetVault, err)
		}

		err = provider.getAllForVault(vault.ID, ref, secretData)
		if err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

// Close closes the client connection.
func (provider *ProviderOnePassword) Close(_ context.Context) error {
	return nil
}

func (provider *ProviderOnePassword) findItem(name string) (*onepassword.Item, error) {
	sortedVaults := sortVaults(provider.vaults)
	for _, vaultName := range sortedVaults {
		vault, err := provider.client.GetVaultByTitle(vaultName)
		if err != nil {
			return nil, fmt.Errorf(errGetVault, err)
		}

		// use GetItemsByTitle instead of GetItemByTitle in order to handle length cases
		items, err := provider.client.GetItemsByTitle(name, vault.ID)
		if err != nil {
			return nil, fmt.Errorf(errGetItem, err)
		}
		switch {
		case len(items) == 1:
			return provider.client.GetItemByUUID(items[0].ID, items[0].Vault.ID)
		case len(items) > 1:
			return nil, fmt.Errorf("%w: '%s', got %d", ErrExpectedOneItem, name, len(items))
		}
	}

	return nil, fmt.Errorf("%w: %s in: %v", ErrKeyNotFound, name, provider.vaults)
}

func (provider *ProviderOnePassword) getField(item *onepassword.Item, property string) ([]byte, error) {
	// default to a field labeled "password"
	fieldLabel := "password"
	if property != "" {
		fieldLabel = property
	}

	if length := countFieldsWithLabel(fieldLabel, item.Fields); length != 1 {
		return nil, fmt.Errorf("%w: '%s' in '%s', got %d", ErrExpectedOneField, fieldLabel, item.Title, length)
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
			return nil, fmt.Errorf(errExpectedOneFieldMsgF, ErrExpectedOneField, field.Label, item.Title, length)
		}

		// caution: do not use client.GetValue here because it has undesirable behavior on keys with a dot in them
		secretData[field.Label] = []byte(field.Value)
	}

	return secretData, nil
}

func (provider *ProviderOnePassword) getAllFields(item onepassword.Item, ref esv1beta1.ExternalSecretFind, secretData map[string][]byte) error {
	i, err := provider.client.GetItemByUUID(item.ID, item.Vault.ID)
	if err != nil {
		return fmt.Errorf(errGetItem, err)
	}
	item = *i
	for _, field := range item.Fields {
		if length := countFieldsWithLabel(field.Label, item.Fields); length != 1 {
			return fmt.Errorf(errExpectedOneFieldMsgF, ErrExpectedOneField, field.Label, item.Title, length)
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

// waitForFunc will wait for OnePassword to _actually_ create/update the secret. OnePassword returns immediately after
// the initial create/update which makes the next call for the same item create/update a new item with the same name. Hence, we'll
// wait for the item to exist or be updated on OnePassword's side as well.
// Ideally we could do bulk operations and handle data with one submit, but that would require re-writing the entire
// push secret controller. For now, this is sufficient.
func (provider *ProviderOnePassword) waitForFunc(ctx context.Context, fn func() error) error {
	// check every .5 seconds
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	done, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var err error
	for {
		select {
		case <-tick.C:
			if err = fn(); err == nil {
				return nil
			}
		case <-done.Done():
			return fmt.Errorf("timeout to wait for function to run successfully; last error was: %w", err)
		}
	}
}

func (provider *ProviderOnePassword) waitForItemToExist(title string) func() error {
	return func() error {
		_, err := provider.findItem(title)

		return err
	}
}

func (provider *ProviderOnePassword) waitForLabelToBeUpdated(title, label string, val []byte) func() error {
	return func() error {
		item, err := provider.findItem(title)
		if err != nil {
			return err
		}

		for _, field := range item.Fields {
			// we found the label with the right value
			if field.Label == label && field.Value == string(val) {
				return nil
			}
		}

		return fmt.Errorf("label %s no found on value with title %s", title, label)
	}
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
