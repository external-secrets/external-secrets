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
		return nil, fmt.Errorf("unable to create session: %w", err)
	}
	switch prov.Service {
	case esv1alpha1.AWSServiceSecretsManager:
		return secretsmanager.New(sess)
	case esv1alpha1.AWSServiceParameterStore:
		return parameterstore.New(sess)
	}
	return nil, fmt.Errorf("unknown AWS Provider Service: %s", prov.Service)
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
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWS AccessKeyID Namespace")
			}
			ke.Namespace = *prov.Auth.SecretRef.AccessKeyID.Namespace
		}
		akSecret := v1.Secret{}
		err := kube.Get(ctx, ke, &akSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch accessKeyID secret: %w", err)
		}
		ke = client.ObjectKey{
			Name:      prov.Auth.SecretRef.SecretAccessKey.Name,
			Namespace: namespace, // default to ExternalSecret namespace
		}
		// only ClusterStore is allowed to set namespace (and then it's required)
		if store.GetObjectKind().GroupVersionKind().Kind == esv1alpha1.ClusterSecretStoreKind {
			if prov.Auth.SecretRef.SecretAccessKey.Namespace == nil {
				return nil, fmt.Errorf("invalid ClusterSecretStore: missing AWS SecretAccessKey Namespace")
			}
			ke.Namespace = *prov.Auth.SecretRef.SecretAccessKey.Namespace
		}
		sakSecret := v1.Secret{}
		err = kube.Get(ctx, ke, &sakSecret)
		if err != nil {
			return nil, fmt.Errorf("could not fetch SecretAccessKey secret: %w", err)
		}
		sak = string(sakSecret.Data[prov.Auth.SecretRef.SecretAccessKey.Key])
		aks = string(akSecret.Data[prov.Auth.SecretRef.AccessKeyID.Key])
		if sak == "" {
			return nil, fmt.Errorf("missing SecretAccessKey")
		}
		if aks == "" {
			return nil, fmt.Errorf("missing AccessKeyID")
		}
	}
	return awssess.New(sak, aks, prov.Region, prov.Role, assumeRoler)
}

// getAWSProvider does the necessary nil checks on the generic store
// it returns the aws provider or an error.
func getAWSProvider(store esv1alpha1.GenericStore) (*esv1alpha1.AWSProvider, error) {
	if store == nil {
		return nil, fmt.Errorf("found nil store")
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf("store is missing spec")
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf("storeSpec is missing provider")
	}
	prov := spc.Provider.AWS
	if prov == nil {
		return nil, fmt.Errorf("invalid provider spec. Missing AWS field in store %s", store.GetObjectMeta().String())
	}
	return prov, nil
}

func init() {
	schema.Register(&Provider{}, &esv1alpha1.SecretStoreProvider{
		AWS: &esv1alpha1.AWSProvider{},
	})
}
