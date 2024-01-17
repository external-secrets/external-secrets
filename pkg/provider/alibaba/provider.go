package alibaba

import (
	"context"
	"fmt"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/keymanagementservice"
	"github.com/external-secrets/external-secrets/pkg/provider/alibaba/parameterstore"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Provider struct{}

var _ esv1beta1.Provider = &Provider{}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) error {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	regionID := alibabaSpec.RegionID

	if regionID == "" {
		return fmt.Errorf("missing alibaba region")
	}

	return auth.ValidateStoreAuth(store)
}

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

// NewClient constructs a new secrets client based on the provided store.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	alibabaSpec := storeSpec.Provider.Alibaba

	if alibabaSpec.Service == esv1beta1.AlibabaCloudParameterStore {
		return parameterstore.NewClient(ctx, store, kube, namespace)
	} else {
		return keymanagementservice.NewClient(ctx, store, kube, namespace)
	}
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Alibaba: &esv1beta1.AlibabaProvider{},
	})
}
