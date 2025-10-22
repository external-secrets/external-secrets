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

	"github.com/external-secrets/external-secrets/pkg/find"
	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
	dclient "github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
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

	kube      kclient.Client
	store     *esv1.DopplerProvider
	namespace string
	storeKind string
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

// DeleteSecret removes a secret from Doppler.
func (c *Client) DeleteSecret(_ context.Context, ref esv1.PushSecretRemoteRef) error {
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

	return nil
}

// SecretExists checks if a secret exists in Doppler.
func (c *Client) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New("not implemented")
}

// PushSecret creates or updates a secret in Doppler.
func (c *Client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1.PushSecretData) error {
	value := secret.Data[data.GetSecretKey()]

	request := dclient.UpdateSecretsRequest{
		Secrets: dclient.Secrets{
			data.GetRemoteKey(): string(value),
		},
		Project: c.project,
		Config:  c.config,
	}

	err := c.doppler.UpdateSecrets(request)
	if err != nil {
		return fmt.Errorf(errPushSecrets, data.GetRemoteKey(), err)
	}

	return nil
}

// GetSecret retrieves a secret from Doppler.
func (c *Client) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	request := dclient.SecretRequest{
		Name:    ref.Key,
		Project: c.project,
		Config:  c.config,
	}

	secret, err := c.doppler.GetSecret(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecret, ref.Key, err)
	}

	return []byte(secret.Value), nil
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
	secrets, err := c.getSecrets(ctx)
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

func (c *Client) getSecrets(_ context.Context) (map[string][]byte, error) {
	request := dclient.SecretsRequest{
		Project:         c.project,
		Config:          c.config,
		NameTransformer: c.nameTransformer,
		Format:          c.format,
	}

	response, err := c.doppler.GetSecrets(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecrets, err)
	}

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
