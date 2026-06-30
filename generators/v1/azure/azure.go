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

// Package azure provides functionality for generating Microsoft Entra ID
// access tokens scoped to a configurable Entra resource (e.g. Azure DevOps).
package azure

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/azure/keyvault"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Generator implements Microsoft Entra ID access token generation.
type Generator struct {
	clientSecretCreds clientSecretCredentialFunc
	clientCertCreds   clientCertificateCredentialFunc
}

type clientSecretCredentialFunc func(tenantID, clientID, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error)

type clientCertificateCredentialFunc func(tenantID, clientID string, certData []byte, options *azidentity.ClientCertificateCredentialOptions) (TokenGetter, error)

// TokenGetter defines an interface for obtaining Microsoft Entra ID access tokens.
type TokenGetter interface {
	GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error)
}

const (
	// tokenKey is the key under which the generated access token is returned.
	tokenKey = "token"

	errNoSpec    = "no config spec provided"
	errParseSpec = "unable to parse spec: %w"
)

// Generate mints a Microsoft Entra ID access token scoped to the configured resource
// using the desired authentication method.
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, crClient client.Client, namespace string) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, err
	}
	return g.generate(ctx, jsonSpec, crClient, namespace, kubeClient)
}

// Cleanup performs any necessary cleanup after token generation. The generator is
// stateless, so this is a no-op.
func (g *Generator) Cleanup(_ context.Context, _ *apiextensions.JSON, _ genv1alpha1.GeneratorProviderState, _ client.Client, _ string) error {
	return nil
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	crClient client.Client,
	namespace string,
	kubeClient kubernetes.Interface,
) (map[string][]byte, genv1alpha1.GeneratorProviderState, error) {
	if jsonSpec == nil {
		return nil, nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, nil, fmt.Errorf(errParseSpec, err)
	}

	resource := res.Spec.Resource
	if resource == "" {
		return nil, nil, errors.New("spec.resource is required")
	}

	var accessToken string
	switch {
	case res.Spec.Auth.ServicePrincipal != nil:
		accessToken, err = g.accessTokenForServicePrincipal(
			ctx,
			crClient,
			namespace,
			res.Spec.EnvironmentType,
			res.Spec.TenantID,
			resource,
			res.Spec.Auth.ServicePrincipal.SecretRef,
		)
	case res.Spec.Auth.ManagedIdentity != nil:
		accessToken, err = accessTokenForManagedIdentity(
			ctx,
			resource,
			res.Spec.Auth.ManagedIdentity.IdentityID,
		)
	case res.Spec.Auth.WorkloadIdentity != nil:
		accessToken, err = accessTokenForWorkloadIdentity(
			ctx,
			crClient,
			kubeClient.CoreV1(),
			res.Spec.EnvironmentType,
			resource,
			res.Spec.Auth.WorkloadIdentity.ServiceAccountRef,
			namespace,
		)
	default:
		return nil, nil, errors.New("invalid auth configuration: one of servicePrincipal, managedIdentity or workloadIdentity must be set")
	}
	if err != nil {
		return nil, nil, err
	}

	return map[string][]byte{
		tokenKey: []byte(accessToken),
	}, nil, nil
}

// scopeForResource builds the azidentity OAuth2 scope for an Entra resource id.
// The adal/workload-identity path uses the bare resource id instead (see
// accessTokenForWorkloadIdentity), matching `az account get-access-token --resource <id>`.
func scopeForResource(resource string) string {
	return strings.TrimSuffix(resource, "/") + "/.default"
}

func (g *Generator) accessTokenForServicePrincipal(
	ctx context.Context,
	crClient client.Client,
	namespace string,
	envType esv1.AzureEnvironmentType,
	tenantID string,
	resource string,
	secretRef genv1alpha1.AzureServicePrincipalAuthSecretRef,
) (string, error) {
	if (secretRef.ClientSecret == nil) == (secretRef.ClientCertificate == nil) {
		return "", errors.New("servicePrincipal auth requires exactly one of clientSecret or clientCertificate")
	}

	clientID, err := secretKeyRef(ctx, crClient, namespace, &secretRef.ClientID)
	if err != nil {
		return "", err
	}

	aadEndpoint := keyvault.AadEndpointForType(envType)
	var creds TokenGetter
	if secretRef.ClientSecret != nil {
		var clientSecret string
		clientSecret, err = secretKeyRef(ctx, crClient, namespace, secretRef.ClientSecret)
		if err != nil {
			return "", err
		}
		opts := &azidentity.ClientSecretCredentialOptions{}
		opts.Cloud.ActiveDirectoryAuthorityHost = aadEndpoint
		creds, err = g.clientSecretCreds(tenantID, clientID, clientSecret, opts)
	} else {
		var certData string
		certData, err = secretKeyRef(ctx, crClient, namespace, secretRef.ClientCertificate)
		if err != nil {
			return "", err
		}
		opts := &azidentity.ClientCertificateCredentialOptions{}
		opts.Cloud.ActiveDirectoryAuthorityHost = aadEndpoint
		creds, err = g.clientCertCreds(tenantID, clientID, []byte(certData), opts)
	}
	if err != nil {
		return "", err
	}

	accessToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scopeForResource(resource)},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

func accessTokenForManagedIdentity(ctx context.Context, resource, identityID string) (string, error) {
	var opts *azidentity.ManagedIdentityCredentialOptions
	if strings.Contains(identityID, "/") {
		opts = &azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ResourceID(identityID),
		}
	} else if identityID != "" {
		opts = &azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(identityID),
		}
	}
	// lacking option ID, az will default to the system-assigned identity.
	creds, err := azidentity.NewManagedIdentityCredential(opts)
	if err != nil {
		return "", err
	}
	accessToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{scopeForResource(resource)},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

func accessTokenForWorkloadIdentity(
	ctx context.Context,
	crClient client.Client,
	kubeClient kcorev1.CoreV1Interface,
	envType esv1.AzureEnvironmentType,
	resource string,
	serviceAccountRef *smmeta.ServiceAccountSelector,
	namespace string,
) (string, error) {
	aadEndpoint := keyvault.AadEndpointForType(envType)
	// The adal token provider takes the bare resource id (no "/.default" suffix),
	// matching `az account get-access-token --resource <id>`.
	// if no serviceAccountRef was provided we expect certain env vars to be present.
	// They are set by the azure workload identity webhook.
	if serviceAccountRef == nil {
		clientID := os.Getenv("AZURE_CLIENT_ID")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		if clientID == "" || tenantID == "" || tokenFilePath == "" {
			return "", errors.New("missing environment variables")
		}
		token, err := os.ReadFile(filepath.Clean(tokenFilePath))
		if err != nil {
			return "", fmt.Errorf("unable to read token file %s: %w", tokenFilePath, err)
		}
		tp, err := keyvault.NewTokenProvider(ctx, string(token), clientID, tenantID, aadEndpoint, resource)
		if err != nil {
			return "", err
		}
		return tp.OAuthToken(), nil
	}
	var sa corev1.ServiceAccount
	err := crClient.Get(ctx, types.NamespacedName{
		Name:      serviceAccountRef.Name,
		Namespace: namespace,
	}, &sa)
	if err != nil {
		return "", err
	}
	clientID, ok := sa.ObjectMeta.Annotations[keyvault.AnnotationClientID]
	if !ok {
		return "", fmt.Errorf("service account is missing annotation: %s", keyvault.AnnotationClientID)
	}
	tenantID, ok := sa.ObjectMeta.Annotations[keyvault.AnnotationTenantID]
	if !ok {
		return "", fmt.Errorf("service account is missing annotation: %s", keyvault.AnnotationTenantID)
	}
	audiences := []string{keyvault.AzureDefaultAudience}
	if len(serviceAccountRef.Audiences) > 0 {
		audiences = append(audiences, serviceAccountRef.Audiences...)
	}
	token, err := keyvault.FetchSAToken(ctx, namespace, serviceAccountRef.Name, audiences, kubeClient)
	if err != nil {
		return "", err
	}
	tp, err := keyvault.NewTokenProvider(ctx, token, clientID, tenantID, aadEndpoint, resource)
	if err != nil {
		return "", err
	}
	return tp.OAuthToken(), nil
}

// secretKeyRef fetches a secret key, honoring the selector namespace where the
// store kind permits it. Generators are not cluster-scoped, so resolvers.EmptyStoreKind
// keeps resolution namespace-local.
func secretKeyRef(ctx context.Context, crClient client.Client, namespace string, secretRef *smmeta.SecretKeySelector) (string, error) {
	value, err := resolvers.SecretKeyRef(ctx, crClient, resolvers.EmptyStoreKind, namespace, secretRef)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func parseSpec(data []byte) (*genv1alpha1.AzureAccessToken, error) {
	var spec genv1alpha1.AzureAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

// NewGenerator creates a new Generator instance.
func NewGenerator() genv1alpha1.Generator {
	return &Generator{
		clientSecretCreds: func(tenantID, clientID, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
			return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, options)
		},
		clientCertCreds: func(tenantID, clientID string, certData []byte, options *azidentity.ClientCertificateCredentialOptions) (TokenGetter, error) {
			certs, key, err := azidentity.ParseCertificates(certData, nil)
			if err != nil {
				return nil, fmt.Errorf("unable to parse service principal certificate: %w", err)
			}
			return azidentity.NewClientCertificateCredential(tenantID, clientID, certs, key, options)
		},
	}
}

// Kind returns the generator kind.
func Kind() string {
	return string(genv1alpha1.GeneratorKindAzureAccessToken)
}
