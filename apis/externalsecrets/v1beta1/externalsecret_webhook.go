//Copyright External Secrets Inc. All Rights Reserved

package v1beta1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *ExternalSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		WithValidator(&ExternalSecretValidator{}).
		Complete()
}
