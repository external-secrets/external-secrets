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

package aws

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	v1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/parameterstore"
	"github.com/external-secrets/external-secrets/pkg/provider/aws/secretsmanager"
	awssess "github.com/external-secrets/external-secrets/pkg/provider/aws/session"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

// Provider satisfies the provider interface.
type Provider struct{}

var log = ctrl.Log.WithName("provider").WithName("aws")

const (
	SecretsManagerEndpointEnv = "AWS_SECRETSMANAGER_ENDPOINT"
	STSEndpointEnv            = "AWS_STS_ENDPOINT"
	SSMEndpointEnv            = "AWS_SSM_ENDPOINT"

	errUnableCreateSession                     = "unable to create session: %w"
	errUnknownProviderService                  = "unknown AWS Provider Service: %s"
	errInvalidClusterStoreMissingAKIDNamespace = "invalid ClusterSecretStore: missing AWS AccessKeyID Namespace"
	errInvalidClusterStoreMissingSAKNamespace  = "invalid ClusterSecretStore: missing AWS SecretAccessKey Namespace"
	errFetchAKIDSecret                         = "could not fetch accessKeyID secret: %w"
	errFetchSAKSecret                          = "could not fetch SecretAccessKey secret: %w"
	errMissingSAK                              = "missing SecretAccessKey"
	errMissingAKID                             = "missing AccessKeyID"
	errNilStore                                = "found nil store"
	errMissingStoreSpec                        = "store is missing spec"
	errMissingProvider                         = "storeSpec is missing provider"
	errInvalidProvider                         = "invalid provider spec. Missing AWS field in store %s"
)

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace, awssess.DefaultSTSProvider)
}

func newClient(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, assumeRoler awssess.STSProvider) (provider.SecretsClient, error) {
	prov, err := getAWSProvider(store)
	if err != nil {
		return nil, err
	}
	sess, err := newSession(ctx, store, kube, namespace, assumeRoler)
	if err != nil {
		return nil, fmt.Errorf(errUnableCreateSession, err)
	}
	switch prov.Service {
	case esv1alpha1.AWSServiceSecretsManager:
		return secretsmanager.New(sess)
	case esv1alpha1.AWSServiceParameterStore:
		return parameterstore.New(sess)
	}
	return nil, fmt.Errorf(errUnknownProviderService, prov.Service)
}

// newSession creates a new aws session based on a store
// it looks up credentials at the provided secrets.
func newSession(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string, assumeRoler awssess.STSProvider) (*session.Session, error) {
	prov, err := getAWSProvider(store)
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
	session, err := awssess.New(sak, aks, prov.Region, prov.Role, assumeRoler)
	if err != nil {
		return nil, err
	}
	session.Config.EndpointResolver = ResolveEndpoint()
	return session, nil
}

// getAWSProvider does the necessary nil checks on the generic store
// it returns the aws provider or an error.
func getAWSProvider(store esv1alpha1.GenericStore) (*esv1alpha1.AWSProvider, error) {
	if store == nil {
		return nil, fmt.Errorf(errNilStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errMissingStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errMissingProvider)
	}
	prov := spc.Provider.AWS
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}
	return prov, nil
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

func init() {
	schema.Register(&Provider{}, &esv1alpha1.SecretStoreProvider{
		AWS: &esv1alpha1.AWSProvider{},
	})
}
