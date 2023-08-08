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
	"time"

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

	errFailedToParseJWTToken               = "failed to parse JWT token: %w"
	errFailedToDetermineJWTTokenExpiration = "conjur only supports jwt tokens that expire and JWT token expiration check failed"
)

// Provider is a provider for Conjur.
type Provider struct {
	StoreKind        string
	kube             client.Client
	store            esv1beta1.GenericStore
	namespace        string
	corev1           typedcorev1.CoreV1Interface
	clientAPI        ClientAPI
	client           Client
	clientExpires    bool
	renewClientAfter time.Time
}

type Connector struct {
	NewConjurProvider func(context context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientApi ClientAPI) (esv1beta1.SecretsClient, error)
}

// NewClient creates a new Conjur client.
func (c *Connector) NewClient(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
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

	return c.NewConjurProvider(context.Background(), store, kube, namespace, clientset.CoreV1(), &ClientAPIImpl{})
}

func newConjurProvider(_ context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, corev1 typedcorev1.CoreV1Interface, clientAPI ClientAPI) (esv1beta1.SecretsClient, error) {
	conjurProvider := &Provider{
		StoreKind: store.GetObjectKind().GroupVersionKind().Kind,
		store:     store,
		kube:      kube,
		namespace: namespace,
		corev1:    corev1,
		clientAPI: clientAPI,
	}

	return conjurProvider, nil
}

func (p *Provider) GetConjurClient(ctx context.Context) (Client, error) {
	// if we already have a client, and it hasn't expired, return it
	if p.client != nil && (!p.clientExpires || time.Now().Before(p.renewClientAfter)) {
		return p.client, nil
	}

	prov, err := util.GetConjurProvider(p.store)

	if err != nil {
		return nil, err
	}

	// maybe in future need a way refresh at some specified interval or on cert verification errors
	// if using a CAProvider to handle updated certs in cases where client does not refresh (apikey example)
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

		conjur, err := p.clientAPI.NewClientFromKey(config,
			authn.LoginPair{
				Login:  conjUser,
				APIKey: conjAPIKey,
			},
		)

		if err != nil {
			return nil, fmt.Errorf(errConjurClient, err)
		}
		// apikey is static, so no need to refresh
		p.client = conjur
		p.clientExpires = false
		return conjur, nil
	} else if prov.Auth.Jwt != nil {
		config.Account = prov.Auth.Jwt.Account

		conjur, clientFromJwtError := p.newClientFromJwt(ctx, config, prov.Auth.Jwt)
		if clientFromJwtError != nil {
			return nil, clientFromJwtError
		}
		p.client = conjur
		// jwt tokens expire, so we need to refresh the client before the token expires
		// expiration is set by the newClientFromJwt function
		p.clientExpires = true
		return conjur, nil
	} else {
		// Should not happen because validate func should catch this
		return nil, fmt.Errorf("no authentication method provided")
	}
}

// GetAllSecrets returns all secrets from the provider.
// NOT IMPLEMENTED.
func (p *Provider) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	conjurClient, err := p.GetConjurClient(ctx)
	if err != nil {
		return nil, err
	}
	secretValue, err := conjurClient.RetrieveSecret(ref.Key)
	if err != nil {
		return nil, err
	}

	return secretValue, nil
}

// PushSecret will write a single secret into the provider.
func (p *Provider) PushSecret(_ context.Context, _ []byte, _ esv1beta1.PushRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

func (p *Provider) DeleteSecret(_ context.Context, _ esv1beta1.PushRemoteRef) error {
	// NOT IMPLEMENTED
	return nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
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
func (p *Provider) Close(_ context.Context) error {
	return nil
}

// Validate validates the provider.
func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

// ValidateStore validates the store.
func (c *Connector) ValidateStore(store esv1beta1.GenericStore) error {
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
func (c *Connector) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) secretKeyRef(ctx context.Context, secretRef *esmeta.SecretKeySelector) (string, error) {
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
func (p *Provider) configMapKeyRef(ctx context.Context, cmRef *esmeta.SecretKeySelector) (string, error) {
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
func (p *Provider) getCA(ctx context.Context, provider *esv1beta1.ConjurProvider) (string, error) {
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
	esv1beta1.Register(&Connector{
		NewConjurProvider: newConjurProvider,
	}, &esv1beta1.SecretStoreProvider{
		Conjur: &esv1beta1.ConjurProvider{},
	})
}
