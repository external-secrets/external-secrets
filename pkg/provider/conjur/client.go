/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package conjur

import (
	"context"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	corev1 "k8s.io/api/core/v1"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/conjur/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var (
	errConjurClient     = "cannot setup new Conjur client: %w"
	errBadCertBundle    = "caBundle failed to base64 decode: %w"
	errBadServiceUser   = "could not get Auth.Apikey.UserRef: %w"
	errBadServiceAPIKey = "could not get Auth.Apikey.ApiKeyRef: %w"

	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"

	errUnableToFetchCAProviderCM     = "unable to fetch Server.CAProvider ConfigMap: %w"
	errUnableToFetchCAProviderSecret = "unable to fetch Server.CAProvider Secret: %w"

	errSecretKeyFmt = "cannot find secret data for key: %q"
)

// Client is a provider for Conjur.
type Client struct {
	StoreKind string
	kube      client.Client
	store     esv1beta1.GenericStore
	namespace string
	corev1    typedcorev1.CoreV1Interface
	clientAPI SecretsClientFactory
	client    SecretsClient
}

func (c *Client) GetConjurClient(ctx context.Context) (SecretsClient, error) {
	// if the client is initialized already, return it
	if c.client != nil {
		return c.client, nil
	}

	prov, err := util.GetConjurProvider(c.store)
	if err != nil {
		return nil, err
	}

	cert, getCertErr := c.getCA(ctx, prov)
	if getCertErr != nil {
		return nil, getCertErr
	}

	config := conjurapi.Config{
		ApplianceURL: prov.URL,
		SSLCert:      cert,
	}

	if prov.Auth.APIKey != nil {
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
	} else if prov.Auth.Jwt != nil {
		config.Account = prov.Auth.Jwt.Account

		conjur, clientFromJwtError := c.newClientFromJwt(ctx, config, prov.Auth.Jwt)
		if clientFromJwtError != nil {
			return nil, fmt.Errorf(errConjurClient, clientFromJwtError)
		}

		c.client = conjur

		return conjur, nil
	} else {
		// Should not happen because validate func should catch this
		return nil, fmt.Errorf("no authentication method provided")
	}
}

// PushSecret will write a single secret into the provider.
func (c *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	// NOT IMPLEMENTED
	return nil
}

func (c *Client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

func (c *Client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// Validate validates the provider.
func (c *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// Close closes the provider.
func (c *Client) Close(_ context.Context) error {
	return nil
}

// configMapKeyRef returns the value of a key in a ConfigMap.
func (c *Client) configMapKeyRef(ctx context.Context, cmRef *esmeta.SecretKeySelector) (string, error) {
	configMap := &corev1.ConfigMap{}
	ref := client.ObjectKey{
		Namespace: c.namespace,
		Name:      cmRef.Name,
	}
	if (c.StoreKind == esv1beta1.ClusterSecretStoreKind) &&
		(cmRef.Namespace != nil) {
		ref.Namespace = *cmRef.Namespace
	}
	err := c.kube.Get(ctx, ref, configMap)
	if err != nil {
		return "", err
	}

	keyBytes, ok := configMap.Data[cmRef.Key]
	if !ok {
		return "", err
	}

	valueStr := strings.TrimSpace(keyBytes)
	return valueStr, nil
}

// getCA try retrieve the CA bundle from the provider CABundle or from the CAProvider.
func (c *Client) getCA(ctx context.Context, provider *esv1beta1.ConjurProvider) (string, error) {
	if provider.CAProvider != nil {
		var ca string
		var err error
		switch provider.CAProvider.Type {
		case esv1beta1.CAProviderTypeConfigMap:
			keySelector := esmeta.SecretKeySelector{
				Name:      provider.CAProvider.Name,
				Namespace: provider.CAProvider.Namespace,
				Key:       provider.CAProvider.Key,
			}
			ca, err = c.configMapKeyRef(ctx, &keySelector)
			if err != nil {
				return "", fmt.Errorf(errUnableToFetchCAProviderCM, err)
			}
		case esv1beta1.CAProviderTypeSecret:
			keySelector := esmeta.SecretKeySelector{
				Name:      provider.CAProvider.Name,
				Namespace: provider.CAProvider.Namespace,
				Key:       provider.CAProvider.Key,
			}
			ca, err = resolvers.SecretKeyRef(
				ctx,
				c.kube,
				c.StoreKind,
				c.namespace,
				&keySelector)
			if err != nil {
				return "", fmt.Errorf(errUnableToFetchCAProviderSecret, err)
			}
		}
		return ca, nil
	}
	certBytes, decodeErr := utils.Decode(esv1beta1.ExternalSecretDecodeBase64, []byte(provider.CABundle))
	if decodeErr != nil {
		return "", fmt.Errorf(errBadCertBundle, decodeErr)
	}
	return string(certBytes), nil
}
