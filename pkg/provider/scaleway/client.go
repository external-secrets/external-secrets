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

	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/tidwall/gjson"

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
		return nil, fmt.Errorf("invalid secret reference: missing colon ':'")
	}

	return &scwSecretRef{
		RefType: key[:sepIndex],
		Value:   key[sepIndex+1:],
	}, nil
}

func (c *client) getSecretByName(ctx context.Context, name string) (*smapi.Secret, error) {
	request := smapi.GetSecretByNameRequest{
		SecretName: name,
	}

	response, err := c.api.GetSecretByName(&request, scw.WithContext(ctx))
	if err != nil {
		//nolint:errorlint
		if _, isErrNotFound := err.(*scw.ResourceNotFoundError); isErrNotFound {
			return nil, errNoSecretForName
		}
		return nil, err
	}

	return response, nil
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

func (c *client) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	scwRef, err := decodeScwSecretRef(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}

	if scwRef.RefType != refTypeName {
		return fmt.Errorf("secrets can only be pushed by name")
	}
	secretName := scwRef.Value

	// First, we do a GetSecretVersion() to resolve the secret id and the last revision number.

	var secretID string
	secretExists := false
	existingSecretVersion := int64(-1)

	secretVersion, err := c.api.GetSecretVersionByName(&smapi.GetSecretVersionByNameRequest{
		SecretName: secretName,
		Revision:   "latest",
	}, scw.WithContext(ctx))
	if err != nil {
		//nolint:errorlint
		if notFoundErr, ok := err.(*scw.ResourceNotFoundError); ok {
			if notFoundErr.Resource == "secret_version" {
				secretExists = true
			}
		} else {
			return err
		}
	} else {
		secretExists = true
		existingSecretVersion = int64(secretVersion.Revision)
	}

	if secretExists {
		if existingSecretVersion != -1 {
			// If the secret exists, we can fetch its last value to see if we have any change to make.

			secretID = secretVersion.SecretID

			data, err := c.accessSpecificSecretVersion(ctx, secretID, secretVersion.Revision)
			if err != nil {
				return err
			}

			if bytes.Equal(data, value) {
				// No change to push.
				return nil
			}
		} else {
			// If the secret exists but has no versions, we need an additional GetSecret() to resolve the secret id.
			// This may happen if a push was interrupted.

			secret, err := c.api.GetSecretByName(&smapi.GetSecretByNameRequest{
				SecretName: secretName,
			}, scw.WithContext(ctx))
			if err != nil {
				return err
			}

			secretID = secret.ID
		}
	} else {
		// If the secret does not exist, we need to create it.

		secret, err := c.api.CreateSecret(&smapi.CreateSecretRequest{
			ProjectID: c.projectID,
			Name:      secretName,
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

	if secretExists && existingSecretVersion != -1 {
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

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	scwRef, err := decodeScwSecretRef(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}

	if scwRef.RefType != refTypeName {
		return fmt.Errorf("secrets can only be pushed by name")
	}
	secretName := scwRef.Value

	secret, err := c.getSecretByName(ctx, secretName)
	if err != nil {
		if errors.Is(err, errNoSecretForName) {
			return nil
		}
		return err
	}

	request := smapi.DeleteSecretRequest{
		SecretID: secret.ID,
	}

	err = c.api.DeleteSecret(&request, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	return nil
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	page := int32(1)
	pageSize := uint32(0)
	_, err := c.api.ListSecrets(&smapi.ListSecretsRequest{
		ProjectID: &c.projectID,
		Page:      &page,
		PageSize:  &pageSize,
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
		Page:      new(int32),
		PageSize:  new(uint32),
	}
	*request.Page = 1
	*request.PageSize = 50

	if ref.Path != nil {
		return nil, fmt.Errorf("searching by path is not supported")
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
		done = totalFetched == uint64(response.TotalCount)

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

	if secretRef.RefType == refTypeID && len(versionSpec) > 0 && '0' <= versionSpec[0] && versionSpec[0] <= '9' {
		secretID := secretRef.Value

		revision, err := strconv.ParseUint(versionSpec, 10, 32)
		if err == nil {
			return c.accessSpecificSecretVersion(ctx, secretID, uint32(revision))
		}
	}

	// otherwise, we do a GetSecret() first to avoid transferring the secret value if it is cached

	var secretID string
	var secretRevision uint32

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
		secretID = response.SecretID
		secretRevision = response.Revision
	case refTypeName:
		request := smapi.GetSecretVersionByNameRequest{
			SecretName: secretRef.Value,
			Revision:   versionSpec,
		}
		response, err := c.api.GetSecretVersionByName(&request, scw.WithContext(ctx))
		if err != nil {
			return nil, err
		}
		secretID = response.SecretID
		secretRevision = response.Revision
	default:
		return nil, fmt.Errorf("invalid secret reference: %q", secretRef.Value)
	}

	return c.accessSpecificSecretVersion(ctx, secretID, secretRevision)
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
