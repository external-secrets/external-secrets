package ovh

import (
	"context"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Provider struct{}

func (p *Provider) NewClient(ctx context.Context, store esv1.GenericStore, kube kclient.Client, namespace string) (esv1.SecretsClient, error) {
	return nil, nil
}

func (p *Provider) Capabilities() esv1.SecretStoreCapabilities {
	return esv1.SecretStoreReadWrite
}

func (p *Provider) ValidateStore(store esv1.GenericStore) (admission.Warnings, error) {
	return nil, nil
}

func init() {
	esv1.Register(&Provider{}, &esv1.SecretStoreProvider{
		Ovh: &esv1.OvhProvider{},
	}, esv1.MaintenanceStatusMaintained)
}
