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

// Package openbao implement a Provider for [OpenBao]
//
// [OpenBao]: https://openbao.org/
package openbao

import (
	"context"
	"net/http"

	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
)

var (
	_ esv1.Provider = &Provider{}
)

// Provider implements the ESO Provider interface for OpenBao.
type Provider struct {
	HTTPClientFactory httpClientFactory
}

type httpClientFactory func() *http.Client

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient implements the Provider interface.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube k8sClient.Client, namespace string) (esv1.SecretsClient, error) {
	spec := store.GetSpec().Provider.OpenBao // if this is somehow nil, there is a bug in the framework

	client := &client{
		storeKind: store.GetKind(),
		store:     spec,
	}

	if client.storeKind != esv1.ClusterSecretStoreKind || namespace != "" || !isReferentSpec(spec) {
		err := client.setup(ctx, kube, namespace, p.HTTPClientFactory)
		if err != nil {
			return nil, err
		}
	}

	return client, nil
}

func isReferentSpec(prov *esv1.OpenBaoProvider) bool {
	if prov.Auth != nil {
		auth := prov.Auth

		if auth.TokenSecretRef != nil && auth.TokenSecretRef.Namespace == nil {
			return true
		}

		if auth.UserPass != nil && auth.UserPass.SecretRef.Namespace == nil {
			return true
		}
	}

	if prov.CAProvider != nil && prov.CAProvider.Namespace == nil {
		return true
	}

	return false
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		HTTPClientFactory: func() *http.Client {
			return &http.Client{
				Transport: http.DefaultTransport.(*http.Transport).Clone(),
			}
		},
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
