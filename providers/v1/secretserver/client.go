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
	// This preserves backward compatibility with the original error message used
	// before the PushSecret feature was added.
	errMsgNoMatchingSecrets = "unable to retrieve secret at this time"
	// errMsgNotFound is returned when a secret is not found in a specific folder.
	errMsgNotFound = "not found"
	// errMsgAmbiguousName is returned by lookupSecretStrict when a plain name
	// matches multiple secrets across folders and no folder scope is provided.
	errMsgAmbiguousName = "multiple secrets found with the same name across different folders; use the 'folderId:<id>/<name>' key format, a path-based key, or a numeric ID to disambiguate"

	// folderPrefix is the prefix used to encode a folder ID in a remote key.
	// Format: "folderId:<id>/<name>" (e.g. "folderId:73/my-secret").
	folderPrefix = "folderId:"
)

// isNotFoundError checks if an error indicates a secret was not found.
// The TSS SDK (v3) returns all errors as plain fmt.Errorf strings with format
// "<StatusCode> <StatusText>: <body>" — no typed/sentinel errors.
//
// This function uses case-insensitive substring matching with explicit exclusions
// for false-positive patterns produced by our own code (e.g. "not found in secret"
// from updateSecret or "not found in secret template" from createSecret) and
// non-404 HTTP errors that happen to contain "not found" in their body.
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	// Our own sentinel from getSecretByName / getSecretByNameStrict.
	if strings.Contains(msg, errMsgNoMatchingSecrets) {
		return true
	}

	// SDK HTTP 404 responses start with "404 ".
	if strings.HasPrefix(msg, "404 ") {
		return true
	}

	// Generic "not found" substring — but exclude false positives.
	if strings.Contains(msg, errMsgNotFound) {
		// Patterns like "field X not found in secret" or "field X not found in secret template"
		// are field-level errors, not secret-not-found errors.
		if strings.Contains(msg, "not found in secret") {
			return false
		}
		// Exclude non-404 HTTP errors that happen to contain "not found" in the body
		// (e.g. "401 Unauthorized: user not found"). The SDK formats all HTTP errors
		// as "<StatusCode> <StatusText>: <body>", so any message starting with a
		// 3-digit code followed by a space is an HTTP error.
		if len(msg) >= 4 && msg[3] == ' ' && msg[0] >= '0' && msg[0] <= '9' && msg[1] >= '0' && msg[1] <= '9' && msg[2] >= '0' && msg[2] <= '9' {
			return false // non-404 HTTP error (404 was already handled above)
		}
		return true
	}

	return false
}

// parseFolderPrefix extracts a folder ID and secret name from a key with the
// format "folderId:<id>/<name>".  If the key does not match the prefix format,
// it returns (0, key, false) so callers can fall through to other resolution
// strategies.
func parseFolderPrefix(key string) (folderID int, name string, hasFolderPrefix bool) {
	if !strings.HasPrefix(key, folderPrefix) {
		return 0, key, false
	}

	rest := strings.TrimPrefix(key, folderPrefix) // "<id>/<name>"

	slashIdx := strings.Index(rest, "/")
	if slashIdx < 0 {
		// "folderId:73" with no slash/name — treat as not having the prefix.
		return 0, key, false
	}

	idStr := rest[:slashIdx]
	secretName := rest[slashIdx+1:]

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		// Non-numeric or non-positive folder ID — treat as not having the prefix.
		return 0, key, false
	}

	if secretName == "" {
		// "folderId:73/" with empty name — treat as not having the prefix.
		return 0, key, false
	}

	return id, secretName, true
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

// GetSecret supports several lookup modes:
//  1. Get the secret using the secret ID in ref.Key (e.g. key: 53974).
//  2. Get the secret using the secret "name" (e.g. key: "secretNameHere").
//     - Secret names must not contain spaces.
//     - If using the secret "name" and multiple secrets are found,
//     the first secret in the array will be the secret returned.
//  3. Get the full secret as a JSON-encoded value by leaving ref.Property empty.
//  4. Get a specific value by using a key from the JSON-formatted secret in
//     Items.0.ItemValue via gjson (supports nested paths like "server.1").
//     If the first field's ItemValue is not valid JSON or the gjson path
//     does not match, fall back to matching ref.Property against each field's
//     Slug or FieldName (useful for multi-field secrets).
func (c *client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.getSecret(ctx, ref)
	if err != nil {
		return nil, err
	}
	if len(secret.Fields) == 0 {
		return nil, errors.New("secret contains no fields")
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

	// Primary path: extract ref.Property from the first field's ItemValue via gjson.
	// This preserves backward compatibility with the original single-field JSON blob pattern.
	val := gjson.Get(string(jsonStr), "Items.0.ItemValue")
	if val.Exists() && gjson.Valid(val.String()) {
		out := gjson.Get(val.String(), ref.Property)
		if out.Exists() {
			return []byte(out.String()), nil
		}
	}

	// Fallback: match ref.Property against field Slug or FieldName.
	// This supports multi-field secrets where fields are accessed by name.
	for index := range secret.Fields {
		if secret.Fields[index].Slug == ref.Property || secret.Fields[index].FieldName == ref.Property {
			return []byte(secret.Fields[index].ItemValue), nil
		}
	}

	return nil, esv1.NoSecretError{}
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

	// Resolve the effective folder ID for both lookups AND creation.
	// A folderId encoded in the remoteKey takes precedence over metadata
	// (delete and existence-check never see metadata, so the prefix must win
	// to keep lookup and creation consistent).
	folderID := 0
	if meta != nil {
		folderID = meta.Spec.FolderID
	}
	if prefixFolderID, _, ok := parseFolderPrefix(data.GetRemoteKey()); ok {
		folderID = prefixFolderID
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

	if meta == nil || meta.Spec.SecretTemplateID <= 0 {
		return errors.New("folderId and secretTemplateId must be provided in metadata to create a new secret")
	}

	// Use the effective folderID (prefix-overridden or metadata-supplied) for creation.
	if folderID <= 0 {
		return errors.New("folderId and secretTemplateId must be provided in metadata to create a new secret")
	}

	createSpec := meta.Spec
	createSpec.FolderID = folderID
	return c.createSecret(data.GetRemoteKey(), data.GetProperty(), string(value), createSpec)
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

	// Strip the "folderId:<id>/" prefix if present so the secret is created
	// with just the plain name.  The folder is already specified in meta.FolderID.
	if _, stripped, ok := parseFolderPrefix(name); ok {
		name = stripped
		// After prefix stripping, the name should be a simple name (no slashes).
		// The folderId format is "folderId:<id>/<name>", not "folderId:<id>/<path>".
		if strings.Contains(name, "/") {
			return fmt.Errorf("invalid secret name %q in folderId prefix: name must not contain path separators", name)
		}
	}

	// For path-based keys (e.g. "/Folder/SubFolder/SecretName"), extract the
	// basename. The folder structure is controlled by meta.FolderID.
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
func (c *client) DeleteSecret(_ context.Context, ref esv1.PushSecretRemoteRef) error {
	secret, err := c.lookupSecretStrict(ref.GetRemoteKey())
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
func (c *client) SecretExists(_ context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	_, err := c.lookupSecretStrict(ref.GetRemoteKey())
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
// Unlike getSecret (used for reads), this refuses ambiguous plain-name matches
// when no folder scope is available, matching the safety behavior of DeleteSecret
// and SecretExists.
func (c *client) findExistingSecret(_ context.Context, key string, folderID int) (*server.Secret, error) {
	// When a folder scope is available (either from prefix or metadata),
	// the lookup is unambiguous — use the regular (non-strict) resolver.
	if folderID > 0 {
		return c.lookupSecret(key, folderID)
	}
	// No folder scope: use strict lookup to reject ambiguous plain names.
	return c.lookupSecretStrict(key)
}

// lookupSecret resolves a secret by path ("/..."), numeric ID, folder-scoped
// name ("folderId:<id>/<name>"), or plain name.
// The folderID scopes name-based lookups (0 = any folder).  A folder prefix
// encoded in the key takes precedence over the folderID argument.
func (c *client) lookupSecret(key string, folderID int) (*server.Secret, error) {
	// 1. Folder-scoped prefix: "folderId:<id>/<name>" — override folderID and
	//    resolve by name within the specified folder.
	if prefixFolderID, name, ok := parseFolderPrefix(key); ok {
		return c.getSecretByName(name, prefixFolderID)
	}

	// 2. Path-based key: fully qualified, no disambiguation needed.
	if strings.HasPrefix(key, "/") {
		return c.api.SecretByPath(key)
	}

	// 3. Numeric key: treat as ID first; fall back to name-based lookup so that
	//    secrets whose name happens to be a numeric string can still be resolved.
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

// lookupSecretStrict resolves a secret like lookupSecret but refuses to
// silently pick the first match when a plain name (no folderId prefix, no
// path, no numeric ID) matches more than one secret across folders.
// This is used by destructive operations (DeleteSecret) and existence checks
// (SecretExists) that must not accidentally act on the wrong secret.
func (c *client) lookupSecretStrict(key string) (*server.Secret, error) {
	// 1. Folder-scoped prefix: unambiguous — delegate directly.
	if prefixFolderID, name, ok := parseFolderPrefix(key); ok {
		return c.getSecretByName(name, prefixFolderID)
	}

	// 2. Path-based key: unambiguous.
	if strings.HasPrefix(key, "/") {
		return c.api.SecretByPath(key)
	}

	// 3. Numeric key: try as ID first; fall back to name-based lookup.
	if id, err := strconv.Atoi(key); err == nil {
		secret, err := c.api.Secret(id)
		if err == nil && secret != nil {
			return secret, nil
		}
		if !isNotFoundError(err) {
			return nil, err
		}
	}

	// 4. Plain name: reject if ambiguous (multiple matches without folder scope).
	return c.getSecretByNameStrict(key)
}

// getSecretByNameStrict searches for a secret by name and returns an error if
// multiple secrets share the same name across different folders.
func (c *client) getSecretByNameStrict(name string) (*server.Secret, error) {
	secrets, err := c.api.Secrets(name, "Name")
	if err != nil {
		return nil, err
	}
	if len(secrets) == 0 {
		return nil, errors.New(errMsgNoMatchingSecrets)
	}
	if len(secrets) > 1 {
		return nil, errors.New(errMsgAmbiguousName)
	}
	return &secrets[0], nil
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
