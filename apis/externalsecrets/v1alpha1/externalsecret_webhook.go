//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

func (alpha *ExternalSecret) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(alpha).
		Complete()
}
