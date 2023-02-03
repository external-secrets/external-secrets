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
	"fmt"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	"github.com/scaleway/scaleway-sdk-go/validation"
	corev1 "k8s.io/api/core/v1"

	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	defaultApiUrl = "https://api.scaleway.com"
)

type Provider struct{}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeClient.Client, namespace string) (esv1beta1.SecretsClient, error) {

	cfg, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	accessKey, err := loadConfigSecret(ctx, cfg.AccessKey, kube, namespace)
	if err != nil {
		return nil, err
	}

	secretKey, err := loadConfigSecret(ctx, cfg.SecretKey, kube, namespace)
	if err != nil {
		return nil, err
	}

	scwClient, err := scw.NewClient(
		scw.WithAPIURL(cfg.ApiUrl),
		scw.WithDefaultRegion(scw.Region(cfg.Region)),
		scw.WithDefaultProjectID(cfg.ProjectId),
		scw.WithAuth(accessKey, secretKey),
	)
	if err != nil {
		return nil, err
	}

	return &client{
		api:       smapi.NewAPI(scwClient),
		projectId: cfg.ProjectId,
	}, nil
}

func loadConfigSecret(ctx context.Context, ref *esv1beta1.ScalewayProviderSecretRef, kube kubeClient.Client, defaultNamespace string) (string, error) {

	var emptySecretKeySelector esmeta.SecretKeySelector

	if ref.SecretRef == emptySecretKeySelector {
		return ref.Value, nil
	}

	namespace := defaultNamespace
	if ref.SecretRef.Namespace != nil {
		namespace = *ref.SecretRef.Namespace
	}

	if ref.SecretRef.Name == "" {
		return "", fmt.Errorf("must specify a value or a reference to a secret")
	}

	if ref.SecretRef.Key == "" {
		return "", fmt.Errorf("must specify a secret key")
	}

	objKey := kubeClient.ObjectKey{
		Namespace: namespace,
		Name:      ref.SecretRef.Name,
	}

	secret := corev1.Secret{}

	err := kube.Get(ctx, objKey, &secret)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[ref.SecretRef.Key]
	if !ok {
		return "", fmt.Errorf("no such key in secret: %v", ref.SecretRef.Key)
	}

	return string(value), nil
}

func validateSecretRef(store esv1beta1.GenericStore, ref *esv1beta1.ScalewayProviderSecretRef) error {

	var emptySecretKeySelector esmeta.SecretKeySelector
	if ref.SecretRef != emptySecretKeySelector {
		if ref.Value != "" {
			return fmt.Errorf("cannot specify both secret reference and value")
		}
		err := utils.ValidateReferentSecretSelector(store, ref.SecretRef)
		if err != nil {
			return err
		}
	} else if ref.Value == "" {
		return fmt.Errorf("must specify either secret refernce or direct value")
	}

	return nil
}

func getConfig(store esv1beta1.GenericStore) (*esv1beta1.ScalewayProvider, error) {

	if store == nil {
		return nil, fmt.Errorf("missing store specification")
	}
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Scaleway == nil {
		return nil, fmt.Errorf("invalid specification for scaleway provider")
	}
	cfg := storeSpec.Provider.Scaleway

	if cfg.ApiUrl == "" {
		cfg.ApiUrl = defaultApiUrl
	} else if !validation.IsURL(cfg.ApiUrl) {
		return nil, fmt.Errorf("invalid api url: %q", cfg.ApiUrl)
	}

	if !validation.IsRegion(cfg.Region) {
		return nil, fmt.Errorf("invalid region: %q", cfg.Region)
	}

	if !validation.IsProjectID(cfg.ProjectId) {
		return nil, fmt.Errorf("invalid project id: %q", cfg.ProjectId)
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

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	_, err := getConfig(store)
	return err
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Scaleway: &esv1beta1.ScalewayProvider{},
	})
}
