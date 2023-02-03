package scaleway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"strings"
)

type client struct {
	api       secretApi
	projectId string
}

func (c *client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {

	request := smapi.AccessSecretVersionRequest{
		SecretID: ref.Key,
		Revision: ref.Version,
	}

	if ref.Version == "" {
		request.Revision = "latest"
	}

	response, err := c.api.AccessSecretVersion(&request, scw.WithContext(ctx))
	if err != nil {
		if _, isNotFoundErr := err.(*scw.ResourceNotFoundError); isNotFoundErr {
			return nil, esv1beta1.NoSecretError{}
		}
		return nil, err
	}

	return response.Data, nil
}

func (c *client) PushSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {

	// TODO: add a tag to secrets managed by kubernetes?

	accessSecretVersionRequest := smapi.AccessSecretVersionRequest{
		SecretID: remoteRef.GetRemoteKey(),
		Revision: "latest",
	}

	secretExistsButHasNoVersion := false

	accessSecretVersionResponse, err := c.api.AccessSecretVersion(&accessSecretVersionRequest, scw.WithContext(ctx))
	if err != nil {
		if notFoundErr, isNotFoundErr := err.(*scw.ResourceNotFoundError); isNotFoundErr {
			if notFoundErr.Resource == "secret_version" {
				secretExistsButHasNoVersion = true
			}
		}
		if !secretExistsButHasNoVersion {
			return err
		}
	}

	if !secretExistsButHasNoVersion && bytes.Equal(accessSecretVersionResponse.Data, value) {
		// no change to push
		return nil
	}

	createSecretVersionRequest := smapi.CreateSecretVersionRequest{
		SecretID: remoteRef.GetRemoteKey(),
		Data:     value,
	}

	_, err = c.api.CreateSecretVersion(&createSecretVersionRequest, scw.WithContext(ctx))
	if err != nil {
		return err
	}

	return nil
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {

	request := smapi.DeleteSecretRequest{
		SecretID: remoteRef.GetRemoteKey(),
	}

	err := c.api.DeleteSecret(&request, scw.WithContext(ctx))
	if err != nil {
		if _, isNotFoundErr := err.(*scw.ResourceNotFoundError); isNotFoundErr {
			return esv1beta1.NoSecretError{}
		}
		return err
	}

	return nil
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {

	// TODO

	page := int32(1)
	pageSize := uint32(0)
	_, err := c.api.ListSecrets(&smapi.ListSecretsRequest{
		ProjectID: &c.projectId,
		Page:      &page,
		PageSize:  &pageSize,
	})
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

	for tag, _ := range ref.Tags {
		request.Tags = append(request.Tags, tag)
	}

	results := map[string][]byte{}

	for {

		response, err := c.api.ListSecrets(&request, scw.WithContext(ctx))
		if err != nil {
			return nil, err
		}

		*request.Page++

		if len(response.Secrets) == 0 {
			break
		}

		for _, secret := range response.Secrets {

			if nameMatcher != nil && !nameMatcher.MatchName(secret.Name) {
				continue
			}

			// TODO: update to latest-enabled when possible

			accessReq := smapi.AccessSecretVersionRequest{
				Region:   secret.Region,
				SecretID: secret.ID,
				Revision: "latest",
			}

			accessResp, err := c.api.AccessSecretVersion(&accessReq, scw.WithContext(ctx))
			if err != nil {
				// TODO: log
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
