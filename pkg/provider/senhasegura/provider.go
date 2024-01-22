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

package senhasegura

import (
	"context"
	"fmt"
	"net/url"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	senhaseguraAuth "github.com/external-secrets/external-secrets/pkg/provider/senhasegura/auth"
	"github.com/external-secrets/external-secrets/pkg/provider/senhasegura/dsm"
)

// https://github.com/external-secrets/external-secrets/issues/644
var _ esv1beta1.Provider = &Provider{}

// Provider struct that satisfier ESO interface.
type Provider struct{}

const (
	errUnknownProviderService     = "unknown senhasegura Provider Service: %s"
	errNilStore                   = "nil store found"
	errMissingStoreSpec           = "store is missing spec"
	errMissingProvider            = "storeSpec is missing provider"
	errInvalidProvider            = "invalid provider spec. Missing senhasegura field in store %s"
	errInvalidSenhaseguraURL      = "invalid senhasegura URL"
	errInvalidSenhaseguraURLHTTPS = "invalid senhasegura URL, must be HTTPS for security reasons"
	errMissingClientID            = "missing senhasegura authentication Client ID"
)

// Capabilities return the provider supported capabilities (ReadOnly, WriteOnly, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadOnly
}

/*
Construct a new secrets client based on provided store.
*/
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	spec := store.GetSpec()
	provider := spec.Provider.Senhasegura

	isoSession, err := senhaseguraAuth.Authenticate(ctx, store, provider, kube, namespace)
	if err != nil {
		return nil, err
	}

	if provider.Module == esv1beta1.SenhaseguraModuleDSM {
		return dsm.New(isoSession)
	}

	return nil, fmt.Errorf(errUnknownProviderService, provider.Module)
}

// Validate store using Validating webhook during secret store creating
// Checks here are usually the best experience for the user, as the SecretStore will not be created until it is a 'valid' one.
// https://github.com/external-secrets/external-secrets/pull/830#discussion_r833278518
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	return nil, validateStore(store)
}

func validateStore(store esv1beta1.GenericStore) error {
	if store == nil {
		return fmt.Errorf(errNilStore)
	}

	spec := store.GetSpec()
	if spec == nil {
		return fmt.Errorf(errMissingStoreSpec)
	}

	if spec.Provider == nil {
		return fmt.Errorf(errMissingProvider)
	}

	provider := spec.Provider.Senhasegura
	if provider == nil {
		return fmt.Errorf(errInvalidProvider, store.GetObjectMeta().String())
	}

	url, err := url.Parse(provider.URL)
	if err != nil {
		return fmt.Errorf(errInvalidSenhaseguraURL)
	}

	// senhasegura doesn't accept requests without SSL/TLS layer for security reasons
	// DSM doesn't provides gRPC schema, only HTTPS
	if url.Scheme != "https" {
		return fmt.Errorf(errInvalidSenhaseguraURLHTTPS)
	}

	if url.Host == "" {
		return fmt.Errorf(errInvalidSenhaseguraURL)
	}

	if provider.Auth.ClientID == "" {
		return fmt.Errorf(errMissingClientID)
	}

	return nil
}

/*
Register SenhaseguraProvider in ESO init.
*/
func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{
		Senhasegura: &esv1beta1.SenhaseguraProvider{},
	})
}
