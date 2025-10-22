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

// Package lockbox implements a Yandex Lockbox provider for External Secrets.
package lockbox

import (
	"context"
	"errors"
	"time"

	"github.com/yandex-cloud/go-sdk/iamkey"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

var log = ctrl.Log.WithName("provider").WithName("yandex").WithName("lockbox")

func adaptInput(store esv1.GenericStore) (*ydxcommon.SecretsClientInput, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexLockbox == nil {
		return nil, errors.New("received invalid Yandex Lockbox SecretStore resource")
	}
	storeSpecYandexLockbox := storeSpec.Provider.YandexLockbox

	var authorizedKey *esmeta.SecretKeySelector
	if storeSpecYandexLockbox.Auth.AuthorizedKey.Name != "" {
		authorizedKey = &storeSpecYandexLockbox.Auth.AuthorizedKey
	}

	var caCertificate *esmeta.SecretKeySelector
	if storeSpecYandexLockbox.CAProvider != nil {
		caCertificate = &storeSpecYandexLockbox.CAProvider.Certificate
	}

	var resourceKeyType ydxcommon.ResourceKeyType
	var folderID string
	policy := storeSpecYandexLockbox.FetchingPolicy
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
			return nil, errors.New("invalid Yandex Lockbox SecretStore: requires either 'byName' or 'byID' policy")
		}
	}

	return &ydxcommon.SecretsClientInput{
		APIEndpoint:     storeSpecYandexLockbox.APIEndpoint,
		AuthorizedKey:   authorizedKey,
		CACertificate:   caCertificate,
		ResourceKeyType: resourceKeyType,
		FolderID:        folderID,
	}, nil
}

func newSecretGetter(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (ydxcommon.SecretGetter, error) {
	lockboxClient, err := client.NewGrpcLockboxClient(ctx, apiEndpoint, authorizedKey, caCertificate)
	if err != nil {
		return nil, err
	}
	return newLockboxSecretGetter(lockboxClient)
}

func init() {
	provider := ydxcommon.InitYandexCloudProvider(
		log,
		clock.NewRealClock(),
		adaptInput,
		newSecretGetter,
		ydxcommon.NewIamToken,
		time.Hour,
	)

	esv1.Register(
		provider,
		&esv1.SecretStoreProvider{
			YandexLockbox: &esv1.YandexLockboxProvider{},
		},
		esv1.MaintenanceStatusMaintained,
	)
}
