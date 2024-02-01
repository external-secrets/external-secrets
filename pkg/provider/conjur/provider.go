// Package conjur provides a Conjur provider for External Secrets.
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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/conjur/util"
	"github.com/external-secrets/external-secrets/pkg/utils"
)

var (
	errConjurClient     = "cannot setup new Conjur client: %w"
	errBadCertBundle    = "caBundle failed to base64 decode: %w"
	errBadServiceUser   = "could not get Auth.Apikey.UserRef: %w"
	errBadServiceAPIKey = "could not get Auth.Apikey.ApiKeyRef: %w"

	errGetKubeSATokenRequest = "cannot request Kubernetes service account token for service account %q: %w"

	errUnableToFetchCAProviderCM     = "unable to fetch Server.CAProvider ConfigMap: %w"
	errUnableToFetchCAProviderSecret = "unable to fetch Server.CAProvider Secret: %w"
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

type Provider struct {
	NewConjurProvider func(context context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientApi SecretsClientFactory) (esv1beta1.SecretsClient, error)
}

// NewClient creates a new Conjur client.
func (c *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	// controller-runtime/client does not support TokenRequest or other subresource APIs
	// so we need to construct our own client and use it to create a TokenRequest
	restCfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, err
	}

	return c.NewConjurProvider(ctx, store, kube, namespace, clientset.CoreV1(), &ClientAPIImpl{})
}

func newConjurProvider(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientAPI SecretsClientFactory) (esv1beta1.SecretsClient, error) {
	return &Client{
		StoreKind: store.GetObjectKind().GroupVersionKind().Kind,
		store:     store,
		kube:      kube,
		namespace: namespace,
		corev1:    corev1,
		clientAPI: clientAPI,
	}, nil
}

func (p *Client) GetConjurClient(ctx context.Context) (SecretsClient, error) {
	// if the client is initialized already, return it
	if p.client != nil {
		return p.client, nil
	}

	prov, err := util.GetConjurProvider(p.store)
	if err != nil {
		return nil, err
	}

	cert, getCertErr := p.getCA(ctx, prov)
	if getCertErr != nil {
		return nil, getCertErr
	}

	config := conjurapi.Config{
		ApplianceURL: prov.URL,
		SSLCert:      cert,
	}

	if prov.Auth.Apikey != nil {
		config.Account = prov.Auth.Apikey.Account
		conjUser, secErr := p.secretKeyRef(ctx, prov.Auth.Apikey.UserRef)
		if secErr != nil {
			return nil, fmt.Errorf(errBadServiceUser, secErr)
		}
		conjAPIKey, secErr := p.secretKeyRef(ctx, prov.Auth.Apikey.APIKeyRef)
		if secErr != nil {
			return nil, fmt.Errorf(errBadServiceAPIKey, secErr)
		}

		conjur, newClientFromKeyError := p.clientAPI.NewClientFromKey(config,
			authn.LoginPair{
				Login:  conjUser,
				APIKey: conjAPIKey,
			},
		)

		if newClientFromKeyError != nil {
			return nil, fmt.Errorf(errConjurClient, newClientFromKeyError)
		}
		p.client = conjur
		return conjur, nil
	} else if prov.Auth.Jwt != nil {
		config.Account = prov.Auth.Jwt.Account

		conjur, clientFromJwtError := p.newClientFromJwt(ctx, config, prov.Auth.Jwt)
		if clientFromJwtError != nil {
			return nil, fmt.Errorf(errConjurClient, clientFromJwtError)
		}

		p.client = conjur

		return conjur, nil
	} else {
		// Should not happen because validate func should catch this
		return nil, fmt.Errorf("no authentication method provided")
	}
}

// GetAllSecrets returns all secrets from the provider.
// NOT IMPLEMENTED.
func (p *Client) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (p *Client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	conjurClient, getConjurClientError := p.GetConjurClient(ctx)
	if getConjurClientError != nil {
		return nil, getConjurClientError
	}
	secretValue, err := conjurClient.RetrieveSecret(ref.Key)
	if err != nil {
		return nil, err
	}

	return secretValue, nil
}

// PushSecret will write a single secret into the provider.
func (p *Client) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1beta1.PushSecretData) error {
	// NOT IMPLEMENTED
	return nil
}

func (p *Client) DeleteSecret(_ context.Context, _ esv1beta1.PushSecretRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

func (p *Client) SecretExists(_ context.Context, _ esv1beta1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	// Gets a secret as normal, expecting secret value to be a json object
	data, err := p.GetSecret(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	// Maps the json data to a string:string map
	kv := make(map[string]string)
	err = json.Unmarshal(data, &kv)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal secret %s: %w", ref.Key, err)
	}

	// Converts values in K:V pairs into bytes, while leaving keys as strings
	secretData := make(map[string][]byte)
	for k, v := range kv {
		secretData[k] = []byte(v)
	}
	return secretData, nil
}

// Close closes the provider.
func (p *Client) Close(_ context.Context) error {
	return nil
}

// Validate validates the provider.
func (p *Client) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// ValidateStore validates the store.
func (c *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov, err := util.GetConjurProvider(store)
	if err != nil {
		return err
	}

	if prov.URL == "" {
		return fmt.Errorf("conjur URL cannot be empty")
	}
	if prov.Auth.Apikey != nil {
		if prov.Auth.Apikey.Account == "" {
			return fmt.Errorf("missing Auth.ApiKey.Account")
		}
		if prov.Auth.Apikey.UserRef == nil {
			return fmt.Errorf("missing Auth.Apikey.UserRef")
		}
		if prov.Auth.Apikey.APIKeyRef == nil {
			return fmt.Errorf("missing Auth.Apikey.ApiKeyRef")
		}
		if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.Apikey.UserRef); err != nil {
			return fmt.Errorf("invalid Auth.Apikey.UserRef: %w", err)
		}
		if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.Apikey.APIKeyRef); err != nil {
			return fmt.Errorf("invalid Auth.Apikey.ApiKeyRef: %w", err)
		}
	}

	if prov.Auth.Jwt != nil {
		if prov.Auth.Jwt.Account == "" {
			return fmt.Errorf("missing Auth.Jwt.Account")
		}
		if prov.Auth.Jwt.ServiceID == "" {
			return fmt.Errorf("missing Auth.Jwt.ServiceID")
		}
		if prov.Auth.Jwt.ServiceAccountRef == nil && prov.Auth.Jwt.SecretRef == nil {
			return fmt.Errorf("must specify Auth.Jwt.SecretRef or Auth.Jwt.ServiceAccountRef")
		}
		if prov.Auth.Jwt.SecretRef != nil {
			if err := utils.ValidateReferentSecretSelector(store, *prov.Auth.Jwt.SecretRef); err != nil {
				return fmt.Errorf("invalid Auth.Jwt.SecretRef: %w", err)
			}
		}
		if prov.Auth.Jwt.ServiceAccountRef != nil {
			if err := utils.ValidateReferentServiceAccountSelector(store, *prov.Auth.Jwt.ServiceAccountRef); err != nil {
				return fmt.Errorf("invalid Auth.Jwt.ServiceAccountRef: %w", err)
			}
		}
	}

	// At least one auth must be configured
	if prov.Auth.Apikey == nil && prov.Auth.Jwt == nil {
		return fmt.Errorf("missing Auth.* configuration")
	}

	return nil
}

// Capabilities returns the provider Capabilities (Read, Write, ReadWrite).
func (c *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Client) secretKeyRef(ctx context.Context, secretRef *esmeta.SecretKeySelector) (string, error) {
	secret := &corev1.Secret{}
	ref := client.ObjectKey{
		Namespace: p.namespace,
		Name:      secretRef.Name,
	}
	if (p.StoreKind == esv1beta1.ClusterSecretStoreKind) &&
		(secretRef.Namespace != nil) {
		ref.Namespace = *secretRef.Namespace
	}
	err := p.kube.Get(ctx, ref, secret)
	if err != nil {
		return "", err
	}

	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", err
	}

	value := string(keyBytes)
	valueStr := strings.TrimSpace(value)
	return valueStr, nil
}

// configMapKeyRef returns the value of a key in a configmap.
func (p *Client) configMapKeyRef(ctx context.Context, cmRef *esmeta.SecretKeySelector) (string, error) {
	configMap := &corev1.ConfigMap{}
	ref := client.ObjectKey{
		Namespace: p.namespace,
		Name:      cmRef.Name,
	}
	if (p.StoreKind == esv1beta1.ClusterSecretStoreKind) &&
		(cmRef.Namespace != nil) {
		ref.Namespace = *cmRef.Namespace
	}
	err := p.kube.Get(ctx, ref, configMap)
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
func (p *Client) getCA(ctx context.Context, provider *esv1beta1.ConjurProvider) (string, error) {
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
			ca, err = p.configMapKeyRef(ctx, &keySelector)
			if err != nil {
				return "", fmt.Errorf(errUnableToFetchCAProviderCM, err)
			}
		case esv1beta1.CAProviderTypeSecret:
			keySelector := esmeta.SecretKeySelector{
				Name:      provider.CAProvider.Name,
				Namespace: provider.CAProvider.Namespace,
				Key:       provider.CAProvider.Key,
			}
			ca, err = p.secretKeyRef(ctx, &keySelector)
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

func init() {
	esv1beta1.Register(&Provider{
		NewConjurProvider: newConjurProvider,
	}, &esv1beta1.SecretStoreProvider{
		Conjur: &esv1beta1.ConjurProvider{},
	})
}
