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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlcfg "sigs.k8s.io/controller-runtime/pkg/client/config"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
)

// Config contains configuration to create a new AWS provider.
type Config struct {
	AssumeRole string
	Region     string
	APIRetries int
}

type SessionCache struct {
	Name            string
	Namespace       string
	Kind            string
	ResourceVersion string
}

var (
	log         = ctrl.Log.WithName("provider").WithName("aws")
	sessions    = make(map[SessionCache]*session.Session)
	EnableCache bool
)

const (
	roleARNAnnotation    = "eks.amazonaws.com/role-arn"
	audienceAnnotation   = "eks.amazonaws.com/audience"
	defaultTokenAudience = "sts.amazonaws.com"

	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterSecretStore: missing AWS AccessKeyID Namespace"
	errInvalidClusterStoreMissingSAKNamespace  = "invalid ClusterSecretStore: missing AWS SecretAccessKey Namespace"
	errFetchAKIDSecret                         = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret                          = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                              = "missing SecretAccessKey"
	errMissingAKID                             = "missing AccessKeyID"
)

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
		creds, err = sessionFromServiceAccount(ctx, prov.Auth, prov.Region, isClusterKind, kube, namespace, jwtProvider)
		if err != nil {
			return nil, err
		}
	}

	// use credentials from sercretRef
	secretRef := prov.Auth.SecretRef
	if secretRef != nil {
		log.V(1).Info("using credentials from secretRef")
		creds, err = sessionFromSecretRef(ctx, prov.Auth, isClusterKind, kube, namespace)
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

	sess, err := getAWSSession(config, EnableCache, store.GetName(), store.GetTypeMeta().Kind, namespace, store.GetObjectMeta().ResourceVersion)
	if err != nil {
		return nil, err
	}

	if prov.Role != "" {
		stsclient := assumeRoler(sess)
		sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, prov.Role))
	}
	log.Info("using aws session", "region", *sess.Config.Region, "credentials", creds)
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
		creds, err = sessionFromServiceAccount(ctx, auth, region, false, kube, namespace, jwtProvider)
		if err != nil {
			return nil, err
		}
	}

	// use credentials from sercretRef
	secretRef := auth.SecretRef
	if secretRef != nil {
		log.V(1).Info("using credentials from secretRef")
		creds, err = sessionFromSecretRef(ctx, auth, false, kube, namespace)
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

func sessionFromSecretRef(ctx context.Context, auth esv1beta1.AWSAuth, isClusterKind bool, kube client.Client, namespace string) (*credentials.Credentials, error) {
	ke := client.ObjectKey{
		Name:      auth.SecretRef.AccessKeyID.Name,
		Namespace: namespace, // default to ExternalSecret namespace
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind {
		if auth.SecretRef.AccessKeyID.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
		}
		ke.Namespace = *auth.SecretRef.AccessKeyID.Namespace
	}
	akSecret := v1.Secret{}
	err := kube.Get(ctx, ke, &akSecret)
	if err != nil {
		return nil, fmt.Errorf(errFetchAKIDSecret, err)
	}
	ke = client.ObjectKey{
		Name:      auth.SecretRef.SecretAccessKey.Name,
		Namespace: namespace, // default to ExternalSecret namespace
	}
	// only ClusterStore is allowed to set namespace (and then it's required)
	if isClusterKind {
		if auth.SecretRef.SecretAccessKey.Namespace == nil {
			return nil, fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
		}
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

	return credentials.NewStaticCredentials(aks, sak, ""), err
}

func sessionFromServiceAccount(ctx context.Context, auth esv1beta1.AWSAuth, region string, isClusterKind bool, kube client.Client, namespace string, jwtProvider jwtProviderFactory) (*credentials.Credentials, error) {
	name := auth.JWTAuth.ServiceAccountRef.Name
	if isClusterKind {
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

// getAWSSession check if an AWS session should be reused
// it returns the aws session or an error.
func getAWSSession(config *aws.Config, enableCache bool, name, kind, namespace, resourceVersion string) (*session.Session, error) {
	tmpSession := SessionCache{
		Name:            name,
		Namespace:       namespace,
		Kind:            kind,
		ResourceVersion: resourceVersion,
	}

	if EnableCache {
		sess, ok := sessions[tmpSession]
		if ok {
			log.Info("reusing aws session", "SecretStore", tmpSession.Name, "namespace", tmpSession.Namespace, "kind", tmpSession.Kind, "resourceversion", tmpSession.ResourceVersion)
			return sess, nil
		}
	}

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

	if EnableCache {
		sessions[tmpSession] = sess
	}
	return sess, nil
}
