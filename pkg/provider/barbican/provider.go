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
package barbican

import (
	"context"
	"fmt"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

// Provider is a secrets provider for OpenStack Barbican.
// It implements the necessary NewClient() and ValidateStore() funcs.
type Provider struct{}

var _ esv1beta1.Provider = &Provider{}

const (
	errInitProvider        = "unable to initialize barbican provider: %s"
	errNilStore            = "found nil store"
	errMissingStoreSpec    = "store is missing spec"
	errMissingProvider     = "storeSpec is missing provider"
	errInvalidProviderSpec = "invalid provider spec. Missing Barbican field in store %s"
)

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Barbican: &esv1beta1.BarbicanProvider{},
	})
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	return newClient(ctx, store, kube, namespace)
}

func newClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	if store == nil {
		return nil, fmt.Errorf(errInitProvider, "nil store")
	}
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.Barbican == nil {
		return nil, fmt.Errorf(errBarbicanStore)
	}
	bStore := storeSpec.Provider.Barbican

	password, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &bStore.Auth.SecretRef.SecretAccessKey)
	if err != nil {
		return nil, err
	}

	authOpts := gophercloud.AuthOptions{
		DomainID:         storeSpec.Provider.Barbican.UserDomain,
		TenantName:       storeSpec.Provider.Barbican.ProjectName,
		IdentityEndpoint: storeSpec.Provider.Barbican.AuthUrl,
	}

	if storeSpec.Provider.Barbican.AuthType == "username" {
		authOpts.Username = storeSpec.Provider.Barbican.Username
		authOpts.Password = password
	} else {
		authOpts.ApplicationCredentialID = storeSpec.Provider.Barbican.AppCredentialID
		authOpts.ApplicationCredentialSecret = password
	}

	endpointOpts := gophercloud.EndpointOpts{
		Type:   "key-manager",
		Name:   storeSpec.Provider.Barbican.ServiceName,
		Region: storeSpec.Provider.Barbican.Region,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	c, err := openstack.NewKeyManagerV1(provider, endpointOpts)
	if err != nil {
		return nil, err
	}

	client := &Client{
		// config:    config,
		client:    c,
		kube:      kube,
		store:     bStore,
		storeKind: store.GetKind(),
		namespace: namespace,
	}

	return client, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	if store == nil {
		return nil, fmt.Errorf(errInvalidStore)
	}
	spc := store.GetSpec()
	if spc == nil {
		return nil, fmt.Errorf(errInvalidStoreSpec)
	}
	if spc.Provider == nil {
		return nil, fmt.Errorf(errInvalidStoreProv)
	}
	g := spc.Provider.Barbican
	if p == nil {
		return nil, fmt.Errorf(errInvalidBarbicanProv)
	}
	if g.Auth.SecretRef != nil {
		if err := utils.ValidateReferentSecretSelector(store, g.Auth.SecretRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	}
	return nil, nil
}

func getPasswordFromSecrets(ctx context.Context, auth esv1beta1.BarbicanAuth, kube kclient.Client, namespace string) (string, error) {
	sr := auth.SecretRef
	if sr == nil {
		return "", fmt.Errorf(errMissingSAK)
	}
	credentialsSecret := &v1.Secret{}
	credentialsSecretName := sr.SecretAccessKey.Name
	objectKey := types.NamespacedName{
		Name:      credentialsSecretName,
		Namespace: namespace,
	}
	err := kube.Get(ctx, objectKey, credentialsSecret)
	if err != nil {
		return "", fmt.Errorf(errFetchSAKSecret, err)
	}
	credentials := credentialsSecret.Data[sr.SecretAccessKey.Key]
	if (credentials == nil) || (len(credentials) == 0) {
		return "", fmt.Errorf(errMissingSAK)
	}

	return string(credentials), nil
}

func isReferentSpec(prov *esv1beta1.BarbicanProvider) bool {
	if prov.Auth.SecretRef != nil &&
		prov.Auth.SecretRef.SecretAccessKey.Namespace == nil {
		return true
	}
	return false
}

func StringPtr(s string) *string {
	return &s
}

// it returns the barbican provider or an error.
func GetBarbicanProvider(store esv1beta1.GenericStore) (*esv1beta1.BarbicanProvider, error) {
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
	prov := spc.Provider.Barbican
	if prov == nil {
		return nil, fmt.Errorf(errInvalidProviderSpec, store.GetObjectMeta().String())
	}
	return prov, nil
}
