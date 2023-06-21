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
	"sigs.k8s.io/controller-runtime/pkg/client"

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
)

// Provider is a provider for Conjur.
type Provider struct {
	ConjurClient Client
	StoreKind    string
	kube         client.Client
	namespace    string
}

// Client is an interface for the Conjur client.
type Client interface {
	RetrieveSecret(secret string) (result []byte, err error)
}

// NewClient creates a new Conjur client.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	prov, err := util.GetConjurProvider(store)
	if err != nil {
		return nil, err
	}
	p.StoreKind = store.GetObjectKind().GroupVersionKind().Kind
	p.kube = kube
	p.namespace = namespace

	certBytes, decodeErr := utils.Decode(esv1beta1.ExternalSecretDecodeBase64, []byte(prov.CABundle))
	if decodeErr != nil {
		return nil, fmt.Errorf(errBadCertBundle, decodeErr)
	}
	cert := string(certBytes)

	config := conjurapi.Config{
		Account:      prov.Auth.Apikey.Account,
		ApplianceURL: prov.URL,
		SSLCert:      cert,
	}

	conjUser, secErr := p.secretKeyRef(ctx, prov.Auth.Apikey.UserRef)
	if secErr != nil {
		return nil, fmt.Errorf(errBadServiceUser, secErr)
	}
	conjAPIKey, secErr := p.secretKeyRef(ctx, prov.Auth.Apikey.APIKeyRef)
	if secErr != nil {
		return nil, fmt.Errorf(errBadServiceAPIKey, secErr)
	}

	conjur, err := conjurapi.NewClientFromKey(config,
		authn.LoginPair{
			Login:  conjUser,
			APIKey: conjAPIKey,
		},
	)

	if err != nil {
		return nil, fmt.Errorf(errConjurClient, err)
	}
	p.ConjurClient = conjur
	return p, nil
}

// GetAllSecrets returns all secrets from the provider.
// NOT IMPLEMENTED.
func (p *Provider) GetAllSecrets(_ context.Context, _ esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	secretValue, err := p.ConjurClient.RetrieveSecret(ref.Key)
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
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
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

	// At least one auth must be configured
	if prov.Auth.Apikey == nil {
		return fmt.Errorf("missing Auth.* configuration")
	}

	return nil
}

// Capabilities returns the provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
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

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Conjur: &esv1beta1.ConjurProvider{},
	})
}
