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

package onepasswordsdk

import (
	"context"
	"errors"
	"fmt"

	"github.com/1password/onepassword-sdk-go"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

const (
	errOnePasswordSdkStore                              = "received invalid 1PasswordSdk SecretStore resource: %w"
	errOnePasswordSdkStoreNilSpec                       = "nil spec"
	errOnePasswordSdkStoreNilSpecProvider               = "nil spec.provider"
	errOnePasswordSdkStoreNilSpecProviderOnePasswordSdk = "nil spec.provider.onepasswordsdk"
	errOnePasswordSdkStoreMissingRefName                = "missing: spec.provider.onepasswordsdk.auth.secretRef.serviceAccountTokenSecretRef.name"
	errOnePasswordSdkStoreMissingRefKey                 = "missing: spec.provider.onepasswordsdk.auth.secretRef.serviceAccountTokenSecretRef.key"
	errVersionNotImplemented                            = "'remoteRef.version' is not implemented in the 1Password SDK provider"
	errNotImplemented                                   = "not implemented"
)

type Provider struct {
	client *onepassword.Client
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config := store.GetSpec().Provider.OnePasswordSDK
	serviceAccountToken, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&config.Auth.ServiceAccountSecretRef,
	)
	if err != nil {
		return nil, err
	}
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithServiceAccountToken(serviceAccountToken),
		// TODO: Set the following to your own integration name and version.
		onepassword.WithIntegrationInfo("My 1Password Integration", "v1.0.0"),
	)
	if err != nil {
		return nil, err
	}
	p.client = client

	return p, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpec))
	}
	if storeSpec.Provider == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpecProvider))
	}
	if storeSpec.Provider.OnePasswordSDK == nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreNilSpecProviderOnePasswordSdk))
	}

	config := storeSpec.Provider.OnePasswordSDK
	if config.Auth.ServiceAccountSecretRef.Name == "" {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreMissingRefName))
	}
	if config.Auth.ServiceAccountSecretRef.Key == "" {
		return nil, fmt.Errorf(errOnePasswordSdkStore, errors.New(errOnePasswordSdkStoreMissingRefKey))
	}

	// check namespace compared to kind
	if err := utils.ValidateSecretSelector(store, config.Auth.ServiceAccountSecretRef); err != nil {
		return nil, fmt.Errorf(errOnePasswordSdkStore, err)
	}

	return nil, nil
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		OnePasswordSDK: &esv1beta1.OnePasswordSDKProvider{},
	})
}
