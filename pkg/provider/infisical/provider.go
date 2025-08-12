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
	machineIdentityLoginViaUniversalAuth         = "MachineIdentityLoginViaUniversalAuth"
	machineIdentityLoginViaAzureAuth             = "MachineIdentityLoginViaAzureAuth"
	machineIdentityLoginViaGcpIdTokenAuth        = "MachineIdentityLoginViaGcpIdTokenAuth"
	machineIdentityLoginViaGcpServiceAccountAuth = "MachineIdentityLoginViaGcpServiceAccountAuth"
	machineIdentityLoginViaJwtAuth               = "MachineIdentityLoginViaJwtAuth"
	machineIdentityLoginViaLdapAuth              = "MachineIdentityLoginViaLdapAuth"
	machineIdentityLoginViaOciAuth               = "MachineIdentityLoginViaOciAuth"
	machineIdentityLoginViaKubernetesAuth        = "MachineIdentityLoginViaKubernetesAuth"
	machineIdentityLoginViaAwsAuth               = "MachineIdentityLoginViaAwsAuth"
	machineIdentityLoginViaTokenAuth             = "MachineIdentityLoginViaTokenAuth"
	revokeAccessToken                            = "RevokeAccessToken"
)

const errSecretDataFormat = "failed to get secret data identityId %w"

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

func performUniversalAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	universalAuthCredentials := infisicalSpec.Auth.UniversalAuthCredentials
	clientID, err := GetStoreSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientID)
	if err != nil {
		return err
	}

	clientSecret, err := GetStoreSecretData(ctx, store, kube, namespace, universalAuthCredentials.ClientSecret)
	if err != nil {
		return err
	}

	_, err = sdkClient.Auth().UniversalAuthLogin(clientID, clientSecret)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaUniversalAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via universal auth %w", err)
	}

	return nil
}

func performAzureAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	azureAuthCredentials := infisicalSpec.Auth.AzureAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, azureAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf("failed to get secret data id %w", err)
	}

	resource := ""
	if azureAuthCredentials.Resource.Name != "" {
		resource, err = GetStoreSecretData(ctx, store, kube, namespace, azureAuthCredentials.Resource)

		if err != nil {
			return fmt.Errorf("failed to get secret data resource %w", err)
		}
	}

	_, err = sdkClient.Auth().AzureAuthLogin(identityID, resource)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaAzureAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via azure auth %w", err)
	}

	return nil
}

func performGcpIdTokenAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	gcpIdTokenAuthCredentials := infisicalSpec.Auth.GcpIdTokenAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, gcpIdTokenAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	_, err = sdkClient.Auth().GcpIdTokenAuthLogin(identityID)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaGcpIdTokenAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via gcp id token auth %w", err)
	}

	return nil
}

func performGcpIamAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	gcpIamAuthCredentials := infisicalSpec.Auth.GcpIamAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, gcpIamAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	serviceAccountKeyFilePath, err := GetStoreSecretData(ctx, store, kube, namespace, gcpIamAuthCredentials.ServiceAccountKeyFilePath)
	if err != nil {
		return fmt.Errorf("failed to get secret data serviceAccountKeyFilePath %w", err)
	}

	_, err = sdkClient.Auth().GcpIamAuthLogin(identityID, serviceAccountKeyFilePath)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaGcpServiceAccountAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via gcp iam auth %w", err)
	}

	return nil
}

func performJwtAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	jwtAuthCredentials := infisicalSpec.Auth.JwtAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, jwtAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	jwt, err := GetStoreSecretData(ctx, store, kube, namespace, jwtAuthCredentials.JWT)
	if err != nil {
		return fmt.Errorf("failed to get secret data jwt %w", err)
	}

	_, err = sdkClient.Auth().JwtAuthLogin(identityID, jwt)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaJwtAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via jwt auth %w", err)
	}

	return nil
}

func performLdapAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	ldapAuthCredentials := infisicalSpec.Auth.LdapAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, ldapAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	ldapPassword, err := GetStoreSecretData(ctx, store, kube, namespace, ldapAuthCredentials.LDAPPassword)
	if err != nil {
		return fmt.Errorf("failed to get secret data ldapPassword %w", err)
	}

	ldapUsername, err := GetStoreSecretData(ctx, store, kube, namespace, ldapAuthCredentials.LDAPUsername)
	if err != nil {
		return fmt.Errorf("failed to get secret data ldapUsername %w", err)
	}

	_, err = sdkClient.Auth().LdapAuthLogin(identityID, ldapPassword, ldapUsername)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaLdapAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via ldap auth %w", err)
	}

	return nil
}

func performOciAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	ociAuthCredentials := infisicalSpec.Auth.OciAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	privateKey, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to get secret data privateKey %w", err)
	}

	var privateKeyPassphrase *string = nil
	if ociAuthCredentials.PrivateKeyPassphrase.Name != "" {
		passphrase, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.PrivateKeyPassphrase)
		if err != nil {
			return fmt.Errorf("failed to get secret data privateKeyPassphrase %w", err)
		}
		privateKeyPassphrase = &passphrase
	}

	fingerprint, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.Fingerprint)
	if err != nil {
		return fmt.Errorf("failed to get secret data fingerprint %w", err)
	}

	userID, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.UserID)
	if err != nil {
		return fmt.Errorf("failed to get secret data userId %w", err)
	}

	tenancyID, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.TenancyID)
	if err != nil {
		return fmt.Errorf("failed to get secret data tenancyId %w", err)
	}

	region, err := GetStoreSecretData(ctx, store, kube, namespace, ociAuthCredentials.Region)
	if err != nil {
		return fmt.Errorf("failed to get secret data region %w", err)
	}

	_, err = sdkClient.Auth().OciAuthLogin(infisicalSdk.OciAuthLoginOptions{
		IdentityID:  identityID,
		PrivateKey:  privateKey,
		Passphrase:  privateKeyPassphrase,
		Fingerprint: fingerprint,
		UserID:      userID,
		TenancyID:   tenancyID,
		Region:      region,
	})
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaOciAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via oci auth %w", err)
	}

	return nil
}

func performKubernetesAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	kubernetesAuthCredentials := infisicalSpec.Auth.KubernetesAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, kubernetesAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	serviceAccountTokenPath := ""
	if kubernetesAuthCredentials.ServiceAccountTokenPath.Name != "" {
		serviceAccountTokenPath, err = GetStoreSecretData(ctx, store, kube, namespace, kubernetesAuthCredentials.ServiceAccountTokenPath)

		if err != nil {
			return fmt.Errorf("failed to get secret data serviceAccountTokenPath %w", err)
		}
	}

	_, err = sdkClient.Auth().KubernetesAuthLogin(identityID, serviceAccountTokenPath)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaKubernetesAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via kubernetes auth %w", err)
	}

	return nil
}

func performAwsAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	awsAuthCredentials := infisicalSpec.Auth.AwsAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, awsAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	_, err = sdkClient.Auth().AwsIamAuthLogin(identityID)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaAwsAuth, err)

	if err != nil {
		return fmt.Errorf("failed to authenticate via aws auth %w", err)
	}

	return nil
}

func performTokenAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	tokenAuthCredentials := infisicalSpec.Auth.TokenAuthCredentials
	accessToken, err := GetStoreSecretData(ctx, store, kube, namespace, tokenAuthCredentials.AccessToken)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	sdkClient.Auth().SetAccessToken(accessToken)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaTokenAuth, err)

	return nil
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

	var loginFn func(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error
	switch {
	case infisicalSpec.Auth.UniversalAuthCredentials != nil:
		loginFn = performUniversalAuthLogin
	case infisicalSpec.Auth.AzureAuthCredentials != nil:
		loginFn = performAzureAuthLogin
	case infisicalSpec.Auth.GcpIdTokenAuthCredentials != nil:
		loginFn = performGcpIdTokenAuthLogin
	case infisicalSpec.Auth.GcpIamAuthCredentials != nil:
		loginFn = performGcpIamAuthLogin
	case infisicalSpec.Auth.JwtAuthCredentials != nil:
		loginFn = performJwtAuthLogin
	case infisicalSpec.Auth.LdapAuthCredentials != nil:
		loginFn = performLdapAuthLogin
	case infisicalSpec.Auth.OciAuthCredentials != nil:
		loginFn = performOciAuthLogin
	case infisicalSpec.Auth.KubernetesAuthCredentials != nil:
		loginFn = performKubernetesAuthLogin
	case infisicalSpec.Auth.AwsAuthCredentials != nil:
		loginFn = performAwsAuthLogin
	case infisicalSpec.Auth.TokenAuthCredentials != nil:
		loginFn = performTokenAuthLogin
	default:
		cancelSdkClient()
		return nil, errors.New("authentication method not found")
	}

	if err := loginFn(ctx, store, infisicalSpec, sdkClient, kube, namespace); err != nil {
		cancelSdkClient()
		return nil, err
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

	return err
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
