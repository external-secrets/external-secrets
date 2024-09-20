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

package bitwarden

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	esmeta "github.com/external-secrets/external-secrets/apis/meta/v1"
	"github.com/external-secrets/external-secrets/pkg/utils"
	"github.com/external-secrets/external-secrets/pkg/utils/resolvers"
)

type Provider struct {
	kube               client.Client
	namespace          string
	store              esv1beta1.GenericStore
	bitwardenSdkClient Client
}

func init() {
	esv1beta1.Register(&Provider{}, &esv1beta1.SecretStoreProvider{BitwardenSecretsManager: &esv1beta1.BitwardenSecretsManagerProvider{}})
}

func (p *Provider) ApplyReferent(spec client.Object, _ esmeta.ReferentCallOrigin, _ string) (client.Object, error) {
	return spec, nil
}

func (p *Provider) Convert(_ esv1beta1.GenericStore) (client.Object, error) {
	return nil, nil
}

func (p *Provider) NewClientFromObj(_ context.Context, _ client.Object, _ client.Client, _ string) (esv1beta1.SecretsClient, error) {
	return nil, fmt.Errorf("not implemented")
}

// NewClient creates a new Bitwarden Secret Manager client.
func (p *Provider) NewClient(ctx context.Context, store esv1beta1.GenericStore, kube client.Client, namespace string) (esv1beta1.SecretsClient, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil || storeSpec.Provider == nil || storeSpec.Provider.BitwardenSecretsManager == nil {
		return nil, errors.New("no store type or wrong store type")
	}

	token, err := resolvers.SecretKeyRef(
		ctx,
		kube,
		store.GetKind(),
		namespace,
		&storeSpec.Provider.BitwardenSecretsManager.Auth.SecretRef.Credentials,
	)
	if err != nil {
		return nil, fmt.Errorf("could not resolve auth credentials: %w", err)
	}

	sdkClient, err := NewSdkClient(ctx,
		kube,
		store.GetKind(),
		namespace,
		storeSpec.Provider.BitwardenSecretsManager,
		token,
	)
	if err != nil {
		return nil, fmt.Errorf("could not create SdkClient: %w", err)
	}

	return &Provider{
		kube:               kube,
		namespace:          namespace,
		store:              store,
		bitwardenSdkClient: sdkClient,
	}, nil
}

// Capabilities returns the provider Capabilities (Read, Write, ReadWrite).
func (p *Provider) Capabilities() esv1beta1.SecretStoreCapabilities {
	return esv1beta1.SecretStoreReadWrite
}

// ValidateStore validates the store.
func (p *Provider) ValidateStore(store esv1beta1.GenericStore) (admission.Warnings, error) {
	storeSpec := store.GetSpec()
	if storeSpec == nil {
		return admission.Warnings{}, errors.New("no store type or wrong store type")
	}

	if storeSpec.Provider == nil {
		return admission.Warnings{}, errors.New("provider not configured")
	}

	bitwardenSpec := storeSpec.Provider.BitwardenSecretsManager
	if bitwardenSpec == nil {
		return admission.Warnings{}, errors.New("bitwarden spec not configured")
	}

	if bitwardenSpec.CAProvider == nil && bitwardenSpec.CABundle == "" {
		return admission.Warnings{
			"Neither CA nor CA bundle is configured; user is expected to provide certificate information via volume mount.",
		}, nil
	}

	return nil, nil
}

// newHTTPSClient creates a new HTTPS client with the given cert.
func newHTTPSClient(ctx context.Context, c client.Client, storeKind, namespace string, provider *esv1beta1.BitwardenSecretsManagerProvider) (*http.Client, error) {
	cert, err := utils.FetchCACertFromSource(ctx, utils.CreateCertOpts{
		CABundle:   []byte(provider.CABundle),
		CAProvider: provider.CAProvider,
		StoreKind:  storeKind,
		Namespace:  namespace,
		Client:     c,
	})
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert)
	if !ok {
		return nil, errors.New("failed to append caBundle")
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12},
	}

	return &http.Client{Transport: tr, Timeout: time.Second * 10}, nil
}
