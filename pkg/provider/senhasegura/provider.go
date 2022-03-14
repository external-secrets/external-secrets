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

package senhasegura

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	senhaseguraAuth "github.com/external-secrets/external-secrets/pkg/provider/senhasegura/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/senhasegura/dsm"
	"github.com/external-secrets/external-secrets/pkg/provider/senhasegura/utils"
)

// Provider struct that satisfier ESO interface.
type Provider struct{}

const (
	errUnknownProviderService = "unknown senhasegura Provider Service: %s"
)

/*
	Construct a new secrets client based on provided store
*/
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	provider, err := utils.GetSenhaseguraProvider(store)
	if err != nil {
		return nil, err
	}

	isoSession, err := senhaseguraAuth.Authenticate(ctx, store, provider, kube, namespace)
	if err != nil {
		return nil, err
	}

	if provider.Module == esv1beta1.SenhaseguraModuleDSM {
		return dsm.New(isoSession)
	}

	return nil, fmt.Errorf(errUnknownProviderService, provider.Module)
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	_, err := utils.GetSenhaseguraProvider(store)
	if err != nil {
		return err
	}
	return nil
}

/*
	Register SenhaseguraProvider in ESO init
*/
func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Senhasegura: &esv1beta1.SenhaseguraProvider{},
	})
}
