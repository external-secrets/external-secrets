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

package onboardbase

import (
	"context"
	"errors"
	"fmt"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	oClient "github.com/external-secrets/external-secrets/pkg/provider/onboardbase/client"
)

const (
	errNewClient        = "unable to create OnboardbaseClient : %s"
	errInvalidStore     = "invalid store: %s"
	errOnboardbaseStore = "missing or invalid Onboardbase SecretStore"
)

// Provider is a Onboardbase secrets provider implementing NewClient and ValidateStore for the esv1.Provider interface.
type Provider struct{}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Client{}
var _ esv1.Provider = &Provider{}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Onboardbase: &esv1.OnboardbaseProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

// Capabilities returns the provider's supported capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// NewClient creates a new Onboardbase client with the provided store configuration.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Onboardbase == nil {
		return nil, errors.New(errOnboardbaseStore)
	}

	onboardbaseStoreSpec := storeSpec.Provider.Onboardbase

	client := &Client{
		kube:      kube,
		store:     onboardbaseStoreSpec,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	if err := client.setAuth(ctx); err != nil {
		return nil, err
	}

	onboardbaseClient, err := oClient.NewOnboardbaseClient(client.onboardbaseAPIKey, client.onboardbasePasscode)
	if err != nil {
		return nil, fmt.Errorf(errNewClient, err)
	}

	client.onboardbase = onboardbaseClient
	client.project = client.store.Project
	client.environment = client.store.Environment

	return client, nil
}

// ValidateStore validates the Onboardbase SecretStore configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	onboardbaseStoreSpec := storeSpec.Provider.Onboardbase
	onboardbaseAPIKeySecretRef := onboardbaseStoreSpec.Auth.OnboardbaseAPIKeyRef
	if err := esutils.ValidateSecretSelector(store, onboardbaseAPIKeySecretRef); err != nil {
		return nil, fmt.Errorf(errInvalidStore, err)
	}

	if onboardbaseAPIKeySecretRef.Name == "" {
		return nil, fmt.Errorf(errInvalidStore, "onboardbaseAPIKey.name cannot be empty")
	}

	onboardbasePasscodeKeySecretRef := onboardbaseStoreSpec.Auth.OnboardbasePasscodeRef
	if err := esutils.ValidateSecretSelector(store, onboardbasePasscodeKeySecretRef); err != nil {
		return nil, fmt.Errorf(errInvalidStore, err)
	}

	if onboardbasePasscodeKeySecretRef.Name == "" {
		return nil, fmt.Errorf(errInvalidStore, "onboardbasePasscode.name cannot be empty")
	}

	return nil, nil
}
