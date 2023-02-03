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
	smapi "github.com/scaleway/scaleway-sdk-go/api/secret/v1alpha1"
	"github.com/scaleway/scaleway-sdk-go/scw"
	corev1 "k8s.io/api/core/v1"

	kubeClient "sigs.k8s.io/controller-runtime/pkg/client"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

var (
	defaultApiUrl = "https://api.scaleway.com"
	// TODO: remove these variables or use more of them, for consistency
	errMissingStore            = fmt.Errorf("missing store provider")
	errMissingScalewayProvider = fmt.Errorf("missing store provider scaleway")
)

type SourceOrigin string

type Config struct {
	ApiUrl    string
	Region    string
	ProjectId string
	AccessKey string
	SecretKey string
}

type Provider struct {
	configs map[string]Config
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeClient.Client, namespace string) (esv1beta1.SecretsClient, error) {

	if p.configs == nil {
		p.configs = make(map[string]Config)
	}

	cfg := p.configs[store.GetName()]

	c, err := getProvider(store)
	if err != nil {
		return nil, err
	}

	cfg = Config{
		ApiUrl:    c.ApiUrl,
		Region:    c.Region,
		ProjectId: c.ProjectId,
	}

	if cfg.ApiUrl == "" {
		cfg.ApiUrl = defaultApiUrl
	}

	cfg.AccessKey, err = loadConfigSecret(ctx, c.AccessKey, kube, namespace)
	if err != nil {
		return nil, err
	}

	cfg.SecretKey, err = loadConfigSecret(ctx, c.SecretKey, kube, namespace)
	if err != nil {
		return nil, err
	}

	p.configs[store.GetName()] = cfg

	scwClient, err := scw.NewClient(
		scw.WithAPIURL(cfg.ApiUrl),
		scw.WithDefaultRegion(scw.Region(cfg.Region)),
		scw.WithDefaultProjectID(cfg.ProjectId),
		scw.WithAuth(cfg.AccessKey, cfg.SecretKey),
	)
	if err != nil {
		return nil, err
	}

	return &client{
		api:       smapi.NewAPI(scwClient),
		projectId: cfg.ProjectId,
	}, nil
}

func getProvider(store esv1beta1.GenericStore) (*esv1beta1.ScalewayProvider, error) {
	if store == nil {
		return nil, errMissingStore
	}
	spc := store.GetSpec()
	if spc == nil || spc.Provider == nil || spc.Provider.Scaleway == nil {
		return nil, errMissingScalewayProvider
	}
	return spc.Provider.Scaleway, nil
}

func loadConfigSecret(ctx context.Context, ref *esv1beta1.ScalewayProviderSecretRef, kube kubeClient.Client, defaultNamespace string) (string, error) {

	if ref.Value != "" {

		if ref.SecretNamespace != "" || ref.SecretName != "" || ref.SecretKey != "" {
			return "", fmt.Errorf("cannot specify both a value and a reference to a secret")
		}

		return ref.Value, nil
	}

	namespace := ref.SecretNamespace
	if namespace == "" {
		namespace = defaultNamespace
	}

	if ref.SecretName == "" {
		return "", fmt.Errorf("must specify a value or a reference to a secret")
	}

	if ref.SecretKey == "" {
		return "", fmt.Errorf("must specify a secret key")
	}

	objKey := kubeClient.ObjectKey{
		Namespace: namespace,
		Name:      ref.SecretName,
	}

	secret := corev1.Secret{}

	err := kube.Get(ctx, objKey, &secret)
	if err != nil {
		return "", err
	}

	value, ok := secret.Data[ref.SecretKey]
	if !ok {
		return "", fmt.Errorf("no such key in secret: %v", ref.SecretKey)
	}

	return string(value), nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	prov := store.GetSpec().Provider.Scaleway
	if prov == nil {
		return nil
	}
	// TODO
	return nil
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Scaleway: &esv1beta1.ScalewayProvider{},
	})
}
