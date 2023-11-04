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
	"strings"

	"github.com/tidwall/gjson"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/find"
	"github.com/external-secrets/external-secrets/pkg/utils"
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

func (p *Provider) NewClient(_ context.Context, store esv1beta1.GenericStore, _ client.Client, _ string) (esv1beta1.SecretsClient, error) {
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
	// We want to remove any FakeSecretStore entry from memory
	// this will ensure SecretStores can delete from memory.
	for key, data := range cfg {
		if data.Origin == FakeSecretStore {
			delete(cfg, key)
		}
	}
	for _, data := range c.Data {
		key := mapKey(data.Key, data.Version)
		cfg[key] = &Data{
			Value:   data.Value,
			Version: data.Version,
			Origin:  FakeSecretStore,
		}
		if data.ValueMap != nil {
			cfg[key].ValueMap = data.ValueMap
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

func (p *Provider) DeleteSecret(_ context.Context, _ esv1beta1.PushRemoteRef) error {
	return nil
}

func (p *Provider) PushSecret(_ context.Context, value []byte, _ corev1.SecretType, _ *apiextensionsv1.JSON, remoteRef esv1beta1.PushRemoteRef) error {
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

// GetAllSecrets returns multiple secrets from the given ExternalSecretFind
// Currently, only the Name operator is supported.
func (p *Provider) GetAllSecrets(_ context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	if ref.Name != nil {
		matcher, err := find.New(*ref.Name)
		if err != nil {
			return nil, err
		}

		latestDataMap := make(map[string]*Data)
		for key, data := range p.config {
			// Reconstruct the "true" key without the version suffix
			// See the mapKey function to know how the provider generates keys
			key = strings.TrimSuffix(key, data.Version)
			if !matcher.MatchName(key) {
				continue
			}

			if d, ok := latestDataMap[key]; ok {
				// Need to get only the latest version
				if d.Version < data.Version {
					latestDataMap[key] = data
				}
			} else {
				latestDataMap[key] = data
			}
		}

		dataMap := make(map[string][]byte)
		for key, data := range latestDataMap {
			dataMap[key] = []byte(data.Value)
		}

		return utils.ConvertKeys(ref.ConversionStrategy, dataMap)
	}
	return nil, fmt.Errorf("unsupported find operator: %#v", ref)
}

// GetSecret returns a single secret from the provider.
func (p *Provider) GetSecret(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	data, ok := p.config[mapKey(ref.Key, ref.Version)]
	if !ok || data.Version != ref.Version {
		return nil, esv1beta1.NoSecretErr
	}

	if ref.Property != "" {
		val := gjson.Get(data.Value, ref.Property)
		if !val.Exists() {
			return nil, esv1beta1.NoSecretErr
		}

		return []byte(val.String()), nil
	}

	return []byte(data.Value), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (p *Provider) GetSecretMap(_ context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, ok := p.config[mapKey(ref.Key, ref.Version)]
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

func (p *Provider) Close(_ context.Context) error {
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

func mapKey(key, version string) string {
	// Add the version suffix to preserve entries with the old versions as well.
	return fmt.Sprintf("%v%v", key, version)
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Fake: &esv1beta1.FakeProvider{},
	})
}
