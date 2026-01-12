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

package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	iam "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	gsmapiv1 "cloud.google.com/go/secretmanager/apiv1"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	"github.com/external-secrets/external-secrets/runtime/constants"
	"github.com/external-secrets/external-secrets/runtime/metrics"
)

const (
	gcpSAAnnotation = "iam.gke.io/gcp-service-account"

	errFetchPodToken  = "unable to fetch pod token: %w"
	errFetchIBToken   = "unable to fetch identitybindingtoken: %w"
	errGenAccessToken = "unable to generate gcp access token: %w"
	errLookupIdentity = "unable to lookup workload identity: %w"
	errNoProjectID    = "unable to find ProjectID in storeSpec"
)

var (
	// defaultUniverseDomain is the domain which will be used in the STS token URL.
	defaultUniverseDomain = "googleapis.com"

	// workloadIdentitySubjectTokenType is the STS token type used in Oauth2.0 token exchange.
	workloadIdentitySubjectTokenType = "urn:ietf:params:oauth:token-type:jwt"

	// workloadIdentitySubjectTokenType is the STS token type used in Oauth2.0 token exchange with AWS.
	workloadIdentitySubjectTokenTypeAWS = "urn:ietf:params:aws:token-type:aws4_request"

	// workloadIdentityTokenGrantType is the grant type for OAuth 2.0 token exchange .
	workloadIdentityTokenGrantType = "urn:ietf:params:oauth:grant-type:token-exchange"

	// workloadIdentityRequestedTokenType is the requested type for OAuth 2.0 access token.
	workloadIdentityRequestedTokenType = "urn:ietf:params:oauth:token-type:access_token"

	// workloadIdentityTokenURL is the token service endpoint.
	workloadIdentityTokenURL = "https://sts.googleapis.com/v1/token"

	// workloadIdentityTokenInfoURL is the STS introspection service endpoint.
	workloadIdentityTokenInfoURL = "https://sts.googleapis.com/v1/introspect"

	// tokenRefreshBuffer is the time before token expiry when we should refresh.
	// Tokens are refreshed 1 minute before they expire to avoid race conditions.
	tokenRefreshBuffer = 1 * time.Minute
)

// workloadIdentity holds all clients and generators needed
// to create a gcp oauth token.
type workloadIdentity struct {
	metadataClient       MetadataClient
	idBindTokenGenerator idBindTokenGenerator
	saTokenGenerator     saTokenGenerator
	// iamClientCreator allows injection of a custom IAM client creator for testing.
	// If nil, the default newIAMClient function is used.
	iamClientCreator func(ctx context.Context, tokenSource oauth2.TokenSource) (IamClient, error)
}

// IamClient provides an interface to the GCP IAM API.
type IamClient interface {
	GenerateAccessToken(ctx context.Context, req *credentialspb.GenerateAccessTokenRequest, opts ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error)
	SignJwt(ctx context.Context, req *credentialspb.SignJwtRequest, opts ...gax.CallOption) (*credentialspb.SignJwtResponse, error)
	Close() error
}

// MetadataClient defines the interface for interacting with GCP Metadata service.
// It provides access to instance metadata and project information.
type MetadataClient interface {
	InstanceAttributeValueWithContext(ctx context.Context, attr string) (string, error)
	ProjectIDWithContext(ctx context.Context) (string, error)
}

// interface to securetoken/identitybindingtoken API.
type idBindTokenGenerator interface {
	Generate(context.Context, *http.Client, string, string, string) (*oauth2.Token, error)
}

// interface to kubernetes serviceaccount token request API.
type saTokenGenerator interface {
	Generate(context.Context, []string, string, string) (*authenticationv1.TokenRequest, error)
}

// workloadIdentityOption is a functional option for configuring workloadIdentity.
type workloadIdentityOption func(*workloadIdentity)

// withSATokenGenerator sets a custom saTokenGenerator (used for testing).
func withSATokenGenerator(satg saTokenGenerator) workloadIdentityOption {
	return func(w *workloadIdentity) {
		w.saTokenGenerator = satg
	}
}

// refreshableTokenSource is an oauth2.TokenSource that automatically refreshes
// the token when it's about to expire. This ensures long-running operations
// don't fail due to token expiration.
type refreshableTokenSource struct {
	mu sync.Mutex
	// refreshFunc is called to generate a new token when needed.
	refreshFunc func(ctx context.Context) (*oauth2.Token, error)
	// currentToken holds the current valid token.
	currentToken *oauth2.Token
	// ctx is used for token refresh operations.
	ctx context.Context
}

// Token returns a valid token, refreshing it if necessary.
// It implements the oauth2.TokenSource interface.
// This method is thread-safe.
func (r *refreshableTokenSource) Token() (*oauth2.Token, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we need to refresh the token
	if r.currentToken == nil || r.shouldRefresh() {
		newToken, err := r.refreshFunc(r.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}
		r.currentToken = newToken
	}
	return r.currentToken, nil
}

// shouldRefresh returns true if the token should be refreshed.
// A token should be refreshed if it will expire within tokenRefreshBuffer.
func (r *refreshableTokenSource) shouldRefresh() bool {
	if r.currentToken == nil {
		return true
	}
	// If token has no expiry, assume it doesn't need refresh
	if r.currentToken.Expiry.IsZero() {
		return false
	}
	// Refresh if token expires within the buffer time
	return time.Until(r.currentToken.Expiry) < tokenRefreshBuffer
}

func newWorkloadIdentity(opts ...workloadIdentityOption) (*workloadIdentity, error) {
	wi := &workloadIdentity{
		metadataClient:       newMetadataClient(),
		idBindTokenGenerator: newIDBindTokenGenerator(),
	}

	// Apply options first (allows tests to inject mocks)
	for _, opt := range opts {
		opt(wi)
	}

	// Only create real SA token generator if not injected
	if wi.saTokenGenerator == nil {
		satg, err := newSATokenGenerator()
		if err != nil {
			return nil, err
		}
		wi.saTokenGenerator = satg
	}

	return wi, nil
}

func (w *workloadIdentity) gcpWorkloadIdentity(ctx context.Context, id *esv1.GCPWorkloadIdentity) (string, string, error) {
	var err error

	projectID := id.ClusterProjectID
	if projectID == "" {
		if projectID, err = w.metadataClient.ProjectIDWithContext(ctx); err != nil {
			return "", "", fmt.Errorf("unable to get project id: %w", err)
		}
	}

	clusterLocation := id.ClusterLocation
	if clusterLocation == "" {
		if clusterLocation, err = w.metadataClient.InstanceAttributeValueWithContext(ctx, "cluster-location"); err != nil {
			return "", "", fmt.Errorf("unable to determine cluster location: %w", err)
		}
	}

	clusterName := id.ClusterName
	if clusterName == "" {
		if clusterName, err = w.metadataClient.InstanceAttributeValueWithContext(ctx, "cluster-name"); err != nil {
			return "", "", fmt.Errorf("unable to determine cluster name: %w", err)
		}
	}

	idPool := fmt.Sprintf("%s.svc.id.goog", projectID)
	idProvider := fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
		projectID,
		clusterLocation,
		clusterName,
	)
	return idPool, idProvider, nil
}

func (w *workloadIdentity) TokenSource(ctx context.Context, auth esv1.GCPSMAuth, isClusterKind bool, kube kclient.Client, namespace string) (oauth2.TokenSource, error) {
	wi := auth.WorkloadIdentity
	if wi == nil {
		return nil, nil
	}
	saKey := types.NamespacedName{
		Name:      wi.ServiceAccountRef.Name,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind && wi.ServiceAccountRef.Namespace != nil {
		saKey.Namespace = *wi.ServiceAccountRef.Namespace
	}

	sa := &v1.ServiceAccount{}
	err := kube.Get(ctx, saKey, sa)
	if err != nil {
		return nil, err
	}

	gcpSA := sa.Annotations[gcpSAAnnotation]

	// Create a refreshable token source that can regenerate tokens when they expire.
	// This is important for long-running operations that may outlive the token TTL.
	refreshFunc := func(refreshCtx context.Context) (*oauth2.Token, error) {
		return w.generateToken(refreshCtx, wi, saKey, gcpSA)
	}

	// Generate initial token to validate configuration
	initialToken, err := refreshFunc(ctx)
	if err != nil {
		return nil, err
	}

	return &refreshableTokenSource{
		refreshFunc:  refreshFunc,
		currentToken: initialToken,
		ctx:          ctx,
	}, nil
}

// getIdentityBindingToken obtains an identity binding token from GKE Workload Identity.
// This is the foundation for both TokenSource (which uses GenerateAccessToken) and
// SignedJWTForVault (which uses SignJwt).
func (w *workloadIdentity) getIdentityBindingToken(ctx context.Context, wi *esv1.GCPWorkloadIdentity, saKey types.NamespacedName) (*oauth2.Token, error) {
	idPool, idProvider, err := w.gcpWorkloadIdentity(ctx, wi)
	if err != nil {
		return nil, fmt.Errorf(errLookupIdentity, err)
	}

	audiences := []string{idPool}
	if len(wi.ServiceAccountRef.Audiences) > 0 {
		audiences = append(audiences, wi.ServiceAccountRef.Audiences...)
	}

	resp, err := w.saTokenGenerator.Generate(ctx, audiences, saKey.Name, saKey.Namespace)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGenerateSAToken, err)
	if err != nil {
		return nil, fmt.Errorf(errFetchPodToken, err)
	}

	idBindToken, err := w.idBindTokenGenerator.Generate(ctx, http.DefaultClient, resp.Status.Token, idPool, idProvider)
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGenerateIDBindToken, err)
	if err != nil {
		return nil, fmt.Errorf(errFetchIBToken, err)
	}

	return idBindToken, nil
}

// generateToken creates a new OAuth2 token using the workload identity flow.
// This is called both for initial token generation and for token refresh.
func (w *workloadIdentity) generateToken(ctx context.Context, wi *esv1.GCPWorkloadIdentity, saKey types.NamespacedName, gcpSA string) (*oauth2.Token, error) {
	idBindToken, err := w.getIdentityBindingToken(ctx, wi, saKey)
	if err != nil {
		return nil, err
	}

	// If no `iam.gke.io/gcp-service-account` annotation is present the
	// identitybindingtoken will be used directly, allowing bindings on secrets
	// of the form "serviceAccount:<project>.svc.id.goog[<namespace>/<sa>]".
	if gcpSA == "" {
		return idBindToken, nil
	}

	// Create IAM client with the token source from Workload Identity
	tokenSource := oauth2.StaticTokenSource(idBindToken)
	iamClientCreator := w.iamClientCreator
	if iamClientCreator == nil {
		iamClientCreator = newIAMClient
	}
	iamClient, err := iamClientCreator(ctx, tokenSource)
	if err != nil {
		return nil, fmt.Errorf("failed to create IAM client: %w", err)
	}
	defer func() {
		_ = iamClient.Close()
	}()

	gcpSAResp, err := iamClient.GenerateAccessToken(ctx, &credentialspb.GenerateAccessTokenRequest{
		Name:  fmt.Sprintf("projects/-/serviceAccounts/%s", gcpSA),
		Scope: gsmapiv1.DefaultAuthScopes(),
	})
	metrics.ObserveAPICall(constants.ProviderGCPSM, constants.CallGCPSMGenerateAccessToken, err)
	if err != nil {
		return nil, fmt.Errorf(errGenAccessToken, err)
	}

	return &oauth2.Token{
		AccessToken: gcpSAResp.GetAccessToken(),
		Expiry:      gcpSAResp.GetExpireTime().AsTime(),
	}, nil
}

// SignedJWTForVault generates a signed JWT for Vault GCP IAM authentication.
// This JWT contains the required claims (sub, aud, exp) and is signed using the
// GCP service account's private key via the IAM SignJwt API.
func (w *workloadIdentity) SignedJWTForVault(ctx context.Context, wi *esv1.GCPWorkloadIdentity, role string, isClusterKind bool, kube kclient.Client, namespace string) (string, error) {
	saKey := types.NamespacedName{
		Name:      wi.ServiceAccountRef.Name,
		Namespace: namespace,
	}

	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind && wi.ServiceAccountRef.Namespace != nil {
		saKey.Namespace = *wi.ServiceAccountRef.Namespace
	}

	sa := &v1.ServiceAccount{}
	err := kube.Get(ctx, saKey, sa)
	if err != nil {
		return "", err
	}

	gcpSA := sa.Annotations[gcpSAAnnotation]
	if gcpSA == "" {
		return "", fmt.Errorf("service account %s/%s is missing required annotation %s", saKey.Namespace, saKey.Name, gcpSAAnnotation)
	}

	// Get the identity binding token directly (single IAM client creation)
	idBindToken, err := w.getIdentityBindingToken(ctx, wi, saKey)
	if err != nil {
		return "", fmt.Errorf("failed to get identity binding token: %w", err)
	}

	// Create IAM client with the identity binding token
	tokenSource := oauth2.StaticTokenSource(idBindToken)
	iamClientCreator := w.iamClientCreator
	if iamClientCreator == nil {
		iamClientCreator = newIAMClient
	}
	iamClient, err := iamClientCreator(ctx, tokenSource)
	if err != nil {
		return "", fmt.Errorf("failed to create IAM client: %w", err)
	}
	defer func() {
		_ = iamClient.Close()
	}()

	// Create JWT payload for Vault IAM auth
	// The audience must be in the format "vault/{role}"
	// Reference: https://support.hashicorp.com/hc/en-us/articles/37175601988499
	// API Docs: https://developer.hashicorp.com/vault/api-docs/auth/gcp
	exp := time.Now().Add(15 * time.Minute).Unix()
	payload := map[string]interface{}{
		"sub": gcpSA,
		"aud": fmt.Sprintf("vault/%s", role),
		"exp": exp,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JWT payload: %w", err)
	}

	// Sign the JWT using the GCP service account
	signResp, err := iamClient.SignJwt(ctx, &credentialspb.SignJwtRequest{
		Name:    fmt.Sprintf("projects/-/serviceAccounts/%s", gcpSA),
		Payload: string(payloadBytes),
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT for Vault GCP IAM auth: role=%q, gcpServiceAccount=%q: %w", role, gcpSA, err)
	}

	return signResp.GetSignedJwt(), nil
}

func (w *workloadIdentity) Close() error {
	// IAM clients are created on-demand and closed immediately after use,
	// so there's nothing to close here. This method exists for interface compatibility.
	return nil
}

func newIAMClient(ctx context.Context, tokenSource oauth2.TokenSource) (IamClient, error) {
	iamOpts := []option.ClientOption{
		option.WithUserAgent("external-secrets-operator"),
		option.WithTokenSource(tokenSource),
		option.WithGRPCConnectionPool(5),
	}
	return iam.NewIamCredentialsClient(ctx, iamOpts...)
}

func newMetadataClient() MetadataClient {
	return metadata.NewClient(&http.Client{
		Timeout: 5 * time.Second,
	})
}

type k8sSATokenGenerator struct {
	corev1 clientcorev1.CoreV1Interface
}

func (g *k8sSATokenGenerator) Generate(ctx context.Context, audiences []string, name, namespace string) (*authenticationv1.TokenRequest, error) {
	// Request a serviceaccount token for the pod
	ttl := int64((15 * time.Minute).Seconds())
	return g.corev1.
		ServiceAccounts(namespace).
		CreateToken(ctx, name,
			&authenticationv1.TokenRequest{
				Spec: authenticationv1.TokenRequestSpec{
					ExpirationSeconds: &ttl,
					Audiences:         audiences,
				},
			},
			metav1.CreateOptions{},
		)
}

// newSATokenGeneratorFunc is a factory function for creating saTokenGenerator.
// It can be overridden in tests to provide a mock implementation.
var newSATokenGeneratorFunc = defaultNewSATokenGenerator

func newSATokenGenerator() (saTokenGenerator, error) {
	return newSATokenGeneratorFunc()
}

func defaultNewSATokenGenerator() (saTokenGenerator, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &k8sSATokenGenerator{
		corev1: clientset.CoreV1(),
	}, nil
}

// Trades the kubernetes token for an identitybindingtoken token.
type gcpIDBindTokenGenerator struct {
	targetURL string
}

func newIDBindTokenGenerator() idBindTokenGenerator {
	return &gcpIDBindTokenGenerator{
		targetURL: workloadIdentityTokenURL,
	}
}

func (g *gcpIDBindTokenGenerator) Generate(ctx context.Context, client *http.Client, k8sToken, idPool, idProvider string) (*oauth2.Token, error) {
	body, err := json.Marshal(map[string]string{
		"grant_type":           workloadIdentityTokenGrantType,
		"subject_token_type":   workloadIdentitySubjectTokenType,
		"requested_token_type": workloadIdentityRequestedTokenType,
		"subject_token":        k8sToken,
		"audience":             fmt.Sprintf("identitynamespace:%s:%s", idPool, idProvider),
		"scope":                CloudPlatformRole,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", g.targetURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("could not get idbindtoken token, status: %v", resp.StatusCode)
	}

	defer func() {
		_ = resp.Body.Close()
	}()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	idBindToken := &oauth2.Token{}
	if err := json.Unmarshal(respBody, idBindToken); err != nil {
		return nil, err
	}
	return idBindToken, nil
}
