/*
Copyright © 2025 ESO Maintainer Team

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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/1password/onepassword-sdk-go"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/kube-openapi/pkg/validation/strfmt"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
)

const (
	fieldPrefix             = "field"
	filePrefix              = "file"
	prefixSplitter          = "/"
	errExpectedOneFieldMsgF = "found more than 1 fields with title '%s' in '%s', got %d"
)

// ErrKeyNotFound is returned when a key is not found in the 1Password Vaults.
var ErrKeyNotFound = errors.New("key not found")

// PushSecretMetadataSpec defines the metadata configuration for pushing secrets to 1Password.
type PushSecretMetadataSpec struct {
	Tags []string `json:"tags,omitempty"`
}

// GetSecret returns a single secret from 1Password provider.
// Follows syntax is used for the ref key: https://developer.1password.com/docs/cli/secret-reference-syntax/
func (p *Provider) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Version != "" {
		return nil, errors.New(errVersionNotImplemented)
	}
	key := p.constructRefKey(ref.Key)

	if cached, ok := p.cacheGet(key); ok {
		return cached, nil
	}

	secret, err := p.client.Secrets().Resolve(ctx, key)
	if err != nil {
		return nil, err
	}

	result := []byte(secret)
	p.cacheAdd(key, result)

	return result, nil
}

// Close closes the client connection.
func (p *Provider) Close(_ context.Context) error {
	return nil
}

// DeleteSecret implements Secret Deletion on the provider when PushSecret.spec.DeletionPolicy=Delete.
func (p *Provider) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	providerItem, err := p.findItem(ctx, ref.GetRemoteKey())
	if err != nil {
		return err
	}

	providerItem.Fields, err = deleteField(providerItem.Fields, ref.GetProperty())
	if err != nil {
		return fmt.Errorf("failed to delete fields: %w", err)
	}

	// There is a chance that there is an empty item left in the section like this: [{ID: Title:}].
	if len(providerItem.Sections) == 1 && providerItem.Sections[0].ID == "" && providerItem.Sections[0].Title == "" {
		providerItem.Sections = nil
	}

	if len(providerItem.Fields) == 0 && len(providerItem.Files) == 0 && len(providerItem.Sections) == 0 {
		// Delete the item if there are no fields, files or sections
		if err = p.client.Items().Delete(ctx, providerItem.VaultID, providerItem.ID); err != nil {
			return fmt.Errorf("failed to delete item: %w", err)
		}
		p.invalidateCacheByPrefix(p.constructRefKey(ref.GetRemoteKey()))
		return nil
	}

	if _, err = p.client.Items().Put(ctx, providerItem); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	p.invalidateCacheByPrefix(p.constructRefKey(ref.GetRemoteKey()))

	return nil
}

func deleteField(fields []onepassword.ItemField, title string) ([]onepassword.ItemField, error) {
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
				return nil, fmt.Errorf("found multiple labels on item %q", title)
			}
			found = true
			continue
		}
		fieldsF = append(fieldsF, item)
	}
	return fieldsF, nil
}

// GetAllSecrets Not Implemented.
func (p *Provider) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errNotImplemented))
}

// GetSecretMap returns multiple k/v pairs from the provider, for dataFrom.extract.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

func (p *Provider) getFields(item onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, field := range item.Fields {
		if property != "" && field.Title != property {
			continue
		}
		if length := countFieldsWithLabel(field.Title, item.Fields); length != 1 {
			return nil, fmt.Errorf(errExpectedOneFieldMsgF, field.Title, item.Title, length)
		}

		// caution: do not use client.GetValue here because it has undesirable behavior on keys with a dot in them
		secretData[field.Title] = []byte(field.Value)
	}

	return secretData, nil
}

func (p *Provider) getFiles(ctx context.Context, item onepassword.Item, property string) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, file := range item.Files {
		if property != "" && file.Attributes.Name != property {
			continue
		}

		contents, err := p.client.Items().Files().Read(ctx, p.vaultID, file.FieldID, file.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		secretData[file.Attributes.Name] = contents
	}

	return secretData, nil
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
func (p *Provider) createItem(ctx context.Context, val []byte, ref esv1.PushSecretData) error {
	mdata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](ref.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}

	label := ref.GetProperty()
	if label == "" {
		label = "password"
	}

	var tags []string
	if mdata != nil && mdata.Spec.Tags != nil {
		tags = mdata.Spec.Tags
	}

	// Create the item
	_, err = p.client.Items().Create(ctx, onepassword.ItemCreateParams{
		Category: onepassword.ItemCategoryServer,
		VaultID:  p.vaultID,
		Title:    ref.GetRemoteKey(),
		Fields: []onepassword.ItemField{
			generateNewItemField(label, string(val)),
		},
		Tags: tags,
	})
	if err != nil {
		return fmt.Errorf("failed to create item: %w", err)
	}

	p.invalidateCacheByPrefix(p.constructRefKey(ref.GetRemoteKey()))

	return nil
}

// updateFieldValue updates the fields value of an item with the given label.
// If the label does not exist, a new field is created. If the label exists but
// the value is different, the value is updated. If the label exists and the
// value is the same, nothing is done.
func updateFieldValue(fields []onepassword.ItemField, title, newVal string) ([]onepassword.ItemField, error) {
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
		return append(fields, generateNewItemField(title, newVal)), nil
	}

	if fields[index].Value != newVal {
		fields[index].Value = newVal
	}

	return fields, nil
}

// generateNewItemField generates a new item field with the given label and value.
func generateNewItemField(title, newVal string) onepassword.ItemField {
	field := onepassword.ItemField{
		Title:     title,
		Value:     newVal,
		FieldType: onepassword.ItemFieldTypeConcealed,
	}

	return field
}

// PushSecret creates or updates a secret in 1Password.
func (p *Provider) PushSecret(ctx context.Context, secret *corev1.Secret, ref esv1.PushSecretData) error {
	val, ok := secret.Data[ref.GetSecretKey()]
	if !ok {
		return fmt.Errorf("secret %s/%s does not contain a key", secret.Namespace, secret.Name)
	}

	title := ref.GetRemoteKey()
	providerItem, err := p.findItem(ctx, title)
	if errors.Is(err, ErrKeyNotFound) {
		if err = p.createItem(ctx, val, ref); err != nil {
			return fmt.Errorf("failed to create item: %w", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to find item: %w", err)
	}

	// TODO: We are only sending info to a specific label on a 1password item.
	// We should change this logic eventually to allow pushing whole kubernetes Secrets to 1password as multiple labels
	// OOTB.
	label := ref.GetProperty()
	if label == "" {
		label = "password"
	}

	mdata, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](ref.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse push secret metadata: %w", err)
	}
	if mdata != nil && mdata.Spec.Tags != nil {
		providerItem.Tags = mdata.Spec.Tags
	}

	providerItem.Fields, err = updateFieldValue(providerItem.Fields, label, string(val))
	if err != nil {
		return fmt.Errorf("failed to update field with value %s: %w", string(val), err)
	}

	if _, err = p.client.Items().Put(ctx, providerItem); err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	p.invalidateCacheByPrefix(p.constructRefKey(title))

	return nil
}

// GetVault retrieves a vault by its title or UUID from 1Password.
func (p *Provider) GetVault(ctx context.Context, titleOrUUID string) (string, error) {
	vaults, err := p.client.VaultsAPI.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list vaults: %w", err)
	}

	for _, v := range vaults {
		if v.Title == titleOrUUID || v.ID == titleOrUUID {
			// cache the ID so we don't have to repeat this lookup.
			p.vaultID = v.ID
			return v.ID, nil
		}
	}

	return "", fmt.Errorf("vault %s not found", titleOrUUID)
}

func (p *Provider) findItem(ctx context.Context, name string) (onepassword.Item, error) {
	if strfmt.IsUUID(name) {
		return p.client.Items().Get(ctx, p.vaultID, name)
	}

	items, err := p.client.Items().List(ctx, p.vaultID)
	if err != nil {
		return onepassword.Item{}, fmt.Errorf("failed to list items: %w", err)
	}

	// We don't stop
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

	return p.client.Items().Get(ctx, p.vaultID, itemUUID)
}

// SecretExists checks if a secret exists in 1Password.
func (p *Provider) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// Validate does nothing here. It would be possible to ping the SDK to prove we're healthy, but
// since the 1password SDK rate-limit is pretty aggressive, we prefer to do nothing.
func (p *Provider) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

func (p *Provider) constructRefKey(key string) string {
	// remove any possible leading slashes because the vaultPrefix already contains it.
	return p.vaultPrefix + strings.TrimPrefix(key, "/")
}

// cacheGet retrieves a value from the cache. Returns false if cache is disabled or key not found.
func (p *Provider) cacheGet(key string) ([]byte, bool) {
	if p.cache == nil {
		return nil, false
	}
	return p.cache.Get(key)
}

// cacheAdd stores a value in the cache. No-op if cache is disabled.
func (p *Provider) cacheAdd(key string, value []byte) {
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
func (p *Provider) invalidateCacheByPrefix(prefix string) {
	if p.cache == nil {
		return
	}

	keys := p.cache.Keys()
	for _, key := range keys {
		if strings.HasPrefix(key, prefix) {
			// if exact match, or ends in `/` or `|` we can remove it.
			// this will clear all fields and properties for this entry.
			if len(key) == len(prefix) || key[len(prefix)] == '/' || key[len(prefix)] == '|' {
				p.cache.Remove(key)
			}
		}
	}
}
