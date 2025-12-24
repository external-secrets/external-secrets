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

package infisical

import (
	"context"
	"errors"
	"fmt"

	"github.com/external-secrets/external-secrets/runtime/metrics"
	"github.com/external-secrets/external-secrets/providers/v1/infisical/constants"
	infisicalSdk "github.com/infisical/go-sdk"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

const (
	machineIdentityLoginViaUniversalAuth         = "MachineIdentityLoginViaUniversalAuth"
	machineIdentityLoginViaAzureAuth             = "MachineIdentityLoginViaAzureAuth"
	machineIdentityLoginViaGCPIDTokenAuth        = "MachineIdentityLoginViaGcpIdTokenAuth"
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

// Provider implements the Infisical external secrets provider.
type Provider struct {
	cancelSdkClient context.CancelFunc
	sdkClient       infisicalSdk.InfisicalClientInterface
	apiScope        *ClientScope
	authMethod      string
}

// ClientScope represents the scope configuration for an Infisical client.
type ClientScope struct {
	EnvironmentSlug        string
	ProjectSlug            string
	Recursive              bool
	SecretPath             string
	ExpandSecretReferences bool
}

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1.SecretsClient = &Provider{}
var _ esv1.Provider = &Provider{}

// Capabilities returns the provider's supported capabilities.
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

func performGcpIDTokenAuthLogin(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error {
	gcpIDTokenAuthCredentials := infisicalSpec.Auth.GcpIDTokenAuthCredentials
	identityID, err := GetStoreSecretData(ctx, store, kube, namespace, gcpIDTokenAuthCredentials.IdentityID)
	if err != nil {
		return fmt.Errorf(errSecretDataFormat, err)
	}

	_, err = sdkClient.Auth().GcpIdTokenAuthLogin(identityID)
	metrics.ObserveAPICall(constants.ProviderName, machineIdentityLoginViaGCPIDTokenAuth, err)

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

	var privateKeyPassphrase *string
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

// NewClient creates a new Infisical client.
func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	storeSpec := store.GetSpec()

	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Infisical == nil {
		return nil, errors.New("invalid infisical store")
	}

	infisicalSpec := storeSpec.Provider.Infisical

	// Fetch CA certificate if configured
	var caCertificate string
	if len(infisicalSpec.CABundle) > 0 || infisicalSpec.CAProvider != nil {
		caCert, err := esutils.FetchCACertFromSource(ctx, esutils.CreateCertOpts{
			CABundle:   infisicalSpec.CABundle,
			CAProvider: infisicalSpec.CAProvider,
			StoreKind:  store.GetObjectKind().GroupVersionKind().Kind,
			Namespace:  namespace,
			Client:     kube,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to get CA certificate: %w", err)
		}
		if caCert != nil {
			caCertificate = string(caCert)
		}
	}

	ctx, cancelSdkClient := context.WithCancel(ctx)

	sdkClient := infisicalSdk.NewInfisicalClient(ctx, infisicalSdk.Config{
		SiteUrl:       infisicalSpec.HostAPI,
		CaCertificate: caCertificate,
	})
	secretPath := infisicalSpec.SecretsScope.SecretsPath
	if secretPath == "" {
		secretPath = "/"
	}

	var loginFn func(ctx context.Context, store esv1.GenericStore, infisicalSpec *esv1.InfisicalProvider, sdkClient infisicalSdk.InfisicalClientInterface, kube kclient.Client, namespace string) error
	var authMethod string
	switch {
	case infisicalSpec.Auth.UniversalAuthCredentials != nil:
		loginFn = performUniversalAuthLogin
		authMethod = machineIdentityLoginViaUniversalAuth
	case infisicalSpec.Auth.AzureAuthCredentials != nil:
		loginFn = performAzureAuthLogin
		authMethod = machineIdentityLoginViaAzureAuth
	case infisicalSpec.Auth.GcpIDTokenAuthCredentials != nil:
		loginFn = performGcpIDTokenAuthLogin
		authMethod = machineIdentityLoginViaGCPIDTokenAuth
	case infisicalSpec.Auth.GcpIamAuthCredentials != nil:
		loginFn = performGcpIamAuthLogin
		authMethod = machineIdentityLoginViaGcpServiceAccountAuth
	case infisicalSpec.Auth.JwtAuthCredentials != nil:
		loginFn = performJwtAuthLogin
		authMethod = machineIdentityLoginViaJwtAuth
	case infisicalSpec.Auth.LdapAuthCredentials != nil:
		loginFn = performLdapAuthLogin
		authMethod = machineIdentityLoginViaLdapAuth
	case infisicalSpec.Auth.OciAuthCredentials != nil:
		loginFn = performOciAuthLogin
		authMethod = machineIdentityLoginViaOciAuth
	case infisicalSpec.Auth.KubernetesAuthCredentials != nil:
		loginFn = performKubernetesAuthLogin
		authMethod = machineIdentityLoginViaKubernetesAuth
	case infisicalSpec.Auth.AwsAuthCredentials != nil:
		loginFn = performAwsAuthLogin
		authMethod = machineIdentityLoginViaAwsAuth
	case infisicalSpec.Auth.TokenAuthCredentials != nil:
		loginFn = performTokenAuthLogin
		authMethod = machineIdentityLoginViaTokenAuth
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
		apiScope: &ClientScope{
			EnvironmentSlug:        infisicalSpec.SecretsScope.EnvironmentSlug,
			ProjectSlug:            infisicalSpec.SecretsScope.ProjectSlug,
			Recursive:              infisicalSpec.SecretsScope.Recursive,
			SecretPath:             secretPath,
			ExpandSecretReferences: infisicalSpec.SecretsScope.ExpandSecretReferences,
		},
		authMethod: authMethod,
	}, nil
}

// Close releases any resources used by the provider.
func (p *Provider) Close(_ context.Context) error {
	p.cancelSdkClient()

	// Don't revoke token if token auth was used
	if p.authMethod == machineIdentityLoginViaTokenAuth {
		return nil
	}

	err := p.sdkClient.Auth().RevokeAccessToken()
	metrics.ObserveAPICall(constants.ProviderName, revokeAccessToken, err)

	return err
}

// GetStoreSecretData retrieves secret data from a Kubernetes secret using the provided reference.
// It handles namespace resolution and returns the secret value as a string.
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

// ValidateStore validates the Infisical SecretStore configuration.
// It checks for required fields and valid authentication settings.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	infisicalStoreSpec := storeSpec.Provider.Infisical
	if infisicalStoreSpec == nil {
		return nil, errors.New("invalid infisical store")
	}

	if infisicalStoreSpec.SecretsScope.EnvironmentSlug == "" || infisicalStoreSpec.SecretsScope.ProjectSlug == "" {
		return nil, errors.New("secretsScope.projectSlug and secretsScope.environmentSlug cannot be empty")
	}

	// Validate CAProvider namespace requirements
	if infisicalStoreSpec.CAProvider != nil {
		if store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind &&
			infisicalStoreSpec.CAProvider.Namespace == nil {
			return nil, errors.New("caProvider.namespace is required for ClusterSecretStore")
		}
		if store.GetObjectKind().GroupVersionKind().Kind == esv1.SecretStoreKind &&
			infisicalStoreSpec.CAProvider.Namespace != nil {
			return nil, errors.New("caProvider.namespace must be empty with SecretStore")
		}
	}

	if infisicalStoreSpec.Auth.UniversalAuthCredentials != nil {
		uaCredential := infisicalStoreSpec.Auth.UniversalAuthCredentials
		// to validate reference authentication
		err := esutils.ValidateReferentSecretSelector(store, uaCredential.ClientID)
		if err != nil {
			return nil, err
		}

		err = esutils.ValidateReferentSecretSelector(store, uaCredential.ClientSecret)
		if err != nil {
			return nil, err
		}

		if uaCredential.ClientID.Key == "" || uaCredential.ClientSecret.Key == "" {
			return nil, errors.New("universalAuthCredentials.clientId and universalAuthCredentials.clientSecret cannot be empty")
		}
	}

	return nil, nil
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		Infisical: &esv1.InfisicalProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
