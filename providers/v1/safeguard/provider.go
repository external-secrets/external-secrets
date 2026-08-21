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

// Package safeguard implements a One Identity Safeguard provider for External Secrets Operator.
package safeguard

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
)

var (
	errMissingStore              = errors.New("missing store specification")
	errInvalidSpec               = errors.New("invalid specification for Safeguard provider")
	errMissingAppliance          = errors.New("appliance must be set")
	errInvalidAppliance          = errors.New("appliance must use the https scheme")
	errMissingA2AAuth            = errors.New("a2a auth must be set")
	errMissingCertificate        = errors.New("a2a certificate must be set")
	errClusterStoreRequiresNamespace = errors.New("when using a ClusterSecretStore, namespaces must be explicitly set on secret references")
)

// Provider implements the esv1.Provider interface for One Identity Safeguard.
type Provider struct{}

var _ esv1.Provider = &Provider{}

// Capabilities returns the provider supported capabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewClient creates a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube client.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}
	if store.GetKind() == esv1.ClusterSecretStoreKind && configDependsOnNamespace(cfg) {
		return nil, errClusterStoreRequiresNamespace
	}

	bootstrap, err := newA2AContext(ctx, store, cfg, kube, namespace)
	if err != nil {
		return nil, err
	}

	return &secretsClient{a2a: bootstrap}, nil
}

// ValidateStore validates the store configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}
	if err := validateAuthRef(store, "certificate", &cfg.Auth.A2A.Certificate); err != nil {
		return nil, err
	}
	if cfg.Auth.A2A.CertificateKey != nil {
		if err := validateAuthRef(store, "certificateKey", cfg.Auth.A2A.CertificateKey); err != nil {
			return nil, err
		}
	}
	if cfg.Auth.A2A.CertificatePassword != nil {
		if err := validateAuthRef(store, "certificatePassword", cfg.Auth.A2A.CertificatePassword); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

func getConfig(store esv1.GenericStore) (*esv1.SafeguardProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Safeguard == nil {
		return nil, errInvalidSpec
	}
	cfg := storeSpec.Provider.Safeguard.DeepCopy()

	appliance, err := normalizeAppliance(cfg.Appliance)
	if err != nil {
		return nil, err
	}
	cfg.Appliance = appliance

	if cfg.Auth.A2A == nil {
		return nil, errMissingA2AAuth
	}
	if cfg.Auth.A2A.Certificate.Value == "" && cfg.Auth.A2A.Certificate.SecretRef == nil {
		return nil, errMissingCertificate
	}
	return cfg, nil
}

func normalizeAppliance(appliance string) (string, error) {
	appliance = strings.TrimSpace(appliance)
	if appliance == "" {
		return "", errMissingAppliance
	}
	if !strings.Contains(appliance, "://") {
		appliance = "https://" + appliance
	}
	parsed, err := url.Parse(appliance)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return "", errInvalidAppliance
	}
	return parsed.String(), nil
}

func configDependsOnNamespace(cfg *esv1.SafeguardProvider) bool {
	if refDependsOnNamespace(&cfg.Auth.A2A.Certificate) {
		return true
	}
	if cfg.Auth.A2A.CertificateKey != nil && refDependsOnNamespace(cfg.Auth.A2A.CertificateKey) {
		return true
	}
	if cfg.Auth.A2A.CertificatePassword != nil && refDependsOnNamespace(cfg.Auth.A2A.CertificatePassword) {
		return true
	}
	return cfg.CAProvider != nil && cfg.CAProvider.Namespace == nil
}

func refDependsOnNamespace(ref *esv1.SafeguardProviderSecretRef) bool {
	return ref != nil && ref.SecretRef != nil && ref.SecretRef.Namespace == nil
}

func validateAuthRef(store esv1.GenericStore, field string, ref *esv1.SafeguardProviderSecretRef) error {
	if ref == nil {
		return nil
	}
	if err := esutils.ValidateValueOrRef(ref.Value, ref.SecretRef, safeguardRefPolicy(store)); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func safeguardRefPolicy(store esv1.GenericStore) esutils.ValueOrRefPolicy[esmeta.SecretKeySelector] {
	return esutils.ValueOrRefPolicy[esmeta.SecretKeySelector]{
		Presence: esutils.RequireValueOrRef,
		ValidateRef: func(ref esmeta.SecretKeySelector) error {
			if err := esutils.ValidateReferentSecretSelector(store, ref); err != nil {
				return err
			}
			if ref.Name == "" {
				return errors.New("must specify a secret name")
			}
			if ref.Key == "" {
				return errors.New("must specify a secret key")
			}
			return nil
		},
	}
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Safeguard: &esv1.SafeguardProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
