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
	"math"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	requeueAfter = time.Second * 300
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
	if apierrors.IsNotFound(err) {
		syncCallsTotal.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "could not get ExternalSecret")
		syncCallsError.With(syncCallsMetricLabels).Inc()
		return ctrl.Result{}, nil
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
  // Check if Secret exists from from this ExternalSecret
	foundSecret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: externalSecret.Spec.Target.Name, Namespace: externalSecret.Namespace}, foundSecret)
	if err != nil {
		// Make this new Secret since it is not found.
		if apierrors.IsNotFound(err) {
			log.Info("Making new Secret")
			secretClient, err := storeProvider.NewClient(ctx, store, r.Client, req.Namespace)
			if err != nil {
				log.Error(err, "could not get provider client")
				conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
				SetExternalSecretCondition(&externalSecret, *conditionSynced)
				err = r.Status().Update(ctx, &externalSecret)
				syncCallsError.With(syncCallsMetricLabels).Inc()
				return ctrl.Result{RequeueAfter: requeueAfter}, nil
			}
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      externalSecret.Spec.Target.Name,
					Namespace: externalSecret.Namespace,
				},
				Data: make(map[string][]byte),
			}

			_, err = ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
				err = controllerutil.SetControllerReference(&externalSecret, &secret.ObjectMeta, r.Scheme)
				if err != nil {
					return fmt.Errorf("could not set ExternalSecret controller reference: %w", err)
				}
				mergeTemplate(secret, externalSecret)
				data, err := r.getProviderSecretData(ctx, secretClient, &externalSecret)
				if err != nil {
					return fmt.Errorf("could not get secret data from provider: %w", err)
				}
				// overwrite data
				for k, v := range data {
					secret.Data[k] = v
				}
				err = template.Execute(externalSecret.Spec.Target.Template, secret, data)
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
	} else {
		if externalSecret.Status.Conditions != nil {
			// Check if it is time to renew. This prevents unnecessary client token creation in secret providers.
			timeUntil := time.Until(externalSecret.Status.Conditions[0].LastTransitionTime.Time.Add(externalSecret.Spec.RefreshInterval.Duration))
			res := math.Signbit(float64(timeUntil))
			if res {
				log.Info("Time to renew. New Secret Client.")
				secretClient, err := storeProvider.NewClient(ctx, store, r.Client, req.Namespace)
				if err != nil {
					log.Error(err, "could not get provider client")
					conditionSynced := NewExternalSecretCondition(esv1alpha1.ExternalSecretReady, corev1.ConditionFalse, esv1alpha1.ConditionReasonSecretSyncedError, err.Error())
					SetExternalSecretCondition(&externalSecret, *conditionSynced)
					err = r.Status().Update(ctx, &externalSecret)
					syncCallsError.With(syncCallsMetricLabels).Inc()
					return ctrl.Result{RequeueAfter: requeueAfter}, nil
				}
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      externalSecret.Spec.Target.Name,
						Namespace: externalSecret.Namespace,
					},
					Data: make(map[string][]byte),
				}
				_, err = ctrl.CreateOrUpdate(ctx, r.Client, secret, func() error {
					err = controllerutil.SetControllerReference(&externalSecret, &secret.ObjectMeta, r.Scheme)
					if err != nil {
						return fmt.Errorf("could not set ExternalSecret controller reference: %w", err)
					}
					mergeTemplate(secret, externalSecret)
					data, err := r.getProviderSecretData(ctx, secretClient, &externalSecret)
					if err != nil {
						return fmt.Errorf("could not get secret data from provider: %w", err)
					}
					// overwrite data
					for k, v := range data {
						secret.Data[k] = v
					}
					err = template.Execute(externalSecret.Spec.Target.Template, secret, data)
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
				externalSecret.Status.Conditions[0].LastTransitionTime = metav1.NewTime(time.Now())
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
		}
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

// we do not want to force-override the label/annotations
// and only copy the necessary key/value pairs.
func mergeTemplate(secret *corev1.Secret, externalSecret esv1alpha1.ExternalSecret) {
	if secret.ObjectMeta.Labels == nil {
		secret.ObjectMeta.Labels = make(map[string]string)
	}
	if secret.ObjectMeta.Annotations == nil {
		secret.ObjectMeta.Annotations = make(map[string]string)
	}
	if externalSecret.Spec.Target.Template == nil {
		mergeMap(secret.ObjectMeta.Labels, externalSecret.ObjectMeta.Labels)
		mergeMap(secret.ObjectMeta.Annotations, externalSecret.ObjectMeta.Annotations)
		return
	}
	// if template is defined: use those labels/annotations
	secret.Type = externalSecret.Spec.Target.Template.Type
	mergeMap(secret.ObjectMeta.Labels, externalSecret.Spec.Target.Template.Metadata.Labels)
	mergeMap(secret.ObjectMeta.Annotations, externalSecret.Spec.Target.Template.Metadata.Annotations)
}

func mergeMap(dest, src map[string]string) {
	for k, v := range src {
		dest[k] = v
	}
}

func (r *Reconciler) getStore(ctx context.Context, externalSecret *esv1alpha1.ExternalSecret) (esv1alpha1.GenericStore, error) {
	ref := types.NamespacedName{
		Name: externalSecret.Spec.SecretStoreRef.Name,
	}

	if externalSecret.Spec.SecretStoreRef.Kind == esv1alpha1.ClusterSecretStoreKind {
		var store esv1alpha1.ClusterSecretStore
		err := r.Get(ctx, ref, &store)
		if err != nil {
			return nil, fmt.Errorf("could not get ClusterSecretStore %q, %w", ref.Name, err)
		}

		return &store, nil
	}

	ref.Namespace = externalSecret.Namespace

	var store esv1alpha1.SecretStore
	err := r.Get(ctx, ref, &store)
	if err != nil {
		return nil, fmt.Errorf("could not get SecretStore %q, %w", ref.Name, err)
	}
	return &store, nil
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

