package scaleway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"strconv"
	"strings"
	"time"
)

var errNoSecretForName = errors.New("no secret for this name")

type client struct {
	api       secretApi
	projectId string
	cache     cache
}

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

	request := smapi.GetSecretRequest{
		SecretName: &name,
	}

	response, err := c.api.GetSecret(&request, scw.WithContext(ctx))
	if err != nil {
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
		if _, isNotFoundErr := err.(*scw.ResourceNotFoundError); isNotFoundErr {
			return nil, esv1beta1.NoSecretError{}
		}
		return nil, err
	}

	return value, nil
}

func (c *client) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {

	// TODO: A cluster-specific prefix should be prepended to the secret name to reduce the probability of a collision.

	scwRef, err := decodeScwSecretRef(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}

	if scwRef.RefType != "name" {
		return fmt.Errorf("secrets can only be pushed by name")
	}
	secretName := scwRef.Value

	// First, we do a GetSecretVersion() to resolve the secret id and the last revision number.

	var secretId string
	secretExists := false
	secretHasVersions := false

	secretVersion, err := c.api.GetSecretVersion(&smapi.GetSecretVersionRequest{
		SecretName: &secretName,
		Revision:   "latest",
	}, scw.WithContext(ctx))
	if err != nil {
		if notFoundErr, ok := err.(*scw.ResourceNotFoundError); ok {
			if notFoundErr.Resource == "secret_version" {
				secretExists = true
			}
		} else {
			return err
		}
	} else {
		secretExists = true
		secretHasVersions = true
	}

	if secretExists {

		if secretHasVersions {

			// If the secret exists, we can fetch its last value to see if we have any change to make.

			secretId = secretVersion.SecretID

			data, err := c.accessSpecificSecretVersion(ctx, secretId, secretVersion.Revision)
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

			secret, err := c.api.GetSecret(&smapi.GetSecretRequest{
				SecretName: &secretName,
			}, scw.WithContext(ctx))
			if err != nil {
				return err
			}

			secretId = secret.ID
		}

	} else {

		// If the secret does not exist, we need to create it.

		secret, err := c.api.CreateSecret(&smapi.CreateSecretRequest{
			ProjectID: c.projectId,
			Name:      secretName,
		}, scw.WithContext(ctx))
		if err != nil {
			return err
		}

		secretId = secret.ID
	}

	// Finally, we push the new secret version.

	createSecretVersionRequest := smapi.CreateSecretVersionRequest{
		SecretID: secretId,
		Data:     value,
	}

	createSecretVersionResponse, err := c.api.CreateSecretVersion(&createSecretVersionRequest, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	c.cache.Put(secretId, createSecretVersionResponse.Revision, value)

	return nil
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {

	scwRef, err := decodeScwSecretRef(remoteRef.GetRemoteKey())
	if err != nil {
		return err
	}

	if scwRef.RefType != "name" {
		return fmt.Errorf("secrets can only be pushed by name")
	}
	secretName := scwRef.Value

	secret, err := c.getSecretByName(ctx, secretName)
	if err != nil {
		if err == errNoSecretForName {
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
		ProjectID: &c.projectId,
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

		var stringValue string
		err := json.Unmarshal(value, &stringValue)
		if err == nil {
			values[key] = []byte(stringValue)
			continue
		}

		values[key] = []byte(strings.TrimSpace(string(value)))
	}

	return values, nil
}

// GetAllSecrets lists secrets matching the given criteria and return their latest versions.
func (c *client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {

	request := smapi.ListSecretsRequest{
		ProjectID: &c.projectId,
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
				SecretID: &secret.ID,
				Revision: "latest_enabled",
			}

			accessResp, err := c.api.AccessSecretVersion(&accessReq, scw.WithContext(ctx))
			if err != nil {
				log.Error(err, "failed to access secret")
				continue
			}

			results[secret.ID] = accessResp.Data
		}
	}

	return results, nil
}

func (c *client) Close(context.Context) error {
	return nil
}

func (c *client) accessSecretVersion(ctx context.Context, secretRef *scwSecretRef, versionSpec string) ([]byte, error) {

	// if we have a secret id and a revision number, we can avoid an extra GetSecret()

	if secretRef.RefType == "id" && len(versionSpec) > 0 && '0' <= versionSpec[0] && versionSpec[0] <= '9' {

		secretId := secretRef.Value

		revision, err := strconv.ParseUint(versionSpec, 10, 32)
		if err == nil {
			return c.accessSpecificSecretVersion(ctx, secretId, uint32(revision))
		}
	}

	// otherwise, we do a GetSecret() first to avoid transferring the secret value if it is cached

	request := smapi.GetSecretVersionRequest{
		Revision: versionSpec,
	}

	switch secretRef.RefType {
	case "id":
		request.SecretID = &secretRef.Value
	case "name":
		request.SecretName = &secretRef.Value
	default:
		return nil, fmt.Errorf("invalid secret reference: %q", secretRef.Value)
	}

	response, err := c.api.GetSecretVersion(&request, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return c.accessSpecificSecretVersion(ctx, response.SecretID, response.Revision)
}

func (c *client) accessSpecificSecretVersion(ctx context.Context, secretId string, revision uint32) ([]byte, error) {

	cachedValue, cacheHit := c.cache.Get(secretId, revision)
	if cacheHit {
		return cachedValue, nil
	}

	request := smapi.AccessSecretVersionRequest{
		SecretID: &secretId,
		Revision: fmt.Sprintf("%d", revision),
	}

	response, err := c.api.AccessSecretVersion(&request, scw.WithContext(ctx))
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}
