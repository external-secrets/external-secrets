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
package fortanix

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/fortanix/sdkms-client-go/sdkms"
	kubeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Provider struct{}

const (
	errCannotResolveSecretKeyRef     = "cannot resolve secret key ref: %w"
	errStoreIsNil                    = "store is nil"
	errNoStoreTypeOrWrongStoreType   = "no store type or wrong store type"
	errApiKeyIsRequired              = "apiKey is required"
	errApiKeySecretRefIsRequired     = "apiKey.secretRef is required"
	errApiKeySecretRefNameIsRequired = "apiKey.secretRef.name is required"
	errApiKeySecretRefKeyIsRequired  = "apiKey.secretRef.key is required"
)

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	apiKey, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, config.ApiKey.SecretRef)
	if err != nil {
		return nil, fmt.Errorf(errCannotResolveSecretKeyRef, err)
	}

	sdkmsClient := sdkms.Client{
		HTTPClient: http.DefaultClient,
		Auth:       sdkms.APIKey(apiKey),
		Endpoint:   config.ApiUrl,
	}

	return &client{
		sdkms: sdkmsClient,
	}, nil
}

func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	_, err := getConfig(store)
	return nil, err
}

func getConfig(store esv1beta1.GenericStore) (*esv1beta1.FortanixProvider, error) {
	if store == nil {
		return nil, errors.New(errStoreIsNil)
	}

	spec := store.GetSpec()
	if spec == nil || spec.Provider == nil || spec.Provider.Fortanix == nil {
		return nil, errors.New(errNoStoreTypeOrWrongStoreType)
	}

	config := spec.Provider.Fortanix

	if config.ApiUrl == "" {
		config.ApiUrl = "https://sdkms.fortanix.com"
	}

	err := validateSecretStoreRef(store, config.ApiKey)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func validateSecretStoreRef(store esv1beta1.GenericStore, ref *esv1beta1.FortanixProviderSecretRef) error {
	if ref == nil {
		return errors.New(errApiKeyIsRequired)
	}

	if ref.SecretRef == nil {
		return errors.New(errApiKeySecretRefIsRequired)
	}

	if ref.SecretRef.Name == "" {
		return errors.New(errApiKeySecretRefNameIsRequired)
	}

	if ref.SecretRef.Key == "" {
		return errors.New(errApiKeySecretRefKeyIsRequired)
	}

	if err := utils.ValidateReferentSecretSelector(store, *ref.SecretRef); err != nil {
		return err
	}

	return nil
}
