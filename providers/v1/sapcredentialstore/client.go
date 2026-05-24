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

package sapcredentialstore

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/metrics"
	"github.com/external-secrets/external-secrets/providers/v1/sapcredentialstore/api"
)

var _ esv1.SecretsClient = &Client{}

const (
	credTypePassword    = "password"
	credTypeKey         = "key"
	credTypeCertificate = "certificate"
	providerName        = "sapCredentialStore"
)

// Client implements esv1.SecretsClient for SAP Credential Store.
type Client struct {
	sapClient api.SAPCSClientInterface
	namespace string
}

// credTypeFromProperty maps the ExternalSecretDataRemoteRef.Property to a SAP CS credential type.
// An empty property defaults to "password".
// "certificate/key" is a special sub-field accessor — the caller handles it.
func credTypeFromProperty(property string) string {
	switch property {
	case credTypeKey:
		return credTypeKey
	case credTypeCertificate, "certificate/key":
		return credTypeCertificate
	default:
		return credTypePassword
	}
}

// GetSecret fetches a single credential value from SAP Credential Store.
//
// ref.Key is the credential name.
// ref.Property is the credential type ("password", "key", "certificate") with "" defaulting to "password".
// The special property "certificate/key" returns the private key PEM sub-field of a certificate credential.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	credType := credTypeFromProperty(ref.Property)
	cred, err := c.sapClient.GetCredential(ctx, c.namespace, credType, ref.Key)
	metrics.ObserveAPICall(providerName, "GetCredential", err)
	if err != nil {
		var notFound *api.NotFoundError
		if errors.As(err, &notFound) {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("sapCredentialStore: GetCredential %s/%s: %w", credType, ref.Key, err)
	}

	if ref.Property == "certificate/key" {
		return []byte(cred.Key), nil
	}
	return []byte(cred.Value), nil
}

// GetSecretMap fetches a credential and returns all its fields as a map.
// Keys are "name", "value", and optionally "username" (password type) and "key" (certificate type).
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	credType := credTypeFromProperty(ref.Property)
	cred, err := c.sapClient.GetCredential(ctx, c.namespace, credType, ref.Key)
	metrics.ObserveAPICall(providerName, "GetCredential", err)
	if err != nil {
		var notFound *api.NotFoundError
		if errors.As(err, &notFound) {
			return nil, esv1.NoSecretError{}
		}
		return nil, fmt.Errorf("sapCredentialStore: GetSecretMap %s/%s: %w", credType, ref.Key, err)
	}

	out := map[string][]byte{
		"name":  []byte(cred.Name),
		"value": []byte(cred.Value),
	}
	if cred.Username != "" {
		out["username"] = []byte(cred.Username)
	}
	if cred.Key != "" {
		out["key"] = []byte(cred.Key)
	}
	return out, nil
}

// GetAllSecrets lists all credentials across all types and returns them keyed as "<type>/<name>".
func (c *Client) GetAllSecrets(ctx context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	result := make(map[string][]byte)

	for _, credType := range []string{credTypePassword, credTypeKey, credTypeCertificate} {
		items, err := c.sapClient.ListCredentials(ctx, c.namespace, credType)
		metrics.ObserveAPICall(providerName, "ListCredentials", err)
		if err != nil {
			return nil, fmt.Errorf("sapCredentialStore: ListCredentials type=%s: %w", credType, err)
		}

		for _, item := range items {
			cred, err := c.sapClient.GetCredential(ctx, c.namespace, credType, item.Name)
			metrics.ObserveAPICall(providerName, "GetCredential", err)
			if err != nil {
				return nil, fmt.Errorf("sapCredentialStore: GetCredential %s/%s: %w", credType, item.Name, err)
			}
			result[credType+"/"+item.Name] = []byte(cred.Value)
		}
	}

	return result, nil
}

// PushSecret writes a Kubernetes Secret value into SAP Credential Store.
// data.GetProperty() determines the credential type (defaults to "password").
// data.GetRemoteKey() is the credential name in SAP CS.
// data.GetSecretKey() is the key within the Kubernetes Secret to read the value from.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	credType := data.GetProperty()
	if credType == "" {
		credType = credTypePassword
	}

	name := data.GetRemoteKey()
	value := secret.Data[data.GetSecretKey()]

	body := &api.CredentialBody{
		Value: string(value),
	}

	err := c.sapClient.PutCredential(ctx, c.namespace, credType, name, body)
	metrics.ObserveAPICall(providerName, "PutCredential", err)
	if err != nil {
		return fmt.Errorf("sapCredentialStore: PushSecret %s/%s: %w", credType, name, err)
	}
	return nil
}

// DeleteSecret removes a credential from SAP Credential Store.
func (c *Client) DeleteSecret(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) error {
	credType := remoteRef.GetProperty()
	if credType == "" {
		credType = credTypePassword
	}

	name := remoteRef.GetRemoteKey()
	err := c.sapClient.DeleteCredential(ctx, c.namespace, credType, name)
	metrics.ObserveAPICall(providerName, "DeleteCredential", err)
	if err != nil {
		return fmt.Errorf("sapCredentialStore: DeleteSecret %s/%s: %w", credType, name, err)
	}
	return nil
}

// SecretExists checks whether a credential exists in SAP Credential Store.
func (c *Client) SecretExists(ctx context.Context, remoteRef esv1.PushSecretRemoteRef) (bool, error) {
	credType := remoteRef.GetProperty()
	if credType == "" {
		credType = credTypePassword
	}

	name := remoteRef.GetRemoteKey()
	exists, err := c.sapClient.CredentialExists(ctx, c.namespace, credType, name)
	metrics.ObserveAPICall(providerName, "CredentialExists", err)
	if err != nil {
		return false, fmt.Errorf("sapCredentialStore: SecretExists %s/%s: %w", credType, name, err)
	}
	return exists, nil
}

// Validate checks whether the provider is reachable. Returns Unknown because validating
// connectivity would require a real credential to exist in the remote store.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultUnknown, nil
}

// Close is a no-op; the HTTP client does not hold persistent connections that need teardown.
func (c *Client) Close(_ context.Context) error {
	return nil
}
