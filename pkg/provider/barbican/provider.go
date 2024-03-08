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

	authOpts := gophercloud.AuthOptions{
		DomainID:         storeSpec.Provider.Barbican.UserDomain,
		TenantName:       storeSpec.Provider.Barbican.ProjectName,
		IdentityEndpoint: storeSpec.Provider.Barbican.AuthUrl,
	}

	if storeSpec.Provider.Barbican.Auth.UserPass != nil {
		authOpts.Username = storeSpec.Provider.Barbican.Auth.UserPass.UserName

		password, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &bStore.Auth.UserPass.PasswordRef.SecretAccessKey)
		if err != nil {
			return nil, err
		}
		authOpts.Password = password
	} else if storeSpec.Provider.Barbican.Auth.AppCredentials != nil {
		authOpts.ApplicationCredentialID = storeSpec.Provider.Barbican.Auth.AppCredentials.ApplicationID
		password, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, &bStore.Auth.AppCredentials.ApplicationSecretRef.SecretAccessKey)
		if err != nil {
			return nil, err
		}
		authOpts.ApplicationCredentialSecret = password
	} else {
		return nil, fmt.Errorf(errInitProvider, "Chossing an authentication strategy is mandatory")
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

	if g.Auth.UserPass != nil {
		if err := utils.ValidateReferentSecretSelector(store, g.Auth.DeepCopy().UserPass.PasswordRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	} else if g.Auth.AppCredentials != nil {
		if err := utils.ValidateReferentSecretSelector(store, g.Auth.DeepCopy().AppCredentials.ApplicationSecretRef.SecretAccessKey); err != nil {
			return nil, fmt.Errorf(errInvalidAuthSecretRef, err)
		}
	} else {
		return nil, fmt.Errorf(errInvalidBarbicanProv)
	}

	return nil, nil
}

func isReferentSpec(prov *esv1beta1.BarbicanProvider) bool {
	if prov.Auth.UserPass != nil && prov.Auth.UserPass.PasswordRef != nil &&
		prov.Auth.UserPass.PasswordRef.SecretAccessKey.Namespace == nil {
		return true
	} else if prov.Auth.UserPass != nil && prov.Auth.AppCredentials.ApplicationSecretRef != nil &&
		prov.Auth.AppCredentials.ApplicationSecretRef.SecretAccessKey.Namespace == nil {
		return true
	}
	return false
}

func StringPtr(s string) *string {
	return &s
}
