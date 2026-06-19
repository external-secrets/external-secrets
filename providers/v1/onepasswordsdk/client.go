/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package onepasswordsdk implements a provider for 1Password secrets management service.
package onepasswordsdk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
	"github.com/external-secrets/external-secrets/runtime/find"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	fieldPrefix       = "field"
	filePrefix        = "file"
	prefixSplitter    = "/"
	vaultCachePrefix  = "vault:"
	itemCachePrefix   = "item:"
	fileCachePrefix   = "file:"
	defaultFieldLabel = "password"

	errMsgUpdateItem       = "failed to update item: %w"
	errMsgCreateItem       = "failed to create item: %w"
	errMsgParsePushMeta    = "failed to parse push secret metadata: %w"
	errMsgExpectedOneField = "found more than 1 fields with title '%s' in '%s', got %d"
	errMsgExpectedOneFile  = "found more than 1 files with title '%s' in '%s', got %d"
	errMsgFieldNotFound    = "field with label '%s' not found in item '%s'"
	errMsgFileNotFound     = "file with title '%s' not found in item '%s'"
)

// ErrKeyNotFound is returned when a key is not found in the 1Password Vaults.
var ErrKeyNotFound = errors.New("key not found")

// nativeIDPattern matches a 1Password unique identifier per the SDK
// docs (^[\da-z]{26}$). Despite being called "UUIDs" in 1Password's SDK and docs,
// they are not RFC 4122 UUIDs.
// https://www.1password.dev/cli/reference#unique-identifiers-ids
var nativeIDPattern = regexp.MustCompile(`^[\da-z]{26}$`)

func isNativeID(s string) bool {
	return nativeIDPattern.MatchString(s)
}

// PushSecretMetadataSpec defines the metadata configuration for pushing secrets to 1Password.
type PushSecretMetadataSpec struct {
	Tags      []string `json:"tags,omitempty"`
	FieldType string   `json:"fieldType,omitempty"`
}

// GetSecret returns a single secret from 1Password provider.
// Follows syntax is used for the ref key: https://developer.1password.com/docs/cli/secret-reference-syntax/
func (p *SecretsClient) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}
	key := p.constructRefKey(ref.Key)

	if cached, ok := p.cacheGet(key); ok {
		return cached, nil
	}

	secret, err := p.client.Secrets().Resolve(ctx, key)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKResolve, err)
	if err != nil {
		return nil, err
	}

	result := []byte(secret)
	p.cacheAdd(key, result)

	return result, nil
}

// Close closes the client connection.
func (p *SecretsClient) Close(_ context.Context) error {
	return nil
}

// DeleteSecret implements Secret Deletion on the provider when PushSecret.spec.DeletionPolicy=Delete.
func (p *SecretsClient) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) (err error) {
	providerItem, err := p.findItem(ctx, ref.GetRemoteKey())
	if errors.Is(err, ErrKeyNotFound) {
		// Since the item no longer exists upstream, it's safe to remove it from the cache.
		p.invalidateItem(providerItem)
		return nil
	}
	if err != nil {
		// do not remove cache entry because the error might be a network problem
		// or something unrelated.
		return err
	}

	defer func() {
		if err == nil {
			// invalidate the cache if there was no error
			p.invalidateItem(providerItem)
		}
	}()

	providerItem.Fields = normalizeItemFields(providerItem.Fields)

	var deleted bool
	providerItem.Fields, deleted, err = deleteField(providerItem.Fields, ref.GetProperty())
	if err != nil {
		return fmt.Errorf("failed to delete fields: %w", err)
	}

	if !deleted {
		// also invalidate the cache on not deleted so we refresh the fields on an item.
		return nil
	}

	// There is a chance that there is an empty item left in the section like this: [{ID: Title:}].
	if len(providerItem.Sections) == 1 && providerItem.Sections[0].ID == "" && providerItem.Sections[0].Title == "" {
		providerItem.Sections = nil
	}

	if len(providerItem.Fields) == 0 && len(providerItem.Files) == 0 && len(providerItem.Sections) == 0 {
		err = p.client.Items().Delete(ctx, providerItem.VaultID, providerItem.ID)
		metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsDelete, err)
		if err != nil {
			return fmt.Errorf("failed to delete item: %w", err)
		}
		return nil
	}

	_, err = p.client.Items().Put(ctx, providerItem)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsPut, err)
	if err != nil {
		return fmt.Errorf(errMsgUpdateItem, err)
	}

	return nil
}

func deleteField(fields []onepassword.ItemField, title string) ([]onepassword.ItemField, bool, error) {
	// This will always iterate over all items,
	// but it's done to ensure that two fields with the same label
	// exist resulting in undefined behavior
	var (
		found   bool
		fieldsF = make([]onepassword.ItemField, 0, len(fields))
	)
	for _, item := range fields {
		if item.Title == title {
			if found {
				return nil, false, fmt.Errorf("found multiple labels on item %q", title)
			}
			found = true
			continue
		}
		fieldsF = append(fieldsF, item)
	}
	return fieldsF, found, nil
}

// GetAllSecrets syncs multiple 1Password Items into a single Kubernetes Secret, for dataFrom.find.
func (p *SecretsClient) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	items, err := p.listItems(ctx)
	if err != nil {
		return nil, err
	}

	// If ref.Tags is set, filter to only items that match the given tags
	if ref.Tags != nil {
		var filteredItems []onepassword.ItemOverview
		for _, item := range items {
			if itemHasTags(ref.Tags, item.Tags) {
				filteredItems = append(filteredItems, item)
			}
		}
		items = filteredItems
	}

	secretData := make(map[string][]byte)
	for _, overview := range items {
		if ref.Path != nil && *ref.Path != overview.Title {
			continue
		}

		if err := p.collectAllSecrets(ctx, overview.Title, ref, secretData); err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

func (p *SecretsClient) collectAllSecrets(ctx context.Context, itemName string, ref esv1.ExternalSecretFind, secretData map[string][]byte) error {
	item, err := p.findItem(ctx, itemName)
	if err != nil {
		return fmt.Errorf("failed to get item %s: %w", itemName, err)
	}
	if err := p.getAllFields(item, ref, secretData); err != nil {
		return fmt.Errorf("failed to get fields for item %s: %w", itemName, err)
	}
	if err := p.getAllFiles(ctx, item, ref, secretData); err != nil {
		return fmt.Errorf("failed to get files for item %s: %w", itemName, err)
	}
	return nil
}

// itemHasTags returns true if all required keys are present in the item's tags.
func itemHasTags(required map[string]string, itemTags []string) bool {
	// Quickly return false if this item has fewer tags than required, since it can't possibly match.
	if len(itemTags) < len(required) {
		return false
	}

	// Use a map to track which required tags we've found in the item's tags.
	matchingTags := make(map[string]string)

	// Loop through item's tags and add any matching tags to the matchingTags map.
	for _, itemTag := range itemTags {
		if _, ok := required[itemTag]; ok {
			matchingTags[itemTag] = required[itemTag]
		}
	}

	// Check if we found all required tags in the item's tags.
	if len(matchingTags) < len(required) {
		return false
	}
	return true
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (p *SecretsClient) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}

	cacheKey := p.constructRefKey(ref.Key) + "|" + ref.Property
	if cached, ok := p.cacheGet(cacheKey); ok {
		var result map[string][]byte
		if err := json.Unmarshal(cached, &result); err == nil {
			return result, nil
		}
		// continue with fresh instead
	}

	item, err := p.findItem(ctx, ref.Key)
	if err != nil {
		return nil, err
	}

	var result map[string][]byte
	propertyType, property := getObjType(item.Category, ref.Property)
	if propertyType == filePrefix {
		result, err = p.getFiles(ctx, item, property)
	} else {
		result, err = p.getFields(item, property)
	}

	if err != nil {
		return nil, err
	}

	if serialized, err := json.Marshal(result); err == nil {
		p.cacheAdd(cacheKey, serialized)
	}

	return result, nil
}

func (p *SecretsClient) listItems(ctx context.Context) ([]onepassword.ItemOverview, error) {
	var items []onepassword.ItemOverview

	cacheKey := vaultCachePrefix + p.vaultID
	if cached, ok := p.cacheGet(cacheKey); ok {
		if err := json.Unmarshal(cached, &items); err == nil {
			return items, nil
		}
	}

	// Vault item list not found in cache - fetch from the API
	items, err := p.client.Items().List(ctx, p.vaultID)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsList, err)
	if err != nil {
		return nil, fmt.Errorf("failed to list items: %w", err)
	}

	// Add the vault list to the cache
	if serialized, err := json.Marshal(items); err == nil {
		p.cacheAdd(cacheKey, serialized)
	} else {
		// If we fail to serialize the items for caching, we can still return the items, so we just log the error and continue.
		fmt.Printf("failed to serialize items for caching: %v\n", err)
	}

	return items, nil
}

// getFields gets the field matching the given property label in an item, or all fields in the item if `property` is not set.
func (p *SecretsClient) getFields(item onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, field := range item.Fields {
		if property != "" && field.Title != property {
			continue
		}
		// Throw error if there are multiple fields with the same label.
		if length := countFieldsWithLabel(field.Title, item.Fields); length != 1 {
			return nil, fmt.Errorf(errMsgExpectedOneField, field.Title, item.Title, length)
		}

		// caution: do not use client.GetValue here because it has undesirable behavior on keys with a dot in them
		secretData[field.Title] = []byte(field.Value)
	}

	return secretData, nil
}

// getAllFields retrieves all fields matching the given ref in an item, and adds them to the given secretData map.
func (p *SecretsClient) getAllFields(item onepassword.Item, ref esv1.ExternalSecretFind, secretData map[string][]byte) error {
	var matcher *find.Matcher
	if ref.Name != nil {
		var err error
		matcher, err = find.New(*ref.Name)
		if err != nil {
			return err
		}
	}

	for _, field := range item.Fields {
		// Throw error if there are multiple fields in this item with the same label.
		if length := countFieldsWithLabel(field.Title, item.Fields); length != 1 {
			return fmt.Errorf(errMsgExpectedOneField, field.Title, item.Title, length)
		}

		// If ref.Name is set, only add fields that match the regex pattern.
		if matcher != nil && !matcher.MatchName(field.Title) {
			continue
		}

		// Throw error if there are multiple fields with the same label.
		if _, found := secretData[field.Title]; found {
			return fmt.Errorf("found multiple labels with the same key '%s'", field.Title)
		}

		secretData[field.Title] = []byte(field.Value)
	}

	return nil
}

// fetchFile retrieves the content of a file, using the cache if possible.
// TODO - Currently, cached files are not invalidated on updates. This should be done as part of the cache refactor.
// See GitHub issue: https://github.com/external-secrets/external-secrets/issues/6444
func (p *SecretsClient) fetchFile(ctx context.Context, itemID, fieldID string, attributes onepassword.FileAttributes) ([]byte, error) {
	cacheKey := fileCachePrefix + p.vaultID + ":" + itemID + ":" + fieldID + ":" + attributes.Name
	if cached, ok := p.cacheGet(cacheKey); ok {
		return cached, nil
	}
	contents, err := p.client.Items().Files().Read(ctx, p.vaultID, fieldID, attributes)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKFilesRead, err)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p.cacheAdd(cacheKey, contents)
	return contents, nil
}

// getFiles gets the file matching the given property label in an item, or all files in the item if `property` is not set.
func (p *SecretsClient) getFiles(ctx context.Context, item onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, file := range item.Files {
		if property != "" && file.Attributes.Name != property {
			continue
		}

		// Throw error if there are multiple files with the same label.
		if length := countFilesWithLabel(file.Attributes.Name, item.Files); length != 1 {
			return nil, fmt.Errorf(errMsgExpectedOneFile, file.Attributes.Name, item.Title, length)
		}

		contents, err := p.fetchFile(ctx, item.ID, file.FieldID, file.Attributes)
		if err != nil {
			return nil, err
		}
		secretData[file.Attributes.Name] = contents
	}

	return secretData, nil
}

// getAllFiles retrieves all files matching the given ref in an item, and adds them to the given secretData map.
func (p *SecretsClient) getAllFiles(ctx context.Context, item onepassword.Item, ref esv1.ExternalSecretFind, secretData map[string][]byte) error {
	var matcher *find.Matcher
	if ref.Name != nil {
		var err error
		matcher, err = find.New(*ref.Name)
		if err != nil {
			return err
		}
	}

	for _, file := range item.Files {
		if matcher != nil && !matcher.MatchName(file.Attributes.Name) {
			continue
		}

		// Throw error if there are multiple files with the same label.
		if _, found := secretData[file.Attributes.Name]; found {
			return fmt.Errorf("found multiple labels with the same key '%s'", file.Attributes.Name)
		}

		contents, err := p.fetchFile(ctx, item.ID, file.FieldID, file.Attributes)
		if err != nil {
			return err
		}
		secretData[file.Attributes.Name] = contents
	}

	return nil
}

func countFieldsWithLabel(fieldLabel string, fields []onepassword.ItemField) int {
	count := 0
	for _, field := range fields {
		if field.Title == fieldLabel {
			count++
		}
	}

	return count
}

func countFilesWithLabel(fileLabel string, files []onepassword.ItemFile) int {
	count := 0
	for _, file := range files {
		if file.Attributes.Name == fileLabel {
			count++
		}
	}

	return count
}

// Clean property string by removing property prefix if needed.
func getObjType(documentType onepassword.ItemCategory, property string) (string, string) {
	if strings.HasPrefix(property, fieldPrefix+prefixSplitter) {
		return fieldPrefix, property[6:]
	}
	if strings.HasPrefix(property, filePrefix+prefixSplitter) {
		return filePrefix, property[5:]
	}

	if documentType == onepassword.ItemCategoryDocument {
		return filePrefix, property
	}

	return fieldPrefix, property
}

// createItem creates a new item in the first vault. If no vaults exist, it returns an error.
func (p *SecretsClient) createItem(ctx context.Context, val []byte, ref esv1.PushSecretData) error {
	mdata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](ref.GetMetadata())
	if err != nil {
		return fmt.Errorf(errMsgParsePushMeta, err)
	}

	label := ref.GetProperty()
	if label == "" {
		label = defaultFieldLabel
	}

	var tags []string
	if mdata != nil && mdata.Spec.Tags != nil {
		tags = mdata.Spec.Tags
	}

	fieldType := onepassword.ItemFieldTypeConcealed
	if mdata != nil {
		fieldType = resolveFieldType(mdata.Spec.FieldType)
	}

	createdItem, err := p.client.Items().Create(ctx, onepassword.ItemCreateParams{
		Category: onepassword.ItemCategoryServer,
		VaultID:  p.vaultID,
		Title:    ref.GetRemoteKey(),
		Fields: []onepassword.ItemField{
			generateNewItemField(label, string(val), fieldType),
		},
		Tags: tags,
	})
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsCreate, err)
	if err != nil {
		return fmt.Errorf(errMsgCreateItem, err)
	}

	p.invalidateItem(createdItem)

	return nil
}

// updateFieldValue updates the fields value of an item with the given label.
// If the label does not exist, a new field is created with the given fieldType. If the label exists but
// the value is different, the value is updated. If the label exists and the
// value is the same, nothing is done.
func updateFieldValue(fields []onepassword.ItemField, title, newVal string, fieldType onepassword.ItemFieldType) ([]onepassword.ItemField, error) {
	// This will always iterate over all items.
	// This is done to ensure that two fields with the same label
	// exist resulting in undefined behavior.
	var (
		found bool
		index int
	)
	for i, item := range fields {
		if item.Title == title {
			if found {
				return nil, fmt.Errorf("found multiple labels with the same key")
			}
			found = true
			index = i
		}
	}
	if !found {
		return append(fields, generateNewItemField(title, newVal, fieldType)), nil
	}

	if fields[index].Value != newVal {
		fields[index].Value = newVal
	}
	if fields[index].FieldType != fieldType {
		fields[index].FieldType = fieldType
	}

	return fields, nil
}

// resolveFieldType maps a 1Password field type name to the SDK constant.
// Case-insensitive. Accepted: text|string, concealed|password, url, email, phone, date, monthYear.
// Defaults to Concealed for empty/unrecognized. OTP and file excluded.
// Reference: https://developer.1password.com/docs/cli/item-fields/#custom-fields
func resolveFieldType(raw string) onepassword.ItemFieldType {
	switch strings.ToLower(raw) {
	case "text", "string":
		return onepassword.ItemFieldTypeText
	case "concealed", "password":
		return onepassword.ItemFieldTypeConcealed
	case "email":
		return onepassword.ItemFieldTypeEmail
	case "url":
		return onepassword.ItemFieldTypeURL
	case "phone":
		return onepassword.ItemFieldTypePhone
	case "date":
		return onepassword.ItemFieldTypeDate
	case "monthyear":
		return onepassword.ItemFieldTypeMonthYear
	}
	return onepassword.ItemFieldTypeConcealed
}

// normalizeItemFields clears empty section IDs because the 1Password SDK rejects items with a SectionID pointer to "" when the section is missing.
func normalizeItemFields(fields []onepassword.ItemField) []onepassword.ItemField {
	for i := range fields {
		if fields[i].SectionID != nil && *fields[i].SectionID == "" {
			fields[i].SectionID = nil
		}
	}
	return fields
}

// generateNewItemField creates an ItemField with ID and Title set to the given title (unique within item), value, and field type.
func generateNewItemField(title, newVal string, fieldType onepassword.ItemFieldType) onepassword.ItemField {
	return onepassword.ItemField{
		ID:        title,
		Title:     title,
		Value:     newVal,
		FieldType: fieldType,
	}
}

// PushSecret creates or updates a secret in 1Password.
func (p *SecretsClient) PushSecret(ctx context.Context, secret *corev1.Secret, ref esv1.PushSecretData) error {
	if ref.GetSecretKey() == "" {
		return p.pushAllKeys(ctx, secret, ref)
	}

	val, ok := secret.Data[ref.GetSecretKey()]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain a key", secret.Namespace, secret.Name)
	}

	title := ref.GetRemoteKey()
	providerItem, err := p.findItem(ctx, title)
	if errors.Is(err, ErrKeyNotFound) {
		return p.createItem(ctx, val, ref)
	} else if err != nil {
		return fmt.Errorf("failed to find item: %w", err)
	}

	providerItem.Fields = normalizeItemFields(providerItem.Fields)

	label := ref.GetProperty()
	if label == "" {
		label = defaultFieldLabel
	}

	mdata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](ref.GetMetadata())
	if err != nil {
		return fmt.Errorf(errMsgParsePushMeta, err)
	}
	if mdata != nil && mdata.Spec.Tags != nil {
		providerItem.Tags = mdata.Spec.Tags
	}

	fieldType := onepassword.ItemFieldTypeConcealed
	if mdata != nil {
		fieldType = resolveFieldType(mdata.Spec.FieldType)
	}

	providerItem.Fields, err = updateFieldValue(providerItem.Fields, label, string(val), fieldType)
	if err != nil {
		return fmt.Errorf("failed to update field with label: %s: %w", label, err)
	}

	_, err = p.client.Items().Put(ctx, providerItem)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsPut, err)
	if err != nil {
		return fmt.Errorf(errMsgUpdateItem, err)
	}

	p.invalidateItem(providerItem)

	return nil
}

// createAllKeysItem creates a new item with all keys from secret.Data.
func (p *SecretsClient) createAllKeysItem(ctx context.Context, secret *corev1.Secret, title string, tags []string, fieldType onepassword.ItemFieldType) error {
	fields := make([]onepassword.ItemField, 0, len(secret.Data))
	for k, v := range secret.Data {
		fields = append(fields, generateNewItemField(k, string(v), fieldType))
	}
	createdItem, err := p.client.Items().Create(ctx, onepassword.ItemCreateParams{
		Category: onepassword.ItemCategoryServer,
		VaultID:  p.vaultID,
		Title:    title,
		Fields:   fields,
		Tags:     tags,
	})
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsCreate, err)
	if err != nil {
		return fmt.Errorf(errMsgCreateItem, err)
	}
	p.invalidateItem(createdItem)
	return nil
}

// pushAllKeys pushes all keys from secret.Data as separate fields on a single 1Password item.
func (p *SecretsClient) pushAllKeys(ctx context.Context, secret *corev1.Secret, ref esv1.PushSecretData) error {
	mdata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](ref.GetMetadata())
	if err != nil {
		return fmt.Errorf(errMsgParsePushMeta, err)
	}

	var tags []string
	if mdata != nil && mdata.Spec.Tags != nil {
		tags = mdata.Spec.Tags
	}

	fieldType := onepassword.ItemFieldTypeConcealed
	if mdata != nil {
		fieldType = resolveFieldType(mdata.Spec.FieldType)
	}

	title := ref.GetRemoteKey()
	providerItem, err := p.findItem(ctx, title)

	if errors.Is(err, ErrKeyNotFound) {
		return p.createAllKeysItem(ctx, secret, title, tags, fieldType)
	}
	if err != nil {
		return fmt.Errorf("failed to find item: %w", err)
	}

	providerItem.Fields = normalizeItemFields(providerItem.Fields)
	if tags != nil {
		providerItem.Tags = tags
	}
	kept := make([]onepassword.ItemField, 0, len(providerItem.Fields))
	for _, f := range providerItem.Fields {
		if v, ok := secret.Data[f.Title]; ok {
			f.Value = string(v)
			f.FieldType = fieldType
			kept = append(kept, f)
		}
	}
	for k, v := range secret.Data {
		if countFieldsWithLabel(k, kept) == 0 {
			kept = append(kept, generateNewItemField(k, string(v), fieldType))
		}
	}
	providerItem.Fields = kept
	_, err = p.client.Items().Put(ctx, providerItem)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsPut, err)
	if err != nil {
		return fmt.Errorf(errMsgUpdateItem, err)
	}
	p.invalidateItem(providerItem)
	return nil
}

// GetVault retrieves a vault by its title or UUID from 1Password.
func (p *SecretsClient) GetVault(ctx context.Context, titleOrUUID string) (string, error) {
	vaults, err := p.client.VaultsAPI.List(ctx)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKVaultsList, err)
	if err != nil {
		return "", fmt.Errorf("failed to list vaults: %w", err)
	}

	for _, v := range vaults {
		if v.Title == titleOrUUID || v.ID == titleOrUUID {
			return v.ID, nil
		}
	}

	return "", fmt.Errorf("vault %s not found", titleOrUUID)
}

// fetchItemByID retrieves an item by its ID, using the cache if possible.
func (p *SecretsClient) fetchItemByID(ctx context.Context, id string) (onepassword.Item, error) {
	cacheKey := itemCachePrefix + p.vaultID + ":" + id
	if cached, ok := p.cacheGet(cacheKey); ok {
		var item onepassword.Item
		if err := json.Unmarshal(cached, &item); err == nil {
			return item, nil
		}
	}

	item, err := p.client.Items().Get(ctx, p.vaultID, id)
	metrics.ObserveAPICall(constants.ProviderOnePasswordSDK, constants.CallOnePasswordSDKItemsGet, err)
	if err != nil {
		return onepassword.Item{}, err
	}

	if serialized, err := json.Marshal(item); err == nil {
		p.cacheAdd(cacheKey, serialized)
	}
	return item, nil
}

// findItem retrieves an item by its title or ID, using the cache if possible.
func (p *SecretsClient) findItem(ctx context.Context, name string) (onepassword.Item, error) {
	cacheKey := itemCachePrefix + p.vaultID + ":" + name
	if cached, ok := p.cacheGet(cacheKey); ok {
		var item onepassword.Item
		if err := json.Unmarshal(cached, &item); err == nil {
			return item, nil
		}
	}

	var item onepassword.Item
	var err error

	if isNativeID(name) {
		item, err = p.fetchItemByID(ctx, name)
		if err != nil {
			if isNotFoundError(err) {
				return onepassword.Item{}, ErrKeyNotFound
			}
			return onepassword.Item{}, err
		}
	} else {
		// If name is not a native item ID, we have to list items and find the matching title.
		items, err := p.listItems(ctx)
		if err != nil {
			return onepassword.Item{}, fmt.Errorf("failed to list items: %w", err)
		}

		// Find the ID of the item matching the given name. Throw an error if there are multiple items with the same name, or if no items are found.
		var itemUUID string
		for _, v := range items {
			if v.Title == name {
				if itemUUID != "" {
					return onepassword.Item{}, fmt.Errorf("found multiple items with name %s", name)
				}
				itemUUID = v.ID
			}
		}
		if itemUUID == "" {
			return onepassword.Item{}, ErrKeyNotFound
		}

		// Fetch the item by ID to get all its details.
		item, err = p.fetchItemByID(ctx, itemUUID)
		if err != nil {
			return onepassword.Item{}, err
		}

		// While fetchItemByID will cache the item by its ID, we also want to cache it by its name.
		if serialized, err := json.Marshal(item); err == nil {
			p.cacheAdd(cacheKey, serialized)
		}
	}

	return item, nil
}

// SecretExists returns true if the item exists, and if a property is specified, if a field with that title exists.
func (p *SecretsClient) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	item, err := p.findItem(ctx, ref.GetRemoteKey())
	if errors.Is(err, ErrKeyNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	property := ref.GetProperty()
	if property == "" {
		return true, nil // item exists; pushAllKeys handles field-level reconciliation
	}

	for _, f := range item.Fields {
		if f.Title == property {
			return true, nil
		}
	}
	return false, nil
}

// Validate does nothing here. It would be possible to ping the SDK to prove we're healthy, but
// since the 1password SDK rate-limit is pretty aggressive, we prefer to do nothing.
func (p *SecretsClient) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

func (p *SecretsClient) constructRefKey(key string) string {
	// remove any possible leading slashes because the vaultPrefix already contains it.
	return p.vaultPrefix + strings.TrimPrefix(key, "/")
}

// cacheGet retrieves a value from the cache. Returns false if cache is disabled or key not found.
func (p *SecretsClient) cacheGet(key string) ([]byte, bool) {
	if p.cache == nil {
		return nil, false
	}
	v, ok := p.cache.Get(key)
	if !ok {
		return nil, false
	}
	return bytes.Clone(v), true
}

// cacheAdd stores a value in the cache. No-op if cache is disabled.
func (p *SecretsClient) cacheAdd(key string, value []byte) {
	if p.cache == nil {
		return
	}
	p.cache.Add(key, value)
}

// invalidateCacheByPrefix removes all cache entries that start with the given prefix.
// This is used to invalidate cache entries when an item is modified or deleted.
// No-op if cache is disabled.
// Why are we using a Prefix? Because items and properties are stored via prefixes using 1Password SDK.
// This means when an item is deleted we delete the fields and properties that belong to the item as well.
// This is a helper for invalidateItem. Do not call directly.
func (p *SecretsClient) invalidateCacheByPrefix(prefix string) {
	if p.cache == nil {
		return
	}

	keys := p.cache.Keys()
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if len(key) == len(prefix) || key[len(prefix)] == '/' || key[len(prefix)] == '|' {
			p.cache.Remove(key)
		}
	}
}

// invalidateItem drops every cache entry tied to an item after a mutation: the
// resolved values (op://...), both the title- and ID-keyed item entries, and the
// vault item list. Mutations are addressed by title, but findItem always resolves
// through the item's UUID and listItems backs every title->UUID lookup, so all
// three must be dropped or reads return stale data.
// No-op if cache is disabled.
func (p *SecretsClient) invalidateItem(item onepassword.Item) {
	if p.cache == nil {
		return
	}

	p.invalidateCacheByPrefix(p.constructRefKey(item.Title))
	if item.ID != "" && item.ID != item.Title {
		p.invalidateCacheByPrefix(p.constructRefKey(item.ID))
	}

	p.cache.Remove(itemCachePrefix + p.vaultID + ":" + item.Title)
	if item.ID != "" {
		p.cache.Remove(itemCachePrefix + p.vaultID + ":" + item.ID)
	}

	p.cache.Remove(vaultCachePrefix + p.vaultID)
}

func isNotFoundError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "couldn't be found") || strings.Contains(msg, "resource not found")
}
