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

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cyberark/conjur-api-go/conjurapi"
	"github.com/cyberark/conjur-api-go/conjurapi/authn"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	errConjurClient          = "cannot setup new Conjur client: %w"
	errMissingStore          = fmt.Errorf("missing store provider")
	errMissingConjurProvider = fmt.Errorf("missing store provider Conjur")
)

type Provider struct {
	ConjurClient Client
}

type Client interface {
	RetrieveSecret(secret string) (result []byte, err error)
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	cfg, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	config := conjurapi.Config{
		Account:      *cfg.ServiceAccount,
		ApplianceURL: *cfg.ServiceURL,
	}
	conjur, err := conjurapi.NewClientFromKey(config,
		authn.LoginPair{
			Login:  *cfg.ServiceUser,
			APIKey: *cfg.ServiceApiKey,
		},
	)

	if err != nil {
		return nil, fmt.Errorf(errConjurClient, err)
	}
	p.ConjurClient = conjur
	return p, nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.ConjurProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Conjur == nil {
		return nil, errMissingConjurProvider
	}
	return spc.Provider.Conjur, nil
}

// Empty GetAllSecrets.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {

	secretValue, err := p.ConjurClient.RetrieveSecret(ref.Key)
	if err != nil {
		return nil, err
	}

	return []byte(secretValue), nil
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

func (p *Provider) Close(ctx context.Context) error {
	return nil
}

func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov := store.GetSpec().Provider.Conjur
	if prov == nil {
		return nil
	}

	return nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Conjur: &esv1beta1.ConjurProvider{},
	})
}
