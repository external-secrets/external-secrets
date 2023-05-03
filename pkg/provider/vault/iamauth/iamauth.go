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

// Mostly sourced from ~/external-secrets/pkg/provider/aws/auth
package iamauth

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	k8scorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/vault/util"
)

var (
	logger = ctrl.Log.WithName("provider").WithName("vault")
)

const (
	roleARNAnnotation    = "eks.amazonaws.com/role-arn"
	audienceAnnotation   = "eks.amazonaws.com/audience"
	defaultTokenAudience = "sts.amazonaws.com"

	STSEndpointEnv                = "AWS_STS_ENDPOINT"
	AWSWebIdentityTokenFileEnvVar = "AWS_WEB_IDENTITY_TOKEN_FILE"

	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterSecretStore: missing AWS AccessKeyID Namespace"
	errInvalidClusterStoreMissingSAKNamespace  = "invalid ClusterSecretStore: missing AWS SecretAccessKey Namespace"
	errFetchAKIDSecret                         = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret                          = "could not fetch SecretAccessKey secret: %w"
	errFetchSTSecret                           = "could not fetch SessionToken secret: %w"
	errMissingSAK                              = "missing SecretAccessKey"
	errMissingAKID                             = "missing AccessKeyID"
)

// DefaultJWTProvider returns a credentials.Provider that calls the AssumeRoleWithWebidentity
// controller-runtime/client does not support TokenRequest or other subresource APIs
// so we need to construct our own client and use it to fetch tokens.
func DefaultJWTProvider(name, namespace, roleArn string, aud []string, region string) (credentials.Provider, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	handlers := defaults.Handlers()
	handlers.Build.PushBack(request.WithAppendUserAgent("external-secrets"))
	awscfg := aws.NewConfig().WithEndpointResolver(ResolveEndpoint())
	if region != "" {
		awscfg.WithRegion(region)
	}
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            *awscfg,
		SharedConfigState: session.SharedConfigDisable,
		Handlers:          handlers,
	})
	if err != nil {
		return nil, err
	}
	tokenFetcher := &authTokenFetcher{
		Namespace:      namespace,
		Audiences:      aud,
		ServiceAccount: name,
		k8sClient:      clientset.CoreV1(),
	}

	return stscreds.NewWebIdentityRoleProviderWithOptions(
		sts.New(sess), roleArn, "external-secrets-provider-vault", tokenFetcher), nil
}

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.
func ResolveEndpoint() endpoints.ResolverFunc {
	customEndpoints := make(map[string]string)
	if v := os.Getenv(STSEndpointEnv); v != "" {
		customEndpoints["sts"] = v
	}
	return ResolveEndpointWithServiceMap(customEndpoints)
}

func ResolveEndpointWithServiceMap(customEndpoints map[string]string) endpoints.ResolverFunc {
	defaultResolver := endpoints.DefaultResolver()
	return func(service, region string, opts ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		if ep, ok := customEndpoints[service]; ok {
			return endpoints.ResolvedEndpoint{
				URL: ep,
			}, nil
		}
		return defaultResolver.EndpointFor(service, region, opts...)
	}
}

// mostly taken from:
// https://github.com/aws/secrets-store-csi-driver-provider-aws/blob/main/auth/auth.go#L140-L145

type authTokenFetcher struct {
	Namespace string
	// Audience is the token aud claim
	// which is verified by the aws oidc provider
	// see: https://github.com/external-secrets/external-secrets/issues/1251#issuecomment-1161745849
	Audiences      []string
	ServiceAccount string
	k8sClient      k8scorev1.CoreV1Interface
}

// FetchToken satisfies the stscreds.TokenFetcher interface
// it is used to generate service account tokens which are consumed by the aws sdk.
func (p authTokenFetcher) FetchToken(ctx credentials.Context) ([]byte, error) {
	logger.V(1).Info("fetching token", "ns", p.Namespace, "sa", p.ServiceAccount)
	tokRsp, err := p.k8sClient.ServiceAccounts(p.Namespace).CreateToken(ctx, p.ServiceAccount, &authv1.TokenRequest{
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
func CredsFromServiceAccount(ctx context.Context, auth esv1beta1.VaultIamAuth, region string, isClusterKind bool, kube kclient.Client, namespace string, jwtProvider util.JwtProviderFactory) (*credentials.Credentials, error) {
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

	jwtProv, err := jwtProvider(name, namespace, roleArn, audiences, region)
	if err != nil {
		return nil, err
	}

	logger.V(1).Info("using credentials via service account", "role", roleArn, "region", region)
	return credentials.NewCredentials(jwtProv), nil
}

func CredsFromControllerServiceAccount(ctx context.Context, saname, ns, region string, kube kclient.Client, jwtProvider util.JwtProviderFactory) (*credentials.Credentials, error) {
	name := saname
	nmspc := ns

	sa := v1.ServiceAccount{}
	err := kube.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: nmspc,
	}, &sa)
	if err != nil {
		return nil, err
	}
	// the service account is expected to have a well-known annotation
	// this is used as input to assumeRoleWithWebIdentity
	roleArn := sa.Annotations[roleARNAnnotation]
	if roleArn == "" {
		return nil, fmt.Errorf("an IAM role must be associated with service account %s (namespace: %s)", name, nmspc)
	}

	tokenAud := sa.Annotations[audienceAnnotation]
	if tokenAud == "" {
		tokenAud = defaultTokenAudience
	}
	audiences := []string{tokenAud}

	jwtProv, err := jwtProvider(name, nmspc, roleArn, audiences, region)
	if err != nil {
		return nil, err
	}

	logger.V(1).Info("using credentials via service account", "role", roleArn, "region", region)
	return credentials.NewCredentials(jwtProv), nil
}

// CredsFromSecretRef pulls access-key / secret-access-key from a secretRef to
// construct a aws.Credentials object
// The namespace of the external secret is used if the ClusterSecretStore does not specify a namespace (referentAuth)
// If the ClusterSecretStore defines a namespace it will take precedence.
func CredsFromSecretRef(ctx context.Context, auth esv1beta1.VaultIamAuth, isClusterKind bool, kube kclient.Client, namespace string) (*credentials.Credentials, error) {
	ke := kclient.ObjectKey{
		Name:      auth.SecretRef.AccessKeyID.Name,
		Namespace: namespace,
	}
	if isClusterKind && auth.SecretRef.AccessKeyID.Namespace != nil {
		ke.Namespace = *auth.SecretRef.AccessKeyID.Namespace
	}
	akSecret := v1.Secret{}
	err := kube.Get(ctx, ke, &akSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAKIDSecret, err)
	}
	ke = kclient.ObjectKey{
		Name:      auth.SecretRef.SecretAccessKey.Name,
		Namespace: namespace,
	}
	if isClusterKind && auth.SecretRef.SecretAccessKey.Namespace != nil {
		ke.Namespace = *auth.SecretRef.SecretAccessKey.Namespace
	}
	sakSecret := v1.Secret{}
	err = kube.Get(ctx, ke, &sakSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchSAKSecret, err)
	}
	sak := string(sakSecret.Data[auth.SecretRef.SecretAccessKey.Key])
	aks := string(akSecret.Data[auth.SecretRef.AccessKeyID.Key])
	if sak == "" {
		return nil, fmt.Errorf(errMissingSAK)
	}
	if aks == "" {
		return nil, fmt.Errorf(errMissingAKID)
	}

	var sessionToken string
	if auth.SecretRef.SessionToken != nil {
		ke = kclient.ObjectKey{
			Name:      auth.SecretRef.SessionToken.Name,
			Namespace: namespace,
		}
		if isClusterKind && auth.SecretRef.SessionToken.Namespace != nil {
			ke.Namespace = *auth.SecretRef.SessionToken.Namespace
		}
		stSecret := v1.Secret{}
		err = kube.Get(ctx, ke, &stSecret)
		if err != nil {
			return nil, fmt.Errorf(errFetchSTSecret, err)
		}
		sessionToken = string(stSecret.Data[auth.SecretRef.SessionToken.Key])
	}

	return credentials.NewStaticCredentials(aks, sak, sessionToken), err
}

type STSProvider func(*session.Session) stsiface.STSAPI

func DefaultSTSProvider(sess *session.Session) stsiface.STSAPI {
	return sts.New(sess)
}

// getAWSSession returns the aws session or an error.
func GetAWSSession(config *aws.Config) (*session.Session, error) {
	handlers := defaults.Handlers()
	handlers.Build.PushBack(request.WithAppendUserAgent("external-secrets"))
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            *config,
		Handlers:          handlers,
		SharedConfigState: session.SharedConfigDisable,
	})
	if err != nil {
		return nil, err
	}

	return sess, nil
}
