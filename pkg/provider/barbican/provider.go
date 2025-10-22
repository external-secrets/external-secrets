/*
Copyright © 2025 ESO Maintainer Team

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

package barbican

import (
	"context"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
)

const (
	errGeneric      = "barbican provider error: %s"
	errMissingField = "barbican provider missing required field: %s"
	errAuthFailed   = "barbican provider authentication failed: %s"
	errClientInit   = "barbican provider client initialization failed: %s"
)

var _ esv1.Provider = &Provider{}

type Provider struct{}

func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errGeneric, errors.New("store is nil"))
	}
	return nil, nil
}

func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func getProvider(store esv1.GenericStore) (*esv1.BarbicanProvider, error) {
	spec := store.GetSpec()
	if spec.Provider == nil || spec.Provider.Barbican == nil {
		return nil, fmt.Errorf(errMissingField, errors.New("provider barbican is nil"))
	}
	return spec.Provider.Barbican, nil
}

func newClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	provider, err := getProvider(store)
	if err != nil {
		return nil, err
	}
	if provider.AuthURL == "" {
		return nil, fmt.Errorf(errMissingField, errors.New("authURL is required"))
	}

	username, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Username.SecretRef)
	if err != nil {
		return nil, fmt.Errorf(errMissingField, err)
	}

	password, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, provider.Password.SecretRef)
	if err != nil {
		return nil, fmt.Errorf(errMissingField, err)
	}

	authopts := gophercloud.AuthOptions{
		IdentityEndpoint: provider.AuthURL,
		TenantName:       provider.TenantName,
		DomainName:       provider.DomainName,
		Username:         username,
		Password:         password,
	}

	auth, err := openstack.AuthenticatedClient(ctx, authopts)
	if err != nil {
		return nil, fmt.Errorf(errAuthFailed, err)
	}

	barbicanClient, err := openstack.NewKeyManagerV1(auth, gophercloud.EndpointOpts{
		Region: provider.Region,
	})
	if err != nil {
		return nil, fmt.Errorf(errClientInit, err)
	}

	c := &Client{
		keyManager: barbicanClient,
	}

	return c, nil
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Barbican: &esv1.BarbicanProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
