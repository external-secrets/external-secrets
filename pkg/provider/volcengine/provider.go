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

package volcengine

import (
	"context"
	"errors"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/volcengine/volcengine-go-sdk/service/kms"
)

var _ esv1.Provider = &Provider{}

// Provider implements the actual SecretsClient interface.
type Provider struct{}

// NewClient implements v1.Provider.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	volcengineProvider, err := getVolcengineProvider(store)
	if err != nil {
		return nil, err
	}

	sess, err := NewSession(ctx, volcengineProvider, kube, namespace)
	if err != nil {
		return nil, err
	}

	kms := kms.New(sess)
	return NewClient(kms), nil
}

// ValidateStore implements v1.Provider.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	volcengineProvider, err := getVolcengineProvider(store)
	if err != nil {
		return nil, err
	}

	if volcengineProvider.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	// Use IRSA as auth is not specified.
	if volcengineProvider.Auth == nil {
		return nil, nil
	}

	return nil, validateAuthSecretRef(store, volcengineProvider.Auth.SecretRef)
}

// Capabilities implements v1.Provider.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// validateAuthSecretRef validates the SecretRef for static credentials.
func validateAuthSecretRef(store esv1.GenericStore, ref *esv1.VolcengineAuthSecretRef) error {
	if ref == nil {
		return errors.New("SecretRef is required when using static credentials")
	}
	if err := esutils.ValidateReferentSecretSelector(store, ref.AccessKeyID); err != nil {
		return fmt.Errorf("invalid AccessKeyID: %w", err)
	}
	if err := esutils.ValidateReferentSecretSelector(store, ref.SecretAccessKey); err != nil {
		return fmt.Errorf("invalid SecretAccessKey: %w", err)
	}
	if ref.Token != nil {
		if err := esutils.ValidateReferentSecretSelector(store, *ref.Token); err != nil {
			return fmt.Errorf("invalid Token: %w", err)
		}
	}
	return nil
}

// getVolcengineProvider gets the VolcengineProvider from the store spec.
func getVolcengineProvider(store esv1.GenericStore) (*esv1.VolcengineProvider, error) {
	spec := store.GetSpec()
	if spec.Provider == nil || spec.Provider.Volcengine == nil {
		return nil, fmt.Errorf("volcengine provider is nil")
	}
	return spec.Provider.Volcengine, nil
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Volcengine: &esv1.VolcengineProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
