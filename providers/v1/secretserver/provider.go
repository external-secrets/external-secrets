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

package secretserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"

	"github.com/DelineaXPM/tss-sdk-go/v3/server"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var (
	errEmptyUserName                 = errors.New("username must not be empty")
	errEmptyPassword                 = errors.New("password must be set")
	errEmptyServerURL                = errors.New("serverURL must be set")
	errMissingStore                  = errors.New("missing store specification")
	errInvalidSpec                   = errors.New("invalid specification for secret server provider")
	errClusterStoreRequiresNamespace = errors.New("when using a ClusterSecretStore, namespaces must be explicitly set")
	errMissingSecretName             = errors.New("must specify a secret name")
	errMissingSecretKey              = errors.New("must specify a secret key")
)

// Provider struct that implements the ESO esv1.Provider.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient creates a new secrets client based on provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kubeClient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}
	if store.GetKind() == esv1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
		// we are not attached to a specific namespace, but some config values are dependent on it
		return nil, errClusterStoreRequiresNamespace
	}

	credentials, err := loadCredentials(ctx, store, cfg, kube, namespace)
	if err != nil {
		return nil, err
	}

	ssConfig := server.Configuration{
		Credentials: credentials,
		ServerURL:   cfg.ServerURL,
	}

	if len(cfg.CABundle) > 0 || cfg.CAProvider != nil {
		cert, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			StoreKind:  store.GetKind(),
			Client:     kube,
			Namespace:  namespace,
			CABundle:   cfg.CABundle,
			CAProvider: cfg.CAProvider,
		})
		if err != nil {
			return nil, err
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(cert) {
			return nil, errors.New("failed to append caBundle")
		}

		ssConfig.TLSClientConfig = &tls.Config{
			RootCAs:    caCertPool,
			MinVersion: tls.VersionTLS12,
		}
	}

	secretServer, err := server.New(ssConfig)
	if err != nil {
		return nil, err
	}

	return &client{
		api: secretServer,
	}, nil
}

func loadConfigSecret(
	ctx context.Context,
	store esv1.GenericStore,
	ref *esv1.SecretServerProviderRef,
	kube kubeClient.Client,
	namespace string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}
	if err := validateStoreSecretRef(store, ref); err != nil {
		return "", err
	}
	return resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, ref.SecretRef)
}

func loadCredentials(ctx context.Context, store esv1.GenericStore, cfg *esv1.SecretServerProvider, kube kubeClient.Client, namespace string) (server.UserCredential, error) {
	if cfg.Token != nil {
		token, err := loadConfigSecret(ctx, store, cfg.Token, kube, namespace)
		if err != nil {
			return server.UserCredential{}, err
		}
		return server.UserCredential{Token: token}, nil
	}

	username, err := loadConfigSecret(ctx, store, cfg.Username, kube, namespace)
	if err != nil {
		return server.UserCredential{}, err
	}
	password, err := loadConfigSecret(ctx, store, cfg.Password, kube, namespace)
	if err != nil {
		return server.UserCredential{}, err
	}
	return server.UserCredential{
		Username: username,
		Password: password,
		Domain:   cfg.Domain,
	}, nil
}

// secretServerCredentialRefPolicy returns the validation policy for Secret Server credential fields.
func secretServerCredentialRefPolicy(store esv1.GenericStore) esutils.ValueOrRefPolicy[esmeta.SecretKeySelector] {
	return esutils.ValueOrRefPolicy[esmeta.SecretKeySelector]{
		Presence:    esutils.RequireValueOrRef,
		ValidateRef: validateSecretServerCredentialSecretRef(store),
	}
}

// validateSecretServerCredentialSecretRef validates a Secret Server credential secret reference against the store scope.
func validateSecretServerCredentialSecretRef(store esv1.GenericStore) func(esmeta.SecretKeySelector) error {
	return func(ref esmeta.SecretKeySelector) error {
		if err := esutils.ValidateReferentSecretSelector(store, ref); err != nil {
			return err
		}
		if ref.Name == "" {
			return errMissingSecretName
		}
		if ref.Key == "" {
			return errMissingSecretKey
		}
		return nil
	}
}

// validateStoreSecretRef validates a Secret Server credential reference against the store scope.
func validateStoreSecretRef(store esv1.GenericStore, ref *esv1.SecretServerProviderRef) error {
	return esutils.ValidateValueOrRef(ref.Value, ref.SecretRef, secretServerCredentialRefPolicy(store))
}

func validateSecretRef(ref *esv1.SecretServerProviderRef) error {
	return esutils.ValidateValueOrRef(ref.Value, ref.SecretRef, esutils.ValueOrRefPolicy[esmeta.SecretKeySelector]{
		Presence:    esutils.RequireValueOrRef,
		ValidateRef: validateSecretServerCredentialSecretRefNameAndKey,
	})
}

// validateSecretServerCredentialSecretRefNameAndKey ensures a Secret Server credential secret reference has both name and key.
func validateSecretServerCredentialSecretRefNameAndKey(ref esmeta.SecretKeySelector) error {
	if ref.Name == "" {
		return errMissingSecretName
	}
	if ref.Key == "" {
		return errMissingSecretKey
	}
	return nil
}

func doesConfigDependOnNamespace(cfg *esv1.SecretServerProvider) bool {
	// Mirror getConfig's precedence: when Token is set, username/password are
	// ignored, so only the token ref can introduce a namespace dependency.
	if cfg.Token != nil {
		return cfg.Token.SecretRef != nil && cfg.Token.SecretRef.Namespace == nil
	}
	if cfg.Username != nil && cfg.Username.SecretRef != nil && cfg.Username.SecretRef.Namespace == nil {
		return true
	}
	if cfg.Password != nil && cfg.Password.SecretRef != nil && cfg.Password.SecretRef.Namespace == nil {
		return true
	}
	return false
}

func getConfig(store esv1.GenericStore) (*esv1.SecretServerProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.SecretServer == nil {
		return nil, errInvalidSpec
	}
	cfg := storeSpec.Provider.SecretServer

	if cfg.ServerURL == "" {
		return nil, errEmptyServerURL
	}

	// Token authentication takes precedence over username/password.
	if cfg.Token != nil {
		if err := validateStoreSecretRef(store, cfg.Token); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	if cfg.Username == nil {
		return nil, errEmptyUserName
	}
	if cfg.Password == nil {
		return nil, errEmptyPassword
	}

	if err := validateStoreSecretRef(store, cfg.Username); err != nil {
		return nil, err
	}
	if err := validateStoreSecretRef(store, cfg.Password); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ValidateStore validates the store's configuration and returns warnings or error.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		SecretServer: &esv1.SecretServerProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
