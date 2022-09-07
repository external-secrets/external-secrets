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
package fake

import (
	"errors"
	"fmt"

	"github.com/1Password/connect-sdk-go/onepassword"
)

// OnePasswordMockClient is a fake connect.Client.
type OnePasswordMockClient struct {
	MockVaults       map[string][]onepassword.Vault
	MockItems        map[string][]onepassword.Item // ID and Title only
	MockItemFields   map[string]map[string][]*onepassword.ItemField
	MockFileContents map[string][]byte
}

// NewMockClient returns an instantiated mock client.
func NewMockClient() *OnePasswordMockClient {
	return &OnePasswordMockClient{
		MockVaults:       map[string][]onepassword.Vault{},
		MockItems:        map[string][]onepassword.Item{},
		MockItemFields:   map[string]map[string][]*onepassword.ItemField{},
		MockFileContents: map[string][]byte{},
	}
}

// GetVaults unused fake.
func (mockClient *OnePasswordMockClient) GetVaults() ([]onepassword.Vault, error) {
	return []onepassword.Vault{}, nil
}

// GetVault unused fake.
func (mockClient *OnePasswordMockClient) GetVault(uuid string) (*onepassword.Vault, error) {
	return &onepassword.Vault{}, nil
}

// GetVaultByUUID unused fake.
func (mockClient *OnePasswordMockClient) GetVaultByUUID(uuid string) (*onepassword.Vault, error) {
	return &onepassword.Vault{}, nil
}

// GetVaultByTitle returns a vault, you must preload, only one.
func (mockClient *OnePasswordMockClient) GetVaultByTitle(uuid string) (*onepassword.Vault, error) {
	if len(mockClient.MockVaults[uuid]) != 0 {
		return &mockClient.MockVaults[uuid][0], nil
	}
	return &onepassword.Vault{}, nil
}

// GetVaultsByTitle returns a list of vaults, you must preload.
func (mockClient *OnePasswordMockClient) GetVaultsByTitle(uuid string) ([]onepassword.Vault, error) {
	return mockClient.MockVaults[uuid], nil
}

// GetItems returns []onepassword.Item, you must preload.
func (mockClient *OnePasswordMockClient) GetItems(vaultUUID string) ([]onepassword.Item, error) {
	return mockClient.MockItems[vaultUUID], nil
}

// GetItem returns a *onepassword.Item, you must preload.
func (mockClient *OnePasswordMockClient) GetItem(itemUUID, vaultUUID string) (*onepassword.Item, error) {
	for _, item := range mockClient.MockItems[vaultUUID] {
		if item.ID == itemUUID {
			// load the fields that GetItemsByTitle does not
			item.Fields = mockClient.MockItemFields[vaultUUID][itemUUID]

			return &item, nil
		}
	}

	return &onepassword.Item{}, errors.New("status 400: Invalid Item UUID")
}

// GetItemByUUID returns a *onepassword.Item, you must preload.
func (mockClient *OnePasswordMockClient) GetItemByUUID(itemUUID, vaultUUID string) (*onepassword.Item, error) {
	for _, item := range mockClient.MockItems[vaultUUID] {
		if item.ID == itemUUID {
			// load the fields that GetItemsByTitle does not
			item.Fields = mockClient.MockItemFields[vaultUUID][itemUUID]

			return &item, nil
		}
	}

	return &onepassword.Item{}, errors.New("status 400: Invalid Item UUID")
}

// GetItemByTitle unused fake.
func (mockClient *OnePasswordMockClient) GetItemByTitle(title, vaultUUID string) (*onepassword.Item, error) {
	return &onepassword.Item{}, nil
}

// GetItemsByTitle returns a list of items, you must preload.
func (mockClient *OnePasswordMockClient) GetItemsByTitle(itemUUID, vaultUUID string) ([]onepassword.Item, error) {
	items := []onepassword.Item{}
	for _, item := range mockClient.MockItems[vaultUUID] {
		if item.Title == itemUUID {
			items = append(items, item)
		}
	}

	return items, nil
}

// CreateItem unused fake.
func (mockClient *OnePasswordMockClient) CreateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	return &onepassword.Item{}, nil
}

// UpdateItem unused fake.
func (mockClient *OnePasswordMockClient) UpdateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	return &onepassword.Item{}, nil
}

// DeleteItem unused fake.
func (mockClient *OnePasswordMockClient) DeleteItem(item *onepassword.Item, vaultUUID string) error {
	return nil
}

// DeleteItemByID unused fake.
func (mockClient *OnePasswordMockClient) DeleteItemByID(itemUUID, vaultQuery string) error {
	return nil
}

// DeleteItemByTitle unused fake.
func (mockClient *OnePasswordMockClient) DeleteItemByTitle(title, vaultQuery string) error {
	return nil
}

// GetFiles unused fake.
func (mockClient *OnePasswordMockClient) GetFiles(itemQuery, vaultQuery string) ([]onepassword.File, error) {
	return []onepassword.File{}, nil
}

// GetFile unused fake.
func (mockClient *OnePasswordMockClient) GetFile(fileUUID, itemUUID, vaultUUID string) (*onepassword.File, error) {
	return &onepassword.File{}, nil
}

// GetFileContent returns file data, you must preload.
func (mockClient *OnePasswordMockClient) GetFileContent(file *onepassword.File) ([]byte, error) {
	value, ok := mockClient.MockFileContents[file.Name]
	if !ok {
		return []byte{}, errors.New("status 400: Invalid File Name")
	}

	return value, nil
}

// DownloadFile unused fake.
func (mockClient *OnePasswordMockClient) DownloadFile(file *onepassword.File, targetDirectory string, overwrite bool) (string, error) {
	return "", nil
}

// LoadStructFromItemByUUID unused fake.
func (mockClient *OnePasswordMockClient) LoadStructFromItemByUUID(config interface{}, itemUUID, vaultQuery string) error {
	return nil
}

// LoadStructFromItemByTitle unused fake.
func (mockClient *OnePasswordMockClient) LoadStructFromItemByTitle(config interface{}, itemTitle, vaultQuery string) error {
	return nil
}

// LoadStructFromItem unused fake.
func (mockClient *OnePasswordMockClient) LoadStructFromItem(config interface{}, itemQuery, vaultQuery string) error {
	return nil
}

// LoadStructunused fake.
func (mockClient *OnePasswordMockClient) LoadStruct(config interface{}) error {
	return nil
}

// // For rigging test cases

// AddPredictableVault adds vaults to the mock client in a predictable way.
func (mockClient *OnePasswordMockClient) AddPredictableVault(name string) *OnePasswordMockClient {
	mockClient.MockVaults[name] = append(mockClient.MockVaults[name], onepassword.Vault{
		ID:   fmt.Sprintf("%s-id", name),
		Name: name,
	})

	return mockClient
}

// AddPredictableItemWithField adds an item and it's fields to the mock client in a predictable way.
func (mockClient *OnePasswordMockClient) AddPredictableItemWithField(vaultName, title, label, value string) *OnePasswordMockClient {
	itemID := fmt.Sprintf("%s-id", title)
	vaultID := fmt.Sprintf("%s-id", vaultName)

	mockClient.MockItems[vaultID] = append(mockClient.MockItems[vaultID], onepassword.Item{
		ID:    itemID,
		Title: title,
		Vault: onepassword.ItemVault{ID: vaultID},
	})

	if mockClient.MockItemFields[vaultID] == nil {
		mockClient.MockItemFields[vaultID] = make(map[string][]*onepassword.ItemField)
	}
	mockClient.MockItemFields[vaultID][itemID] = append(mockClient.MockItemFields[vaultID][itemID], &onepassword.ItemField{
		Label: label,
		Value: value,
	})

	return mockClient
}

// AppendVault appends a onepassword.Vault to the mock client.
func (mockClient *OnePasswordMockClient) AppendVault(name string, vault onepassword.Vault) *OnePasswordMockClient {
	mockClient.MockVaults[name] = append(mockClient.MockVaults[name], vault)

	return mockClient
}

// AppendItem appends a onepassword.Item to the mock client.
func (mockClient *OnePasswordMockClient) AppendItem(vaultID string, item onepassword.Item) *OnePasswordMockClient {
	mockClient.MockItems[vaultID] = append(mockClient.MockItems[vaultID], item)

	return mockClient
}

// AppendItemField appends a onepassword.ItemField to the mock client.
func (mockClient *OnePasswordMockClient) AppendItemField(vaultID, itemID string, itemField onepassword.ItemField) *OnePasswordMockClient {
	if mockClient.MockItemFields[vaultID] == nil {
		mockClient.MockItemFields[vaultID] = make(map[string][]*onepassword.ItemField)
	}
	mockClient.MockItemFields[vaultID][itemID] = append(mockClient.MockItemFields[vaultID][itemID], &itemField)

	return mockClient
}

// SetFileContents adds file contents to the mock client.
func (mockClient *OnePasswordMockClient) SetFileContents(name string, contents []byte) *OnePasswordMockClient {
	// no need to test or mock same file names in different vaults, because we only GetFileContent after findItem, which already tests getting the right item from the right vault
	mockClient.MockFileContents[name] = contents

	return mockClient
}
