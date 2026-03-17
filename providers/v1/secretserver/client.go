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

package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
)

const (
	// errMsgNoMatchingSecrets is returned by getSecretByName when a search returns zero results.
	errMsgNoMatchingSecrets = "no matching secrets"
	// errMsgNotFound is returned when a secret is not found in a specific folder.
	errMsgNotFound = "not found"
)

// isNotFoundError checks if an error indicates a secret was not found.
// Uses case-insensitive substring matching since tss-sdk-go v3 has no typed errors.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, errMsgNoMatchingSecrets) || strings.Contains(msg, errMsgNotFound)
}

// PushSecretMetadataSpec contains metadata information for pushing secrets to Delinea Secret Server.
type PushSecretMetadataSpec struct {
	FolderID         int `json:"folderId"`
	SecretTemplateID int `json:"secretTemplateId"`
}

type client struct {
	api secretAPI
}

var _ esv1.SecretsClient = &client{}

// GetSecret supports two types:
//  1. Get the secrets using the secret ID in ref.key i.e. key: 53974
//  2. Get the secret using the secret "name" i.e. key: "secretNameHere"
//     - Secret names must not contain spaces.
//     - If using the secret "name" and multiple secrets are found ...
//     the first secret in the array will be the secret returned.
//  3. get the full secret as json-encoded value
//     by leaving the ref.Property empty.
//  4. get a specific value by using a key from the json formatted secret in Items.0.ItemValue.
//     Nested values are supported by specifying a gjson expression
func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Return nil if secret contains no fields
	if secret.Fields == nil {
		return nil, nil
	}
	jsonStr, err := json.Marshal(secret)
	if err != nil {
		return nil, err
	}
	// Intentionally fetch and return the full secret as raw JSON when no specific property is provided.
	// This requires calling the API to retrieve the entire secret object.
	if ref.Property == "" {
		return jsonStr, nil
	}

	// extract first "field" i.e. Items.0.ItemValue, data from secret using gjson
	val := gjson.Get(string(jsonStr), "Items.0.ItemValue")
	if val.Exists() && gjson.Valid(val.String()) {
		// extract specific value from data directly above using gjson
		out := gjson.Get(val.String(), ref.Property)
		if out.Exists() {
			return []byte(out.String()), nil
		}
	}

	// More general case Fields is an array in DelineaXPM/tss-sdk-go/v3/server
	// https://github.com/DelineaXPM/tss-sdk-go/blob/571e5674a8103031ad6f873453db27959ec1ca67/server/secret.go#L23
	secretMap := make(map[string]string)
	for index := range secret.Fields {
		secretMap[secret.Fields[index].FieldName] = secret.Fields[index].ItemValue
		secretMap[secret.Fields[index].Slug] = secret.Fields[index].ItemValue
	}

	out, ok := secretMap[ref.Property]
	if !ok {
		return nil, esv1.NoSecretError{}
	}

	return []byte(out), nil
}

// PushSecret creates or updates a secret in Delinea Secret Server.
func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if data.GetRemoteKey() == "" {
		return errors.New("remote key must be defined")
	}

	value, err := esutils.ExtractSecretData(data, secret)
	if err != nil {
		return fmt.Errorf("failed to extract secret data: %w", err)
	}

	if !utf8.Valid(value) {
		return errors.New("secret value is not valid UTF-8")
	}

	meta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Look up the secret to see if it exists
	folderID := 0
	if meta != nil {
		folderID = meta.Spec.FolderID
	}

	existingSecret, err := c.findExistingSecret(ctx, data.GetRemoteKey(), folderID)
	if err != nil {
		if !isNotFoundError(err) {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		existingSecret = nil
	}

	if existingSecret != nil {
		// Update existing secret
		return c.updateSecret(existingSecret, data.GetProperty(), string(value))
	}

	if meta == nil || meta.Spec.FolderID <= 0 || meta.Spec.SecretTemplateID <= 0 {
		return errors.New("folderId and secretTemplateId must be provided in metadata to create a new secret")
	}

	return c.createSecret(data.GetRemoteKey(), data.GetProperty(), string(value), meta.Spec)
}

// updateSecret updates an existing secret in Delinea Secret Server.
func (c *client) updateSecret(secret *server.Secret, property, value string) error {
	if property == "" {
		// If property is empty, put the JSON value in the first field, matching GetSecretMap logic
		if len(secret.Fields) > 0 {
			secret.Fields[0].ItemValue = value
		} else {
			return errors.New("secret has no fields to update")
		}
	} else {
		found := false
		for i, field := range secret.Fields {
			if field.Slug == property || field.FieldName == property {
				secret.Fields[i].ItemValue = value
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("field %s not found in secret", property)
		}
	}

	_, err := c.api.UpdateSecret(*secret)
	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}
	return nil
}

// createSecret creates a new secret in Delinea Secret Server.
// Only the targeted field is populated; other required template fields
// may cause an API error.
func (c *client) createSecret(name, property, value string, meta PushSecretMetadataSpec) error {
	template, err := c.api.SecretTemplate(meta.SecretTemplateID)
	if err != nil {
		return fmt.Errorf("failed to get secret template: %w", err)
	}

	if strings.HasSuffix(name, "/") {
		return fmt.Errorf("invalid secret name %q: name must not be empty or end with a trailing slash", name)
	}

	normalizedName := strings.TrimPrefix(name, "/")
	if strings.Contains(normalizedName, "/") {
		parts := strings.Split(normalizedName, "/")
		normalizedName = parts[len(parts)-1]
	}

	newSecret := server.Secret{
		Name:             normalizedName,
		FolderID:         meta.FolderID,
		SecretTemplateID: meta.SecretTemplateID,
		Fields:           make([]server.SecretField, 0),
	}

	if property == "" {
		// No property specified: use the first template field.
		if len(template.Fields) == 0 {
			return errors.New("secret template has no fields")
		}
		newSecret.Fields = append(newSecret.Fields, server.SecretField{
			FieldID:   template.Fields[0].SecretTemplateFieldID,
			ItemValue: value,
		})
	} else {
		// Use the field matching the specified property.
		fieldID, found := findTemplateFieldID(template, property)
		if !found {
			return fmt.Errorf("field %s not found in secret template", property)
		}
		newSecret.Fields = append(newSecret.Fields, server.SecretField{
			FieldID:   fieldID,
			ItemValue: value,
		})
	}

	_, err = c.api.CreateSecret(newSecret)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}
	return nil
}

// DeleteSecret deletes a secret in Delinea Secret Server.
func (c *client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	remoteRef := esv1.ExternalSecretDataRemoteRef{
		Key: ref.GetRemoteKey(),
	}
	secret, err := c.getSecret(ctx, remoteRef)
	if err != nil {
		// If already deleted/not found, ignore
		if isNotFoundError(err) {
			return nil
		}
		return fmt.Errorf("failed to get secret for deletion: %w", err)
	}

	err = c.api.DeleteSecret(secret.ID)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// SecretExists checks if a secret exists in Delinea Secret Server.
func (c *client) SecretExists(ctx context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	remoteRef := esv1.ExternalSecretDataRemoteRef{
		Key: ref.GetRemoteKey(),
	}
	_, err := c.getSecret(ctx, remoteRef)
	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if secret exists: %w", err)
	}
	return true, nil
}

// Validate not supported at this time.
func (c *client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetSecretMap retrieves the secret referenced by ref from the Secret Server API
// and returns it as a map of byte slices.
func (c *client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	// Ensure secret has fields before indexing into them
	if len(secret.Fields) == 0 {
		return nil, errors.New("secret contains no fields")
	}

	secretData := make(map[string]any)

	err = json.Unmarshal([]byte(secret.Fields[0].ItemValue), &secretData)
	if err != nil {
		// Do not return the raw error as json.Unmarshal errors may contain
		// sensitive secret data in the error message
		return nil, errors.New("failed to unmarshal secret: invalid JSON format")
	}

	data := make(map[string][]byte)
	for k, v := range secretData {
		data[k], err = esutils.GetByteValue(v)
		if err != nil {
			return nil, err
		}
	}
	return data, nil
}

// GetAllSecrets is not supported. The tss-sdk-go v3 SDK search is hard-capped
// at 30 results with no pagination, no tag filtering, and no folder enumeration.
func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("getting all secrets is not supported by Delinea Secret Server")
}

func (c *client) Close(context.Context) error {
	return nil
}

// getSecret retrieves the secret referenced by ref from the Secret Server API.
func (c *client) getSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (*server.Secret, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not supported")
	}

	return c.lookupSecret(ref.Key, 0)
}

// findExistingSecret looks up a secret for PushSecret's find-or-create logic.
// Unlike getSecret, it supports folder-scoped disambiguation via folderID.
func (c *client) findExistingSecret(_ context.Context, key string, folderID int) (*server.Secret, error) {
	return c.lookupSecret(key, folderID)
}

// lookupSecret resolves a secret by path ("/..."), numeric ID, or name.
// The folderID scopes name-based lookups (0 = any folder).
func (c *client) lookupSecret(key string, folderID int) (*server.Secret, error) {
	// Path-based key: fully qualified, no disambiguation needed.
	if strings.HasPrefix(key, "/") {
		return c.api.SecretByPath(key)
	}

	// Numeric key: treat as ID first; fall back to name-based lookup so that
	// secrets whose name happens to be a numeric string can still be resolved.
	if id, err := strconv.Atoi(key); err == nil {
		secret, err := c.api.Secret(id)
		if err == nil && secret != nil {
			return secret, nil
		}
		if !isNotFoundError(err) {
			return nil, err
		}
	}

	return c.getSecretByName(key, folderID)
}

func (c *client) getSecretByName(name string, folderID int) (*server.Secret, error) {
	secrets, err := c.api.Secrets(name, "Name")
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return nil, errors.New(errMsgNoMatchingSecrets)
	}

	// No folder constraint: return the first match.
	if folderID == 0 {
		return &secrets[0], nil
	}

	// Find the first secret matching the requested folder.
	for i, s := range secrets {
		if s.FolderID == folderID {
			return &secrets[i], nil
		}
	}
	return nil, errors.New(errMsgNotFound)
}

func findTemplateFieldID(template *server.SecretTemplate, property string) (int, bool) {
	fieldID, found := template.FieldSlugToId(property)
	if found {
		return fieldID, true
	}

	// fallback check if they used name instead of slug
	for _, f := range template.Fields {
		if f.Name == property || f.FieldSlugName == property {
			return f.SecretTemplateFieldID, true
		}
	}

	return 0, false
}
