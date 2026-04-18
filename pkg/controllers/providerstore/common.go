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

package providerstore

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	esv2alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v2alpha1"
	"github.com/external-secrets/external-secrets/runtime/clientmanager"
)

func setReadyCondition(store esv2alpha1.GenericStore, status corev1.ConditionStatus, reason, message string) {
	condition := esv2alpha1.ProviderStoreCondition{
		Type:               esv2alpha1.ProviderStoreReady,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}

	current := store.GetStoreStatus()
	for i := range current.Conditions {
		if current.Conditions[i].Type != condition.Type {
			continue
		}
		if current.Conditions[i].Status == condition.Status {
			condition.LastTransitionTime = current.Conditions[i].LastTransitionTime
		}
		current.Conditions[i] = condition
		store.SetStoreStatus(current)
		return
	}

	current.Conditions = append(current.Conditions, condition)
	store.SetStoreStatus(current)
}

func validateStore(ctx context.Context, kubeClient client.Client, store esv2alpha1.GenericStore, sourceNamespace string) error {
	mgr := clientmanager.NewManager(kubeClient, "", false)
	defer func() {
		_ = mgr.Close(ctx)
	}()

	secretClient, err := mgr.Get(ctx, esv1.SecretStoreRef{
		Name: store.GetName(),
		Kind: store.GetKind(),
	}, sourceNamespace, nil)
	if err != nil {
		return err
	}

	_, err = secretClient.Validate()
	return err
}

func assertRuntimeClassReady(ctx context.Context, kubeClient client.Client, runtimeRef esv2alpha1.StoreRuntimeRef) error {
	runtimeKind := runtimeRef.Kind
	if runtimeKind == "" {
		runtimeKind = "ClusterProviderClass"
	}
	if runtimeKind != "ClusterProviderClass" {
		return fmt.Errorf("unsupported runtimeRef kind %q", runtimeKind)
	}

	var runtimeClass esv1alpha1.ClusterProviderClass
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: runtimeRef.Name}, &runtimeClass); err != nil {
		return fmt.Errorf("failed to get ClusterProviderClass %q: %w", runtimeRef.Name, err)
	}

	condition := meta.FindStatusCondition(runtimeClass.Status.Conditions, "Ready")
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return fmt.Errorf("ClusterProviderClass %q is not ready", runtimeRef.Name)
	}

	return nil
}

func updateStatus(ctx context.Context, statusWriter client.StatusWriter, store esv2alpha1.GenericStore, requeueAfter time.Duration, log logr.Logger) (ctrl.Result, error) {
	if err := statusWriter.Update(ctx, store); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: requeueAfter}, nil
}
