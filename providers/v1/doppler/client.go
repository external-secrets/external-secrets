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

// Package doppler implements a provider for Doppler secrets management.
package doppler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	dclient "github.com/external-secrets/external-secrets/providers/v1/doppler/client"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/find"
)

const (
	customBaseURLEnvVar                                = "DOPPLER_BASE_URL"
	verifyTLSOverrideEnvVar                            = "DOPPLER_VERIFY_TLS"
	errGetSecret                                       = "could not get secret %s: %w"
	errGetSecrets                                      = "could not get secrets %w"
	errDeleteSecrets                                   = "could not delete secrets %s: %w"
	errPushSecrets                                     = "could not push secrets %s: %w"
	errUnmarshalSecretMap                              = "unable to unmarshal secret %s: %w"
	secretsDownloadFileKey                             = "DOPPLER_SECRETS_FILE"
	errDopplerTokenSecretName                          = "missing auth.secretRef.dopplerToken.name"
	errInvalidClusterStoreMissingDopplerTokenNamespace = "missing auth.secretRef.dopplerToken.namespace"
)

// Client implements the SecretsClient interface for Doppler.
type Client struct {
	doppler         SecretsClientInterface
	dopplerToken    string
	project         string
	config          string
	nameTransformer string
	format          string

	kube        kclient.Client
	corev1      typedcorev1.CoreV1Interface
	store       *esv1.DopplerProvider
	namespace   string
	storeKind   string
	storeName   string
	oidcManager *OIDCTokenManager
}

// SecretsClientInterface defines the required Doppler Client methods.
type SecretsClientInterface interface {
	BaseURL() *url.URL
	Authenticate() error
	GetSecret(request dclient.SecretRequest) (*dclient.SecretResponse, error)
	GetSecrets(request dclient.SecretsRequest) (*dclient.SecretsResponse, error)
	UpdateSecrets(request dclient.UpdateSecretsRequest) error
}

func (c *Client) setAuth(ctx context.Context) error {
	if c.store.Auth.SecretRef != nil {
		token, err := resolvers.SecretKeyRef(
			ctx,
			c.kube,
			c.storeKind,
			c.namespace,
			&c.store.Auth.SecretRef.DopplerToken)
		if err != nil {
			return err
		}
		c.dopplerToken = token
	} else if c.store.Auth.OIDCConfig != nil {
		token, err := c.oidcManager.Token(ctx)
		if err != nil {
			return fmt.Errorf("failed to get OIDC token: %w", err)
		}
		c.dopplerToken = token
	} else {
		return errors.New("no authentication method configured: either secretRef or oidcConfig must be specified")
	}
	return nil
}

func (c *Client) refreshAuthIfNeeded(ctx context.Context) error {
	if c.store != nil && c.store.Auth != nil && c.store.Auth.OIDCConfig != nil && c.oidcManager != nil {
		token, err := c.oidcManager.Token(ctx)
		if err != nil {
			return fmt.Errorf("failed to refresh OIDC token: %w", err)
		}
		if doppler, ok := c.doppler.(*dclient.DopplerClient); ok {
			doppler.DopplerToken = token
		}
	}
	return nil
}

// Validate validates the Doppler client configuration.
func (c *Client) Validate() (esv1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := c.doppler.BaseURL().String()

	if err := esutils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1.ValidationResultError, err
	}

	if err := c.doppler.Authenticate(); err != nil {
		return esv1.ValidationResultError, err
	}

	return esv1.ValidationResultReady, nil
}

func (c *Client) storeIdentity() storeIdentity {
	return storeIdentity{
		namespace: c.namespace,
		name:      c.storeName,
		kind:      c.storeKind,
	}
}

// DeleteSecret removes a secret from Doppler.
func (c *Client) DeleteSecret(ctx context.Context, ref esv1.PushSecretRemoteRef) error {
	if err := c.refreshAuthIfNeeded(ctx); err != nil {
		return err
	}
	request := dclient.UpdateSecretsRequest{
		ChangeRequests: []dclient.Change{
			{
				Name:         ref.GetRemoteKey(),
				OriginalName: ref.GetRemoteKey(),
				ShouldDelete: true,
			},
		},
		Project: c.project,
		Config:  c.config,
	}

	err := c.doppler.UpdateSecrets(request)
	if err != nil {
		return fmt.Errorf(errDeleteSecrets, ref.GetRemoteKey(), err)
	}

	etagCache.invalidate(c.storeIdentity())

	return nil
}

// SecretExists checks if a secret exists in Doppler.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// PushSecret creates or updates a secret in Doppler.
func (c *Client) PushSecret(ctx context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	if err := c.refreshAuthIfNeeded(ctx); err != nil {
		return err
	}
	request := dclient.UpdateSecretsRequest{
		Secrets: dclient.Secrets{
			data.GetRemoteKey(): string(secret.Data[data.GetSecretKey()]),
		},
		Project: c.project,
		Config:  c.config,
	}

	err := c.doppler.UpdateSecrets(request)
	if err != nil {
		return fmt.Errorf(errPushSecrets, data.GetRemoteKey(), err)
	}

	etagCache.invalidate(c.storeIdentity())

	return nil
}

// GetSecret retrieves a secret from Doppler.
func (c *Client) GetSecret(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if err := c.refreshAuthIfNeeded(ctx); err != nil {
		return nil, err
	}

	var etag string
	cached, hasCached := etagCache.get(c.storeIdentity(), ref.Key)
	if hasCached {
		etag = cached.etag
	}

	request := dclient.SecretRequest{
		Name:    ref.Key,
		Project: c.project,
		Config:  c.config,
		ETag:    etag,
	}

	response, err := c.doppler.GetSecret(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecret, ref.Key, err)
	}

	if !response.Modified && hasCached {
		return []byte(cached.secrets[ref.Key]), nil
	}

	etagCache.set(c.storeIdentity(), ref.Key, &cacheEntry{
		etag:    response.ETag,
		secrets: dclient.Secrets{response.Name: response.Value},
	})

	return []byte(response.Value), nil
}

// GetSecretMap retrieves a secret from Doppler and returns it as a map.
func (c *Client) GetSecretMap(ctx context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := c.GetSecret(ctx, ref)
	if err != nil {
		return nil, err
	}

	kv := make(map[string]json.RawMessage)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf(errUnmarshalSecretMap, ref.Key, err)
	}

	secretData := make(map[string][]byte)
	for k, v := range kv {
		var strVal string
		err = json.Unmarshal(v, &strVal)
		if err == nil {
			secretData[k] = []byte(strVal)
		} else {
			secretData[k] = v
		}
	}
	return secretData, nil
}

// GetAllSecrets retrieves all secrets from Doppler that match the given criteria.
func (c *Client) GetAllSecrets(ctx context.Context, ref esv1.ExternalSecretFind) (map[string][]byte, error) {
	secrets, err := c.secrets(ctx)
	selected := map[string][]byte{}

	if err != nil {
		return nil, err
	}

	if ref.Name == nil && ref.Path == nil {
		return secrets, nil
	}

	var matcher *find.Matcher
	if ref.Name != nil {
		m, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}
		matcher = m
	}

	for key, value := range secrets {
		if (matcher != nil && !matcher.MatchName(key)) || (ref.Path != nil && !strings.HasPrefix(key, *ref.Path)) {
			continue
		}
		selected[key] = value
	}

	return selected, nil
}

// Close implements cleanup operations for the Doppler client.
func (c *Client) Close(_ context.Context) error {
	return nil
}

func (c *Client) secrets(ctx context.Context) (map[string][]byte, error) {
	if err := c.refreshAuthIfNeeded(ctx); err != nil {
		return nil, err
	}

	var etag string
	cached, hasCached := etagCache.get(c.storeIdentity(), "")
	if hasCached {
		etag = cached.etag
	}

	request := dclient.SecretsRequest{
		Project:         c.project,
		Config:          c.config,
		NameTransformer: c.nameTransformer,
		Format:          c.format,
		ETag:            etag,
	}

	response, err := c.doppler.GetSecrets(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecrets, err)
	}

	if !response.Modified && hasCached {
		if c.format != "" {
			return map[string][]byte{
				secretsDownloadFileKey: cached.body,
			}, nil
		}
		return externalSecretsFormat(cached.secrets), nil
	}

	etagCache.set(c.storeIdentity(), "", &cacheEntry{
		etag:    response.ETag,
		secrets: response.Secrets,
		body:    response.Body,
	})

	if c.format != "" {
		return map[string][]byte{
			secretsDownloadFileKey: response.Body,
		}, nil
	}

	return externalSecretsFormat(response.Secrets), nil
}

func externalSecretsFormat(secrets dclient.Secrets) map[string][]byte {
	converted := make(map[string][]byte, len(secrets))
	for key, value := range secrets {
		converted[key] = []byte(value)
	}
	return converted
}
