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

package keepersecurity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

const (
	errKeeperSecuritySecretsNotFound            = "unable to find secrets. %w"
	errKeeperSecuritySecretNotFound             = "unable to find secret %s. Error: %w"
	errKeeperSecuritySecretNotUnique            = "more than 1 secret %s found"
	errKeeperSecurityNoSecretsFound             = "no secrets found"
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
	errFieldNotFound                            = "secret %s does not contain any custom field with label %s"

	externalSecretType = "externalSecrets"
	secretType         = "secret"
	// LoginType represents the login field type.
	LoginType = "login"
	// LoginTypeExpr is the regex expression for matching login/username fields.
	LoginTypeExpr = "login|username"
	// PasswordType represents the password field type.
	PasswordType = "password"
	// URLTypeExpr is the regex expression for matching URL/baseurl fields.
	URLTypeExpr = "url|baseurl"
	// URLType represents the URL field type.
	URLType = "url"
)

// Client represents a KeeperSecurity client that can interact with the KeeperSecurity API.
type Client struct {
	ksmClient SecurityClient
	folderID  string
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

// Field represents a KeeperSecurity field with its type and value.
type Field struct {
	Type  string `json:"type"`
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

// GetSecret retrieves a secret from Keeper Security by its ID.
func (c *Client) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	record, err := c.findSecretByID(ref.Key)
	if err != nil {
		return nil, err
	}
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}
	// GetSecret retrieves a secret from Keeper Security by its ID.
	// If ref.Property is specified, it returns only that property's value.

	return secret.getItem(ref)
}

// GetSecretMap retrieves a secret from Keeper Security and returns it as a map.
func (c *Client) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	record, err := c.findSecretByID(ref.Key)
	if err != nil {
		return nil, err
	}
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}
	// GetSecretMap retrieves a secret from Keeper Security and returns it as a map.
	// If ref.Property is specified, it returns only that property as a map entry.

	return secret.getItems(ref)
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
	if data.GetSecretKey() == "" {
		return errors.New("pushing the whole secret is not yet implemented")
	}

	// Close implements cleanup operations for the Keeper Security client
	value := secret.Data[data.GetSecretKey()]
	parts, err := c.buildSecretNameAndKey(data)
	if err != nil {
		return err
		// PushSecret creates or updates a secret in Keeper Security.
		// Currently only supports pushing individual secret values, not entire secrets.
	}

	record, err := c.findSecretByName(parts[0])
	if err != nil {
		return err
	}

	if record != nil {
		if record.Type() != externalSecretType {
			return fmt.Errorf(errInvalidSecretType, externalSecretType, record.Title(), record.Type())
		}
		return c.updateSecret(record, parts[1], value)
	}

	_, err = c.createSecret(parts[0], parts[1], value)
	return err
}

// DeleteSecret removes a secret from Keeper Security.
func (c *Client) DeleteSecret(_ context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	parts, err := c.buildSecretNameAndKey(remoteRef)
	if err != nil {
		return err
	}
	secret, err := c.findSecretByName(parts[0])
	if err != nil {
		return err
	} else if secret == nil {
		// DeleteSecret removes a secret from Keeper Security.
		// Returns nil if the secret doesn't exist (already deleted).
		return nil // not found == already deleted (success)
	}

	if secret.Type() != externalSecretType {
		return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
	}
	_, err = c.ksmClient.DeleteSecrets([]string{secret.Uid})
	return err
}

// SecretExists checks if a secret exists in Keeper Security.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *Client) buildSecretNameAndKey(remoteRef esv1.PushSecretRemoteRef) ([]string, error) {
	parts := strings.Split(remoteRef.GetRemoteKey(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf(errInvalidRemoteRefKey, remoteRef.GetRemoteKey())
	}
	// SecretExists checks if a secret exists in Keeper Security.
	// This method is not implemented yet.

	return parts, nil
}

func (c *Client) createSecret(name, key string, value []byte) (string, error) {
	normalizedKey := strings.ToLower(key)
	externalSecretRecord := ksm.NewRecordCreate(externalSecretType, name)
	login := regexp.MustCompile(LoginTypeExpr)
	pass := regexp.MustCompile(PasswordType)
	url := regexp.MustCompile(URLTypeExpr)

	switch {
	case login.MatchString(normalizedKey):
		externalSecretRecord.Fields = append(externalSecretRecord.Fields,
			ksm.NewLogin(string(value)),
		)
	case pass.MatchString(normalizedKey):
		externalSecretRecord.Fields = append(externalSecretRecord.Fields,
			ksm.NewPassword(string(value)),
		)
	case url.MatchString(normalizedKey):
		externalSecretRecord.Fields = append(externalSecretRecord.Fields,
			ksm.NewUrl(string(value)),
		)
	default:
		field := ksm.KeeperRecordField{Type: secretType, Label: key}
		externalSecretRecord.Custom = append(externalSecretRecord.Custom,
			ksm.Secret{KeeperRecordField: field, Value: []string{string(value)}},
		)
	}

	return c.ksmClient.CreateSecretWithRecordData("", c.folderID, externalSecretRecord)
}

func (c *Client) updateSecret(secret *ksm.Record, key string, value []byte) error {
	normalizedKey := strings.ToLower(key)
	login := regexp.MustCompile(LoginTypeExpr)
	pass := regexp.MustCompile(PasswordType)
	url := regexp.MustCompile(URLTypeExpr)
	custom := false

	switch {
	case login.MatchString(normalizedKey):
		secret.SetFieldValueSingle(LoginType, string(value))
	case pass.MatchString(normalizedKey):
		secret.SetPassword(string(value))
	case url.MatchString(normalizedKey):
		secret.SetFieldValueSingle(URLType, string(value))
	default:
		custom = true
	}
	if custom {
		field := secret.GetCustomFieldValueByLabel(key)
		if field != "" {
			secret.SetCustomFieldValueSingle(key, string(value))
		} else {
			return fmt.Errorf(errFieldNotFound, secret.Title(), key)
		}
	}

	return c.ksmClient.Save(secret)
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
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretsNotFound, err)
	}

	return records, nil
}

func (c *Client) findSecretByID(id string) (*ksm.Record, error) {
	records, err := c.ksmClient.GetSecrets([]string{id})
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
	}

	if len(records) == 0 {
		return nil, errors.New(errKeeperSecurityNoSecretsFound)
	}

	return records[0], nil
}

func (c *Client) findSecretByName(name string) (*ksm.Record, error) {
	records, err := c.ksmClient.GetSecretsByTitle(name)
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
		fields[field.Type]++
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
		if field.Type == key && field.Type != keeperSecurityFileRef && field.Type != keeperSecurityMfa && len(field.Value) > 0 {
			return getFieldValue(field.Value), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *Secret) getFields() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, field := range s.Fields {
		if len(field.Value) > 0 {
			secretData[field.Type] = getFieldValue(field.Value)
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
