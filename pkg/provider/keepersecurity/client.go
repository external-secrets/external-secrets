package keepersecurity

import (
	"context"
	"encoding/json"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	ksm "github.com/keeper-security/secrets-manager-go/core"
	"regexp"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sort"
)

const (
	errKeeperSecuritySecretNotFound             = "Unable to find secret %s. Error: %w"
	errKeeperSecuritySecretNotUnique            = "More than 1 secret %s found"
	errKeeperSecurityInvalidSecretInvalidFormat = "Invalid secret. Invalid format: %w"
	errKeeperSecurityInvalidSecretDuplicatedKey = "Invalid Secret. Keys are not unique"
	errKeeperSecurityInvalidProperty            = "Invalid Property. Secret %s does not have any key matching %s"
	errKeeperSecurityNoFields                   = "Invalid Secret. Secret %s does not contain any valid field/file"
	keeperSecurityFileRef                       = "fileRef"
	keeperSecurityMfa                           = "oneTimeCode"
	errTagsNotImplemented                       = "'find.tags' is not implemented in the KeeperSecurity provider"
	errPathNotImplemented                       = "'find.path' is not implemented in the KeeperSecurity provider"
	errInvalidJsonSecret                        = "Invalid Secret. Secret %s can not be converted to JSON. %w"
	errInvalidRegex                             = "find.name.regex. Invalid Regular expresion %s. %w"
)

type Client struct {
	ksmClient *ksm.SecretsManager
	kube      kclient.Client
	store     *esv1beta1.KeeperSecurityProvider
	namespace string
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
	secret, err := c.getValidKeeperSecret(record)

	if err != nil {
		return nil, err
	}

	return secret.getField(ref)
}

func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	record, err := c.findSecretById(ref.Key)
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}

	return secret.getFields(ref)
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
		secretData[secret.Title], err = secret.getField(esv1beta1.ExternalSecretDataRemoteRef{})
		if err != nil {
			return nil, err
		}
	}

	return secretData, nil
}

func (c *Client) Close(ctx context.Context) error {

	return nil
}

func (c *Client) getValidKeeperSecret(secret *ksm.Record) (*KeeperSecuritySecret, error) {

	keeperSecret := KeeperSecuritySecret{}
	err := json.Unmarshal([]byte(secret.RawJson), &keeperSecret)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecurityInvalidSecretInvalidFormat, err)
	}
	keeperSecret.addFiles(secret.Files)
	if !keeperSecret.validate() {
		return nil, fmt.Errorf(errKeeperSecurityInvalidSecretDuplicatedKey)
	}

	return &keeperSecret, nil
}

func (c *Client) findSecrets() ([]*ksm.Record, error) {

	records, err := c.ksmClient.GetSecrets([]string{})
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, err)
	}

	return records, nil
}

func (c *Client) findSecretById(id string) (*ksm.Record, error) {

	records, err := c.ksmClient.GetSecrets([]string{id})
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
	}

	if len(records) != 1 {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotUnique, id)
	}

	return records[0], nil
}

func (s *KeeperSecuritySecret) validate() bool {
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
	keys := make([]string, 0, len(fields))

	for key := range fields {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return fields[keys[i]] > fields[keys[j]]
	})

	return fields[keys[0]] == 1
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

func (s *KeeperSecuritySecret) getField(ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if ref.Property == "" {
		secret, err := s.toString()
		return []byte(secret), err
	}

	for _, field := range s.Fields {
		if field.Type == ref.Property && len(field.Value) > 0 {
			return []byte(field.Value[0]), nil
		}
	}
	for _, customField := range s.Custom {
		if customField.Label == ref.Property && len(customField.Value) > 0 {
			return []byte(customField.Value[0]), nil
		}
	}
	for _, file := range s.Files {
		if file.Title == ref.Property {
			return []byte(file.Content), nil
		}
	}

	return nil, fmt.Errorf(errKeeperSecurityInvalidProperty, s.Title, ref.Property)
}

func (s *KeeperSecuritySecret) getFields(ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secretData := make(map[string][]byte)
	for _, field := range s.Fields {
		if ref.Property != "" && field.Type != ref.Property {
			continue
		}
		if len(field.Value) > 0 && field.Type != keeperSecurityFileRef && field.Type != keeperSecurityMfa {
			secretData[field.Type] = []byte(field.Value[0])
		}
	}
	for _, customField := range s.Custom {
		if ref.Property != "" && customField.Label != ref.Property {
			continue
		}
		if len(customField.Value) > 0 {
			secretData[customField.Label] = []byte(customField.Value[0])
		}
	}
	for _, file := range s.Files {
		if ref.Property != "" && file.Title != ref.Property {
			continue
		}
		secretData[file.Title] = []byte(file.Content)
	}

	if len(secretData) == 0 {
		return nil, fmt.Errorf(errKeeperSecurityNoFields, s.Title)
	}

	return secretData, nil
}

func (s *KeeperSecuritySecret) toString() (string, error) {

	secretJson, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf(errInvalidJsonSecret, s.Title, err)
	}

	return string(secretJson), nil
}
