package scaleway

import (
	"context"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

type client struct {
	api            secretApi
	organizationId string
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
	// TODO
	return fmt.Errorf("PushSecret not implemented")
}

func (c *client) DeleteSecret(ctx context.Context, remoteRef esv1beta1.PushRemoteRef) error {
	// TODO
	return fmt.Errorf("DeleteSecret not implemented")
}

func (c *client) Validate() (esv1beta1.ValidationResult, error) {
	// TODO
	return esv1beta1.ValidationResultReady, nil
}

func (c *client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// TODO
	return nil, fmt.Errorf("GetSecretMap not implemented")
}

func (c *client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {

	request := smapi.ListSecretsRequest{
		OrganizationID: &c.organizationId, // TODO: scope to project or orga?
		Page:           new(int32),
		PageSize:       new(uint32),
	}
	*request.PageSize = 50

	// TODO: validate the name now?
	if ref.Name != nil {
		// TODO
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
