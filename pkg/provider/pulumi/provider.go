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

package pulumi

import (
	"context"
	"errors"
	"fmt"

	esc "github.com/pulumi/esc-sdk/sdk/go"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
)

// Provider implements the esv1.Provider interface for Pulumi ESC.
type Provider struct{}

var _ esv1.Provider = &Provider{}

const (
	errClusterStoreRequiresNamespace = "cluster store requires namespace"
	errCannotResolveSecretKeyRef     = "cannot resolve secret key ref: %w"
	errStoreIsNil                    = "store is nil"
	errNoStoreTypeOrWrongStoreType   = "no store type or wrong store type"
	errOrganizationIsRequired        = "organization is required"
	errEnvironmentIsRequired         = "environment is required"
	errProjectIsRequired             = "project is required"
	errSecretRefNameIsRequired       = "secretRef.name is required"
	errSecretRefKeyIsRequired        = "secretRef.key is required"
)

// NewClient creates a new Pulumi ESC client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}
	storeKind := store.GetKind()
	if storeKind == esv1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
		return nil, errors.New(errClusterStoreRequiresNamespace)
	}
	accessToken, err := loadAccessTokenSecret(ctx, cfg.AccessToken, kube, storeKind, namespace)
	if err != nil {
		return nil, err
	}
	configuration := esc.NewConfiguration()
	configuration.UserAgent = "external-secrets-operator"
	configuration.Servers = esc.ServerConfigurations{
		esc.ServerConfiguration{
			URL: cfg.APIURL,
		},
	}
	authCtx := esc.NewAuthContext(accessToken)
	escClient := esc.NewClient(configuration)
	return &client{
		escClient:    *escClient,
		authCtx:      authCtx,
		project:      cfg.Project,
		environment:  cfg.Environment,
		organization: cfg.Organization,
	}, nil
}

func loadAccessTokenSecret(ctx context.Context, ref *esv1.PulumiProviderSecretRef, kube kclient.Client, storeKind, namespace string) (string, error) {
	acctoken, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, ref.SecretRef)
	if err != nil {
		return "", fmt.Errorf(errCannotResolveSecretKeyRef, err)
	}
	return acctoken, nil
}

func doesConfigDependOnNamespace(cfg *esv1.PulumiProvider) bool {
	if cfg.AccessToken.SecretRef != nil && cfg.AccessToken.SecretRef.Namespace == nil {
		return true
	}
	return false
}

func getConfig(store esv1.GenericStore) (*esv1.PulumiProvider, error) {
	if store == nil {
		return nil, errors.New(errStoreIsNil)
	}
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.Pulumi == nil {
		return nil, errors.New(errNoStoreTypeOrWrongStoreType)
	}
	cfg := spec.Provider.Pulumi

	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.pulumi.com/api/esc"
	}

	if cfg.Organization == "" {
		return nil, errors.New(errOrganizationIsRequired)
	}
	if cfg.Environment == "" {
		return nil, errors.New(errEnvironmentIsRequired)
	}
	if cfg.Project == "" {
		return nil, errors.New(errProjectIsRequired)
	}
	err := validateStoreSecretRef(store, cfg.AccessToken)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func validateStoreSecretRef(store esv1.GenericStore, ref *esv1.PulumiProviderSecretRef) error {
	if ref != nil {
		if err := esutils.ValidateReferentSecretSelector(store, *ref.SecretRef); err != nil {
			return err
		}
	}
	return validateSecretRef(ref)
}

func validateSecretRef(ref *esv1.PulumiProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.SecretRef.Name == "" {
			return errors.New(errSecretRefNameIsRequired)
		}
		if ref.SecretRef.Key == "" {
			return errors.New(errSecretRefKeyIsRequired)
		}
	}
	return nil
}

// ValidateStore validates the store's configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

// Capabilities returns the provider's esv1.SecretStoreCapabilities.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Pulumi: &esv1.PulumiProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
