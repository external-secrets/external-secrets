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

package fake

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	errMissingStore        = fmt.Errorf("missing store provider")
	errMissingFakeProvider = fmt.Errorf("missing store provider fake")
	errMissingKeyField     = "key must be set in data %v"
	errMissingValueField   = "at least one of value or valueMap must be set in data %v"
)

type SourceOrigin string

const (
	FakeSecretStore SourceOrigin = "SecretStore"
	FakeSetSecret   SourceOrigin = "SetSecret"
)

type Data struct {
	Value    string
	Version  string
	ValueMap map[string]string
	Origin   SourceOrigin
}
type Config map[string]*Data
type Provider struct {
	config   Config
	database map[string]Config
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	if p.database == nil {
		p.database = make(map[string]Config)
	}
	c, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	cfg := p.database[store.GetName()]
	if cfg == nil {
		cfg = Config{}
	}
	for _, data := range c.Data {
		mapKey := fmt.Sprintf("%v%v", data.Key, data.Version)
		cfg[mapKey] = &Data{
			Value:   data.Value,
			Version: data.Version,
			Origin:  FakeSecretStore,
		}
		if data.ValueMap != nil {
			cfg[mapKey].ValueMap = data.ValueMap
		}
	}
	p.database[store.GetName()] = cfg
	return &Provider{
		config: cfg,
	}, nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.FakeProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Fake == nil {
		return nil, errMissingFakeProvider
	}
	return spc.Provider.Fake, nil
}

// Not Implemented SetSecret.
func (p *Provider) SetSecret(ctx context.Context, value []byte, remoteRef esv1beta1.PushRemoteRef) error {
	currentData, ok := p.config[remoteRef.GetRemoteKey()]
	if !ok {
		p.config[remoteRef.GetRemoteKey()] = &Data{
			Value:  string(value),
			Origin: FakeSetSecret,
		}
		return nil
	}
	if currentData.Origin != FakeSetSecret {
		return fmt.Errorf("key already exists")
	}
	currentData.Value = string(value)
	return nil
}

// Empty GetAllSecrets.
func (p *Provider) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	// TO be implemented
	return nil, fmt.Errorf("GetAllSecrets not implemented")
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	mapKey := fmt.Sprintf("%v%v", ref.Key, ref.Version)
	data, ok := p.config[mapKey]
	if !ok || data.Version != ref.Version {
		return nil, esv1beta1.NoSecretErr
	}
	return []byte(data.Value), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	mapKey := fmt.Sprintf("%v%v", ref.Key, ref.Version)
	data, ok := p.config[mapKey]
	if !ok || data.Version != ref.Version || data.ValueMap == nil {
		return nil, esv1beta1.NoSecretErr
	}
	return convertMap(data.ValueMap), nil
}

func convertMap(in map[string]string) map[string][]byte {
	m := make(map[string][]byte)
	for k, v := range in {
		m[k] = []byte(v)
	}
	return m
}

func (p *Provider) Close(ctx context.Context) error {
	return nil
}

func (p *Provider) Validate() (esv1beta1.ValidationResult, error) {
	return esv1beta1.ValidationResultReady, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov := store.GetSpec().Provider.Fake
	if prov == nil {
		return nil
	}
	for pos, data := range prov.Data {
		if data.Key == "" {
			return fmt.Errorf(errMissingKeyField, pos)
		}
		if data.Value == "" && data.ValueMap == nil {
			return fmt.Errorf(errMissingValueField, pos)
		}
	}
	return nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Fake: &esv1beta1.FakeProvider{},
	})
}
