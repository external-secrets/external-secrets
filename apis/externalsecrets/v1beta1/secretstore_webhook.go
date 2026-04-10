/*
Copyright © The ESO Authors

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

package v1beta1

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupWebhookWithManager configures the webhook manager for the SecretStore.
func (c *SecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, c).
		WithDefaulter(&secretStoreDefaulter{}).
		WithValidator(&GenericStoreValidator{}).
		Complete()
}

// SetupWebhookWithManager configures the webhook manager for the ClusterSecretStore.
func (c *ClusterSecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, c).
		WithDefaulter(&clusterSecretStoreDefaulter{}).
		WithValidator(&GenericClusterStoreValidator{}).
		Complete()
}

type secretStoreDefaulter struct{}

func (d *secretStoreDefaulter) Default(_ context.Context, store *SecretStore) error {
	if store.Spec.RuntimeRef != nil && store.Spec.RuntimeRef.Kind == "" {
		store.Spec.RuntimeRef.Kind = StoreRuntimeRefKindProviderClass
	}
	return nil
}

type clusterSecretStoreDefaulter struct{}

func (d *clusterSecretStoreDefaulter) Default(_ context.Context, store *ClusterSecretStore) error {
	if store.Spec.RuntimeRef != nil && store.Spec.RuntimeRef.Kind == "" {
		store.Spec.RuntimeRef.Kind = StoreRuntimeRefKindClusterProviderClass
	}
	return nil
}
