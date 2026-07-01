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

package certificatemanager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	mwssdk "go.mws.cloud/go-sdk/mws"
	certmanagerclient "go.mws.cloud/go-sdk/service/certmanager/client"
	certmanagersdk "go.mws.cloud/go-sdk/service/certmanager/sdk"
	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	errNotImplemented   = errors.New("not implemented")
	errUnsetChainedCert = errors.New("chainedCert is not set for this certificate")
)

var _ esv1.SecretsClient = (*Client)(nil)

// Client implements the Secrets Client interface for MWS Certificate Manager.
type Client struct {
	sdk         *mwssdk.SDK
	certificate *certmanagersdk.Certificate

	project string
}

// GetSecret gets the secret from MWS Certificate Manager.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	var (
		certificateName     = ref.Key
		certificateProperty = ref.Property
	)

	content, err := c.certificate.GetCertificateContent(ctx, certmanagerclient.GetCertificateContentRequest{
		Project: c.project,
		Name:    certificateName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate content: %w", err)
	}

	if certificateProperty == "" {
		//nolint:gosec // PrivateKey is a part of the secret
		buffer, err := json.Marshal(content)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal certificate content: %w", err)
		}

		return buffer, nil
	}

	switch certificateProperty {
	case "certificate":
		return []byte(content.Certificate), nil

	case "privateKey":
		return []byte(content.PrivateKey), nil

	case "chainedCert":
		if content.ChainedCert == nil {
			return nil, errUnsetChainedCert
		}

		return []byte(*content.ChainedCert), nil
	}

	return nil, fmt.Errorf("invalid certificate property %s", certificateProperty)
}

// GetSecretMap gets the secret map from MWS Certificate Manager.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	var certificateName = ref.Key

	content, err := c.certificate.GetCertificateContent(ctx, certmanagerclient.GetCertificateContentRequest{
		Project: c.project,
		Name:    certificateName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate content: %w", err)
	}

	secretMap := make(map[string][]byte, 3)

	secretMap["certificate"] = []byte(content.Certificate)
	secretMap["privateKey"] = []byte(content.PrivateKey)

	if content.ChainedCert != nil {
		secretMap["chainedCert"] = []byte(*content.ChainedCert)
	}

	return secretMap, nil
}

// GetAllSecrets gets all secrets from MWS Certificate Manager.
func (c *Client) GetAllSecrets(context.Context, esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errNotImplemented
}

// PushSecret pushes the secret to MWS Certificate Manager.
func (c *Client) PushSecret(context.Context, *corev1.Secret, esv1.PushSecretData) error {
	return errNotImplemented
}

// DeleteSecret deletes the secret from MWS Certificate Manager.
func (c *Client) DeleteSecret(context.Context, esv1.PushSecretRemoteRef) error {
	return errNotImplemented
}

// SecretExists checks does the secret exist in MWS Certificate Manager.
func (c *Client) SecretExists(context.Context, esv1.PushSecretRemoteRef) (bool, error) {
	return false, errNotImplemented
}

// Validate validates the client configuration.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}

// Close closes the client.
func (c *Client) Close(ctx context.Context) error {
	if err := c.sdk.Close(ctx); err != nil {
		return fmt.Errorf("failed to close mws sdk: %w", err)
	}

	return nil
}
