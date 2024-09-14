//Copyright External Secrets Inc. All Rights Reserved

package certificatemanager

import (
	"context"
	"errors"
	"time"

	"github.com/yandex-cloud/go-sdk/iamkey"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/certificatemanager/client"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
)

var log = ctrl.Log.WithName("provider").WithName("yandex").WithName("certificatemanager")

func adaptInput(store esv1beta1.GenericStore) (*common.SecretsClientInput, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexCertificateManager == nil {
		return nil, errors.New("received invalid Yandex Certificate Manager SecretStore resource")
	}
	storeSpecYandexCertificateManager := storeSpec.Provider.YandexCertificateManager

	if storeSpecYandexCertificateManager.Auth.AuthorizedKey.Name == "" {
		return nil, errors.New("invalid Yandex Certificate Manager SecretStore resource: missing AuthorizedKey Name")
	}

	var caCertificate *esmeta.SecretKeySelector
	if storeSpecYandexCertificateManager.CAProvider != nil {
		caCertificate = &storeSpecYandexCertificateManager.CAProvider.Certificate
	}

	return &common.SecretsClientInput{
		APIEndpoint:   storeSpecYandexCertificateManager.APIEndpoint,
		AuthorizedKey: storeSpecYandexCertificateManager.Auth.AuthorizedKey,
		CACertificate: caCertificate,
	}, nil
}

func newSecretGetter(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (common.SecretGetter, error) {
	grpcClient, err := client.NewGrpcCertificateManagerClient(ctx, apiEndpoint, authorizedKey, caCertificate)
	if err != nil {
		return nil, err
	}
	return newCertificateManagerSecretGetter(grpcClient)
}

func init() {
	provider := common.InitYandexCloudProvider(
		log,
		clock.NewRealClock(),
		adaptInput,
		newSecretGetter,
		common.NewIamToken,
		time.Hour,
	)

	esv1beta1.Register(
		provider,
		&esv1beta1.SecretStoreProvider{
			YandexCertificateManager: &esv1beta1.YandexCertificateManagerProvider{},
		},
	)
}
