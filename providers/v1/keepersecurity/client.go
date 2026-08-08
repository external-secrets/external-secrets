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
	"strings"
	"time"

	ksm "github.com/keeper-security/secrets-manager-go/core"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
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
	ksmClient          SecurityClient
	folderID           string
	getByTitleFallback bool
	// cacheKey identifies this store's record set in the shared cache (empty -> folderID).
	cacheKey string
	// cacheTTL >0 enables the shared record cache for this client (0 -> env KEEPER_RECORD_CACHE_TTL_MS).
	cacheTTL time.Duration
}

// recordCacheTTL resolves the effective cache TTL (client field, else env).
func (c *Client) recordCacheTTL() time.Duration {
	if c.cacheTTL > 0 {
		return c.cacheTTL
	}
	return cacheTTL()
}

// recordCacheKey identifies this store's record set in the shared cache.
func (c *Client) recordCacheKey() string {
	if c.cacheKey != "" {
		return c.cacheKey
	}
	return c.folderID
}

// getAllRecords fetches the full record set once (with throttle/429 retry) and,
// when caching is enabled, serves a reconcile wave of N ExternalSecrets from a
// single backend get_secret call instead of N.
func (c *Client) getAllRecords(ctx context.Context) ([]*ksm.Record, error) {
	key := c.recordCacheKey()
	ttl := c.recordCacheTTL()
	if recs, ok := sharedRecordCache.get(key, ttl); ok {
		return recs, nil
	}
	records, err := getSecretsWithRetry(ctx, c.ksmClient, []string{})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityGetSecrets, err)
	if err != nil {
		return nil, err
	}
	if ttl > 0 {
		sharedRecordCache.set(key, records)
	}
	return records, nil
}

// getAllFolders fetches the folder hierarchy (with throttle/429 retry), cached
// alongside records since folders change rarely. Used for dataFrom.find.path.
func (c *Client) getAllFolders(ctx context.Context) ([]*ksm.KeeperFolder, error) {
	key := c.recordCacheKey()
	ttl := c.recordCacheTTL()
	if f, ok := sharedFolderCache.get(key, ttl); ok {
		return f, nil
	}
	folders, err := getFoldersWithRetry(ctx, c.ksmClient)
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityGetFolders, err)
	if err != nil {
		return nil, err
	}
	if ttl > 0 {
		sharedFolderCache.set(key, folders)
	}
	return folders, nil
}

// recordFolderUID returns the UID of the folder directly containing the record
// (the subfolder if present, else the shared folder).
func recordFolderUID(r *ksm.Record) string {
	if uid := r.InnerFolderUid(); uid != "" {
		return uid
	}
	return r.FolderUid()
}

// SecurityClient defines the interface for interacting with KeeperSecurity's API.
type SecurityClient interface {
	GetSecrets(filter []string) ([]*ksm.Record, error)
	GetFolders() ([]*ksm.KeeperFolder, error)
	FindNotation(records []*ksm.Record, notation string) ([]interface{}, error)
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

// Validate checks the store is actually usable by performing a real (cache-backed)
// read against Keeper. A transient rate-limit is reported as Unknown (ignored by
// ESO) rather than failing the store; auth/config errors are reported as Error.
// With the record cache enabled this is effectively free (served from cache).
func (c *Client) Validate() (esv1.ValidationResult, error) {
	if _, err := c.getAllRecords(context.Background()); err != nil {
		if isRateLimited(err) {
			return esv1.ValidationResultUnknown, err
		}
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// GetSecret retrieves a secret from Keeper Security by ID or name.
// It first attempts to find the secret by ID, then falls back to name lookup.
// The name lookup must be opted in by setting getByTitleFallback on the provider.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	// Keeper Notation (e.g. keeper://<uid|title>/field/password[0], /custom_field/..., /file/...)
	// is resolved against the (cached) record set by the SDK — no extra backend call.
	if isKeeperNotation(ref.Key) {
		return c.getByNotation(ctx, ref.Key)
	}
	secret, err := c.findByIDWithNameFallback(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	return secret.getItem(ref)
}

// isKeeperNotation reports whether key is a Keeper Notation expression. We require
// the explicit "keeper://" prefix so a bare record UID/title is never mistaken for
// notation (maximum backwards-compat safety).
func isKeeperNotation(key string) bool {
	return strings.HasPrefix(key, "keeper://")
}

// getByNotation resolves a Keeper Notation expression against all records the
// app can read (served from cache when enabled).
func (c *Client) getByNotation(ctx context.Context, notation string) ([]byte, error) {
	records, err := c.getAllRecords(ctx)
	if err != nil {
		return nil, err
	}
	values, err := c.ksmClient.FindNotation(records, notation)
	if err != nil {
		return nil, fmt.Errorf("keeper notation %q: %w", notation, err)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("keeper notation %q returned no value", notation)
	}
	return notationValueToBytes(values[0])
}

func notationValueToBytes(v interface{}) ([]byte, error) {
	switch t := v.(type) {
	case string:
		return []byte(t), nil
	case []byte:
		return t, nil
	default:
		return json.Marshal(v)
	}
}

// GetSecretMap retrieves a secret from Keeper Security and returns it as a map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	secret, err := c.findByIDWithNameFallback(ctx, ref.Key)
	if err != nil {
		return nil, err
	}
	return secret.getItems(ref)
}

// It first attempts to find the secret by ID, then falls back to name lookup.
// The name lookup must be opted in by setting getByTitleFallback on the provider.
func (c *Client) findByIDWithNameFallback(ctx context.Context, key string) (*Secret, error) {
	record, err := c.findSecretByID(ctx, key)
	if err != nil {
		return nil, err
	}

	if record == nil && c.getByTitleFallback {
		var records []*ksm.Record
		ferr := withRateLimitRetry(ctx, func() error {
			var e error
			records, e = c.ksmClient.GetSecretsByTitle(key)
			return e
		})
		metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityGetSecretsByTitle, ferr)
		if ferr != nil {
			return nil, ferr
		}

		if len(records) > 1 {
			return nil, errors.New(errKeeperSecuritySecretNotUnique)
		} else if len(records) == 1 {
			record = records[0]
		}
	}

	if record == nil {
		return nil, errors.New(errKeeperSecurityNoSecretsFound)
	}

	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// GetAllSecrets retrieves all secrets from Keeper Security that match the given criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Tags != nil {
		return nil, errors.New(errTagsNotImplemented)
	}

	records, err := c.findSecrets(ctx)
	if err != nil {
		return nil, err
	}
	matchPath, err := c.folderPathMatcher(ctx, ref)
	if err != nil {
		return nil, err
	}
	nameRe, err := compileFindName(ref)
	if err != nil {
		return nil, err
	}

	secretData := make(map[string][]byte)
	for _, record := range records {
		if !matchPath(record) {
			continue
		}
		if title, value, ok := c.findRecordValue(record, nameRe); ok {
			secretData[title] = value
		}
	}

	return secretData, nil
}

// folderPathMatcher returns a predicate for dataFrom.find.path; when no path is
// set it matches every record.
func (c *Client) folderPathMatcher(ctx context.Context, ref esv1.ExternalSecretFind) (func(*ksm.Record) bool, error) {
	if ref.Path == nil {
		return func(*ksm.Record) bool { return true }, nil
	}
	folders, err := c.getAllFolders(ctx)
	if err != nil {
		return nil, err
	}
	tree := buildFolderTree(folders)
	path := *ref.Path
	return func(r *ksm.Record) bool {
		return pathMatchesPrefix(tree.pathOf(recordFolderUID(r)), path)
	}, nil
}

// compileFindName compiles dataFrom.find.name.regexp (nil when unset).
func compileFindName(ref esv1.ExternalSecretFind) (*regexp.Regexp, error) {
	if ref.Name == nil || ref.Name.RegExp == "" {
		return nil, nil
	}
	re, err := regexp.Compile(ref.Name.RegExp)
	if err != nil {
		return nil, fmt.Errorf(errInvalidRegex, ref.Name.RegExp, err)
	}
	return re, nil
}

// findRecordValue resolves a record to its (title, value) for find. It is
// best-effort: records that can't be represented (or don't match the name regexp)
// are skipped (ok=false) rather than failing the whole ExternalSecret — important
// for find.path, which sweeps entire folders.
func (c *Client) findRecordValue(record *ksm.Record, nameRe *regexp.Regexp) (string, []byte, bool) {
	secret, err := c.getValidKeeperSecret(record)
	if err != nil {
		return "", nil, false
	}
	if nameRe != nil && !nameRe.MatchString(secret.Title) {
		return "", nil, false
	}
	value, err := secret.getItem(esv1.ExternalSecretDataRemoteRef{})
	if err != nil {
		return "", nil, false
	}
	return secret.Title, value, true
}

// Close implements cleanup operations for the Keeper Security client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// PushSecret creates or updates a secret in Keeper Security.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if data.GetSecretKey() == "" {
		return errors.New("pushing the whole secret is not yet implemented")
	}

	value := secret.Data[data.GetSecretKey()]
	parts, err := c.buildSecretNameAndKey(data)
	if err != nil {
		return err
	}

	record, err := c.findSecretByName(ctx, parts[0])
	if err != nil {
		return err
	}

	if record != nil {
		if record.Type() != externalSecretType {
			return fmt.Errorf(errInvalidSecretType, externalSecretType, record.Title(), record.Type())
		}
		return c.updateSecret(ctx, record, parts[1], value)
	}

	_, err = c.createSecret(ctx, parts[0], parts[1], value)
	return err
}

// DeleteSecret removes a secret from Keeper Security.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	parts, err := c.buildSecretNameAndKey(remoteRef)
	if err != nil {
		return err
	}
	secret, err := c.findSecretByName(ctx, parts[0])
	if err != nil {
		return err
	} else if secret == nil {
		// not found == already deleted (success)
		return nil
	}

	if secret.Type() != externalSecretType {
		return fmt.Errorf(errInvalidSecretType, externalSecretType, secret.Title(), secret.Type())
	}
	err = withRateLimitRetry(ctx, func() error {
		_, derr := c.ksmClient.DeleteSecrets([]string{secret.Uid})
		return derr
	})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityDeleteSecrets, err)
	if err == nil {
		sharedRecordCache.invalidate(c.recordCacheKey())
	}
	return err
}

// SecretExists checks if a secret already exists in Keeper Security.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	parts, err := c.buildSecretNameAndKey(remoteRef)
	if err != nil {
		return false, err
	}
	record, err := c.findSecretByName(ctx, parts[0])
	if err != nil {
		return false, err
	}
	return record != nil, nil
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

func (c *Client) createSecret(ctx context.Context, name, key string, value []byte) (string, error) {
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

	var uid string
	err := withRateLimitRetry(ctx, func() error {
		var e error
		uid, e = c.ksmClient.CreateSecretWithRecordData("", c.folderID, externalSecretRecord)
		return e
	})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityCreateSecretWithRecordData, err)
	if err == nil {
		sharedRecordCache.invalidate(c.recordCacheKey())
	}
	return uid, err
}

func (c *Client) updateSecret(ctx context.Context, secret *ksm.Record, key string, value []byte) error {
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

	err := withRateLimitRetry(ctx, func() error {
		return c.ksmClient.Save(secret)
	})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecuritySave, err)
	if err == nil {
		sharedRecordCache.invalidate(c.recordCacheKey())
	}
	return err
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

func (c *Client) findSecrets(ctx context.Context) ([]*ksm.Record, error) {
	records, err := c.getAllRecords(ctx)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretsNotFound, err)
	}

	return records, nil
}

func (c *Client) findSecretByID(ctx context.Context, id string) (*ksm.Record, error) {
	// Cached path: one shared get_secret returns all records; filter by UID.
	// Collapses a reconcile wave of N ExternalSecrets into a single backend call.
	if c.recordCacheTTL() > 0 {
		records, err := c.getAllRecords(ctx)
		if err != nil {
			return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
		}
		for _, r := range records {
			if r.Uid == id {
				return r, nil
			}
		}
		return nil, nil
	}

	// Default (uncached) path, now resilient to throttle (403) / 429.
	records, err := getSecretsWithRetry(ctx, c.ksmClient, []string{id})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityGetSecrets, err)
	if err != nil {
		return nil, fmt.Errorf(errKeeperSecuritySecretNotFound, id, err)
	}

	if len(records) == 0 {
		return nil, nil
	}

	return records[0], nil
}

func (c *Client) findSecretByName(ctx context.Context, name string) (*ksm.Record, error) {
	var records []*ksm.Record
	err := withRateLimitRetry(ctx, func() error {
		var e error
		records, e = c.ksmClient.GetSecretsByTitle(name)
		return e
	})
	metrics.ObserveAPICall(constants.ProviderKeeperSecurity, constants.CallKeeperSecurityGetSecretsByTitle, err)
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
