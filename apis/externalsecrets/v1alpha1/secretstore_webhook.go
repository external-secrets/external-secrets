//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func (c *SecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}

func (c *ClusterSecretStore) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(c).
		Complete()
}
