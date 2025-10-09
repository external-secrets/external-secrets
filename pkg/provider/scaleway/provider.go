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

package scaleway

import (
	"context"
	"errors"
	"fmt"

	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1beta1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/validation"
	ctrl "sigs.k8s.io/controller-runtime"
	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/external-secrets/external-secrets/pkg/esutils/resolvers"
)

var (
	defaultAPIURL = "https://api.scaleway.com"
	log           = ctrl.Log.WithName("provider").WithName("scaleway")
)

var _ esv1.Provider = &Provider{}

// Provider is a Scaleway provider implementation that satisfies the esv1.Provider interface.
type Provider struct{}

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
		return nil, errors.New("when using a ClusterSecretStore, namespaces must be explicitly set")
	}

	accessKey, err := loadConfigSecret(ctx, cfg.AccessKey, kube, namespace, store.GetKind())
	if err != nil {
		return nil, err
	}

	secretKey, err := loadConfigSecret(ctx, cfg.SecretKey, kube, namespace, store.GetKind())
	if err != nil {
		return nil, err
	}

	scwClient, err := scw.NewClient(
		scw.WithAPIURL(cfg.APIURL),
		scw.WithDefaultRegion(scw.Region(cfg.Region)),
		scw.WithDefaultProjectID(cfg.ProjectID),
		scw.WithAuth(accessKey, secretKey),
		scw.WithUserAgent("external-secrets"),
	)
	if err != nil {
		return nil, err
	}

	return &client{
		api:       smapi.NewAPI(scwClient),
		projectID: cfg.ProjectID,
		cache:     newCache(),
	}, nil
}

func loadConfigSecret(ctx context.Context, ref *esv1.ScalewayProviderSecretRef, kube kubeClient.Client, defaultNamespace, storeKind string) (string, error) {
	if ref.SecretRef == nil {
		return ref.Value, nil
	}
	return resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		defaultNamespace,
		ref.SecretRef,
	)
}

func validateSecretRef(store esv1.GenericStore, ref *esv1.ScalewayProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.Value != "" {
			return errors.New("cannot specify both secret reference and value")
		}
		err := esutils.ValidateReferentSecretSelector(store, *ref.SecretRef)
		if err != nil {
			return err
		}
	} else if ref.Value == "" {
		return errors.New("must specify either secret reference or direct value")
	}

	return nil
}

func doesConfigDependOnNamespace(cfg *esv1.ScalewayProvider) bool {
	if cfg.AccessKey.SecretRef != nil && cfg.AccessKey.SecretRef.Namespace == nil {
		return true
	}

	if cfg.SecretKey.SecretRef != nil && cfg.SecretKey.SecretRef.Namespace == nil {
		return true
	}

	return false
}

func getConfig(store esv1.GenericStore) (*esv1.ScalewayProvider, error) {
	if store == nil {
		return nil, errors.New("missing store specification")
	}
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Scaleway == nil {
		return nil, errors.New("invalid specification for scaleway provider")
	}
	cfg := storeSpec.Provider.Scaleway

	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	} else if !validation.IsURL(cfg.APIURL) {
		return nil, fmt.Errorf("invalid api url: %q", cfg.APIURL)
	}

	if !validation.IsRegion(cfg.Region) {
		return nil, fmt.Errorf("invalid region: %q", cfg.Region)
	}

	if !validation.IsProjectID(cfg.ProjectID) {
		return nil, fmt.Errorf("invalid project id: %q", cfg.ProjectID)
	}

	err := validateSecretRef(store, cfg.AccessKey)
	if err != nil {
		return nil, err
	}

	err = validateSecretRef(store, cfg.SecretKey)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

// ValidateStore validates the store's configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Scaleway: &esv1.ScalewayProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
