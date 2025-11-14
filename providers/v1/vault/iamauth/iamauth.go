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

// Package iamauth provides utilities for AWS IAM authentication using Kubernetes Service Accounts.
// Mostly sourced from ~/external-secrets/pkg/provider/aws/auth
package iamauth

import (
	"context"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	smithy "github.com/aws/smithy-go/endpoints"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsutil "github.com/external-secrets/external-secrets/providers/v1/aws/util"
	vaultutil "github.com/external-secrets/external-secrets/providers/v1/vault/util"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

var (
	logger = ctrl.Log.WithName("provider").WithName("vault")
)

const (
	roleARNAnnotation    = "eks.amazonaws.com/role-arn"
	audienceAnnotation   = "eks.amazonaws.com/audience"
	defaultTokenAudience = "sts.amazonaws.com"

	// STSEndpointEnv is the environment variable that can be used to override the default STS endpoint.
	STSEndpointEnv = "AWS_STS_ENDPOINT"
	// AWSWebIdentityTokenFileEnvVar is the environment variable that points to the service account token file.
	AWSWebIdentityTokenFileEnvVar = "AWS_WEB_IDENTITY_TOKEN_FILE"
	// AWSContainerCredentialsFullURIEnvVar is the environment variable that points to the full credentials URI for ECS tasks.
	AWSContainerCredentialsFullURIEnvVar = "AWS_CONTAINER_CREDENTIALS_FULL_URI"
)

// DefaultJWTProvider returns a credentials.Provider that calls the AssumeRoleWithWebidentity
// controller-runtime/client does not support TokenRequest or other subresource APIs
// so we need to construct our own client and use it to fetch tokens.
func DefaultJWTProvider(ctx context.Context, name, namespace, roleArn string, aud []string, region string) (aws.CredentialsProvider, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	var loadCfgOpts []func(*config.LoadOptions) error
	loadCfgOpts = append(loadCfgOpts,
		config.WithAppID("external-secrets"),
		config.WithSharedConfigFiles([]string{}),
		config.WithSharedCredentialsFiles([]string{}),
	)
	if region != "" {
		loadCfgOpts = append(loadCfgOpts, config.WithRegion(region))
	}

	awscfg, err := config.LoadDefaultConfig(context.TODO(), loadCfgOpts...)
	if err != nil {
		return nil, awsutil.SanitizeErr(err)
	}

	tokenFetcher := &authTokenFetcher{
		Namespace:      namespace,
		Audiences:      aud,
		ServiceAccount: name,
		Context:        ctx,
		k8sClient:      clientset.CoreV1(),
	}

	stsClient := sts.NewFromConfig(awscfg, func(o *sts.Options) {
		o.EndpointResolverV2 = customEndpointResolver{}
	})

	return stscreds.NewWebIdentityRoleProvider(
		stsClient, roleArn, tokenFetcher, func(opts *stscreds.WebIdentityRoleOptions) {
			opts.RoleSessionName = "external-secrets-provider-vault"
		}), nil
}

// customEndpointResolver implements sts.EndpointResolverV2 for custom STS endpoint.
type customEndpointResolver struct{}

// ResolveEndpoint resolves the STS endpoint using custom configuration if available.
func (r customEndpointResolver) ResolveEndpoint(ctx context.Context, params sts.EndpointParameters) (smithy.Endpoint, error) {
	if v := os.Getenv(STSEndpointEnv); v != "" {
		uri, err := url.Parse(v)
		if err != nil {
			return smithy.Endpoint{}, err
		}
		return smithy.Endpoint{
			URI: *uri,
		}, nil
	}
	// Fall back to default resolver
	return sts.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, params)
}

// mostly taken from:
// https://github.com/aws/secrets-store-csi-driver-provider-aws/blob/main/auth/auth.go#L140-L145

type authTokenFetcher struct {
	Context   context.Context
	Namespace string
	// Audience is the token aud claim
	// which is verified by the aws oidc provider
	// see: https://github.com/external-secrets/external-secrets/issues/1251#issuecomment-1161745849
	Audiences      []string
	ServiceAccount string
	k8sClient      k8scorev1.CoreV1Interface
}

// GetIdentityToken satisfies the stscreds.IdentityTokenRetriever interface
// it is used to generate service account tokens which are consumed by the aws sdk.
func (p *authTokenFetcher) GetIdentityToken() ([]byte, error) {
	logger.V(1).Info("fetching token", "ns", p.Namespace, "sa", p.ServiceAccount)
	tokRsp, err := p.k8sClient.ServiceAccounts(p.Namespace).CreateToken(p.Context, p.ServiceAccount, &authv1.TokenRequest{
		Spec: authv1.TokenRequestSpec{
			Audiences: p.Audiences,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("error creating service account token: %w", err)
	}
	return []byte(tokRsp.Status.Token), nil
}

// CredsFromServiceAccount uses a Kubernetes Service Account to acquire temporary
// credentials using aws.AssumeRoleWithWebIdentity. It will assume the role defined
// in the ServiceAccount annotation.
// If the ClusterSecretStore does not define a namespace it will use the namespace from the ExternalSecret (referentAuth).
// If the ClusterSecretStore defines the namespace it will take precedence.
func CredsFromServiceAccount(ctx context.Context, auth esv1.VaultIamAuth, region string, isClusterKind bool, kube kclient.Client, namespace string, jwtProvider vaultutil.JwtProviderFactory) (aws.CredentialsProvider, error) {
	name := auth.JWTAuth.ServiceAccountRef.Name
	if isClusterKind && auth.JWTAuth.ServiceAccountRef.Namespace != nil {
		namespace = *auth.JWTAuth.ServiceAccountRef.Namespace
	}
	sa := v1.ServiceAccount{}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, &sa)
	if err != nil {
		return nil, err
	}
	// the service account is expected to have a well-known annotation
	// this is used as input to assumeRoleWithWebIdentity
	roleArn := sa.Annotations[roleARNAnnotation]
	if roleArn == "" {
		return nil, fmt.Errorf("an IAM role must be associated with service account %s (namespace: %s)", name, namespace)
	}

	tokenAud := sa.Annotations[audienceAnnotation]
	if tokenAud == "" {
		tokenAud = defaultTokenAudience
	}
	audiences := []string{tokenAud}
	if len(auth.JWTAuth.ServiceAccountRef.Audiences) > 0 {
		audiences = append(audiences, auth.JWTAuth.ServiceAccountRef.Audiences...)
	}

	jwtProv, err := jwtProvider(ctx, name, namespace, roleArn, audiences, region)
	if err != nil {
		return nil, err
	}

	logger.V(1).Info("using credentials via service account", "role", roleArn, "region", region)
	return jwtProv, nil
}

// CredsFromControllerServiceAccount uses a Kubernetes Service Account to acquire temporary
// credentials using aws.AssumeRoleWithWebIdentity. It will assume the role defined
// in the ServiceAccount annotation.
// The namespace of the controller service account is used.
func CredsFromControllerServiceAccount(ctx context.Context, saName, ns, region string, kube kclient.Client, jwtProvider vaultutil.JwtProviderFactory) (aws.CredentialsProvider, error) {
	sa := v1.ServiceAccount{}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      saName,
		Namespace: ns,
	}, &sa)
	if err != nil {
		return nil, err
	}
	// the service account is expected to have a well-known annotation
	// this is used as input to assumeRoleWithWebIdentity
	roleArn := sa.Annotations[roleARNAnnotation]
	if roleArn == "" {
		return nil, fmt.Errorf("an IAM role must be associated with service account %s (namespace: %s)", saName, ns)
	}

	tokenAud := sa.Annotations[audienceAnnotation]
	if tokenAud == "" {
		tokenAud = defaultTokenAudience
	}
	audiences := []string{tokenAud}

	jwtProv, err := jwtProvider(ctx, saName, ns, roleArn, audiences, region)
	if err != nil {
		return nil, err
	}

	logger.V(1).Info("using credentials via service account", "role", roleArn, "region", region)
	return jwtProv, nil
}

// CredsFromSecretRef pulls access-key / secret-access-key from a secretRef to
// construct a aws.Credentials object
// The namespace of the external secret is used if the ClusterSecretStore does not specify a namespace (referentAuth)
// If the ClusterSecretStore defines a namespace it will take precedence.
func CredsFromSecretRef(ctx context.Context, auth esv1.VaultIamAuth, storeKind string, kube kclient.Client, namespace string) (aws.CredentialsProvider, error) {
	akid, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&auth.SecretRef.AccessKeyID,
	)
	if err != nil {
		return nil, err
	}
	sak, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		&auth.SecretRef.SecretAccessKey,
	)
	if err != nil {
		return nil, err
	}

	// session token is optional
	sessionToken, _ := resolvers.SecretKeyRef(
		ctx,
		kube,
		storeKind,
		namespace,
		auth.SecretRef.SessionToken,
	)
	return credentials.NewStaticCredentialsProvider(akid, sak, sessionToken), err
}

// STSProvider is a function type that returns an STS client.
type STSProvider func(*aws.Config) *sts.Client

// DefaultSTSProvider returns the default sts client.
func DefaultSTSProvider(cfg *aws.Config) *sts.Client {
	return sts.NewFromConfig(*cfg, func(o *sts.Options) {
		o.EndpointResolverV2 = customEndpointResolver{}
	})
}

// GetAWSConfig returns the aws config or an error.
func GetAWSConfig(cfg *aws.Config) (*aws.Config, error) {
	return cfg, nil
}
