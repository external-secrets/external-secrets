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
	"fmt"

	infisicalSdk "github.com/infisical/go-sdk"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/metrics"
	"github.com/external-secrets/external-secrets/pkg/provider/infisical/constants"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	machineIdentityLoginViaUniversalAuth = "MachineIdentityLoginViaUniversalAuth"
	machineIdentityLoginViaAzureAuth     = "MachineIdentityLoginViaAzureAuth"
	getSecretsV3                         = "GetSecretsV3"
	getSecretByKeyV3                     = "GetSecretByKeyV3"
	revokeAccessToken                    = "RevokeAccessToken"
)

type Provider struct {
	cancelSdkClient context.CancelFunc
	sdkClient       infisicalSdk.InfisicalClientInterface
	apiScope        *InfisicalClientScope
}

type InfisicalClientScope struct {
	EnvironmentSlug        string
	ProjectSlug            string
	Recursive              bool
	SecretPath             string
	ExpandSecretReferences bool
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Provider{}
var _ esv1.Provider = &Provider{}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Infisical: &esv1.InfisicalProvider{},
	}, esv1.MaintenanceStatusMaintained)
}

func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Infisical == nil {
		return nil, errors.New("invalid infisical store")
	}

	infisicalSpec := storeSpec.Provider.Infisical

	ctx, cancelSdkClient := context.WithCancel(ctx)

	sdkClient := infisicalSdk.NewInfisicalClient(ctx, infisicalSdk.Config{
		SiteUrl: infisicalSpec.HostAPI,
	})
	secretPath := infisicalSpec.SecretsScope.SecretsPath
	if secretPath == "" {
		secretPath = "/"
	}

	if infisicalSpec.Auth.UniversalAuthCredentials != nil {
		universalAuthCredentials := infisicalSpec.Auth.UniversalAuthCredentials
		clientID, err := GetStoreSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientID)
		if err != nil {
			cancelSdkClient()
			return nil, err
		}

		clientSecret, err := GetStoreSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientSecret)
		if err != nil {
			cancelSdkClient()
			return nil, err
		}

		_, err = sdkClient.Auth().UniversalAuthLogin(clientID, clientSecret)
		metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaUniversalAuth, err)

		if err != nil {
			cancelSdkClient()
			return nil, fmt.Errorf("failed to authenticate via universal auth %w", err)
		}
	} else if infisicalSpec.Auth.AzureAuthCredentials != nil {
		azureAuthCredentials := infisicalSpec.Auth.AzureAuthCredentials
		identityID, err := GetStoreSecretData(ctx, store, kube, namespace, azureAuthCredentials.IdentityID)
		if err != nil {
			cancelSdkClient()
			return nil, err
		}

		resource := ""
		if azureAuthCredentials.Resource.Name != "" {
			resource, err = GetStoreSecretData(ctx, store, kube, namespace, azureAuthCredentials.Resource)
			if err != nil {
				cancelSdkClient()
				return nil, err
			}
		}

		_, err = sdkClient.Auth().AzureAuthLogin(identityID, resource)
		metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaAzureAuth, err)

		if err != nil {
			cancelSdkClient()
			return nil, fmt.Errorf("failed to authenticate via azure auth %w", err)
		}
	} else {
		cancelSdkClient()
		return &Provider{}, errors.New("authentication method not found")
	}

	return &Provider{
		cancelSdkClient: cancelSdkClient,
		sdkClient:       sdkClient,

		apiScope: &InfisicalClientScope{
			EnvironmentSlug:        infisicalSpec.SecretsScope.EnvironmentSlug,
			ProjectSlug:            infisicalSpec.SecretsScope.ProjectSlug,
			Recursive:              infisicalSpec.SecretsScope.Recursive,
			SecretPath:             secretPath,
			ExpandSecretReferences: infisicalSpec.SecretsScope.ExpandSecretReferences,
		},
	}, nil
}

func (p *Provider) Close(ctx context.Context) error {
	p.cancelSdkClient()
	err := p.sdkClient.Auth().RevokeAccessToken()
	metrics.ObserveAPICall(constants.ProviderName, revokeAccessToken, err)

	if err != nil {
		return err
	}

	return nil
}

func GetStoreSecretData(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string, secret esmeta.SecretKeySelector) (string, error) {
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

func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	infisicalStoreSpec := storeSpec.Provider.Infisical
	if infisicalStoreSpec == nil {
		return nil, errors.New("invalid infisical store")
	}

	if infisicalStoreSpec.SecretsScope.EnvironmentSlug == "" || infisicalStoreSpec.SecretsScope.ProjectSlug == "" {
		return nil, errors.New("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty")
	}

	if infisicalStoreSpec.Auth.UniversalAuthCredentials != nil {
		uaCredential := infisicalStoreSpec.Auth.UniversalAuthCredentials
		// to validate reference authentication
		err := utils.ValidateReferentSecretSelector(store, uaCredential.ClientID)
		if err != nil {
			return nil, err
		}

		err = utils.ValidateReferentSecretSelector(store, uaCredential.ClientSecret)
		if err != nil {
			return nil, err
		}

		if uaCredential.ClientID.Key == "" || uaCredential.ClientSecret.Key == "" {
			return nil, errors.New("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty")
		}
	}

	return nil, nil
}
