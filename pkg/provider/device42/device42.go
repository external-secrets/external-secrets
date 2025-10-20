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

// Package device42 implements a provider for Device42 password management.
package device42

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
)

const (
	errNotImplemented                         = "not implemented"
	errUninitializedProvider                  = "unable to get device42 client"
	errCredSecretName                         = "credentials are empty"
	errInvalidClusterStoreMissingSAKNamespace = "invalid clusterStore missing SAK namespace"
	errFetchSAKSecret                         = "couldn't find secret on cluster: %w"
	errMissingSAK                             = "missing credentials while setting auth"
)

// Client defines the interface for interacting with Device42 passwords.
type Client interface {
	GetSecret(secretID string) (D42Password, error)
}

// Device42 implements the Provider interface for Device42.
type Device42 struct {
	client Client
}

// ValidateStore validates the Device42 provider configuration.
func (p *Device42) ValidateStore(esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

// Capabilities returns the provider's supported capabilities (ReadOnly).
func (p *Device42) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

// Client for interacting with kubernetes.
type device42Client struct {
	kube      kclient.Client
	store     *esv1.Device42Provider
	namespace string
	storeKind string
}

// Provider implements the external-secrets provider for Device42.
type Provider struct{}

// NewDevice42Provider returns a reference to a new instance of a 'Device42' struct.
func NewDevice42Provider() *Device42 {
	return &Device42{}
}

func (c *device42Client) getAuth(ctx context.Context) (string, string, error) {
	credentialsSecret := &corev1.Secret{}
	credentialsSecretName := c.store.Auth.SecretRef.Credentials.Name
	if credentialsSecretName == "" {
		return "", "", errors.New(errCredSecretName)
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
	if len(username) == 0 || len(password) == 0 {
		return "", "", errors.New(errMissingSAK)
	}

	return string(username), string(password), nil
}

// NewClient creates a new Device42 client.
func (p *Device42) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Device42 == nil {
		return nil, errors.New("no store type or wrong store type")
	}
	storeSpecDevice42 := storeSpec.Provider.Device42

	cliStore := device42Client{
		kube:      kube,
		store:     storeSpecDevice42,
		namespace: namespace,
		storeKind: store.GetObjectKind().GroupVersionKind().Kind,
	}

	username, password, err := cliStore.getAuth(ctx)
	if err != nil {
		return nil, err
	}
	// Create a new client using credentials and options
	p.client = NewAPI(storeSpecDevice42.Host, username, password, "443")

	return p, nil
}

// SecretExists checks if a secret exists in Device42.
func (p *Device42) SecretExists(_ context.Context, _ esv1.PushSecretRemoteRef) (bool, error) {
	return false, errors.New(errNotImplemented)
}

// Validate validates the Device42 provider configuration.
func (p *Device42) Validate() (esv1.ValidationResult, error) {
	timeout := 15 * time.Second
	url := fmt.Sprintf("https://%s:%s", p.client.(*API).baseURL, p.client.(*API).hostPort)

	if err := esutils.NetworkValidate(url, timeout); err != nil {
		return esv1.ValidationResultError, err
	}
	return esv1.ValidationResultReady, nil
}

// PushSecret creates or updates a secret in Device42.
func (p *Device42) PushSecret(_ context.Context, _ *corev1.Secret, _ esv1.PushSecretData) error {
	return errors.New(errNotImplemented)
}

// GetAllSecrets retrieves multiple secrets from Device42.
func (p *Device42) GetAllSecrets(_ context.Context, _ esv1.ExternalSecretFind) (map[string][]byte, error) {
	return nil, errors.New(errNotImplemented)
}

// DeleteSecret removes a secret from Device42.
func (p *Device42) DeleteSecret(_ context.Context, _ esv1.PushSecretRemoteRef) error {
	return errors.New(errNotImplemented)
}

// GetSecret retrieves a secret from Device42.
func (p *Device42) GetSecret(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) ([]byte, error) {
	if esutils.IsNil(p.client) {
		return nil, errors.New(errUninitializedProvider)
	}

	data, err := p.client.GetSecret(ref.Key)
	if err != nil {
		return nil, err
	}
	return []byte(data.Password), nil
}

// GetSecretMap retrieves a secret from Device42 and returns it as a map.
func (p *Device42) GetSecretMap(_ context.Context, ref esv1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	data, err := p.client.GetSecret(ref.Key)
	if err != nil {
		return nil, fmt.Errorf("error getting secret %s: %w", ref.Key, err)
	}

	return data.ToMap(), nil
}

// Close implements cleanup operations for the Device42 client.
func (p *Device42) Close(_ context.Context) error {
	return nil
}

func init() {
	esv1.Register(&Device42{}, &esv1.SecretStoreProvider{
		Device42: &esv1.Device42Provider{},
	}, esv1.MaintenanceStatusNotMaintained)
}
