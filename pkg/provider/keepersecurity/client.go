package keepersecurity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ksm "github.com/keeper-security/secrets-manager-go/core"
	"golang.org/x/exp/maps"
	"regexp"
	"strings"
)

const (
	errKeeperSecuritySecretsNotFound            = "Unable to find secrets. %w"
	errKeeperSecuritySecretNotFound             = "Unable to find secret %s. Error: %w"
	errKeeperSecuritySecretNotUnique            = "More than 1 secret %s found"
	errKeeperSecurityNoSecretsFound             = "No secrets found"
	errKeeperSecurityInvalidSecretInvalidFormat = "Invalid secret. Invalid format: %w"
	errKeeperSecurityInvalidSecretDuplicatedKey = "Invalid Secret. Following keys are duplicated %s"
	errKeeperSecurityInvalidProperty            = "Invalid Property. Secret %s does not have any key matching %s"
	errKeeperSecurityInvalidField               = "Invalid Field. Key %s does not exists"
	errKeeperSecurityNoFields                   = "Invalid Secret. Secret %s does not contain any valid field/file"
	keeperSecurityFileRef                       = "fileRef"
	keeperSecurityMfa                           = "oneTimeCode"
	errTagsNotImplemented                       = "'find.tags' is not implemented in the KeeperSecurity provider"
	errPathNotImplemented                       = "'find.path' is not implemented in the KeeperSecurity provider"
	errInvalidJsonSecret                        = "Invalid Secret. Secret %s can not be converted to JSON. %w"
	errInvalidRegex                             = "find.name.regex. Invalid Regular expresion %s. %w"
	errInvalidRemoteRefKey                      = "match.remoteRef.remoteKey. Invalid format. Format should match secretName/key got %s"
	errInvalidSecretType                        = "ESO can only push/delete %s record types. Secret %s is type %s"
	errFieldNotFound                            = "Secret %s does not contain any custom field with label %s"

	externalSecretType = "externalSecrets"
	secretType         = "secret"
	LoginType          = "login"
	LoginTypeExpr      = "login|username"
	PasswordType       = "password"
	UrlTypeExpr        = "url|baseurl"
	UrlType            = "url"
)

type Client struct {
	ksmClient KeeperSecurityClient
	folderID  string
}

type KeeperSecurityClient interface {
	GetSecrets(filter []string) ([]*ksm.Record, error)
	GetSecretByTitle(recordTitle string) (*ksm.Record, error)
	CreateSecretWithRecordData(recUID, folderUID string, recordData *ksm.RecordCreate) (string, error)
	DeleteSecrets(recrecordUids []string) (map[string]string, error)
	Save(record *ksm.Record) error
}

type KeeperSecurityField struct {
	Type  string   `json:"type"`
	Value []string `json:"value"`
}

type KeeperSecurityCustomField struct {
	Type  string   `json:"type"`
	Label string   `json:"label"`
	Value []string `json:"value"`
}

type KeeperSecurityFile struct {
	Title   string `json:"type"`
	Content string `json:"content"`
}

type KeeperSecuritySecret struct {
	Title  string                      `json:"title"`
	Type   string                      `json:"type"`
	Fields []KeeperSecurityField       `json:"fields"`
	Custom []KeeperSecurityCustomField `json:"custom"`
	Files  []KeeperSecurityFile        `json:"files"`
}

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {

	return esv1beta1.ValidationResultReady, nil
}

func (c *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	record, err := c.findSecretById(ref.Key)
	if err != nil {
		return nil, err
	}
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}

	return secret.getItem(ref)
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	record, err := c.findSecretById(ref.Key)
	if err != nil {
		return nil, err
	}
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}

	return secret.getItems(ref)
}

func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, fmt.Errorf(errTagsNotImplemented)
	}
	if ref.Path != nil {
		return nil, fmt.Errorf(errPathNotImplemented)
	}
	secretData := make(map[string][]byte)
	records, err := c.findSecrets()
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
		secretData[secret.Title], err = secret.getItem(esv1beta1.ExternalSecretDataRemoteRef{})
		if err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

func (c *Client) Close(ctx context.Context) error {

	return nil
}

func (c *Client) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	parts, err := c.buildSecretNameAndKey(remoteRef)
	if err != nil {
		return err
	}
	secret, err := c.findSecretByName(parts[0])
	if err != nil {
		_, err = c.createSecret(parts[0], parts[1], value)
		if err != nil {
			return err
		}
	}
	if secret != nil {
		if secret.Type() != externalSecretType {
			return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
		}
		err = c.updateSecret(secret, parts[1], value)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {

	parts, err := c.buildSecretNameAndKey(remoteRef)
	if err != nil {
		return err
	}
	secret, err := c.findSecretByName(parts[0])
	if err != nil {
		return err
	}
	if secret.Type() != externalSecretType {
		return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
	}
	_, err = c.ksmClient.DeleteSecrets([]string{secret.Uid})
	if err != nil {
		return nil
	}

	return nil
}

func (c *Client) buildSecretNameAndKey(remoteRef esv1beta1.PushRemoteRef) ([]string, error) {
	parts := strings.Split(remoteRef.GetRemoteKey(), "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf(errInvalidRemoteRefKey, remoteRef.GetRemoteKey())
	}

	return parts, nil
}

func (c *Client) createSecret(name string, key string, value []byte) (string, error) {
	normalizedKey := strings.ToLower(key)
	externalSecretRecord := ksm.NewRecordCreate(externalSecretType, name)
	login := regexp.MustCompile(LoginTypeExpr)
	pass := regexp.MustCompile(PasswordType)
	url := regexp.MustCompile(UrlTypeExpr)

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
	url := regexp.MustCompile(UrlTypeExpr)

	switch {
	case login.MatchString(normalizedKey):
		secret.SetFieldValueSingle(LoginType, string(value))
	case pass.MatchString(normalizedKey):
		secret.SetPassword(string(value))
	case url.MatchString(normalizedKey):
		secret.SetFieldValueSingle(UrlType, string(value))
	default:
		field := secret.GetCustomFieldValueByLabel(key)
		if field == "" {
			return fmt.Errorf(errFieldNotFound, secret.Title(), key)
		} else {
			secret.SetCustomFieldValueSingle(key, string(value))
		}
	}

	return c.ksmClient.Save(secret)
}

func (c *Client) getValidKeeperSecret(secret *ksm.Record) (*KeeperSecuritySecret, error) {

	keeperSecret := KeeperSecuritySecret{}
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

func (c *Client) findSecretById(id string) (*ksm.Record, error) {

	records, err := c.ksmClient.GetSecrets([]string{id})
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
	}

	if len(records) == 0 {
		return nil, errors.New(errKeeperSecurityNoSecretsFound)
	}
	if len(records) >= 1 {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotUnique, id)
	}

	return records[0], nil
}

func (c *Client) findSecretByName(name string) (*ksm.Record, error) {

	record, err := c.ksmClient.GetSecretByTitle(name)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (s *KeeperSecuritySecret) validate() error {
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
	duplicates := []string{}
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

func (s *KeeperSecuritySecret) addFiles(keeperFiles []*ksm.KeeperFile) {
	for _, f := range keeperFiles {
		s.Files = append(
			s.Files,
			KeeperSecurityFile{
				Title:   f.Title,
				Content: string(f.GetFileData()),
			},
		)
	}
}

func (s *KeeperSecuritySecret) getItem(ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Property != "" {
		return s.getProperty(ref.Property)
	}
	secret, err := s.toString()

	return []byte(secret), err
}

func (s *KeeperSecuritySecret) getItems(ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

func (s *KeeperSecuritySecret) getField(key string) ([]byte, error) {
	for _, field := range s.Fields {
		if field.Type == key && field.Type != keeperSecurityFileRef && field.Type != keeperSecurityMfa && len(field.Value) > 0 {
			return []byte(field.Value[0]), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *KeeperSecuritySecret) getFields() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, field := range s.Fields {
		if len(field.Value) > 0 {
			secretData[field.Type] = []byte(field.Value[0])
		}
	}

	return secretData
}

func (s *KeeperSecuritySecret) getCustomField(key string) ([]byte, error) {
	for _, field := range s.Custom {
		if field.Label == key && len(field.Value) > 0 {
			return []byte(field.Value[0]), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *KeeperSecuritySecret) getCustomFields() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, field := range s.Custom {
		if len(field.Value) > 0 {
			secretData[field.Label] = []byte(field.Value[0])
		}
	}

	return secretData
}

func (s *KeeperSecuritySecret) getFile(key string) ([]byte, error) {
	for _, file := range s.Files {
		if file.Title == key {
			return []byte(file.Content), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidField, key)
}

func (s *KeeperSecuritySecret) getProperty(key string) ([]byte, error) {
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

func (s *KeeperSecuritySecret) getFiles() map[string][]byte {
	secretData := make(map[string][]byte)
	for _, file := range s.Files {
		secretData[file.Title] = []byte(file.Content)
	}

	return secretData
}

func (s *KeeperSecuritySecret) toString() (string, error) {

	secretJson, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf(errInvalidJsonSecret, s.Title, err)
	}

	return string(secretJson), nil
}
