/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or impliec.
See the License for the specific language governing permissions and
limitations under the License.
*/

package doppler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	dClient "github.com/external-secrets/external-secrets/pkg/provider/doppler/client"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
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

type Client struct {
	doppler         SecretsClientInterface
	dopplerToken    string
	project         string
	config          string
	nameTransformer string
	format          string

	kube      kclient.Client
	store     *esv1beta1.DopplerProvider
	namespace string
	storeKind string
}

// SecretsClientInterface defines the required Doppler Client methods.
type SecretsClientInterface interface {
	BaseURL() *url.URL
	Authenticate() error
	GetSecret(request dClient.SecretRequest) (*dClient.SecretResponse, error)
	GetSecrets(request dClient.SecretsRequest) (*dClient.SecretsResponse, error)
	UpdateSecrets(request dClient.UpdateSecretsRequest) error
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

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := c.doppler.BaseURL().String()

	if err := utils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	if err := c.doppler.Authenticate(); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (c *Client) DeleteSecret(_ context.Context, ref esv1beta1.PushSecretRemoteRef) error {
	request := dClient.UpdateSecretsRequest{
		ChangeRequests: []dClient.Change{
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

func (c *Client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

func (c *Client) PushSecret(_ context.Context, secret *corev1.Secret, data esv1beta1.PushSecretData) error {
	value := secret.Data[data.GetSecretKey()]

	request := dClient.UpdateSecretsRequest{
		Secrets: dClient.Secrets{
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

func (c *Client) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	request := dClient.SecretRequest{
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

func (c *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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

func (c *Client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
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

func (c *Client) Close(_ context.Context) error {
	return nil
}

func (c *Client) getSecrets(_ context.Context) (map[string][]byte, error) {
	request := dClient.SecretsRequest{
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

func externalSecretsFormat(secrets dClient.Secrets) map[string][]byte {
	converted := make(map[string][]byte, len(secrets))
	for key, value := range secrets {
		converted[key] = []byte(value)
	}
	return converted
}
