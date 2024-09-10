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

package acr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/go-autorest/autorest/azure"
	corev1 "k8s.io/api/core/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	kcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/yaml"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	smmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	prov "github.com/external-secrets/external-secrets/apis/providers/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider/azure/keyvault"
)

type Generator struct {
	clientSecretCreds clientSecretCredentialFunc
}

type clientSecretCredentialFunc func(tenantID string, clientID string, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error)

type TokenGetter interface {
	GetToken(ctx context.Context, opts policy.TokenRequestOptions) (azcore.AccessToken, error)
}

const (
	defaultLoginUsername = "00000000-0000-0000-0000-000000000000"

	errNoSpec     = "no config spec provided"
	errParseSpec  = "unable to parse spec: %w"
	errCreateSess = "unable to create aws session: %w"
	errGetToken   = "unable to get authorization token: %w"
)

// Generate generates a token that can be used to authenticate against Azure Container Registry.
// First, an Azure Active Directory access token is obtained with the desired authentication method.
// This AAD access token will be used to authenticate against ACR.
// Depending on the generator spec it generates an ACR access token or an ACR refresh token.
// * access tokens are scoped to a specific repository or action (pull,push)
// * refresh tokens can are scoped to whatever policy is attached to the identity that creates the acr refresh token
// details can be found here: https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview
func (g *Generator) Generate(ctx context.Context, jsonSpec *apiextensions.JSON, crClient client.Client, namespace string) (map[string][]byte, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	g.clientSecretCreds = func(tenantID, clientID, clientSecret string, options *azidentity.ClientSecretCredentialOptions) (TokenGetter, error) {
		return azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, options)
	}

	return g.generate(
		ctx,
		jsonSpec,
		crClient,
		namespace,
		kubeClient,
		fetchACRAccessToken,
		fetchACRRefreshToken)
}

func (g *Generator) generate(
	ctx context.Context,
	jsonSpec *apiextensions.JSON,
	crClient client.Client,
	namespace string,
	kubeClient kubernetes.Interface,
	fetchAccessToken accessTokenFetcher,
	fetchRefreshToken refreshTokenFetcher) (map[string][]byte, error) {
	if jsonSpec == nil {
		return nil, errors.New(errNoSpec)
	}
	res, err := parseSpec(jsonSpec.Raw)
	if err != nil {
		return nil, fmt.Errorf(errParseSpec, err)
	}
	var accessToken string
	// pick authentication strategy to create an AAD access token
	if res.Spec.Auth.ServicePrincipal != nil {
		accessToken, err = g.accessTokenForServicePrincipal(
			ctx,
			crClient,
			namespace,
			res.Spec.EnvironmentType,
			res.Spec.TenantID,
			res.Spec.Auth.ServicePrincipal.SecretRef.ClientID,
			res.Spec.Auth.ServicePrincipal.SecretRef.ClientSecret,
		)
	} else if res.Spec.Auth.ManagedIdentity != nil {
		accessToken, err = accessTokenForManagedIdentity(
			ctx,
			res.Spec.EnvironmentType,
			res.Spec.Auth.ManagedIdentity.IdentityID,
		)
	} else if res.Spec.Auth.WorkloadIdentity != nil {
		accessToken, err = accessTokenForWorkloadIdentity(
			ctx,
			crClient,
			kubeClient.CoreV1(),
			res.Spec.EnvironmentType,
			res.Spec.Auth.WorkloadIdentity.ServiceAccountRef,
			namespace,
		)
	} else {
		return nil, errors.New("unexpeted configuration")
	}
	if err != nil {
		return nil, err
	}
	var acrToken string
	acrToken, err = fetchRefreshToken(accessToken, res.Spec.TenantID, res.Spec.ACRRegistry)
	if err != nil {
		return nil, err
	}
	if res.Spec.Scope != "" {
		acrToken, err = fetchAccessToken(acrToken, res.Spec.TenantID, res.Spec.ACRRegistry, res.Spec.Scope)
		if err != nil {
			return nil, err
		}
	}

	return map[string][]byte{
		"username": []byte(defaultLoginUsername),
		"password": []byte(acrToken),
	}, nil
}

type accessTokenFetcher func(acrRefreshToken, tenantID, registryURL, scope string) (string, error)

func fetchACRAccessToken(acrRefreshToken, _, registryURL, scope string) (string, error) {
	formData := url.Values{
		"grant_type":    {"refresh_token"},
		"service":       {registryURL},
		"scope":         {scope},
		"refresh_token": {acrRefreshToken},
	}
	res, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/token", registryURL), formData)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("could not generate access token, unexpected status code: %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return "", err
	}
	accessToken, ok := payload["access_token"]
	if !ok {
		return "", errors.New("unable to get token")
	}
	return accessToken, nil
}

type refreshTokenFetcher func(aadAccessToken, tenantID, registryURL string) (string, error)

func fetchACRRefreshToken(aadAccessToken, tenantID, registryURL string) (string, error) {
	// https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md#overview
	// https://docs.microsoft.com/en-us/azure/container-registry/container-registry-authentication?tabs=azure-cli
	formData := url.Values{
		"grant_type":   {"access_token"},
		"service":      {registryURL},
		"tenant":       {tenantID},
		"access_token": {aadAccessToken},
	}
	res, err := http.PostForm(fmt.Sprintf("https://%s/oauth2/exchange", registryURL), formData)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("count not generate refresh token, unexpected status code %d, expected %d", res.StatusCode, http.StatusOK)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	var payload map[string]string
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return "", err
	}
	refreshToken, ok := payload["refresh_token"]
	if !ok {
		return "", errors.New("unable to get token")
	}
	return refreshToken, nil
}

func accessTokenForWorkloadIdentity(ctx context.Context, crClient client.Client, kubeClient kcorev1.CoreV1Interface, envType prov.AzureEnvironmentType, serviceAccountRef *smmeta.ServiceAccountSelector, namespace string) (string, error) {
	aadEndpoint := keyvault.AadEndpointForType(envType)
	scope := keyvault.ServiceManagementEndpointForType(envType)
	// if no serviceAccountRef was provided
	// we expect certain env vars to be present.
	// They are set by the azure workload identity webhook.
	if serviceAccountRef == nil {
		clientID := os.Getenv("AZURE_CLIENT_ID")
		tenantID := os.Getenv("AZURE_TENANT_ID")
		tokenFilePath := os.Getenv("AZURE_FEDERATED_TOKEN_FILE")
		if clientID == "" || tenantID == "" || tokenFilePath == "" {
			return "", errors.New("missing environment variables")
		}
		token, err := os.ReadFile(tokenFilePath)
		if err != nil {
			return "", fmt.Errorf("unable to read token file %s: %w", tokenFilePath, err)
		}
		tp, err := keyvault.NewTokenProvider(ctx, string(token), clientID, tenantID, aadEndpoint, scope)
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
		return "", fmt.Errorf("service account is missing annoation: %s", keyvault.AnnotationClientID)
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
	tp, err := keyvault.NewTokenProvider(ctx, token, clientID, tenantID, aadEndpoint, scope)
	if err != nil {
		return "", err
	}
	return tp.OAuthToken(), nil
}

func accessTokenForManagedIdentity(ctx context.Context, envType prov.AzureEnvironmentType, identityID string) (string, error) {
	// handle workload identity
	creds, err := azidentity.NewManagedIdentityCredential(
		&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ResourceID(identityID),
		},
	)
	if err != nil {
		return "", err
	}
	aud := audienceForType(envType)
	accessToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{aud},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

func (g *Generator) accessTokenForServicePrincipal(ctx context.Context, crClient client.Client, namespace string, envType prov.AzureEnvironmentType, tenantID string, idRef, secretRef smmeta.SecretKeySelector) (string, error) {
	cid, err := secretKeyRef(ctx, crClient, namespace, idRef)
	if err != nil {
		return "", err
	}
	csec, err := secretKeyRef(ctx, crClient, namespace, secretRef)
	if err != nil {
		return "", err
	}
	aadEndpoint := keyvault.AadEndpointForType(envType)
	p := azidentity.ClientSecretCredentialOptions{}
	p.Cloud.ActiveDirectoryAuthorityHost = aadEndpoint
	creds, err := g.clientSecretCreds(
		tenantID,
		cid,
		csec,
		&p)
	if err != nil {
		return "", err
	}
	aud := audienceForType(envType)
	accessToken, err := creds.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{aud},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

// secretKeyRef fetches a secret key.
func secretKeyRef(ctx context.Context, crClient client.Client, namespace string, secretRef smmeta.SecretKeySelector) (string, error) {
	var secret corev1.Secret
	ref := types.NamespacedName{
		Namespace: namespace,
		Name:      secretRef.Name,
	}
	err := crClient.Get(ctx, ref, &secret)
	if err != nil {
		return "", fmt.Errorf("unable to find namespace=%q secret=%q %w", ref.Namespace, ref.Name, err)
	}
	keyBytes, ok := secret.Data[secretRef.Key]
	if !ok {
		return "", fmt.Errorf("unable to find key=%q secret=%q namespace=%q", secretRef.Key, secretRef.Name, namespace)
	}
	value := strings.TrimSpace(string(keyBytes))
	return value, nil
}

func audienceForType(t prov.AzureEnvironmentType) string {
	suffix := ".default"
	switch t {
	case prov.AzureEnvironmentChinaCloud:
		return azure.ChinaCloud.TokenAudience + suffix
	case prov.AzureEnvironmentGermanCloud:
		return azure.GermanCloud.TokenAudience + suffix
	case prov.AzureEnvironmentUSGovernmentCloud:
		return azure.USGovernmentCloud.TokenAudience + suffix
	case prov.AzureEnvironmentPublicCloud, "":
		return azure.PublicCloud.TokenAudience + suffix
	}
	return azure.PublicCloud.TokenAudience + suffix
}

func parseSpec(data []byte) (*genv1alpha1.ACRAccessToken, error) {
	var spec genv1alpha1.ACRAccessToken
	err := yaml.Unmarshal(data, &spec)
	return &spec, err
}

func init() {
	genv1alpha1.Register(genv1alpha1.ACRAccessTokenKind, &Generator{})
}
