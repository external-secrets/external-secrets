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

package cerberus

import (
	"context"
	"fmt"
	"strings"

	cerberussdkauth "github.com/Nike-Inc/cerberus-go-client/v3/auth"
	cerberussdk "github.com/Nike-Inc/cerberus-go-client/v3/cerberus"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	awsauth "github.com/external-secrets/external-secrets/pkg/provider/aws/auth"
	cerberusauth "github.com/external-secrets/external-secrets/pkg/provider/cerberus/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/cerberus/util"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.Provider = &Provider{}

// Provider satisfies the provider interface.
type Provider struct{}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	cerberusProvider, err := util.GetCerberusProvider(store)
	if err != nil {
		return nil, err
	}
	sess, err := cerberusauth.New(ctx, store, kube, namespace, awsauth.DefaultSTSProvider, awsauth.DefaultJWTProvider)
	if err != nil {
		return nil, err
	}

	authMethod, err := cerberussdkauth.NewSTSAuth(cerberusProvider.CerberusURL, cerberusProvider.Region)
	if err != nil {
		return nil, err
	}

	sdkclient, err := cerberussdk.NewClient(authMethod.WithCredentials(sess.Config.Credentials), nil)
	if err != nil {
		return nil, err
	}

	sdbList, err := sdkclient.SDB().List()
	if err != nil {
		return nil, err
	}

	for _, sdb := range sdbList {
		if strings.EqualFold(sdb.Name, cerberusProvider.SDB) {
			return &cerberus{
				client: &cerberusClient{*sdkclient},
				sdb:    sdb,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find SDB %s", cerberusProvider.SDB)
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	cerberusProvider, err := getCerberusProvider(store)
	if err != nil {
		return err
	}

	// TODO implement me

	_ = cerberusProvider

	return nil
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func getCerberusProvider(store esv1beta1.GenericStore) (*esv1beta1.CerberusProvider, error) {
	spec := store.GetSpec()
	if spec == nil {
		return nil, fmt.Errorf("could not find spec")
	}

	provider := spec.Provider
	if provider == nil {
		return nil, fmt.Errorf("could not find provider")
	}

	cerberusProvider := provider.Cerberus
	if cerberusProvider == nil {
		return nil, fmt.Errorf("could not find cerberus provider")
	}

	return cerberusProvider, nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Cerberus: &esv1beta1.CerberusProvider{},
	})
}
