/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

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

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

var (
	defaultAPIURL = "https://api.scaleway.com"
	log           = ctrl.Log.WithName("provider").WithName("scaleway")
)

type Provider struct{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) ApplyReferent(spec kubeClient.Object, _ esmeta.ReferentCallOrigin, _ string) (kubeClient.Object, error) {
	return spec, nil
}
func (p *Provider) Convert(_ esv1beta1.GenericStore) (kubeClient.Object, error) {
	return nil, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ kubeClient.Object, _ kubeClient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeClient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	if store.GetKind() == esv1beta1.ClusterSecretStoreKind && doesConfigDependOnNamespace(cfg) {
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

func loadConfigSecret(ctx context.Context, ref *esv1beta1.ScalewayProviderSecretRef, kube kubeClient.Client, defaultNamespace, storeKind string) (string, error) {
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

func validateSecretRef(store esv1beta1.GenericStore, ref *esv1beta1.ScalewayProviderSecretRef) error {
	if ref.SecretRef != nil {
		if ref.Value != "" {
			return errors.New("cannot specify both secret reference and value")
		}
		err := utils.ValidateReferentSecretSelector(store, *ref.SecretRef)
		if err != nil {
			return err
		}
	} else if ref.Value == "" {
		return errors.New("must specify either secret reference or direct value")
	}

	return nil
}

func doesConfigDependOnNamespace(cfg *esv1beta1.ScalewayProvider) bool {
	if cfg.AccessKey.SecretRef != nil && cfg.AccessKey.SecretRef.Namespace == nil {
		return true
	}

	if cfg.SecretKey.SecretRef != nil && cfg.SecretKey.SecretRef.Namespace == nil {
		return true
	}

	return false
}

func getConfig(store esv1beta1.GenericStore) (*esv1beta1.ScalewayProvider, error) {
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

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Scaleway: &esv1beta1.ScalewayProvider{},
	})
}
