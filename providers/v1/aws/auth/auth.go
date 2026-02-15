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

// Package auth provides authentication functionality for the AWS provider, handling
// various authentication methods including static credentials, IAM roles,
// and web identity tokens.
package auth

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	stsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	awsutil "github.com/external-secrets/external-secrets/providers/v1/aws/util"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
)

// Config contains configuration to create a new AWS provider.
type Config struct {
	AssumeRole string
	Region     string
	APIRetries int
}

var (
	log = ctrl.Log.WithName("provider").WithName("aws")
)

const (
	roleARNAnnotation    = "eks.amazonaws.com/role-arn"
	audienceAnnotation   = "eks.amazonaws.com/audience"
	defaultTokenAudience = "sts.amazonaws.com"

	errFetchAKIDSecret = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret  = "could not fetch SecretAccessKey secret: %w"
	errFetchSTSecret   = "could not fetch SessionToken secret: %w"
)

// Opts define options for New function.
type Opts struct {
	Store       esv1.GenericStore
	Kube        client.Client
	Namespace   string
	AssumeRoler STSProvider
	JWTProvider jwtProviderFactory
}

// New creates a new aws config based on the provided store
// it uses the following authentication mechanisms in order:
// * service-account token authentication via AssumeRoleWithWebIdentity
// * static credentials from a Kind=Secret, optionally with doing a AssumeRole.
// * sdk default provider chain, see: https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default
func New(ctx context.Context, opts Opts) (*aws.Config, error) {
	prov, err := awsutil.GetAWSProvider(opts.Store)
	if err != nil {
		return nil, err
	}
	var credsProvider aws.CredentialsProvider
	isClusterKind := opts.Store.GetObjectKind().GroupVersionKind().Kind == esv1.ClusterSecretStoreKind

	credsProvider, err = constructCredsProvider(ctx, prov, isClusterKind, opts)
	if err != nil {
		return nil, err
	}

	// global endpoint resolver is deprecated, should we EndpointResolverV2 field on service client options
	var loadCfgOpts []func(*config.LoadOptions) error
	if credsProvider != nil {
		loadCfgOpts = append(loadCfgOpts, config.WithCredentialsProvider(credsProvider))
	}
	if prov.Region != "" {
		loadCfgOpts = append(loadCfgOpts, config.WithRegion(prov.Region))
	}

	return createConfiguration(prov, opts.AssumeRoler, loadCfgOpts)
}

func createConfiguration(prov *esv1.AWSProvider, assumeRoler STSProvider, loadCfgOpts []func(*config.LoadOptions) error) (*aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), loadCfgOpts...)
	if err != nil {
		return nil, err
	}

	for _, aRole := range prov.AdditionalRoles {
		stsclient := assumeRoler(&cfg)
		cfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, aRole)
	}

	sessExtID := prov.ExternalID
	sessTransitiveTagKeys := prov.TransitiveTagKeys
	sessTags := make([]stsTypes.Tag, len(prov.SessionTags))
	for i, tag := range prov.SessionTags {
		sessTags[i] = stsTypes.Tag{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		}
	}
	if prov.Role != "" {
		stsclient := assumeRoler(&cfg)
		if sessExtID != "" || sessTags != nil {
			cfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, prov.Role, setAssumeRoleOptionFn(sessExtID, sessTags, sessTransitiveTagKeys))
		} else {
			cfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, prov.Role)
		}
	}
	log.Info("using aws config", "region", cfg.Region, "external id", sessExtID, "credentials", cfg.Credentials)

	return &cfg, nil
}

func setAssumeRoleOptionFn(sessExtID string, sessTags []stsTypes.Tag, sessTransitiveTagKeys []string) func(p *stscreds.AssumeRoleOptions) {
	return func(p *stscreds.AssumeRoleOptions) {
		if sessExtID != "" {
			p.ExternalID = aws.String(sessExtID)
		}
		if sessTags != nil {
			p.Tags = sessTags
			if len(sessTransitiveTagKeys) > 0 {
				p.TransitiveTagKeys = sessTransitiveTagKeys
			}
		}
	}
}

func constructCredsProvider(ctx context.Context, prov *esv1.AWSProvider, isClusterKind bool, opts Opts) (aws.CredentialsProvider, error) {
	switch {
	case prov.Auth.JWTAuth != nil:
		return credsFromServiceAccount(ctx, prov.Auth, prov.Region, isClusterKind, opts.Kube, opts.Namespace, opts.JWTProvider)
	case prov.Auth.SecretRef != nil:
		log.V(1).Info("using credentials from secretRef")
		return credsFromSecretRef(ctx, prov.Auth, opts.Store.GetKind(), opts.Kube, opts.Namespace)
	default:
		return nil, nil
	}
}

// NewGeneratorSession creates a new aws session based on the provided store
// it uses the following authentication mechanisms in order:
// * service-account token authentication via AssumeRoleWithWebIdentity
// * static credentials from a Kind=Secret, optionally with doing a AssumeRole.
// * sdk default provider chain, see: https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default
func NewGeneratorSession(
	ctx context.Context,
	auth esv1.AWSAuth,
	role, region string,
	kube client.Client,
	namespace string,
	assumeRoler STSProvider,
	jwtProvider jwtProviderFactory,
) (*aws.Config, error) {
	var (
		credsProvider aws.CredentialsProvider
		err           error
	)

	// use credentials via service account token
	jwtAuth := auth.JWTAuth
	if jwtAuth != nil {
		credsProvider, err = credsFromServiceAccount(ctx, auth, region, false, kube, namespace, jwtProvider)
		if err != nil {
			return nil, err
		}
	}

	// use credentials from secretRef
	secretRef := auth.SecretRef
	if secretRef != nil {
		log.V(1).Info("using credentials from secretRef")
		credsProvider, err = credsFromSecretRef(ctx, auth, "", kube, namespace)
		if err != nil {
			return nil, err
		}
	}
	awscfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	if credsProvider != nil {
		awscfg.Credentials = credsProvider
	}
	if region != "" {
		awscfg.Region = region
	}

	if role != "" {
		stsclient := assumeRoler(&awscfg)
		awscfg.Credentials = stscreds.NewAssumeRoleProvider(stsclient, role)
	}
	log.Info("using aws config", "region", awscfg.Region, "credentials", awscfg.Credentials)
	return &awscfg, nil
}

// credsFromSecretRef pulls access-key / secret-access-key from a secretRef to
// construct a aws.Credentials object
// The namespace of the external secret is used if the ClusterSecretStore does not specify a namespace (referentAuth)
// If the ClusterSecretStore defines a namespace it will take precedence.
func credsFromSecretRef(ctx context.Context, auth esv1.AWSAuth, storeKind string, kube client.Client, namespace string) (aws.CredentialsProvider, error) {
	sak, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &auth.SecretRef.SecretAccessKey)
	if err != nil {
		return nil, fmt.Errorf(errFetchSAKSecret, err)
	}
	aks, err := resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, &auth.SecretRef.AccessKeyID)
	if err != nil {
		return nil, fmt.Errorf(errFetchAKIDSecret, err)
	}

	var sessionToken string
	if auth.SecretRef.SessionToken != nil {
		sessionToken, err = resolvers.SecretKeyRef(ctx, kube, storeKind, namespace, auth.SecretRef.SessionToken)
		if err != nil {
			return nil, fmt.Errorf(errFetchSTSecret, err)
		}
	}
	var credsProvider aws.CredentialsProvider = credentials.NewStaticCredentialsProvider(aks, sak, sessionToken)

	return credsProvider, nil
}

// credsFromServiceAccount uses a Kubernetes Service Account to acquire temporary
// credentials using aws.AssumeRoleWithWebIdentity. It will assume the role defined
// in the ServiceAccount annotation.
// If the ClusterSecretStore does not define a namespace it will use the namespace from the ExternalSecret (referentAuth).
// If the ClusterSecretStore defines the namespace it will take precedence.
func credsFromServiceAccount(
	ctx context.Context,
	auth esv1.AWSAuth,
	region string,
	isClusterKind bool,
	kube client.Client,
	namespace string,
	jwtProvider jwtProviderFactory,
) (aws.CredentialsProvider, error) {
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

	log.V(1).Info("using credentials via service account", "role", roleArn, "region", region)

	return jwtProv, nil
}

type jwtProviderFactory func(name, namespace, roleArn string, aud []string, region string) (aws.CredentialsProvider, error)

// DefaultJWTProvider returns a credentials.Provider that calls the AssumeRoleWithWebidentity
// controller-runtime/client does not support TokenRequest or other subresource APIs
// so we need to construct our own client and use it to fetch tokens.
func DefaultJWTProvider(name, namespace, roleArn string, aud []string, region string) (aws.CredentialsProvider, error) {
	cfg, err := ctrlcfg.GetConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	awscfg, err := config.LoadDefaultConfig(context.TODO(), config.WithAppID("external-secrets"),
		config.WithRegion(region),
		config.WithSharedConfigFiles([]string{}), // Disable shared config files:
		config.WithSharedCredentialsFiles([]string{}))

	if err != nil {
		return nil, err
	}

	tokenFetcher := authTokenFetcher{
		Namespace:      namespace,
		Audiences:      aud,
		ServiceAccount: name,
		k8sClient:      clientset.CoreV1(),
	}
	stsClient := sts.NewFromConfig(awscfg, func(o *sts.Options) {
		o.EndpointResolverV2 = customEndpointResolver{}
	})
	return stscreds.NewWebIdentityRoleProvider(
		stsClient, roleArn, tokenFetcher, func(opts *stscreds.WebIdentityRoleOptions) {
			opts.RoleSessionName = "external-secrets-provider-aws"
		}), nil
}

// STSprovider defines the interface for interacting with AWS STS API operations.
// This allows for mocking STS operations during testing.
type STSprovider interface {
	AssumeRole(ctx context.Context, params *sts.AssumeRoleInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleOutput, error)
	AssumeRoleWithSAML(ctx context.Context, params *sts.AssumeRoleWithSAMLInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithSAMLOutput, error)
	AssumeRoleWithWebIdentity(ctx context.Context, params *sts.AssumeRoleWithWebIdentityInput, optFns ...func(*sts.Options)) (*sts.AssumeRoleWithWebIdentityOutput, error)
	AssumeRoot(ctx context.Context, params *sts.AssumeRootInput, optFns ...func(*sts.Options)) (*sts.AssumeRootOutput, error)
	DecodeAuthorizationMessage(ctx context.Context, params *sts.DecodeAuthorizationMessageInput, optFns ...func(*sts.Options)) (*sts.DecodeAuthorizationMessageOutput, error)
}

// STSProvider is a function type that returns an STSprovider implementation.
// Used to inject custom or mock STS clients.
type STSProvider func(*aws.Config) STSprovider

// DefaultSTSProvider creates and returns a new STS client from the provided AWS config.
func DefaultSTSProvider(cfg *aws.Config) STSprovider {
	stsClient := sts.NewFromConfig(*cfg, func(o *sts.Options) {
		o.EndpointResolverV2 = customEndpointResolver{}
	})
	return stsClient
}
