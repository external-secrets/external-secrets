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
package workloadidentity

import (
	"context"
	"fmt"
	"time"

	iam "cloud.google.com/go/iam/credentials/apiv1"
	"cloud.google.com/go/iam/credentials/apiv1/credentialspb"
	"github.com/googleapis/gax-go/v2"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	stsv1 "google.golang.org/api/sts/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"grpc.go4.org/credentials/oauth"
	authenticationv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	clientcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	ServiceAccountAnnotation = "iam.gke.io/gcp-service-account"

	errFetchKSAToken       = "failed to fetch kubernetes service account token: %w"
	errFetchGCPAccessToken = "failed to fetch GCP access token: %w"
	errImpersonateGSA      = "failed to impersonate GCP service account %s: %w"
)

// Provider provides oauth2.TokenSource(s) using Workload Identity. It exchanges
// Kubernetes ServiceAccount tokens for GCP access tokens, optionally
// impersonating a GCP service account.
type Provider struct {
	gcpTokenGenerator    GoogleTokenGenerator
	iam                  IamClient
	identityProvider     string
	ksaTokenGenerator    KSATokenGenerator
	workloadIdentityPool string
}

// IamClient is an interface for the Google IAM client.
type IamClient interface {
	GenerateAccessToken(ctx context.Context, req *credentialspb.GenerateAccessTokenRequest, opts ...gax.CallOption) (*credentialspb.GenerateAccessTokenResponse, error)
	Close() error
}

// GoogleTokenGenerator generates a GCP access token using a Kubernetes ServiceAccount token.
type GoogleTokenGenerator interface {
	Generate(ctx context.Context, ksaToken string, idPool string, idProvider string) (*oauth2.Token, error)
}

// KSATokenGenerator generates a TokenRequest for a Kubernetes ServiceAccount.
type KSATokenGenerator interface {
	Generate(ctx context.Context, name, namespace string, aud ...string) (*authenticationv1.TokenRequest, error)
}

// NewProvider creates a new Provider.
func NewProvider(ctx context.Context, projectID string, idp IdentityProvider, opts ...Option) (*Provider, error) {
	f := &Provider{
		identityProvider:     idp(projectID),
		workloadIdentityPool: fmt.Sprintf("%s.svc.id.goog", projectID),
	}

	for _, opt := range opts {
		opt(f)
	}

	var err error
	if f.ksaTokenGenerator == nil {
		f.ksaTokenGenerator, err = newKSATokenGenerator()
		if err != nil {
			return nil, err
		}
	}

	if f.gcpTokenGenerator == nil {
		f.gcpTokenGenerator, err = newGCPAccessTokenGenerator(ctx)
		if err != nil {
			return nil, err
		}
	}

	if f.iam == nil {
		f.iam, err = newIAMClient(ctx)
		if err != nil {
			return nil, err
		}
	}

	return f, nil
}

// Option is a functional option for Provider.
type Option func(*Provider)

// WithGCPTokenGenerator allows the caller to provide a custom GoogleTokenGenerator.
func WithGCPTokenGenerator(g GoogleTokenGenerator) Option {
	return func(p *Provider) {
		p.gcpTokenGenerator = g
	}
}

// WithIAMClient allows the caller to provide a custom IamClient.
func WithIAMClient(c IamClient) Option {
	return func(p *Provider) {
		p.iam = c
	}
}

// WithKSATokenGenerator allows the caller to provide a custom KSATokenGenerator.
func WithKSATokenGenerator(k KSATokenGenerator) Option {
	return func(p *Provider) {
		p.ksaTokenGenerator = k
	}
}

// IdentityProvider returns a fully qualified identity provider URI for a
// project.  See FleetIdentityProvider and ClusterIdentityProvider.
type IdentityProvider func(projectID string) string

// FleetIdentityProvider returns a workload identity provider URI for a fleet.
func FleetIdentityProvider(membership string) func(string) string {
	return func(projectID string) string {
		return fmt.Sprintf("https://gkehub.googleapis.com/projects/%s/locations/global/memberships/%s",
			projectID,
			membership)
	}
}

// ClusterIdentityProvider returns a workload identity provider URI for a single
// cluster.
func ClusterIdentityProvider(clusterName, clusterLocation string) func(string) string {
	return func(projectID string) string {
		return fmt.Sprintf("https://container.googleapis.com/v1/projects/%s/locations/%s/clusters/%s",
			projectID,
			clusterLocation,
			clusterName)
	}
}

// TokenSource returns a new oauth2.TokenSource using Workload Identity. If the
// ServiceAccount referenced in saKey is annotated with
// iam.gke.io/gcp-service-account, the returned TokenSource will impersonate
// that GCP service account. aud is an optional list of audiences to include in
// the request to create a service account token in the TokenRequest API.
func (p *Provider) TokenSource(ctx context.Context, kube kclient.Client, saKey types.NamespacedName, aud ...string) (oauth2.TokenSource, error) {
	sa := &v1.ServiceAccount{}
	err := kube.Get(ctx, saKey, sa)
	if err != nil {
		return nil, err
	}

	audiences := []string{p.workloadIdentityPool}
	if len(aud) > 0 {
		audiences = append(audiences, aud...)
	}

	resp, err := p.ksaTokenGenerator.Generate(ctx, saKey.Name, saKey.Namespace, audiences...)
	if err != nil {
		return nil, fmt.Errorf(errFetchKSAToken, err)
	}

	gcpAccessToken, err := p.gcpTokenGenerator.Generate(ctx, resp.Status.Token, p.workloadIdentityPool, p.identityProvider)
	if err != nil {
		return nil, fmt.Errorf(errFetchGCPAccessToken, err)
	}

	gcpSA := sa.Annotations[ServiceAccountAnnotation]
	// If no `iam.gke.io/gcp-service-account` annotation is present, no service
	// account impersonation will be performed.  gcpAccessToken will be used
	// directly, allowing IAM bindings of the form
	// "serviceAccount:<pool_id>.svc.id.goog[<namespace>/<ksa>]".
	if gcpSA == "" {
		return oauth2.StaticTokenSource(gcpAccessToken), nil
	}

	gcpSAResp, err := p.iam.GenerateAccessToken(ctx, &credentialspb.GenerateAccessTokenRequest{
		Name:  fmt.Sprintf("projects/-/serviceAccounts/%s", gcpSA),
		Scope: []string{"https://www.googleapis.com/auth/cloud-platform"},
	}, gax.WithGRPCOptions(grpc.PerRPCCredentials(oauth.TokenSource{TokenSource: oauth2.StaticTokenSource(gcpAccessToken)})))
	if err != nil {
		return nil, fmt.Errorf(errImpersonateGSA, gcpSA, err)
	}

	return oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: gcpSAResp.GetAccessToken(),
	}), nil
}

func (p *Provider) Close() error {
	if p.iam != nil {
		return p.iam.Close()
	}
	return nil
}

func newIAMClient(ctx context.Context) (IamClient, error) {
	iamOpts := []option.ClientOption{
		option.WithUserAgent("external-secrets-operator"),
		// tell the secretmanager library to not add transport-level ADC since
		// we need to override on a per call basis
		option.WithoutAuthentication(),
		// grpc oauth TokenSource credentials require transport security, so
		// this must be set explicitly even though TLS is used
		option.WithGRPCDialOption(grpc.WithTransportCredentials(credentials.NewTLS(nil))),
		option.WithGRPCConnectionPool(5),
	}
	return iam.NewIamCredentialsClient(ctx, iamOpts...)
}

type K8sSATokenGenerator struct {
	Corev1 clientcorev1.CoreV1Interface
}

func (g *K8sSATokenGenerator) Generate(ctx context.Context, name, namespace string, audiences ...string) (*authenticationv1.TokenRequest, error) {
	// Create a serviceaccount token through the TokenRequest API.
	ttl := int64((15 * time.Minute).Seconds())
	return g.Corev1.
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

func newKSATokenGenerator() (KSATokenGenerator, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &K8sSATokenGenerator{
		Corev1: clientset.CoreV1(),
	}, nil
}

// Exchanges a Kubernetes service account token for a GCP access token.
type gcpAccessTokenGenerator struct {
	sts *stsv1.Service
}

func newGCPAccessTokenGenerator(ctx context.Context) (GoogleTokenGenerator, error) {
	sts, err := stsv1.NewService(ctx,
		option.WithUserAgent("external-secrets-operator"),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(credentials.NewTLS(nil))),
		option.WithGRPCConnectionPool(5),
	)
	if err != nil {
		return nil, err
	}
	return &gcpAccessTokenGenerator{
		sts: sts,
	}, nil
}

func (g *gcpAccessTokenGenerator) Generate(ctx context.Context, ksaToken, idPool, idProvider string) (*oauth2.Token, error) {
	res, err := g.sts.V1.Token(&stsv1.GoogleIdentityStsV1ExchangeTokenRequest{
		GrantType:          "urn:ietf:params:oauth:grant-type:token-exchange",
		Audience:           fmt.Sprintf("identitynamespace:%s:%s", idPool, idProvider),
		RequestedTokenType: "urn:ietf:params:oauth:token-type:access_token",
		SubjectToken:       ksaToken,
		SubjectTokenType:   "urn:ietf:params:oauth:token-type:jwt",
		Scope:              "https://www.googleapis.com/auth/cloud-platform",
	}).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("exchange kubernetes service account token for gcp access token: %w", err)
	}

	return &oauth2.Token{
		AccessToken: res.AccessToken,
		TokenType:   res.TokenType,
	}, nil
}
