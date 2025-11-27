/*
Copyright © 2025 ESO Maintainer Team

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

package conjur

import (
	"context"
	"errors"
	"fmt"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/providers/v1/conjur/util"
)

var (
	errConjurClient          = "cannot setup new Conjur client: %w"
	errBadServiceUser        = "could not get Auth.Apikey.UserRef: %w"
	errBadServiceAPIKey      = "could not get Auth.Apikey.ApiKeyRef: %w"
	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"
	errSecretKeyFmt          = "cannot find secret data for key: %q"
)

// Client is a provider for Conjur.
type Client struct {
	StoreKind string
	kube      client.Client
	store     esv1.GenericStore
	namespace string
	corev1    typedcorev1.CoreV1Interface
	clientAPI SecretsClientFactory
	client    SecretsClient
}

// GetConjurClient returns an authenticated Conjur client.
// If a client is already initialized, it returns the existing client.
// Otherwise, it creates a new client based on the authentication method specified.
func (c *Client) GetConjurClient(ctx context.Context) (SecretsClient, error) {
	// if the client is initialized already, return it
	if c.client != nil {
		return c.client, nil
	}

	prov, err := conjurutil.GetConjurProvider(c.store)
	if err != nil {
		return nil, err
	}

	cert, getCertErr := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
		CABundle:   []byte(prov.CABundle),
		CAProvider: prov.CAProvider,
		StoreKind:  c.store.GetKind(),
		Namespace:  c.namespace,
		Client:     c.kube,
	})
	if getCertErr != nil {
		return nil, getCertErr
	}

	config := conjurapi.Config{
		ApplianceURL: prov.URL,
		SSLCert:      string(cert),
		// disable credential storage, as it depends on a writable
		// file system, which we can't rely on - it would fail.
		CredentialStorage: conjurapi.CredentialStorageNone,
	}

	if prov.Auth.APIKey != nil {
		return c.conjurClientFromAPIKey(ctx, config, prov)
	}
	if prov.Auth.Jwt != nil {
		return c.conjurClientFromJWT(ctx, config, prov)
	}
	// Should not happen because validate func should catch this
	return nil, errors.New("no authentication method provided")
}

// PushSecret will write a single secret into the provider.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	// NOT IMPLEMENTED
	return nil
}

// DeleteSecret removes a secret from the provider.
func (c *Client) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

// SecretExists checks if a secret exists in the provider.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// Validate validates the provider configuration.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// Close closes the provider.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// conjurClientFromAPIKey creates a new Conjur client using API key authentication.
func (c *Client) conjurClientFromAPIKey(ctx context.Context, config conjurapi.Config, prov *esv1.ConjurProvider) (SecretsClient, error) {
	config.Account = prov.Auth.APIKey.Account
	conjUser, secErr := resolvers.SecretKeyRef(
		ctx,
		c.kube,
		c.StoreKind,
		c.namespace, prov.Auth.APIKey.UserRef)
	if secErr != nil {
		return nil, fmt.Errorf(errBadServiceUser, secErr)
	}
	conjAPIKey, secErr := resolvers.SecretKeyRef(
		ctx,
		c.kube,
		c.StoreKind,
		c.namespace,
		prov.Auth.APIKey.APIKeyRef)
	if secErr != nil {
		return nil, fmt.Errorf(errBadServiceAPIKey, secErr)
	}

	conjur, newClientFromKeyError := c.clientAPI.NewClientFromKey(config,
		authn.LoginPair{
			Login:  conjUser,
			APIKey: conjAPIKey,
		},
	)

	if newClientFromKeyError != nil {
		return nil, fmt.Errorf(errConjurClient, newClientFromKeyError)
	}
	c.client = conjur
	return conjur, nil
}

func (c *Client) conjurClientFromJWT(ctx context.Context, config conjurapi.Config, prov *esv1.ConjurProvider) (SecretsClient, error) {
	config.AuthnType = "jwt"
	config.Account = prov.Auth.Jwt.Account
	config.JWTHostID = prov.Auth.Jwt.HostID
	config.ServiceID = prov.Auth.Jwt.ServiceID

	jwtToken, getJWTError := c.getJWTToken(ctx, prov.Auth.Jwt)
	if getJWTError != nil {
		return nil, getJWTError
	}

	config.JWTContent = jwtToken

	conjur, clientError := c.clientAPI.NewClientFromJWT(config)
	if clientError != nil {
		return nil, fmt.Errorf(errConjurClient, clientError)
	}

	c.client = conjur
	return conjur, nil
}
