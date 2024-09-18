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

package lockbox

import (
	"context"
	"errors"
	"time"

	"github.com/yandex-cloud/go-sdk/iamkey"
	ctrl "sigs.k8s.io/controller-runtime"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/common/clock"
	"github.com/external-secrets/external-secrets/pkg/provider/yandex/lockbox/client"
)

var log = ctrl.Log.WithName("provider").WithName("yandex").WithName("lockbox")

func adaptInput(store esv1beta1.GenericStore) (*common.SecretsClientInput, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.YandexLockbox == nil {
		return nil, errors.New("received invalid Yandex Lockbox SecretStore resource")
	}
	storeSpecYandexLockbox := storeSpec.Provider.YandexLockbox

	if storeSpecYandexLockbox.Auth.AuthorizedKey.Name == "" {
		return nil, errors.New("invalid Yandex Lockbox SecretStore resource: missing AuthorizedKey Name")
	}

	var caCertificate *esmeta.SecretKeySelector
	if storeSpecYandexLockbox.CAProvider != nil {
		caCertificate = &storeSpecYandexLockbox.CAProvider.Certificate
	}

	return &common.SecretsClientInput{
		APIEndpoint:   storeSpecYandexLockbox.APIEndpoint,
		AuthorizedKey: storeSpecYandexLockbox.Auth.AuthorizedKey,
		CACertificate: caCertificate,
	}, nil
}

func newSecretGetter(ctx context.Context, apiEndpoint string, authorizedKey *iamkey.Key, caCertificate []byte) (common.SecretGetter, error) {
	lockboxClient, err := client.NewGrpcLockboxClient(ctx, apiEndpoint, authorizedKey, caCertificate)
	if err != nil {
		return nil, err
	}
	return newLockboxSecretGetter(lockboxClient)
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
			YandexLockbox: &esv1beta1.YandexLockboxProvider{},
		},
	)
}
