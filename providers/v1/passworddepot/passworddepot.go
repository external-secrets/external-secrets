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

// Package passworddepot implements a SecretStore provider for PasswordDepot.
package passworddepot

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

// Requires PASSWORDDEPOT_TOKEN and PASSWORDDEPOT_PROJECT_ID to be set in environment variables

const (
	errPasswordDepotCredSecretName            = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errMissingSAK                             = "missing credentials while setting auth"
	errUninitalizedPasswordDepotProvider      = "provider passworddepot is not initialized"
	errNotImplemented                         = "%s not implemented"
)

// Client defines the interface for interacting with the PasswordDepot API.
type Client interface {
	GetSecret(database, key string) (SecretEntry, error)
}

// PasswordDepot Provider struct with reference to a PasswordDepot client and a projectID.
type PasswordDepot struct {
	client   Client
	database string
}

// ValidateStore validates the PasswordDepot SecretStore resource configuration.
func (p *PasswordDepot) ValidateStore(esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *PasswordDepot) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// Client for interacting with kubernetes cluster...?
type passwordDepotClient struct {
	kube      kclient.Client
	store     *esv1.PasswordDepotProvider
	namespace string
	storeKind string
}

// Provider represents the PasswordDepot provider configuration.
type Provider struct{}

func (c *passwordDepotClient) getAuth(ctx context.Context) (string, string, error) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.Credentials.Name
	if credentialsSecretName == "" {
		return "", "", errors.New(errPasswordDepotCredSecretName)
	}
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: c.namespace,
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if c.storeKind == esv1.ClusterSecretStoreKind {
		if c.store.Auth.SecretRef.Credentials.Namespace == nil {
			return "", "", errors.New(errInvalidClusterStoreMissingSAKNamespace)
		}
		objectKey.Namespace = *c.store.Auth.SecretRef.Credentials.Namespace
	}

	err := c.kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return "", "", fmt.Errorf(errFetchSAKSecret, err)
	}

	username := credentialsSecret.Data["username"]
	password := credentialsSecret.Data["password"]
	if (username == nil) || (len(username) == 0 || password == nil) || (len(password) == 0) {
		return "", "", errors.New(errMissingSAK)
	}

	return string(username), string(password), nil
}

// NewClient constructs a new secrets client based on the provided store.
func (p *PasswordDepot) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.PasswordDepot == nil {
		return nil, errors.New("no store type or wrong store type")
	}
	storeSpecPasswordDepot := storeSpec.Provider.PasswordDepot

	cliStore := passwordDepotClient{
		kube:      kube,
		store:     storeSpecPasswordDepot,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	username, password, err := cliStore.getAuth(ctx)
	if err != nil {
		return nil, err
	}

	// Create a new PasswordDepot client using credentials and options
	passworddepotClient, err := NewAPI(ctx, storeSpecPasswordDepot.Host, username, password, "8714")
	if err != nil {
		return nil, err
	}

	p.client = passworddepotClient
	p.database = storeSpecPasswordDepot.Database

	return p, nil
}

// SecretExists checks if the secret exists in the PasswordDepot. This method is not implemented
// as PasswordDepot is read-only.
func (p *PasswordDepot) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, fmt.Errorf(errNotImplemented, "SecretExists")
}

// Validate performs validation of the PasswordDepot provider configuration.
func (p *PasswordDepot) Validate() (esv1.ValidationResult, error) {
	return esv1.ValidationResultReady, nil
}

// PushSecret is not implemented for PasswordDepot as it is read-only.
func (p *PasswordDepot) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return fmt.Errorf(errNotImplemented, "PushSecret")
}

// GetAllSecrets retrieves all secrets from PasswordDepot that match the given criteria.
func (p *PasswordDepot) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, fmt.Errorf(errNotImplemented, "GetAllSecrets")
}

// DeleteSecret is not implemented for PasswordDepot as it is read-only.
func (p *PasswordDepot) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return fmt.Errorf(errNotImplemented, "DeleteSecret")
}

// GetSecret retrieves a secret from PasswordDepot.
func (p *PasswordDepot) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if esutils.IsNil(p.client) {
		return nil, errors.New(errUninitalizedPasswordDepotProvider)
	}

	data, err := p.client.GetSecret(p.database, ref.Key)
	if err != nil {
		return nil, err
	}
	mappedData := data.ToMap()
	value, ok := mappedData[ref.Property]
	if !ok {
		return nil, errors.New("key not found in secret data")
	}

	return value, nil
}

// GetSecretMap retrieves a secret and returns it as a map of key/value pairs.
func (p *PasswordDepot) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := p.client.GetSecret(p.database, ref.Key)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	return data.ToMap(), nil
}

// Close implements cleanup operations for the PasswordDepot provider.
func (p *PasswordDepot) Close(_ context.Context) error {
	return nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &PasswordDepot{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		PasswordDepot: &esv1.PasswordDepotProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
