package awsengine

import (
	"context"
	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	kclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.SecretsClient = &client{}
var _ esv1beta1.Provider = &provider{}

type client struct {
}

func (c client) GetSecret(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) Validate() (esv1beta1.ValidationResult, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) GetSecretMap(ctx context.Context, ref esv1beta1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) GetAllSecrets(ctx context.Context, ref esv1beta1.ExternalSecretFind) (map[string][]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (c client) Close(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

type provider struct {
}

func (p provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	//TODO implement me
	panic("implement me")
}

func (p provider) ValidateStore(store esv1beta1.GenericStore) error {
	//TODO implement me
	panic("implement me")
}
