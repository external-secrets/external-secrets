//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func (c *SecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		WithValidator(&GenericStoreValidator{}).
		Complete()
}

func (c *ClusterSecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		WithValidator(&GenericStoreValidator{}).
		Complete()
}
