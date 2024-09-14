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

package auth

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/defaults"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/sts/stsiface"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/cache"
	"github.com/external-secrets/external-secrets/pkg/feature"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

// Config contains configuration to create a new AWS provider.
type Config struct {
	AssumeRole string
	Region     string
	APIRetries int
}

var (
	log                = ctrl.Log.WithName("provider").WithName("aws")
	enableSessionCache bool
	sessionCache       *cache.Cache[*session.Session]
)

const (
	roleARNAnnotation    = "eks.amazonaws.com/role-arn"
	audienceAnnotation   = "eks.amazonaws.com/audience"
	defaultTokenAudience = "sts.amazonaws.com"

	errFetchAKIDSecret = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret  = "could not fetch SecretAccessKey secret: %w"
	errFetchSTSecret   = "could not fetch SessionToken secret: %w"
)

func init() {
	fs := pflag.NewFlagSet("aws-auth", pflag.ExitOnError)
	fs.BoolVar(&enableSessionCache, "experimental-enable-aws-session-cache", false, "Enable experimental AWS session cache. External secret will reuse the AWS session without creating a new one on each request.")
	feature.Register(feature.Feature{
		Flags: fs,
	})
	sessionCache = cache.Must[*session.Session](1024, nil)
}

// New creates a new aws session based on the provided store
// it uses the following authentication mechanisms in order:
// * service-account token authentication via AssumeRoleWithWebIdentity
// * static credentials from a Kind=Secret, optionally with doing a AssumeRole.
// * sdk default provider chain, see: https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default
func New(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string, assumeRoler STSProvider, jwtProvider jwtProviderFactory) (*session.Session, error) {
	prov, err := util.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}
	var creds *credentials.Credentials
	isClusterKind := store.GetObjectKind().GroupVersionKind().Kind == esv1beta1.ClusterSecretStoreKind

	// use credentials via service account token
	jwtAuth := prov.Auth.JWTAuth
	if jwtAuth != nil {
		creds, err = credsFromServiceAccount(ctx, prov.Auth, prov.Region, isClusterKind, kube, namespace, jwtProvider)
		if err != nil {
			return nil, err
		}
	}

	// use credentials from secretRef
	secretRef := prov.Auth.SecretRef
	if secretRef != nil {
		log.V(1).Info("using credentials from secretRef")
		creds, err = credsFromSecretRef(ctx, prov.Auth, store.GetKind(), kube, namespace)
		if err != nil {
			return nil, err
		}
	}

	config := aws.NewConfig().WithEndpointResolver(ResolveEndpoint())
	if creds != nil {
		config.WithCredentials(creds)
	}
	if prov.Region != "" {
		config.WithRegion(prov.Region)
	}

	sess, err := getAWSSession(config, enableSessionCache, store.GetName(), store.GetTypeMeta().Kind, namespace, store.GetObjectMeta().ResourceVersion)
	if err != nil {
		return nil, err
	}

	for _, aRole := range prov.AdditionalRoles {
		stsclient := assumeRoler(sess)
		sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, aRole))
	}

	sessExtID := prov.ExternalID
	sessTransitiveTagKeys := prov.TransitiveTagKeys
	sessTags := make([]*sts.Tag, len(prov.SessionTags))
	for i, tag := range prov.SessionTags {
		sessTags[i] = &sts.Tag{
			Key:   aws.String(tag.Key),
			Value: aws.String(tag.Value),
		}
	}
	if prov.Role != "" {
		stsclient := assumeRoler(sess)
		if sessExtID != "" || sessTags != nil {
			var setAssumeRoleOptions = func(p *stscreds.AssumeRoleProvider) {
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
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, prov.Role, setAssumeRoleOptions))
		} else {
			sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, prov.Role))
		}
	}
	log.Info("using aws session", "region", *sess.Config.Region, "external id", sessExtID, "credentials", creds)
	return sess, nil
}

// NewSession creates a new aws session based on the provided store
// it uses the following authentication mechanisms in order:
// * service-account token authentication via AssumeRoleWithWebIdentity
// * static credentials from a Kind=Secret, optionally with doing a AssumeRole.
// * sdk default provider chain, see: https://docs.aws.amazon.com/sdk-for-java/v1/developer-guide/credentials.html#credentials-default
func NewGeneratorSession(ctx context.Context, auth esv1beta1.AWSAuth, role, region string, kube client.Client, namespace string, assumeRoler STSProvider, jwtProvider jwtProviderFactory) (*session.Session, error) {
	var creds *credentials.Credentials
	var err error

	// use credentials via service account token
	jwtAuth := auth.JWTAuth
	if jwtAuth != nil {
		creds, err = credsFromServiceAccount(ctx, auth, region, false, kube, namespace, jwtProvider)
		if err != nil {
			return nil, err
		}
	}

	// use credentials from secretRef
	secretRef := auth.SecretRef
	if secretRef != nil {
		log.V(1).Info("using credentials from secretRef")
		creds, err = credsFromSecretRef(ctx, auth, "", kube, namespace)
		if err != nil {
			return nil, err
		}
	}

	config := aws.NewConfig().WithEndpointResolver(ResolveEndpoint())
	if creds != nil {
		config.WithCredentials(creds)
	}
	if region != "" {
		config.WithRegion(region)
	}

	sess, err := getAWSSession(config, false, "", "", "", "")
	if err != nil {
		return nil, err
	}

	if role != "" {
		stsclient := assumeRoler(sess)
		sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, role))
	}
	log.Info("using aws session", "region", *sess.Config.Region, "credentials", creds)
	return sess, nil
}

// credsFromSecretRef pulls access-key / secret-access-key from a secretRef to
// construct a aws.Credentials object
// The namespace of the external secret is used if the ClusterSecretStore does not specify a namespace (referentAuth)
// If the ClusterSecretStore defines a namespace it will take precedence.
func credsFromSecretRef(ctx context.Context, auth esv1beta1.AWSAuth, storeKind string, kube client.Client, namespace string) (*credentials.Credentials, error) {
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

	return credentials.NewStaticCredentials(aks, sak, sessionToken), err
}

// credsFromServiceAccount uses a Kubernetes Service Account to acquire temporary
// credentials using aws.AssumeRoleWithWebIdentity. It will assume the role defined
// in the ServiceAccount annotation.
// If the ClusterSecretStore does not define a namespace it will use the namespace from the ExternalSecret (referentAuth).
// If the ClusterSecretStore defines the namespace it will take precedence.
func credsFromServiceAccount(ctx context.Context, auth esv1beta1.AWSAuth, region string, isClusterKind bool, kube client.Client, namespace string, jwtProvider jwtProviderFactory) (*credentials.Credentials, error) {
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
	return credentials.NewCredentials(jwtProv), nil
}

type jwtProviderFactory func(name, namespace, roleArn string, aud []string, region string) (credentials.Provider, error)

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
		sts.New(sess), roleArn, "external-secrets-provider-aws", tokenFetcher), nil
}

type STSProvider func(*session.Session) stsiface.STSAPI

func DefaultSTSProvider(sess *session.Session) stsiface.STSAPI {
	return sts.New(sess)
}

// getAWSSession checks if an AWS session should be reused
// it returns the aws session or an error.
func getAWSSession(config *aws.Config, enableCache bool, name, kind, namespace, resourceVersion string) (*session.Session, error) {
	key := cache.Key{
		Name:      name,
		Namespace: namespace,
		Kind:      kind,
	}

	if enableCache {
		sess, ok := sessionCache.Get(resourceVersion, key)
		if ok {
			log.Info("reusing aws session", "SecretStore", key.Name, "namespace", key.Namespace, "kind", key.Kind, "resourceversion", resourceVersion)
			return sess, nil
		}
	}

	handlers := defaults.Handlers()
	handlers.Build.PushBack(request.WithAppendUserAgent("external-secrets"))
	sess, err := session.NewSessionWithOptions(session.Options{
		Config:   *config,
		Handlers: handlers,
	})
	if err != nil {
		return nil, err
	}

	if enableCache {
		sessionCache.Add(resourceVersion, key, sess.Copy())
	}
	return sess, nil
}
