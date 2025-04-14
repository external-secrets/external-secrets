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

package onboardbase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	onboardbaseClient "github.com/external-secrets/external-secrets/pkg/provider/onboardbase/client"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	errGetSecret                                            = "could not get secret %s: %s"
	errGetSecrets                                           = "could not get secrets %s"
	errUnmarshalSecretMap                                   = "unable to unmarshal secret %s: %w"
	errOnboardbaseAPIKeySecretName                          = "missing auth.secretRef.onboardbaseAPIKey.name"
	errInvalidClusterStoreMissingOnboardbaseAPIKeyNamespace = "missing auth.secretRef.onboardbaseAPIKey.namespace"
	errFetchOnboardbaseAPIKeySecret                         = "unable to find find OnboardbaseAPIKey secret: %w"
	errMissingOnboardbaseAPIKey                             = "auth.secretRef.onboardbaseAPIKey.key '%s' not found in secret '%s'"
	errMissingOnboardbasePasscode                           = "auth.secretRef.onboardbasePasscode.key '%s' not found in secret '%s'"
	errSecretKeyFmt                                         = "cannot find property %s in secret data for key: %q"
)

type Client struct {
	onboardbase         SecretsClientInterface
	onboardbaseAPIKey   string
	onboardbasePasscode string
	project             string
	environment         string

	kube      kclient.Client
	store     *esv1beta1.OnboardbaseProvider
	namespace string
	storeKind string
}

// SecretsClientInterface defines the required Onboardbase Client methods.
type SecretsClientInterface interface {
	BaseURL() *url.URL
	Authenticate() error
	GetSecret(request onboardbaseClient.SecretRequest) (*onboardbaseClient.SecretResponse, error)
	DeleteSecret(request onboardbaseClient.SecretRequest) error
	GetSecrets(request onboardbaseClient.SecretsRequest) (*onboardbaseClient.SecretsResponse, error)
}

func (c *Client) setAuth(ctx context.Context) error {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.OnboardbaseAPIKeyRef.Name
	if credentialsSecretName == "" {
		return errors.New(errOnboardbaseAPIKeySecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1beta1.ClusterSecretStoreKind {
		if c.store.Auth.OnboardbaseAPIKeyRef.Namespace == nil {
			return errors.New(errInvalidClusterStoreMissingOnboardbaseAPIKeyNamespace)
		}
		objectKey.Namespace = *c.store.Auth.OnboardbaseAPIKeyRef.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return fmt.Errorf(errFetchOnboardbaseAPIKeySecret, err)
	}

	onboardbaseAPIKey := credentialsSecret.Data[c.store.Auth.OnboardbaseAPIKeyRef.Key]
	if (onboardbaseAPIKey == nil) || (len(onboardbaseAPIKey) == 0) {
		return fmt.Errorf(errMissingOnboardbaseAPIKey, c.store.Auth.OnboardbaseAPIKeyRef.Key, credentialsSecretName)
	}
	c.onboardbaseAPIKey = string(onboardbaseAPIKey)

	onboardbasePasscode := credentialsSecret.Data[c.store.Auth.OnboardbasePasscodeRef.Key]
	if (onboardbasePasscode == nil) || (len(onboardbasePasscode) == 0) {
		return fmt.Errorf(errMissingOnboardbasePasscode, c.store.Auth.OnboardbasePasscodeRef.Key, credentialsSecretName)
	}

	c.onboardbasePasscode = string(onboardbasePasscode)

	return nil
}

func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	timeout := 15 * time.Second
	clientURL := c.onboardbase.BaseURL().String()

	if err := utils.NetworkValidate(clientURL, timeout); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	if err := c.onboardbase.Authenticate(); err != nil {
		return esv1beta1.ValidationResultError, err
	}

	return esv1beta1.ValidationResultReady, nil
}

func (c *Client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	// not implemented
	return nil
}

func (c *Client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	// not implemented
	return false, nil
}

func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	// not implemented
	return nil
}

func (c *Client) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	request := onboardbaseClient.SecretRequest{
		Project:     c.project,
		Environment: c.environment,
		Name:        ref.Key,
	}

	secret, err := c.onboardbase.GetSecret(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecret, ref.Key, err)
	}

	value := secret.Value

	if ref.Property != "" {
		jsonRes := gjson.Get(secret.Value, ref.Property)
		if !jsonRes.Exists() {
			return nil, fmt.Errorf(errSecretKeyFmt, ref.Property, ref.Key)
		}
		value = jsonRes.Raw
	}

	return []byte(value), nil
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
	if len(ref.Tags) > 0 {
		return nil, errors.New("find by tags not supported")
	}

	secrets, err := c.getSecrets(ctx)

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

	selected := map[string][]byte{}
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
	request := onboardbaseClient.SecretsRequest{
		Project:     c.project,
		Environment: c.environment,
	}

	response, err := c.onboardbase.GetSecrets(request)
	if err != nil {
		return nil, fmt.Errorf(errGetSecrets, err)
	}

	return externalSecretsFormat(response.Secrets), nil
}

func externalSecretsFormat(secrets onboardbaseClient.Secrets) map[string][]byte {
	converted := make(map[string][]byte, len(secrets))
	for key, value := range secrets {
		converted[key] = []byte(value)
	}
	return converted
}
