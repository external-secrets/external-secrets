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
package noop

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"
	"github.com/external-secrets/external-secrets/pkg/provider/schema"
)

// Noop is a provider that does nothing.
type Noop struct{}

// New constructs a Noop Provider.
func (sm *Noop) New(ctx context.Context, store esv1alpha1.GenericStore, kube client.Client, namespace string) (provider.Provider, error) {
	return sm, nil // stub
}

// GetSecret returns a single secret from the provider.
func (sm *Noop) GetSecret(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) ([]byte, error) {
	return []byte("NOOP"), nil
}

// GetSecretMap returns multiple k/v pairs from the provider.
func (sm *Noop) GetSecretMap(ctx context.Context, ref esv1alpha1.ExternalSecretDataRemoteRef) (map[string][]byte, error) {
	return map[string][]byte{
		"noop": []byte("NOOP"),
	}, nil
}

func init() {
	schema.Register(&Noop{}, &esv1alpha1.SecretStoreProvider{
		NOOP: &esv1alpha1.NOOPProvider{},
	})
}
