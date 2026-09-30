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

package keepersecurity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	errKeeperSecuritySecretsNotFound            = "unable to find secrets. %w"
	errKeeperSecuritySecretNotFound             = "unable to find secret %s. Error: %w"
	errKeeperSecuritySecretNotUnique            = "more than 1 secret %s found"
	errKeeperSecurityRecordNotFound             = "%w: no record matched %s"
	errKeeperSecurityInvalidSecretInvalidFormat = "invalid secret. Invalid format: %w"
	errKeeperSecurityInvalidSecretDuplicatedKey = "invalid Secret. Following keys are duplicated %s"
	errKeeperSecurityInvalidProperty            = "invalid Property. Secret %s does not have any key matching %s"
	errKeeperSecurityInvalidField               = "invalid Field. Key %s does not exists"
	errKeeperSecurityNoFields                   = "invalid Secret. Secret %s does not contain any valid field/file"
	keeperSecurityFileRef                       = "fileRef"
	keeperSecurityMfa                           = "oneTimeCode"
	errTagsNotImplemented                       = "'find.tags' is not implemented in the KeeperSecurity provider"
	errPathNotImplemented                       = "'find.path' is not implemented in the KeeperSecurity provider"
	errInvalidJSONSecret                        = "invalid Secret. Secret %s can not be converted to JSON. %w"
	errInvalidRegex                             = "find.name.regex. Invalid Regular expresion %s. %w"
	errInvalidRemoteRefKey                      = "match.remoteRef.remoteKey. Invalid format. Format should match secretName/key got %s"
	errInvalidSecretType                        = "ESO can only push/delete records of type %s. Secret %s is type %s"
	errKeeperSecurityMissingFolderIDForCreate   = "folderID must be set on the SecretStore to create a new Keeper Security record"
	errKeeperSecurityUnexpectedFieldState       = "keepersecurity: unexpected field state (property=%t, secretKey=%t)"
	errKeeperSecurityWholeRecordKeyHasSlash     = "match.remoteRef.remoteKey. Whole-record push does not support '/' in the remote key, got %s"
	errKeeperSecurityNonUTF8Value               = "secret key %q is not valid UTF-8 and cannot be stored in a Keeper property"

	externalSecretType = "externalSecrets"
	secretType         = "secret"
	// LoginType represents the login field type.
	LoginType = "login"
	// PasswordType represents the password field type.
	PasswordType = "password"
	// URLType represents the URL field type.
	URLType = "url"
)

// Client represents a KeeperSecurity client that can interact with the KeeperSecurity API.
type Client struct {
	ksmClient          SecurityClient
	folderID           string
	getByTitleFallback bool
}

// SecurityClient defines the interface for interacting with KeeperSecurity's API.
type SecurityClient interface {
	GetSecrets(filter []string) ([]*ksm.Record, error)
	GetSecretByTitle(recordTitle string) (*ksm.Record, error)
	GetSecretsByTitle(recordTitle string) (records []*ksm.Record, err error)
	CreateSecretWithRecordData(recUID, folderUID string, recordData *ksm.RecordCreate) (string, error)
	DeleteSecrets(recrecordUids []string) (map[string]string, error)
	Save(record *ksm.Record) error
}

// Field represents a KeeperSecurity field with its type, label (optional), and value.
type Field struct {
	Type  string `json:"type"`
	Label string `json:"label,omitempty"`
	Value []any  `json:"value"`
}

// CustomField represents a custom field in KeeperSecurity with its type, label and value.
type CustomField struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Value []any  `json:"value"`
}

// File represents a file stored in KeeperSecurity with its title and content.
type File struct {
	Title   string `json:"type"`
	Content string `json:"content"`
}

// Secret represents a KeeperSecurity secret with its metadata and content.
type Secret struct {
	Title  string        `json:"title"`
	Type   string        `json:"type"`
	Fields []Field       `json:"fields"`
	Custom []CustomField `json:"custom"`
	Files  []File        `json:"files"`
}

// Validate performs validation of the Keeper Security client configuration.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// GetSecret retrieves a secret from Keeper Security by ID or name.
// It first attempts to find the secret by ID, then falls back to name lookup.
// The name lookup must be opted in by setting getByTitleFallback on the provider.
// A record that does not exist yields esv1.NoSecretErr, which is what the
// reconciler keys deletionPolicy off.
func (c *Client) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secret, err := c.findByIDWithNameFallback(ref.Key)
	if err != nil {
		return nil, err
	}
	return secret.getItem(ref)
}

// GetSecretMap retrieves a secret from Keeper Security and returns it as a map.
func (c *Client) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.findByIDWithNameFallback(ref.Key)
	if err != nil {
		return nil, err
	}
	return secret.getItems(ref)
}

// It first attempts to find the secret by ID, then falls back to name lookup.
// The name lookup must be opted in by setting getByTitleFallback on the provider.
func (c *Client) findByIDWithNameFallback(key string) (*Secret, error) {
	record, err := c.findSecretByID(key)
	if err != nil {
		return nil, err
	}

	if record == nil && c.getByTitleFallback {
		records, err := c.ksmClient.GetSecretsByTitle(key)
		metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityGetSecretsByTitle, err)
		if err != nil {
			return nil, err
		}

		if len(records) > 1 {
			return nil, errors.New(errKeeperSecuritySecretNotUnique)
		} else if len(records) == 1 {
			record = records[0]
		}
	}

	if record == nil {
		// Only a genuinely absent record gets the sentinel; the API failures
		// wrapped by findSecretByID/GetSecretsByTitle above must stay generic so
		// an outage is not mistaken for a deletion.
		return nil, fmt.Errorf(errKeeperSecurityRecordNotFound, esv1.NoSecretErr, key)
	}

	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetAllSecrets retrieves all secrets from Keeper Security that match the given criteria.
func (c *Client) GetAllSecrets(_ context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errors.New(errTagsNotImplemented)
	}
	if ref.Path != nil {
		return nil, errors.New(errPathNotImplemented)
	}
	secretData := make(map[string][]byte)
	records, err := c.findSecrets()
	// GetAllSecrets retrieves all secrets from Keeper Security that match the given criteria.
	// Currently supports filtering by name pattern only.
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		secret, err := c.getValidKeeperSecret(record)
		if err != nil {
			return nil, err
		}
		match, err := regexp.MatchString(ref.Name.RegExp, secret.Title)
		if err != nil {
			return nil, fmt.Errorf(errInvalidRegex, ref.Name.RegExp, err)
		}
		if !match {
			continue
		}
		secretData[secret.Title], err = secret.getItem(esv1.ExternalSecretDataRemoteRef{})
		if err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

// Close implements cleanup operations for the Keeper Security client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// PushSecret creates or updates a secret in Keeper Security.
func (c *Client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	keeperRecordData, err := c.buildRecord(secret, data)
	if err != nil {
		return err
	}

	record, err := c.findSecretByName(keeperRecordData.Title)
	if err != nil {
		return err
	}

	if record != nil {
		if record.Type() != externalSecretType {
			return fmt.Errorf(errInvalidSecretType, externalSecretType, record.Title(), record.Type())
		}
		return c.updateSecret(record, keeperRecordData, isWholeSecretData(data))
	}

	_, err = c.createSecret(keeperRecordData)
	return err
}

func isWholeSecretData(data esv1.PushSecretData) bool {
	return data.GetProperty() == "" && data.GetSecretKey() == ""
}

// buildRecord builds the Keeper record as outlined by the rules in
// https://external-secrets.io/latest/guides/pushsecrets/
func (c *Client) buildRecord(secret *corev1.Secret, data esv1.PushSecretData) (*Secret, error) {
	hasProperty := data.GetProperty() != ""
	hasSecretKey := data.GetSecretKey() != ""

	switch {
	case !hasProperty && !hasSecretKey:
		return buildWholeRecord(secret, data)
	case hasProperty && !hasSecretKey:
		return buildPropertyRecord(secret, data)
	case hasProperty && hasSecretKey:
		return buildPropertyFromSecretKeyRecord(secret, data)
	case !hasProperty && hasSecretKey:
		return c.buildLegacyRecord(secret, data)
	}

	return nil, fmt.Errorf(errKeeperSecurityUnexpectedFieldState, hasProperty, hasSecretKey)
}

func buildWholeRecord(secret *corev1.Secret, data esv1.PushSecretData) (*Secret, error) {
	remoteKey := data.GetRemoteKey()
	if strings.Contains(remoteKey, "/") {
		return nil, fmt.Errorf(errKeeperSecurityWholeRecordKeyHasSlash, remoteKey)
	}
	return buildSecret(remoteKey, secret.Data), nil
}

func buildPropertyRecord(secret *corev1.Secret, data esv1.PushSecretData) (*Secret, error) {
	// json.Marshal base64-encodes []byte values, so convert to strings first to
	// keep the stored property readable by ExternalSecrets.
	stringData := make(map[string]string, len(secret.Data))
	for key, value := range secret.Data {
		// string(value) would silently corrupt non-UTF-8 bytes (json.Marshal
		// replaces them with U+FFFD), so reject them instead.
		if !utf8.Valid(value) {
			return nil, fmt.Errorf(errKeeperSecurityNonUTF8Value, key)
		}
		stringData[key] = string(value)
	}
	secretContent, err := json.Marshal(stringData)
	if err != nil {
		return nil, err
	}

	return buildSecret(data.GetRemoteKey(), map[string][]byte{data.GetProperty(): secretContent}), nil
}

func buildPropertyFromSecretKeyRecord(secret *corev1.Secret, data esv1.PushSecretData) (*Secret, error) {
	secretValue, ok := secret.Data[data.GetSecretKey()]
	if !ok {
		return nil, fmt.Errorf(errKeeperSecurityNoFields, data.GetSecretKey())
	}

	return buildSecret(data.GetRemoteKey(), map[string][]byte{data.GetProperty(): secretValue}), nil
}

func (c *Client) buildLegacyRecord(secret *corev1.Secret, data esv1.PushSecretData) (*Secret, error) {
	parts, err := buildSecretNameAndKey(data)
	if err != nil {
		return nil, err
	}

	value := secret.Data[data.GetSecretKey()]
	fieldType, isStandard := keeperFieldType(parts[1])
	if !isStandard {
		return buildSecret(parts[0], map[string][]byte{parts[1]: value}), nil
	}

	// Legacy record/key references historically map standard keys to Keeper's
	// default fields. Keep them unlabelled so existing consumers see the same
	// Login, Password, and URL fields.
	return &Secret{
		Type:  externalSecretType,
		Title: parts[0],
		Fields: []Field{{
			Type:  fieldType,
			Value: []any{string(value)},
		}},
		Custom: []CustomField{},
	}, nil
}

func buildSecret(title string, data map[string][]byte) *Secret {
	recordData := Secret{Type: externalSecretType, Title: title, Fields: []Field{}, Custom: []CustomField{}}
	// Iterate in sorted order so whole-record replacement keeps a stable field order.
	for _, key := range slices.Sorted(maps.Keys(data)) {
		value := data[key]
		fieldType, isStandard := keeperFieldType(key)
		if isStandard {
			recordData.Fields = append(recordData.Fields, Field{Type: fieldType, Label: key, Value: []any{string(value)}})
		} else {
			recordData.Custom = append(recordData.Custom, CustomField{Type: secretType, Label: key, Value: []any{string(value)}})
		}
	}

	return &recordData
}

func keeperFieldType(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "login", "username":
		return LoginType, true
	case PasswordType:
		return PasswordType, true
	case "url", "baseurl":
		return URLType, true
	default:
		return secretType, false
	}
}

// DeleteSecret removes a secret from Keeper Security.
func (c *Client) DeleteSecret(_ context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	target := resolvePushTarget(remoteRef)
	if target.fieldKey == "" {
		return c.deleteWholeSecret(target.secretName)
	}
	return c.deleteSecretField(target.secretName, target.fieldKey)
}

type keeperPushTarget struct {
	secretName string
	fieldKey   string
}

func resolvePushTarget(remoteRef esv1.PushSecretRemoteRef) keeperPushTarget {
	if property := remoteRef.GetProperty(); property != "" {
		return keeperPushTarget{secretName: remoteRef.GetRemoteKey(), fieldKey: property}
	}
	parts, err := buildSecretNameAndKey(remoteRef)
	if err != nil {
		return keeperPushTarget{secretName: remoteRef.GetRemoteKey()}
	}

	return keeperPushTarget{secretName: parts[0], fieldKey: legacyFieldKey(parts[1])}
}

func legacyFieldKey(key string) string {
	if fieldType, isStandard := keeperFieldType(key); isStandard {
		return fieldType
	}
	return key
}

func (c *Client) deleteWholeSecret(secretName string) error {
	secret, err := c.findSecretByName(secretName)
	if err != nil {
		return err
	}
	if secret == nil {
		return nil
	}
	if secret.Type() != externalSecretType {
		return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
	}
	_, err = c.ksmClient.DeleteSecrets([]string{secret.Uid})
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityDeleteSecrets, err)
	return err
}

func (c *Client) deleteSecretField(secretName, fieldKey string) error {
	secret, err := c.findSecretByName(secretName)
	if err != nil {
		return err
	}
	if secret == nil {
		return nil
	}
	if secret.Type() != externalSecretType {
		return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
	}
	if !removeKeeperRecordField(secret, fieldKey) {
		return nil
	}
	if keeperRecordIsEmpty(secret) {
		_, err = c.ksmClient.DeleteSecrets([]string{secret.Uid})
		metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityDeleteSecrets, err)
		return err
	}

	syncRecordRawJSON(secret)
	err = c.ksmClient.Save(secret)
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecuritySave, err)
	return err
}

func removeKeeperRecordField(record *ksm.Record, key string) bool {
	fieldType, isStandard := keeperFieldType(key)
	section := "custom"
	if isStandard {
		section = "fields"
	}
	fields, ok := record.RecordDict[section].([]any)
	if !ok {
		return false
	}

	updatedFields := make([]any, 0, len(fields))
	removed := false
	for _, field := range fields {
		fieldMap, ok := field.(map[string]any)
		if !removed && ok && (isStandard && fieldMap["type"] == fieldType && matchesStandardField(fieldMap, key, fieldType) || !isStandard && fieldMap["label"] == key) {
			removed = true
			continue
		}
		updatedFields = append(updatedFields, field)
	}
	if removed {
		record.RecordDict[section] = updatedFields
	}

	return removed
}

func keeperRecordIsEmpty(record *ksm.Record) bool {
	for _, section := range []string{"fields", "custom"} {
		if fields, ok := record.RecordDict[section].([]any); ok && len(fields) > 0 {
			return false
		}
	}
	return true
}

// SecretExists checks if a secret exists in Keeper Security.
func (c *Client) SecretExists(_ context.Context, ref esv1.PushSecretRemoteRef) (bool, error) {
	target := resolvePushTarget(ref)
	record, err := c.findSecretByName(target.secretName)
	if err != nil || record == nil {
		return record != nil, err
	}
	if target.fieldKey == "" {
		return true, nil
	}

	return keeperRecordHasField(record, target.fieldKey), nil
}

func keeperRecordHasField(record *ksm.Record, key string) bool {
	fieldType, isStandard := keeperFieldType(key)
	if !isStandard {
		return len(record.GetCustomFieldsByLabel(key)) > 0
	}
	for _, field := range record.GetFieldsByType(fieldType) {
		if matchesStandardField(field, key, fieldType) {
			return true
		}
	}

	return false
}

func buildSecretNameAndKey(remoteRef esv1.PushSecretRemoteRef) ([]string, error) {
	parts := strings.Split(remoteRef.GetRemoteKey(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf(errInvalidRemoteRefKey, remoteRef.GetRemoteKey())
	}
	return parts, nil
}

func (c *Client) createSecret(secret *Secret) (string, error) {
	externalSecretRecord := ksm.NewRecordCreate(externalSecretType, secret.Title)
	if c.folderID == "" {
		return "", errors.New(errKeeperSecurityMissingFolderIDForCreate)
	}
	for _, field := range secret.Fields {
		externalSecretRecord.Fields = append(externalSecretRecord.Fields, field)
	}
	for _, field := range secret.Custom {
		externalSecretRecord.Custom = append(externalSecretRecord.Custom, field)
	}

	uid, err := c.ksmClient.CreateSecretWithRecordData("", c.folderID, externalSecretRecord)
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityCreateSecretWithRecordData, err)
	return uid, err
}

func (c *Client) updateSecret(currentRecord *ksm.Record, recordData *Secret, replaceFields bool) error {
	if replaceFields {
		if err := replaceKeeperRecordFields(currentRecord, recordData); err != nil {
			return err
		}
	} else if err := mergeKeeperRecordFields(currentRecord, recordData); err != nil {
		return err
	}

	err := c.ksmClient.Save(currentRecord)
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecuritySave, err)
	return err
}

func replaceKeeperRecordFields(currentRecord *ksm.Record, recordData *Secret) error {
	desiredJSON, err := json.Marshal(recordData)
	if err != nil {
		return err
	}
	var desiredRecord map[string]any
	if err := json.Unmarshal(desiredJSON, &desiredRecord); err != nil {
		return err
	}
	currentRecord.RecordDict["fields"] = desiredRecord["fields"]
	currentRecord.RecordDict["custom"] = desiredRecord["custom"]
	syncRecordRawJSON(currentRecord)
	return nil
}

func syncRecordRawJSON(record *ksm.Record) {
	record.RawJson = ksm.DictToJson(record.RecordDict)
}

func mergeKeeperRecordFields(currentRecord *ksm.Record, recordData *Secret) error {
	for _, field := range recordData.Fields {
		if err := mergeStandardField(currentRecord, field); err != nil {
			return err
		}
	}

	for _, field := range recordData.Custom {
		if err := mergeCustomField(currentRecord, field); err != nil {
			return err
		}
	}

	return nil
}

func mergeStandardField(currentRecord *ksm.Record, field Field) error {
	key := field.Label
	if key == "" {
		key = field.Type
	}
	if len(currentRecord.GetFieldsByLabel(field.Label)) > 0 {
		return currentRecord.SetStandardFieldValue(key, field.Value)
	}
	if (field.Label == "" || field.Label == field.Type) && hasUnlabelledStandardField(currentRecord, field.Type) {
		return currentRecord.SetStandardFieldValue(field.Type, field.Value)
	}
	return insertStandardField(currentRecord, field)
}

func mergeCustomField(currentRecord *ksm.Record, field CustomField) error {
	if len(currentRecord.GetCustomFieldsByLabel(field.Label)) > 0 {
		return currentRecord.SetCustomFieldValue(field.Label, field.Value)
	}
	values, err := customFieldStringValues(field)
	if err != nil {
		return err
	}
	return currentRecord.AddCustomField(ksm.Secret{KeeperRecordField: ksm.KeeperRecordField{Type: secretType, Label: field.Label}, Value: values})
}

func customFieldStringValues(field CustomField) ([]string, error) {
	values := make([]string, len(field.Value))
	for index, value := range field.Value {
		stringValue, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("custom field %q requires string values", field.Label)
		}
		values[index] = stringValue
	}
	return values, nil
}

func insertStandardField(record *ksm.Record, field Field) error {
	value := ""
	if len(field.Value) > 0 {
		var ok bool
		value, ok = field.Value[0].(string)
		if !ok {
			return fmt.Errorf("standard field %q requires string values", field.Type)
		}
	}
	var keeperField any
	switch field.Type {
	case LoginType:
		keeperField = ksm.NewLogin(value)
		keeperField.(*ksm.Login).Label = field.Label
	case PasswordType:
		keeperField = ksm.NewPassword(value)
		keeperField.(*ksm.Password).Label = field.Label
	case URLType:
		keeperField = ksm.NewUrl(value)
		keeperField.(*ksm.Url).Label = field.Label
	default:
		return fmt.Errorf("unsupported standard field type %q", field.Type)
	}
	if err := record.InsertField("fields", keeperField); err != nil {
		return err
	}
	key := field.Label
	if key == "" {
		key = field.Type
	}
	return record.SetStandardFieldValue(key, field.Value)
}

func hasUnlabelledStandardField(record *ksm.Record, fieldType string) bool {
	for _, field := range record.GetFieldsByType(fieldType) {
		if matchesStandardField(field, fieldType, fieldType) {
			return true
		}
	}
	return false
}

func matchesStandardField(field map[string]any, key, fieldType string) bool {
	if label, ok := field["label"].(string); ok && label != "" {
		return label == key
	}
	return key == fieldType
}

func (c *Client) getValidKeeperSecret(secret *ksm.Record) (*Secret, error) {
	keeperSecret := Secret{}
	err := json.Unmarshal([]byte(secret.RawJson), &keeperSecret)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecurityInvalidSecretInvalidFormat, err)
	}
	keeperSecret.addFiles(secret.Files)
	err = keeperSecret.validate()
	if err != nil {
		return nil, err
	}

	return &keeperSecret, nil
}

func (c *Client) findSecrets() ([]*ksm.Record, error) {
	records, err := c.ksmClient.GetSecrets([]string{})
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityGetSecrets, err)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretsNotFound, err)
	}

	return records, nil
}

func (c *Client) findSecretByID(id string) (*ksm.Record, error) {
	records, err := c.ksmClient.GetSecrets([]string{id})
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityGetSecrets, err)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records[0], nil
}

func (c *Client) findSecretByName(name string) (*ksm.Record, error) {
	records, err := c.ksmClient.GetSecretsByTitle(name)
	metrics.ObserveAPICall(ProviderKeeperSecurity, CallKeeperSecurityGetSecretsByTitle, err)
	if err != nil {
		return nil, err
	}

	// filter in-place, preserve only records of type externalSecretType
	n := 0
	for _, record := range records {
		if record.Type() == externalSecretType {
			records[n] = record
			n++
		}
	}
	records = records[:n]

	// record not found is not an error - handled differently:
	// PushSecret will create new record instead
	// DeleteSecret will consider record already deleted (no error)
	if len(records) == 0 {
		return nil, nil
	} else if len(records) == 1 {
		return records[0], nil
	}

	// len(records) > 1
	return nil, fmt.Errorf(errKeeperSecuritySecretNotUnique, name)
}

func (s *Secret) validate() error {
	fields := make(map[string]int)
	for _, field := range s.Fields {
		fieldKey := field.Label
		if fieldKey == "" {
			fieldKey = field.Type
		}
		fields[fieldKey]++
	}

	for _, customField := range s.Custom {
		fields[customField.Label]++
	}

	for _, file := range s.Files {
		fields[file.Title]++
	}
	var duplicates []string
	for key, ocurrences := range fields {
		if ocurrences > 1 {
			duplicates = append(duplicates, key)
		}
	}
	if len(duplicates) != 0 {
		return fmt.Errorf(errKeeperSecurityInvalidSecretDuplicatedKey, strings.Join(duplicates, ", "))
	}

	return nil
}

func (s *Secret) addFiles(keeperFiles []*ksm.KeeperFile) {
	for _, f := range keeperFiles {
		s.Files = append(
			s.Files,
			File{
				Title:   f.Title,
				Content: string(f.GetFileData()),
			},
		)
	}
}

func (s *Secret) getItem(ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Property != "" {
		return s.getProperty(ref.Property)
	}
	secret, err := s.toString()

	return []byte(secret), err
}

func (s *Secret) getItems(ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	if ref.Property != "" {
		value, err := s.getProperty(ref.Property)
		if err != nil {
			return nil, err
		}
		secretData[ref.Property] = value

		return secretData, nil
	}

	fields := s.getFields()
	maps.Copy(secretData, fields)
	customFields := s.getCustomFields()
	maps.Copy(secretData, customFields)
	files := s.getFiles()
	maps.Copy(secretData, files)

	if len(secretData) == 0 {
		return nil, fmt.Errorf(errKeeperSecurityNoFields, s.Title)
	}

	return secretData, nil
}

func getFieldValue(value []any) []byte {
	if len(value) < 1 {
		return []byte{}
	}
	if len(value) == 1 {
		res, _ := json.Marshal(value[0])
		if str, ok := value[0].(string); ok {
			res = []byte(str)
		}
		return res
	}
	res, _ := json.Marshal(value)
	return res
}

func (s *Secret) getField(key string) ([]byte, error) {
	for _, field := range s.Fields {
		fieldKey := field.Label
		if fieldKey == "" {
			fieldKey = field.Type
		}
		if fieldKey == key && field.Type != keeperSecurityFileRef && field.Type != keeperSecurityMfa && len(field.Value) > 0 {
			return getFieldValue(field.Value), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *Secret) getFields() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, field := range s.Fields {
		if len(field.Value) > 0 {
			fieldKey := field.Label
			if fieldKey == "" {
				fieldKey = field.Type
			}
			secretData[fieldKey] = getFieldValue(field.Value)
		}
	}

	return secretData
}

func (s *Secret) getCustomField(key string) ([]byte, error) {
	for _, field := range s.Custom {
		if field.Label == key && len(field.Value) > 0 {
			return getFieldValue(field.Value), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *Secret) getCustomFields() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, field := range s.Custom {
		if len(field.Value) > 0 {
			secretData[field.Label] = getFieldValue(field.Value)
		}
	}

	return secretData
}

func (s *Secret) getFile(key string) ([]byte, error) {
	for _, file := range s.Files {
		if file.Title == key {
			return []byte(file.Content), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *Secret) getProperty(key string) ([]byte, error) {
	field, _ := s.getField(key)
	if field != nil {
		return field, nil
	}
	customField, _ := s.getCustomField(key)
	if customField != nil {
		return customField, nil
	}
	file, _ := s.getFile(key)
	if file != nil {
		return file, nil
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidProperty, s.Title, key)
}

func (s *Secret) getFiles() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, file := range s.Files {
		secretData[file.Title] = []byte(file.Content)
	}

	return secretData
}

func (s *Secret) toString() (string, error) {
	secretJSON, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf(errInvalidJSONSecret, s.Title, err)
	}

	return string(secretJSON), nil
}
