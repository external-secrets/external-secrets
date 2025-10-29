package smop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"time"

	cg "github.com/BeyondTrust/platform-secrets-manager/apiclient/clientgen"
	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	corev1 "k8s.io/api/core/v1"
)

var ErrNotImplemented = errors.New("not implemented")

// Client implements the SecretsClient interface for SMoP.
type Client struct {
	smopClient      SecretsClientInterface
	store     *esv1.SmopProvider
}

// SecretsClientInterface defines the required SMoP Client methods.
type SecretsClientInterface interface {
	BaseURL() *url.URL
	SetBaseURL(urlStr string) error
	GetSecret(ctx context.Context, name string, folderPath *string) (*cg.KV, error)
	GetSecrets(ctx context.Context, folderPath *string) (*[]cg.KVListItem, error)
}

// Validate checks if the client is configured correctly
// and is able to retrieve secrets from the SMOP provider.
// If the validation result is unknown it will be ignored.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := c.smopClient.BaseURL().String()

	if err := esutils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1.ValidationResultError, err
	}

	// --TODO: validate auth?

	return esv1.ValidationResultReady, nil
}

// GetSecret returns a single secret from the SMOP provider
// 	if GetSecret returns an error with type NoSecretError
// 	then the secret entry will be deleted depending on the deletionPolicy.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	folderPath := c.store.FolderPath
	
	secret, err := c.smopClient.GetSecret(ctx, ref.Key, &folderPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %w", err)
	}

	secretBytes, err := json.Marshal(secret.Secret)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal secret: %w", err)
    }

    return secretBytes, nil
}

/////////////////////////
// NOT YET IMPLEMENTED //
/////////////////////////

// PushSecret will write a single secret into the SMOP provider.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	return ErrNotImplemented
}

// DeleteSecret will delete the secret from the SMOP provider.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error{
	return ErrNotImplemented
}

// SecretExists checks if a secret is already present in the SMOP provider at the given location.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	return false, ErrNotImplemented
}

// GetSecretMap returns multiple k/v pairs from the SMOP provider.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return nil, ErrNotImplemented
}

// GetAllSecrets retrieves all secrets from SMoP that match the given criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, ErrNotImplemented
}

// Close implements cleanup operations for the SMoP client.
func (c *Client) Close(ctx context.Context) error {
	return nil
}
