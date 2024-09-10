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
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Provider struct{}

const (
	errCannotResolveSecretKeyRef     = "cannot resolve secret key ref: %w"
	errStoreIsNil                    = "store is nil"
	errNoStoreTypeOrWrongStoreType   = "no store type or wrong store type"
	errAPIKeyIsRequired              = "apiKey is required"
	errAPIKeySecretRefIsRequired     = "apiKey.secretRef is required"
	errAPIKeySecretRefNameIsRequired = "apiKey.secretRef.name is required"
	errAPIKeySecretRefKeyIsRequired  = "apiKey.secretRef.key is required"
)

var _ esv1beta1.Provider = &Provider{}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Fortanix: &esv1beta1.FortanixProvider{},
	})
}
func (p *Provider) ApplyReferent(spec kubeclient.Object, _ esmeta.ReferentCallOrigin, _ string) (kubeclient.Object, error) {
	return spec, nil
}

func (p *Provider) Convert(_ esv1beta1.GenericStore) (kubeclient.Object, error) {
	return nil, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ kubeclient.Object, _ kubeclient.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube kubeclient.Client, namespace string) (esv1beta1.SecretsClient, error) {
	config, err := getConfig(store)
	if err != nil {
		return nil, err
	}

	apiKey, err := resolvers.SecretKeyRef(ctx, kube, store.GetKind(), namespace, config.APIKey.SecretRef)
	if err != nil {
		return nil, fmt.Errorf(errCannotResolveSecretKeyRef, err)
	}

	sdkmsClient := sdkms.Client{
		HTTPClient: http.DefaultClient,
		Auth:       sdkms.APIKey(apiKey),
		Endpoint:   config.APIURL,
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

	if config.APIURL == "" {
		config.APIURL = "https://sdkms.fortanix.com"
	}

	err := validateSecretStoreRef(store, config.APIKey)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func validateSecretStoreRef(store esv1beta1.GenericStore, ref *esv1beta1.FortanixProviderSecretRef) error {
	if ref == nil {
		return errors.New(errAPIKeyIsRequired)
	}

	if ref.SecretRef == nil {
		return errors.New(errAPIKeySecretRefIsRequired)
	}

	if ref.SecretRef.Name == "" {
		return errors.New(errAPIKeySecretRefNameIsRequired)
	}

	if ref.SecretRef.Key == "" {
		return errors.New(errAPIKeySecretRefKeyIsRequired)
	}

	return utils.ValidateReferentSecretSelector(store, *ref.SecretRef)
}
