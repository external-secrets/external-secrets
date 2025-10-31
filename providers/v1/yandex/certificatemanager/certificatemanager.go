/*
Copyright © 2025 ESO Maintainer Team

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package certificatemanager implements the Yandex Cloud Certificate Manager provider for External Secrets Operator.
package certificatemanager

import (
	"context"
	"errors"
	"time"

	"github.com/yandex-cloud/go-sdk/iamkey"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/certificatemanager/client"
	ydxcommon "github.com/external-secrets/external-secrets/providers/v1/yandex/common"
	"github.com/external-secrets/external-secrets/providers/v1/yandex/common/clock"
)

var log = ctrl.Log.WithName("provider").WithName("yandex").WithName("certificatemanager")

func adaptInput(store esv1.GenericStore) (*ydxcommon.SecretsClientInput, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexCertificateManager == nil {
		return nil, errors.New("received invalid Yandex Certificate Manager SecretStore resource")
	}
	storeSpecYandexCertificateManager := storeSpec.Provider.YandexCertificateManager

	var authorizedKey *esmeta.SecretKeySelector
	if storeSpecYandexCertificateManager.Auth.AuthorizedKey.Name != "" {
		authorizedKey = &storeSpecYandexCertificateManager.Auth.AuthorizedKey
	}

	var caCertificate *esmeta.SecretKeySelector
	if storeSpecYandexCertificateManager.CAProvider != nil {
		caCertificate = &storeSpecYandexCertificateManager.CAProvider.Certificate
	}

	var resourceKeyType ydxcommon.ResourceKeyType
	var folderID string
	policy := storeSpecYandexCertificateManager.FetchingPolicy
	if policy != nil {
		switch {
		case policy.ByName != nil:
			if policy.ByName.FolderID == "" {
				return nil, errors.New("folderID is required when fetching policy is 'byName'")
			}
			resourceKeyType = ydxcommon.ResourceKeyTypeName
			folderID = policy.ByName.FolderID

		case policy.ByID != nil:
			resourceKeyType = ydxcommon.ResourceKeyTypeID

		default:
			return nil, errors.New("invalid Yandex Certificate Manager SecretStore: requires either 'byName' or 'byID' policy")
		}
	}

	return &ydxcommon.SecretsClientInput{
		APIEndpoint:     storeSpecYandexCertificateManager.APIEndpoint,
		AuthorizedKey:   authorizedKey,
		CACertificate:   caCertificate,
		ResourceKeyType: resourceKeyType,
		FolderID:        folderID,
	}, nil
}

func newSecretGetter(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (ydxcommon.SecretGetter, error) {
	grpcClient, err := client.NewGrpcCertificateManagerClient(ctx, apiEndpoint, authorizedKey, caCertificate)
	if err != nil {
		return nil, err
	}
	return newCertificateManagerSecretGetter(grpcClient)
}

// NewProvider creates a new Provider instance.
func NewProvider() esv1.Provider {
	return ydxcommon.InitYandexCloudProvider(
		log,
		clock.NewRealClock(),
		adaptInput,
		newSecretGetter,
		ydxcommon.NewIamToken,
		time.Hour,
	)
}

// ProviderSpec returns the provider specification for registration.
func ProviderSpec() *esv1.SecretStoreProvider {
	return &esv1.SecretStoreProvider{
		YandexCertificateManager: &esv1.YandexCertificateManagerProvider{},
	}
}

// MaintenanceStatus returns the maintenance status of the provider.
func MaintenanceStatus() esv1.MaintenanceStatus {
	return esv1.MaintenanceStatusMaintained
}
