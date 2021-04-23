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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	esv1alpha1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/provider"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
	schema "github.com/external-secrets/external-secrets/pkg/provider/schema"
	"github.com/external-secrets/external-secrets/pkg/template"
	utils "github.com/external-secrets/external-secrets/pkg/utils"
)

const (
	requeueAfter = time.Second * 30
)

// Reconciler reconciles a ExternalSecret object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	ControllerClass string
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ExternalSecret", req.NamespacedName)

	syncCallsMetricLabels := prometheus.Labels{"name": req.Name, "namespace": req.Namespace}

	var externalSecret esv1alpha1.ExternalSecret

	err := r.Get(ctx, req.NamespacedName, &externalSecret)
	if err != nil {
		log.Error(err, "could not get ExternalSecret")
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	store, err := r.getStore(ctx, &externalSecret)
	if err != nil {
		log.Error(err, "could not get store reference")
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)

		err = r.Status().Update(ctx, &externalSecret)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	log = log.WithValues("SecretStore", store.GetNamespacedName())

	// check if store should be handled by this controller instance
	if !shouldProcessStore(store, r.ControllerClass) {
		log.Info("skippig unmanaged store")
		return ctrl.Result{}, nil
	}

	storeProvider, err := schema.GetProvider(store)
	if err != nil {
		log.Error(err, "could not get store provider")
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	secretClient, err := storeProvider.NewClient(ctx, store, r.Client, req.Namespace)
	if err != nil {
		log.Error(err, "could not get provider client")
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)
		err = r.Status().Update(ctx, &externalSecret)
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}
	secret := defaultSecret(externalSecret)
	_, err = ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
		err = controllerutil.SetControllerReference(&externalSecret, &secret.ObjectMeta, r.Scheme)
		if err != nil {
			return fmt.Errorf("could not set ExternalSecret controller reference: %w", err)
		}
		data, err := r.getProviderSecretData(ctx, secretClient, &externalSecret)
		if err != nil {
			return fmt.Errorf("could not get secret data from provider: %w", err)
		}
		// overwrite data
		for k, v := range data {
			secret.Data[k] = v
		}
		err = template.Execute(secret, data)
		if err != nil {
			return fmt.Errorf("could not execute template: %w", err)
		}
		return nil
	})

	if err != nil {
		log.Error(err, "could not reconcile ExternalSecret")
		conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
		SetExternalSecretCondition(&externalSecret, *conditionSynced)
		err = r.Status().Update(ctx, &externalSecret)
		if err != nil {
			log.Error(err, "unable to update status")
		}
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{RequeueAfter: requeueAfter}, nil
	}

	dur := time.Hour
	if externalSecret.Spec.RefreshInterval != nil {
		dur = externalSecret.Spec.RefreshInterval.Duration
	}

	conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionTrue, esv1alpha1.ConditionReasonSecretSynced, "Secret was synced")
	SetExternalSecretCondition(&externalSecret, *conditionSynced)
	externalSecret.Status.RefreshTime = metav1.NewTime(time.Now())
	err = r.Status().Update(ctx, &externalSecret)
	if err != nil {
		log.Error(err, "unable to update status")
	}

	syncCallsTotal.With(syncCallsMetricLabels).Inc()

	return ctrl.Result{
		RequeueAfter: dur,
	}, nil
}

func shouldProcessStore(store esv1alpha1.GenericStore, class string) bool {
	if store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}
	return false
}

func defaultSecret(es esv1alpha1.ExternalSecret) *corev1.Secret {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        es.Spec.Target.Name,
			Namespace:   es.Namespace,
			Labels:      es.Labels,
			Annotations: es.Annotations,
		},
		Data: make(map[string][]byte),
	}

	if es.Spec.Target.Template != nil {
		secret.Type = es.Spec.Target.Template.Type
		secret.Data = es.Spec.Target.Template.Data
		secret.ObjectMeta.Labels = es.Spec.Target.Template.Metadata.Labels
		secret.ObjectMeta.Annotations = es.Spec.Target.Template.Metadata.Annotations
	}

	return secret
}

func (r *Reconciler) getStore(ctx context.Context, externalSecret *esv1alpha1.ExternalSecret) (esv1alpha1.GenericStore, error) {
	// TODO: Implement getting ClusterSecretStore
	var secretStore esv1alpha1.SecretStore

	ref := types.NamespacedName{
		Name:      externalSecret.Spec.SecretStoreRef.Name,
		Namespace: externalSecret.Namespace,
	}

	err := r.Get(ctx, ref, &secretStore)
	if err != nil {
		return nil, fmt.Errorf("could not get SecretStore %q, %w", ref.Name, err)
	}

	return &secretStore, nil
}

func (r *Reconciler) getProviderSecretData(ctx context.Context, providerClient provider.SecretsClient, externalSecret *esv1alpha1.ExternalSecret) (map[string][]byte, error) {
	providerData := make(map[string][]byte)

	for _, remoteRef := range externalSecret.Spec.DataFrom {
		secretMap, err := providerClient.GetSecretMap(ctx, remoteRef)
		if err != nil {
			return nil, fmt.Errorf("key %q from ExternalSecret %q: %w", remoteRef.Key, externalSecret.Name, err)
		}

		providerData = utils.Merge(providerData, secretMap)
	}

	for _, secretRef := range externalSecret.Spec.Data {
		secretData, err := providerClient.GetSecret(ctx, secretRef.RemoteRef)
		if err != nil {
			return nil, fmt.Errorf("key %q from ExternalSecret %q: %w", secretRef.RemoteRef.Key, externalSecret.Name, err)
		}

		providerData[secretRef.SecretKey] = secretData
	}

	return providerData, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&esv1alpha1.ExternalSecret{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}
