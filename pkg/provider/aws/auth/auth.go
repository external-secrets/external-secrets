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
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	awssess "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/util"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config contains configuration to create a new AWS provider.
type Config struct {
	AssumeRole string
	Region     string
	APIRetries int
}

var log = ctrl.Log.WithName("provider").WithName("aws")

const (
	SecretsManagerEndpointEnv = "AWS_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "AWS_STS_ENDPOINT"
	SSMEndpointEnv            = "AWS_SSM_ENDPOINT"

	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterSecretStore: missing AWS AccessKeyID Namespace"
	errInvalidClusterStoreMissingSAKNamespace  = "invalid ClusterSecretStore: missing AWS SecretAccessKey Namespace"
	errFetchAKIDSecret                         = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret                          = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                              = "missing SecretAccessKey"
	errMissingAKID                             = "missing AccessKeyID"
)

// New creates a new aws session based on a store
// it looks up credentials at the provided secrets.
func New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, assumeRoler STSProvider) (*session.Session, error) {
	prov, err := util.GetAWSProvider(store)
	if err != nil {
		return nil, err
	}
	var sak, aks string
	// use provided credentials via secret reference
	if prov.Auth != nil {
		log.V(1).Info("fetching secrets for authentication")
		ke := client.ObjectKey{
			Name:      prov.Auth.SecretRef.AccessKeyID.Name,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// only ClusterStore is allowed to set namespace (and then it's required)
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if prov.Auth.SecretRef.AccessKeyID.Namespace == nil {
				return nil, fmt.Errorf(errInvalidClusterStoreMissingAKIDNamespace)
			}
			ke.Namespace = *prov.Auth.SecretRef.AccessKeyID.Namespace
		}
		akSecret := v1.Secret{}
		err := kube.Get(ctx, ke, &akSecret)
		if err != nil {
			return nil, fmt.Errorf(errFetchAKIDSecret, err)
		}
		ke = client.ObjectKey{
			Name:      prov.Auth.SecretRef.SecretAccessKey.Name,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// only ClusterStore is allowed to set namespace (and then it's required)
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if prov.Auth.SecretRef.SecretAccessKey.Namespace == nil {
				return nil, fmt.Errorf(errInvalidClusterStoreMissingSAKNamespace)
			}
			ke.Namespace = *prov.Auth.SecretRef.SecretAccessKey.Namespace
		}
		sakSecret := v1.Secret{}
		err = kube.Get(ctx, ke, &sakSecret)
		if err != nil {
			return nil, fmt.Errorf(errFetchSAKSecret, err)
		}
		sak = string(sakSecret.Data[prov.Auth.SecretRef.SecretAccessKey.Key])
		aks = string(akSecret.Data[prov.Auth.SecretRef.AccessKeyID.Key])
		if sak == "" {
			return nil, fmt.Errorf(errMissingSAK)
		}
		if aks == "" {
			return nil, fmt.Errorf(errMissingAKID)
		}
	}
	session, err := NewSession(sak, aks, prov.Region, prov.Role, assumeRoler)
	if err != nil {
		return nil, err
	}
	session.Config.EndpointResolver = ResolveEndpoint()
	return session, nil
}

// New creates a new aws session based on the supported input methods.
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials
func NewSession(sak, aks, region, role string, stsprovider STSProvider) (*awssess.Session, error) {
	config := aws.NewConfig()
	sessionOpts := awssess.Options{
		Config: *config,
	}
	if sak != "" && aks != "" {
		sessionOpts.Config.Credentials = credentials.NewStaticCredentials(aks, sak, "")
		sessionOpts.SharedConfigState = awssess.SharedConfigDisable
	}
	sess, err := awssess.NewSessionWithOptions(sessionOpts)
	if err != nil {
		return nil, fmt.Errorf("unable to create aws session: %w", err)
	}
	if region != "" {
		log.V(1).Info("using region", "region", region)
		sess.Config.WithRegion(region)
	}

	if role != "" {
		log.V(1).Info("assuming role", "role", role)
		stsclient := stsprovider(sess)
		sess.Config.WithCredentials(stscreds.NewCredentialsWithClient(stsclient, role))
	}
	sess.Handlers.Build.PushBack(request.WithAppendUserAgent("external-secrets"))
	return sess, nil
}

type STSProvider func(*awssess.Session) stscreds.AssumeRoler

func DefaultSTSProvider(sess *awssess.Session) stscreds.AssumeRoler {
	return sts.New(sess)
}

// ResolveEndpoint returns a ResolverFunc with
// customizable endpoints.
func ResolveEndpoint() endpoints.ResolverFunc {
	customEndpoints := make(map[string]string)
	if v := os.Getenv(SecretsManagerEndpointEnv); v != "" {
		customEndpoints["secretsmanager"] = v
	}
	if v := os.Getenv(SSMEndpointEnv); v != "" {
		customEndpoints["ssm"] = v
	}
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
