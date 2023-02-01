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

package scaleway

import (
	"context"
	"fmt"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"

	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	errMissingStore            = fmt.Errorf("missing store provider")
	errMissingScalewayProvider = fmt.Errorf("missing store provider scaleway")
	errMissingKeyField         = "key must be set in data %v"
	errMissingValueField       = "at least one of value or valueMap must be set in data %v"
)

type SourceOrigin string

const (
	ScalewaySecretStore SourceOrigin = "SecretStore"
	ScalewaySetSecret   SourceOrigin = "SetSecret"
)

type Config struct {
	ApiUrl         string
	Region         string
	OrganizationId string
	ProjectId      string
	AccessKey      string
	Secretkey      string
}

type Provider struct {
	configs map[string]Config
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeClient.Client, namespace string) (esv1beta1.SecretsClient, error) {

	if p.configs == nil {
		p.configs = make(map[string]Config)
	}

	cfg := p.configs[store.GetName()]

	c, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	cfg = Config{
		ApiUrl:         c.ApiUrl,
		Region:         c.Region,
		OrganizationId: c.OrganizationId,
		ProjectId:      c.ProjectId,
		AccessKey:      c.AccessKey,
		Secretkey:      c.Secretkey,
	}

	p.configs[store.GetName()] = cfg

	scwClient, err := scw.NewClient(
		scw.WithAPIURL(cfg.ApiUrl),
		scw.WithDefaultRegion(scw.Region(cfg.Region)),
		scw.WithDefaultOrganizationID(cfg.OrganizationId),
		scw.WithDefaultProjectID(cfg.ProjectId),
		scw.WithAuth(cfg.AccessKey, cfg.Secretkey),
	)
	if err != nil {
		return nil, err
	}

	return &client{
		api:            smapi.NewAPI(scwClient),
		organizationId: cfg.OrganizationId,
	}, nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.ScalewayProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Scaleway == nil {
		return nil, errMissingScalewayProvider
	}
	return spc.Provider.Scaleway, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov := store.GetSpec().Provider.Scaleway
	if prov == nil {
		return nil
	}
	// TODO
	return nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Scaleway: &esv1beta1.ScalewayProvider{},
	})
}
