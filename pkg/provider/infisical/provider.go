/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implieclient.
See the License for the specific language governing permissions and
limitations under the License.
*/

package infisical

import (
	"context"
	"errors"

	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/api"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/constants"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	Logger = ctrl.Log.WithName("provider").WithName(constants.ProviderName)
)

type Provider struct {
	apiClient api.InfisicalApis
	apiScope  *InfisicalClientScope
}

type InfisicalClientScope struct {
	SecretPath      string
	ProjectSlug     string
	EnvironmentSlug string
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &Provider{}
var _ esv1beta1.Provider = &Provider{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Infisical: &esv1beta1.InfisicalProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Infisical == nil {
		return nil, errors.New("Invalid infisical store")
	}

	infisicalSpec := storeSpec.Provider.Infisical

	apiClient, err := api.NewApiClient(infisicalSpec.HostAPI)
	if err != nil {
		return nil, err
	}

	if infisicalSpec.Auth.Type == constants.UniversalAuth {
		universalAuthCredentials := infisicalSpec.Auth.UniversalAuthCredentials
		clientId, err := GetSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientId)
		if err != nil {
			return nil, err
		}

		clientSecret, err := GetSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientSecret)
		if err != nil {
			return nil, err
		}

		tokenManager := NewMachineTokenManger(apiClient, clientId, clientSecret)
		apiClient.SetTokenManager(tokenManager)

		if err := tokenManager.StartLifeCycle(); err != nil {
			return nil, err
		}

		return &Provider{
			apiClient: apiClient,
			apiScope: &InfisicalClientScope{
				SecretPath:      infisicalSpec.SecretsScope.SecretsPath,
				ProjectSlug:     infisicalSpec.SecretsScope.ProjectSlug,
				EnvironmentSlug: infisicalSpec.SecretsScope.EnvironmentSlug,
			},
		}, nil
	}

	return &Provider{}, errors.New("Authentication method not found")
}

func GetSecretData(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string, secret esmeta.SecretKeySelector) (string, error) {
	secretRef := esmeta.SecretKeySelector{
		Name: secret.Name,
		Key:  secret.Key,
	}
	if secret.Namespace != nil {
		secretRef.Namespace = secret.Namespace
	}

	secretData, err := resolvers.SecretKeyRef(ctx, kube, store.GetObjectKind().GroupVersionKind().Kind, namespace, &secretRef)
	if err != nil {
		return "", err
	}
	return secretData, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	infisicalStoreSpec := storeSpec.Provider.Infisical

	if infisicalStoreSpec.SecretsScope.EnvironmentSlug == "" || infisicalStoreSpec.SecretsScope.ProjectSlug == "" {
		return nil, errors.New("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty")
	}

	if infisicalStoreSpec.Auth.Type == constants.UniversalAuth {
		uaCredential := infisicalStoreSpec.Auth.UniversalAuthCredentials
		err := utils.ValidateSecretSelector(store, uaCredential.ClientId)
		if err != nil {
			return nil, err
		}
		if uaCredential.ClientId.Key == "" || uaCredential.ClientSecret.Key == "" {
			return nil, errors.New("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty")
		}
	}

	return nil, nil
}
