/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scaleway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
)

var errNoSecretForName = errors.New("no secret for this name")

type client struct {
	api       secretAPI
	projectID string
	cache     cache
}

const (
	refTypeName = "name"
	refTypeID   = "id"
	refTypePath = "path"
)

type scwSecretRef struct {
	RefType string
	Value   string
}

func (r scwSecretRef) String() string {
	return fmt.Sprintf("%s:%s", r.RefType, r.Value)
}

func decodeScwSecretRef(key string) (*scwSecretRef, error) {
	sepIndex := strings.IndexRune(key, ':')
	if sepIndex < 0 {
		return nil, errors.New("invalid secret reference: missing colon ':'")
	}

	return &scwSecretRef{
		RefType: key[:sepIndex],
		Value:   key[sepIndex+1:],
	}, nil
}

func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	scwRef, err := decodeScwSecretRef(ref.Key)
	if err != nil {
		return nil, err
	}

	versionSpec := "latest_enabled"
	if ref.Version != "" {
		versionSpec = ref.Version
	}

	value, err := c.accessSecretVersion(ctx, scwRef, versionSpec)
	if err != nil {
		//nolint:errorlint
		if _, isNotFoundErr := err.(*scw.ResourceNotFoundError); isNotFoundErr {
			return nil, esv1beta1.NoSecretError{}
		} else if errors.Is(err, errNoSecretForName) {
			return nil, esv1beta1.NoSecretError{}
		}
		return nil, err
	}

	if ref.Property != "" {
		extracted, err := extractJSONProperty(value, ref.Property)
		if err != nil {
			return nil, err
		}

		value = extracted
	}

	return value, nil
}

func (c *client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	if data.GetSecretKey() == "" {
		return errors.New("pushing the whole secret is not yet implemented")
	}

	value := secret.Data[data.GetSecretKey()]
	scwRef, err := decodeScwSecretRef(data.GetRemoteKey())
	if err != nil {
		return err
	}

	listSecretReq := &smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      scw.Int32Ptr(1),
		PageSize:  scw.Uint32Ptr(1),
	}
	var secretName string
	secretPath := "/"

	switch scwRef.RefType {
	case refTypeName:
		listSecretReq.Name = &scwRef.Value
		secretName = scwRef.Value
	case refTypePath:
		name, path, ok := splitNameAndPath(scwRef.Value)
		if !ok {
			return errors.New("ref is not a path")
		}
		listSecretReq.Name = &name
		listSecretReq.Path = &path
		secretName = name
		secretPath = path
	default:
		return errors.New("secrets can only be pushed by name or path")
	}

	var secretID string
	existingSecretVersion := int64(-1)

	// list secret by ref
	listSecrets, err := c.api.ListSecrets(listSecretReq, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	// secret exists
	if len(listSecrets.Secrets) > 0 {
		secretID = listSecrets.Secrets[0].ID

		// get the latest version
		secretVersion, err := c.api.GetSecretVersion(&smapi.GetSecretVersionRequest{
			SecretID: secretID,
			Revision: "latest",
		}, scw.WithContext(ctx))
		if err != nil {
			var errNotNound *scw.ResourceNotFoundError
			if !errors.As(err, &errNotNound) {
				return err
			}
		} else {
			existingSecretVersion = int64(secretVersion.Revision)
		}

		if existingSecretVersion != -1 {
			data, err := c.accessSpecificSecretVersion(ctx, secretID, secretVersion.Revision)
			if err != nil {
				return err
			}

			if bytes.Equal(data, value) {
				// No change to push.
				return nil
			}
		}
	} else {
		secret, err := c.api.CreateSecret(&smapi.CreateSecretRequest{
			ProjectID: c.projectID,
			Name:      secretName,
			Path:      &secretPath,
		}, scw.WithContext(ctx))
		if err != nil {
			return err
		}

		secretID = secret.ID
	}

	// Finally, we push the new secret version.

	createSecretVersionRequest := smapi.CreateSecretVersionRequest{
		SecretID: secretID,
		Data:     value,
	}

	createSecretVersionResponse, err := c.api.CreateSecretVersion(&createSecretVersionRequest, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	c.cache.Put(secretID, createSecretVersionResponse.Revision, value)

	if existingSecretVersion != -1 {
		_, err := c.api.DisableSecretVersion(&smapi.DisableSecretVersionRequest{
			SecretID: secretID,
			Revision: fmt.Sprintf("%d", existingSecretVersion),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushSecretRemoteRef) error {
	scwRef, err := decodeScwSecretRef(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}

	listSecretReq := &smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      scw.Int32Ptr(1),
		PageSize:  scw.Uint32Ptr(1),
	}

	switch scwRef.RefType {
	case refTypeName:
		listSecretReq.Name = &scwRef.Value
	case refTypePath:
		name, path, ok := splitNameAndPath(scwRef.Value)
		if !ok {
			return errors.New("ref is not a path")
		}
		listSecretReq.Name = &name
		listSecretReq.Path = &path

	default:
		return errors.New("secrets can only be deleted by name or path")
	}

	listSecrets, err := c.api.ListSecrets(listSecretReq, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	if len(listSecrets.Secrets) == 0 {
		return nil
	}

	request := smapi.DeleteSecretRequest{
		SecretID: listSecrets.Secrets[0].ID,
	}

	err = c.api.DeleteSecret(&request, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	return nil
}

func (c *client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.api.ListSecrets(&smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      scw.Int32Ptr(1),
		PageSize:  scw.Uint32Ptr(0),
	}, scw.WithContext(ctx))
	if err != nil {
		return esv1beta1.ValidationResultError, nil
	}

	return esv1beta1.ValidationResultReady, nil
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	rawData, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	structuredData := make(map[string]json.RawMessage)

	err = json.Unmarshal(rawData, &structuredData)
	if err != nil {
		return nil, err
	}

	values := make(map[string][]byte)

	for key, value := range structuredData {
		values[key] = jsonToSecretData(value)
	}

	return values, nil
}

// GetAllSecrets lists secrets matching the given criteria and return their latest versions.
func (c *client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	request := smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      scw.Int32Ptr(1),
		PageSize:  scw.Uint32Ptr(50),
	}

	if ref.Path != nil {
		request.Path = ref.Path
	}

	var nameMatcher *find.Matcher
	if ref.Name != nil {
		var err error
		nameMatcher, err = find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
	}

	for tag := range ref.Tags {
		request.Tags = append(request.Tags, tag)
	}

	results := map[string][]byte{}

	for done := false; !done; {
		response, err := c.api.ListSecrets(&request, scw.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		totalFetched := uint64(*request.Page-1)*uint64(*request.PageSize) + uint64(len(response.Secrets))
		done = totalFetched == response.TotalCount

		*request.Page++

		for _, secret := range response.Secrets {
			if nameMatcher != nil && !nameMatcher.MatchName(secret.Name) {
				continue
			}

			accessReq := smapi.AccessSecretVersionRequest{
				Region:   secret.Region,
				SecretID: secret.ID,
				Revision: "latest_enabled",
			}

			accessResp, err := c.api.AccessSecretVersion(&accessReq, scw.WithContext(ctx))
			if err != nil {
				log.Error(err, "failed to access secret")
				continue
			}

			results[secret.Name] = accessResp.Data
		}
	}

	return results, nil
}

func (c *client) Close(context.Context) error {
	return nil
}

func (c *client) accessSecretVersion(ctx context.Context, secretRef *scwSecretRef, versionSpec string) ([]byte, error) {
	// if we have a secret id and a revision number, we can avoid an extra GetSecret()

	if secretRef.RefType == refTypeID && versionSpec != "" && '0' <= versionSpec[0] && versionSpec[0] <= '9' {
		secretID := secretRef.Value

		revision, err := strconv.ParseUint(versionSpec, 10, 32)
		if err == nil {
			return c.accessSpecificSecretVersion(ctx, secretID, uint32(revision))
		}
	}

	// otherwise, we do a GetSecret() first to avoid transferring the secret value if it is cached
	request := &smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      scw.Int32Ptr(1),
		PageSize:  scw.Uint32Ptr(1),
	}

	switch secretRef.RefType {
	case refTypeID:
		request := smapi.GetSecretVersionRequest{
			SecretID: secretRef.Value,
			Revision: versionSpec,
		}
		response, err := c.api.GetSecretVersion(&request, scw.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		return c.accessSpecificSecretVersion(ctx, response.SecretID, response.Revision)
	case refTypeName:
		request.Name = &secretRef.Value

	case refTypePath:
		name, path, ok := splitNameAndPath(secretRef.Value)
		if !ok {
			return nil, errors.New("ref is not a path")
		}

		request.Name = &name
		request.Path = &path
	default:
		return nil, fmt.Errorf("invalid secret reference: %q", secretRef.Value)
	}

	response, err := c.api.ListSecrets(request, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	if len(response.Secrets) == 0 {
		return nil, errNoSecretForName
	}

	secretID := response.Secrets[0].ID

	secretVersion, err := c.api.GetSecretVersion(&smapi.GetSecretVersionRequest{
		SecretID: secretID,
		Revision: versionSpec,
	}, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return c.accessSpecificSecretVersion(ctx, secretID, secretVersion.Revision)
}

func (c *client) accessSpecificSecretVersion(ctx context.Context, secretID string, revision uint32) ([]byte, error) {
	cachedValue, cacheHit := c.cache.Get(secretID, revision)
	if cacheHit {
		return cachedValue, nil
	}

	request := smapi.AccessSecretVersionRequest{
		SecretID: secretID,
		Revision: fmt.Sprintf("%d", revision),
	}

	response, err := c.api.AccessSecretVersion(&request, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func jsonToSecretData(value json.RawMessage) []byte {
	var stringValue string
	err := json.Unmarshal(value, &stringValue)
	if err == nil {
		return []byte(stringValue)
	}

	return []byte(strings.TrimSpace(string(value)))
}

func extractJSONProperty(secretData []byte, property string) ([]byte, error) {
	result := gjson.Get(string(secretData), property)

	if !result.Exists() {
		return nil, esv1beta1.NoSecretError{}
	}

	return jsonToSecretData(json.RawMessage(result.Raw)), nil
}

func splitNameAndPath(ref string) (name, path string, ok bool) {
	if !strings.HasPrefix(ref, "/") {
		return
	}

	s := strings.Split(ref, "/")
	name = s[len(s)-1]
	if len(s) == 2 {
		path = "/"
	} else {
		path = strings.Join(s[:len(s)-1], "/")
	}
	ok = true
	return
}
