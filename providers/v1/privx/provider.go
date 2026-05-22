/*
Copyright © 2026 SSH Communications

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

package privx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SSHcom/privx-sdk-go/v2/api/vault"
	"github.com/SSHcom/privx-sdk-go/v2/oauth"
	privxapi "github.com/SSHcom/privx-sdk-go/v2/restapi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	v1 "github.com/external-secrets/external-secrets/apis/meta/v1"
)

var (
	// ErrNotImplemented is returned when the requested functionality is not implemented.
	ErrNotImplemented = errors.New("not implemented")

	// ErrInvalidJSON is returned when JSON decoding fails.
	ErrInvalidJSON = errors.New("invalid JSON")

	// ErrEmptyAudience is returned when the audience value is empty.
	ErrEmptyAudience = errors.New("audience is empty")

	// ErrReadNamespace is returned when reading the namespace fails.
	ErrReadNamespace = errors.New("failed to read namespace")

	// ErrReadServiceAccount is returned when reading the service account name fails.
	ErrReadServiceAccount = errors.New("failed to read serviceaccount name")

	// ErrInClusterConfig is returned when creating in-cluster Kubernetes config fails.
	ErrInClusterConfig = errors.New("failed to create in-cluster config")

	// ErrKubernetesClient is returned when creating the Kubernetes client fails.
	ErrKubernetesClient = errors.New("failed to create kubernetes client")

	// ErrCreateToken is returned when creating a service account token fails.
	ErrCreateToken = errors.New("failed to create serviceaccount token")

	// ErrEmptyReturnedToken is returned when the token returned by the API is empty.
	ErrEmptyReturnedToken = errors.New("empty token returned")

	// ErrInvalidJWTFormat is returned when the JWT format is invalid.
	ErrInvalidJWTFormat = errors.New("invalid jwt format")

	// ErrDecodeJWTPayload is returned when decoding the JWT payload fails.
	ErrDecodeJWTPayload = errors.New("failed to decode jwt payload")

	// ErrParseJWTPayload is returned when parsing the JWT payload JSON fails.
	ErrParseJWTPayload = errors.New("failed to parse jwt payload json")

	// ErrServiceAccountNameNotFound is returned when the service account name is not found in JWT claims.
	ErrServiceAccountNameNotFound = errors.New("serviceaccount name not found in jwt claims")
)

// ErrNoStoreAuth is an error returned which details what is missing from authorisation.
type ErrNoStoreAuth struct {
	Field string
}

func (e ErrNoStoreAuth) Error() string {
	if e.Field == "" {
		return "no PrivX authorisation from SecretStore definition"
	}
	return fmt.Sprintf("no PrivX authorisation from SecretStore definition (missing %s)", e.Field)
}

// Check during compile that we implement the interface.
var _ esv1.Provider = (*Provider)(nil)

// Provider implements the ESO Provider interface for PrivX.
type Provider struct {
	// newClient is the function that actually returns a new client.
	// Tests can replace this function to create a fake client.
	newClient func(
		ctx context.Context,
		store esv1.GenericStore,
		kube kclient.Client,
		namespace string,
	) (esv1.SecretsClient, error)
}

// readSecretValue gets a Kubernetes Secret as a string.
func readSecretValue(
	ctx context.Context,
	client kclient.Client,
	namespace string,
	ref v1.SecretKeySelector,
) (string, error) {
	var secret corev1.Secret
	if err := client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      ref.Name,
	}, &secret); err != nil {
		return "", err
	}

	b, ok := secret.Data[ref.Key]
	if !ok {
		return "", fmt.Errorf("secret %s/%s missing key %q", namespace, ref.Name, ref.Key)
	}

	return string(b), nil
}

// privxAuth creates authentication from information in the Store specification.
func privxAuth(
	ctx context.Context,
	kube kclient.Client,
	namespace string,
	privxSpec *esv1.PrivxProvider,
) (privxapi.Authorizer, error) {
	if privxSpec.Auth != nil && privxSpec.Auth.OAuth != nil {
		return privxAuthOAuth(ctx, kube, namespace, privxSpec)
	}
	return privxAuthJWT(ctx, kube, namespace, privxSpec)
}

// privxAuthOAuth authenticates using OAuth2 API client credentials.
func privxAuthOAuth(
	ctx context.Context,
	kube kclient.Client,
	namespace string,
	privxSpec *esv1.PrivxProvider,
) (privxapi.Authorizer, error) {
	auth := privxapi.New(
		privxapi.BaseURL(privxSpec.Host),
	)

	clientID, err := readSecretValue(ctx, kube, namespace, privxSpec.Auth.OAuth.ApiClientIDRef)
	if err != nil {
		return nil, err
	}

	clientSecret, err := readSecretValue(ctx, kube, namespace, privxSpec.Auth.OAuth.ApiClientSecretRef)
	if err != nil {
		return nil, err
	}

	oAuthAccess, err := readSecretValue(ctx, kube, namespace, privxSpec.Auth.OAuth.ClientIDRef)
	if err != nil {
		return nil, err
	}

	oAuthSecret, err := readSecretValue(ctx, kube, namespace, privxSpec.Auth.OAuth.ClientSecretRef)
	if err != nil {
		return nil, err
	}

	return oauth.With(
		auth,
		oauth.Access(clientID),
		oauth.Secret(clientSecret),
		oauth.Digest(oAuthAccess, oAuthSecret),
	), nil
}

// privxAuthJWT authenticates using a signed JWT token exchanged for a PrivX access token.
func privxAuthJWT(
	ctx context.Context,
	kube kclient.Client,
	namespace string,
	privxSpec *esv1.PrivxProvider,
) (privxapi.Authorizer, error) {
	token, err := obtainJWTToken(ctx, kube, namespace, privxSpec)
	if err != nil {
		return nil, err
	}

	// Logging
	decoded, err := decodeJWT(token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT: %w", err)
	}
	logger := log.FromContext(ctx)
	logger.V(1).Info("JWT token", "claims", decoded)

	// Then exchange the token for a PrivX token
	req := ExchangeTokenRequest{Token: token}
	tokenResponse, err := ExchangeToken(ctx, nil, privxSpec.Host, req)
	if err != nil {
		return nil, err
	}

	logTokenResponse(logger, tokenResponse)
	return oauth.WithToken("Bearer " + tokenResponse.AccessToken), nil
}

// obtainJWTToken creates a signed JWT using either an explicit key or a Kubernetes service account token.
func obtainJWTToken(
	ctx context.Context,
	kube kclient.Client,
	namespace string,
	privxSpec *esv1.PrivxProvider,
) (string, error) {
	if privxSpec.Auth != nil && privxSpec.Auth.JWTAuth != nil {
		// JWT private key given, use it to sign a JWT (PrivX 43 and earlier)
		return createSignedJWT(
			ctx,
			kube,
			namespace,
			privxSpec.Auth.JWTAuth.PublicKeyRef,
			privxSpec.Auth.JWTAuth.Iss,
			privxSpec.Auth.JWTAuth.Sub,
			privxSpec.Host,
			15*time.Minute,
			map[string]any{},
		)
	}
	// No OAuth tokens, no explicit key — use Kubernetes service account JWT (PrivX 44+)
	return getAudienceJWTFromPod(ctx, privxSpec.Host, 15*time.Minute)
}

// privxAPI creates a working PrivX API connection from information in the Store specification.
func privxAPI(
	ctx context.Context,
	kube kclient.Client,
	namespace string,
	privxSpec *esv1.PrivxProvider,
) (privxapi.Connector, error) {
	auth, err := privxAuth(ctx, kube, namespace, privxSpec)
	if err != nil {
		return nil, err
	}

	return privxapi.New(
		privxapi.BaseURL(privxSpec.Host),
		privxapi.Auth(auth),
	), nil
}

// NewClient returns a new PrivX Client.
func (p *Provider) NewClient(
	ctx context.Context,
	store esv1.GenericStore,
	kube kclient.Client,
	namespace string,
) (esv1.SecretsClient, error) {
	if p.newClient == nil {
		return newRealClient(ctx, store, kube, namespace)
	}
	return p.newClient(ctx, store, kube, namespace)
}

// newRealClient returns a new PrivX Client.
func newRealClient(
	ctx context.Context,
	store esv1.GenericStore,
	kube kclient.Client,
	namespace string,
) (esv1.SecretsClient, error) {
	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.PrivX == nil {
		return nil, ErrNoStoreAuth{Field: "spec.provider.privx"}
	}

	config := spec.Provider.PrivX
	conn, err := privxAPI(ctx, kube, namespace, config)
	if err != nil {
		return nil, err
	}

	client := SecretsClient{
		conn: conn,
		vault: &sdkVaultClient{
			v: vault.New(conn),
		},
		store:             store,
		kube:              kube,
		namespace:         namespace,
		defaultReadRoles:  config.DefaultReadRoles,
		defaultWriteRoles: config.DefaultWriteRoles,
	}
	return &client, nil
}

// ValidateStore checks the configuration.
func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	if store.GetSpec().Provider == nil {
		return nil, ErrNoStoreAuth{Field: "spec.provider"}
	}
	provider := store.GetSpec().Provider
	if provider.PrivX == nil {
		return nil, ErrNoStoreAuth{Field: "spec.provider.privx"}
	}

	privx := provider.PrivX

	// with JWT, no auth fields necessary
	// if privx.Auth == nil {
	// 	return nil, ErrNoStoreAuth{Field: "spec.provider.privx.auth"}
	// }

	if privx.Host == "" {
		return nil, ErrNoStoreAuth{Field: "spec.provider.privx.host"}
	}

	return nil, nil
}

// Capabilities indicates, if store is read-only or read-write.
func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return &Provider{
		newClient: newRealClient,
	}
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		PrivX: &esv1.PrivxProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
