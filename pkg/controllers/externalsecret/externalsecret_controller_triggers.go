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

package externalsecret

import (
	"context"
	"encoding/json"
	"fmt"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

// secretEventHandler returns an event handler for Secret events
// WARNING: the secrets are expected to be PartialObjectMetadata, so this should only be used with WatchesMetadata().
func (r *Reconciler) secretEventHandler() handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &handler.TypedFuncs[client.Object, reconcile.Request]{
		CreateFunc:  r.secretEventHandlerCreateFunc,
		UpdateFunc:  r.secretEventHandlerUpdateFunc,
		DeleteFunc:  r.secretEventHandlerDeleteFunc,
		GenericFunc: r.secretEventHandlerGenericFunc,
	}
}

func (r *Reconciler) secretEventHandlerCreateFunc(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if r.EnableTriggerInClusterSecrets {
		r.triggerDataRefreshForTriggerSecret(ctx, e.Object)
	}

	if r.secretIsManaged(e.Object) {
		r.triggerReconcileForTargetSecret(ctx, q, e.Object)
	}
}

func (r *Reconciler) secretEventHandlerUpdateFunc(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if r.EnableTriggerInClusterSecrets {
		r.triggerDataRefreshForTriggerSecret(ctx, e.ObjectNew)
	}

	// NOTE: we need to check both the old and new object, because the update could have changed the "managed" label
	if r.secretIsManaged(e.ObjectOld) || r.secretIsManaged(e.ObjectNew) {
		r.triggerReconcileForTargetSecret(ctx, q, e.ObjectNew)
	}
}

func (r *Reconciler) secretEventHandlerDeleteFunc(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if r.EnableTriggerInClusterSecrets {
		r.triggerDataRefreshForTriggerSecret(ctx, e.Object)
	}

	if r.secretIsManaged(e.Object) {
		r.triggerReconcileForTargetSecret(ctx, q, e.Object)
	}
}

func (r *Reconciler) secretEventHandlerGenericFunc(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	if r.secretIsManaged(e.Object) {
		r.triggerReconcileForTargetSecret(ctx, q, e.Object)
	}
}

// triggerDataRefresh triggers a data refresh for an ExternalSecret by annotating the ExternalSecret.
func (r *Reconciler) triggerDataRefresh(ctx context.Context, esName, esNamespace string, triggerCause *RefreshTriggerCause) error {
	// convert the trigger cause to JSON
	triggerCauseJSON, err := json.Marshal(triggerCause)
	if err != nil {
		return fmt.Errorf("failed to marshal trigger cause: %w", err)
	}

	// trigger a data refresh by annotating the ExternalSecret
	es := &esv1beta1.ExternalSecret{}
	es.SetName(esName)
	es.SetNamespace(esNamespace)
	patch := client.MergeFrom(es.DeepCopy())
	es.SetAnnotations(map[string]string{
		esv1beta1.AnnotationRefreshTrigger: string(triggerCauseJSON),
	})
	err = r.Patch(ctx, es, patch)
	if err != nil {
		return fmt.Errorf("failed to patch ExternalSecret with refresh trigger: %w", err)
	}

	return nil
}

// triggerReconcile adds a reconcile request for an ExternalSecret to the workqueue.
func (r *Reconciler) triggerReconcile(q workqueue.TypedRateLimitingInterface[reconcile.Request], esName, esNamespace string) {
	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      esName,
			Namespace: esNamespace,
		},
	}
	q.Add(req)
}

// triggerDataRefreshForTriggerSecret finds all ExternalSecrets that reference a secret in their
// `status.triggers.inClusterSecrets` and triggers a data refresh by annotating the ExternalSecret.
func (r *Reconciler) triggerDataRefreshForTriggerSecret(ctx context.Context, obj client.Object) {
	externalSecretsList := &esv1beta1.ExternalSecretList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexESTriggerSecretsField, fmt.Sprintf("%s/%s", obj.GetNamespace(), obj.GetName())),
	}
	err := r.List(ctx, externalSecretsList, listOps)
	if err != nil {
		r.Log.Error(err, "failed to list ExternalSecrets triggered by Secret", "secretName", obj.GetName(), "secretNamespace", obj.GetNamespace())
	}

	// return early if no ExternalSecrets are triggered by this secret
	if len(externalSecretsList.Items) == 0 {
		return
	}

	// create a trigger cause for the refresh
	triggerCause := &RefreshTriggerCause{
		InClusterSecret: &RefreshTriggerInClusterSecret{
			Name:            obj.GetName(),
			Namespace:       obj.GetNamespace(),
			ResourceVersion: obj.GetResourceVersion(),
		},
	}

	// trigger a reconcile for each ExternalSecret
	for i := range externalSecretsList.Items {
		// use index to avoid copying the object
		es := externalSecretsList.Items[i]

		// don't trigger a refresh if this ExternalSecret manages the trigger secret
		// to avoid infinite loops
		esTargetSecretName := es.Spec.Target.Name
		if esTargetSecretName == "" {
			esTargetSecretName = es.GetName()
		}
		if esTargetSecretName == obj.GetName() && es.GetNamespace() == obj.GetNamespace() {
			continue
		}

		// annotate the ExternalSecret to trigger a data refresh
		err = r.triggerDataRefresh(ctx, es.GetName(), es.GetNamespace(), triggerCause)
		if err != nil {
			r.Log.Error(err, "failed to trigger data refresh for ExternalSecret", "esName", es.GetName(), "esNamespace", es.GetNamespace())
		}
	}
}

// triggerReconcileForTargetSecret find all ExternalSecrets that manage a secret, and triggers a reconcile for each.
func (r *Reconciler) triggerReconcileForTargetSecret(ctx context.Context, q workqueue.TypedRateLimitingInterface[reconcile.Request], obj client.Object) {
	// find all ExternalSecrets that manage this secret
	externalSecretsList := &esv1beta1.ExternalSecretList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(indexESTargetSecretNameField, obj.GetName()),
		Namespace:     obj.GetNamespace(),
	}
	err := r.List(ctx, externalSecretsList, listOps)
	if err != nil {
		r.Log.Error(err, "failed to list ExternalSecrets which manage Secret", "secretName", obj.GetName(), "secretNamespace", obj.GetNamespace())
	}

	// trigger a reconcile for each ExternalSecret
	for i := range externalSecretsList.Items {
		// use index to avoid copying the object
		es := externalSecretsList.Items[i]

		r.triggerReconcile(q, es.GetName(), es.GetNamespace())
	}
}

// secretIsManaged returns true if the secret is managed by an ExternalSecret.
func (r *Reconciler) secretIsManaged(obj client.Object) bool {
	value, hasLabel := obj.GetLabels()[esv1beta1.LabelManaged]
	return hasLabel && value == esv1beta1.LabelManagedValue
}

// RefreshTriggerInClusterSecret contains information about an in-cluster secret which triggered the refresh.
type RefreshTriggerInClusterSecret struct {
	// Name is the name of the secret which triggered the refresh.
	Name string `json:"name"`

	// Namespace is the namespace of the secret which triggered the refresh.
	Namespace string `json:"namespace"`

	// ResourceVersion is the resource version of the secret which triggered the refresh.
	ResourceVersion string `json:"resourceVersion"`
}

// RefreshTriggerCause indicates the cause of a data refresh trigger.
// It will be JSON serialized and stored in the ExternalSecret's `refresh-trigger` annotation.
type RefreshTriggerCause struct {
	InClusterSecret *RefreshTriggerInClusterSecret `json:"inClusterSecret,omitempty"`
}
