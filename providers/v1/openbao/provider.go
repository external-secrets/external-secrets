/*
Copyright © The ESO Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package openbao

import (
	"context"
	"net/http"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/openbao/openbao/api/v2"
)

var (
	_      esv1.Provider = &Provider{}
	logger               = ctrl.Log.WithName("provider").WithName("openbao")
)

// Provider implements the ESO Provider interface for OpenBao.
type Provider struct {
	HTTPClient *http.Client
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec().Provider.OpenBao // if this is somehow nil, there is a bug in the framework

	config := api.DefaultConfig()
	config.HttpClient = p.HTTPClient
	config.Address = spec.Server

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	if spec.Auth != nil && spec.Auth.TokenSecretRef != nil {
		token, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, spec.Auth.TokenSecretRef)
		if err != nil {
			return nil, err
		}

		client.SetToken(token)
	}

	path := "kv"
	if spec.Path != nil {
		path = *spec.Path
	}

	return &Client{
		path:   path,
		client: client,
		useV1:  spec.Version == esv1.OpenBaoKVStoreV1,
	}, nil
}

func isReferentSpec(prov *esv1.OpenBaoProvider) bool {
	if prov.Auth == nil {
		return false
	}

	if prov.Auth.TokenSecretRef != nil && prov.Auth.TokenSecretRef.Namespace == nil {
		return true
	}

	return false
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		HTTPClient: http.DefaultClient,
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		OpenBao: &esv1.OpenBaoProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
