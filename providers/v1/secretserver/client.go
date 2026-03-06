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

package secretserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/metadata"
)

const (
	errMsgUnableToRetrieve = "unable to retrieve secret"
	errMsgNotFound         = "not found"
)

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

	// Look up the secret to see if it exists
	remoteRef := esv1.ExternalSecretDataRemoteRef{
		Key: data.GetRemoteKey(),
	}
	existingSecret, err := c.getSecret(ctx, remoteRef)
	if err != nil {
		if !strings.Contains(err.Error(), errMsgUnableToRetrieve) && !strings.Contains(err.Error(), errMsgNotFound) {
			return fmt.Errorf("failed to get secret: %w", err)
		}
		existingSecret = nil
	}

	if existingSecret != nil {
		// Update existing secret
		return c.updateSecret(existingSecret, data.GetProperty(), string(value))
	}

	// Secret doesn't exist, create it
	meta, err := metadata.ParseMetadataParameters[PushSecretMetadataSpec](data.GetMetadata())
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	if meta == nil || meta.Spec.FolderID <= 0 || meta.Spec.SecretTemplateID <= 0 {
		return errors.New("folderId and secretTemplateId must be provided in metadata to create a new secret")
	}

	return c.createSecret(data.GetRemoteKey(), data.GetProperty(), string(value), meta.Spec)
}

// updateSecret updates an existing secret in Delinea Secret Server.
func (c *client) updateSecret(secret *server.Secret, property string, value string) error {
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
func (c *client) createSecret(name, property, value string, meta PushSecretMetadataSpec) error {
	template, err := c.api.SecretTemplate(meta.SecretTemplateID)
	if err != nil {
		return fmt.Errorf("failed to get secret template: %w", err)
	}

	newSecret := server.Secret{
		Name:             name,
		FolderID:         meta.FolderID,
		SecretTemplateID: meta.SecretTemplateID,
		Fields:           make([]server.SecretField, 0),
	}

	if property == "" {
		// Populate the first field of the template with the whole JSON
		if len(template.Fields) == 0 {
			return errors.New("secret template has no fields")
		}
		newSecret.Fields = append(newSecret.Fields, server.SecretField{
			FieldID:   template.Fields[0].SecretTemplateFieldID,
			ItemValue: value,
		})
	} else {
		// Populate the specific property
		fieldId, found := findTemplateFieldID(template, property)
		if !found {
			return fmt.Errorf("field %s not found in secret template", property)
		}
		newSecret.Fields = append(newSecret.Fields, server.SecretField{
			FieldID:   fieldId,
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
		if strings.Contains(err.Error(), errMsgUnableToRetrieve) || strings.Contains(err.Error(), errMsgNotFound) {
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
		if strings.Contains(err.Error(), errMsgUnableToRetrieve) || strings.Contains(err.Error(), errMsgNotFound) {
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

// GetAllSecrets not supported at this time.
func (c *client) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New("getting all secrets is not supported by Delinea Secret Server at this time")
}

func (c *client) Close(context.Context) error {
	return nil
}

// getSecret retrieves the secret referenced by ref from the Vault API.
func (c *client) getSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (*server.Secret, error) {
	if ref.Version != "" {
		return nil, errors.New("specifying a version is not supported")
	}

	// If the ref.Key looks like a full path (starts with "/"), fetch by path.
	// Example: "/Folder/Subfolder/SecretName"
	if strings.HasPrefix(ref.Key, "/") {
		s, err := c.api.SecretByPath(ref.Key)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	// Otherwise try converting it to an ID
	id, err := strconv.Atoi(ref.Key)
	if err != nil {
		s, err := c.api.Secrets(ref.Key, "Name")
		if err != nil {
			return nil, err
		}
		if len(s) == 0 {
			return nil, errors.New("unable to retrieve secret at this time")
		}

		return &s[0], nil
	}
	return c.api.Secret(id)
}

func findTemplateFieldID(template *server.SecretTemplate, property string) (int, bool) {
	fieldId, found := template.FieldSlugToId(property)
	if found {
		return fieldId, true
	}

	// fallback check if they used name instead of slug
	for _, f := range template.Fields {
		if f.Name == property || f.FieldSlugName == property {
			return f.SecretTemplateFieldID, true
		}
	}

	return 0, false
}
