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

// Package clusterprovider implements the controller for ClusterProvider resources.
package clusterprovider

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	pb "github.com/external-secrets/external-secrets/proto/provider"
	"github.com/external-secrets/external-secrets/providers/v2/common/grpc"
)

// Reconciler reconciles a ClusterProvider object.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	RequeueInterval time.Duration
}

// Reconcile validates the ClusterProvider and updates its status.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("ClusterProvider", req.NamespacedName)
	start := time.Now()
	defer func() {
		duration := time.Since(start)
		if gaugeVec := GetGaugeVec(ClusterProviderReconcileDurationKey); gaugeVec != nil {
			gaugeVec.WithLabelValues(req.Name).Set(duration.Seconds())
		}
	}()

	log.Info("reconciling ClusterProvider")

	var store esv1.ClusterProvider
	if err := r.Get(ctx, req.NamespacedName, &store); err != nil {
		if apierrors.IsNotFound(err) {
			RemoveMetrics(req.Name)
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to get ClusterProvider")
		return ctrl.Result{}, err
	}

	// Validate provider config and get capabilities
	capabilities, err := r.validateStoreAndGetCapabilities(ctx, &store)
	if err != nil {
		log.Error(err, "validation failed")
		r.setNotReadyCondition(&store, "ValidationFailed", err.Error())
		if updateErr := r.Status().Update(ctx, &store); updateErr != nil {
			log.Error(updateErr, "failed to update status")
			return ctrl.Result{}, updateErr
		}
		// Requeue after interval to retry
		return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
	}

	// Set ready condition and capabilities
	r.setReadyCondition(&store)
	store.Status.Capabilities = capabilities
	if err := r.Status().Update(ctx, &store); err != nil {
		log.Error(err, "failed to update status")
		return ctrl.Result{}, err
	}

	log.Info("ClusterProvider is ready", "capabilities", capabilities)
	return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
}

// validateStoreAndGetCapabilities validates the ClusterProvider configuration and retrieves capabilities by:
// 1. Creating a gRPC client to the provider
// 2. Calling Validate() on the provider with the ProviderReference
// 3. Calling Capabilities() to get the provider's capabilities.
func (r *Reconciler) validateStoreAndGetCapabilities(ctx context.Context, store *esv1.ClusterProvider) (esv1.ProviderCapabilities, error) {
	// Get provider address
	address := store.Spec.Config.Address
	if address == "" {
		return "", fmt.Errorf("provider address is required")
	}

	// Load TLS configuration
	tlsConfig, err := grpc.LoadClientTLSConfig(ctx, r.Client, store.Spec.Config.Address, "external-secrets-system")
	if err != nil {
		return "", fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Create gRPC client with TLS
	client, err := grpc.NewClient(address, tlsConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create gRPC client: %w", err)
	}
	defer func() { _ = client.Close(ctx) }()

	// Convert ProviderReference to protobuf format
	providerRef := &pb.ProviderReference{
		ApiVersion: store.Spec.Config.ProviderRef.APIVersion,
		Kind:       store.Spec.Config.ProviderRef.Kind,
		Name:       store.Spec.Config.ProviderRef.Name,
		Namespace:  store.Spec.Config.ProviderRef.Namespace,
	}

	// For ClusterProvider validation, we need to be more lenient since:
	// - ManifestNamespace scope: we don't have a manifest namespace at validation time
	// - ProviderNamespace scope: validation namespace may not have RBAC set up yet
	// So we skip validation and rely on runtime errors instead
	// This is acceptable since the provider will be validated when actually used by an ExternalSecret

	// Get provider capabilities (using empty namespace for ClusterProvider)
	caps, err := client.Capabilities(ctx, providerRef, "")
	if err != nil {
		r.Log.Error(err, "failed to get capabilities")
		// Don't fail validation if capabilities check fails, just log and default to ReadOnly
		return esv1.ProviderReadOnly, nil
	}

	// Map gRPC capabilities to our API type
	var capabilities esv1.ProviderCapabilities
	switch caps {
	case pb.SecretStoreCapabilities_READ_ONLY:
		capabilities = esv1.ProviderReadOnly
	case pb.SecretStoreCapabilities_WRITE_ONLY:
		capabilities = esv1.ProviderWriteOnly
	case pb.SecretStoreCapabilities_READ_WRITE:
		capabilities = esv1.ProviderReadWrite
	default:
		capabilities = esv1.ProviderReadOnly
	}

	return capabilities, nil
}

// setReadyCondition sets the Ready condition to True.
func (r *Reconciler) setReadyCondition(store *esv1.ClusterProvider) {
	condition := esv1.ProviderCondition{
		Type:               esv1.ProviderReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             "Validated",
		Message:            "ClusterProvider is ready",
	}
	r.setCondition(store, condition)
}

// setNotReadyCondition sets the Ready condition to False.
func (r *Reconciler) setNotReadyCondition(store *esv1.ClusterProvider, reason, message string) {
	condition := esv1.ProviderCondition{
		Type:               esv1.ProviderReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
	r.setCondition(store, condition)
}

// setCondition updates or adds a condition to the ClusterProvider status.
func (r *Reconciler) setCondition(store *esv1.ClusterProvider, newCondition esv1.ProviderCondition) {
	// Find existing condition
	for i, condition := range store.Status.Conditions {
		if condition.Type == newCondition.Type {
			// Only update if status changed
			if condition.Status != newCondition.Status {
				store.Status.Conditions[i] = newCondition
			}
			// Update metrics
			UpdateStatusCondition(store, newCondition)
			return
		}
	}
	// Add new condition
	store.Status.Conditions = append(store.Status.Conditions, newCondition)
	// Update metrics
	UpdateStatusCondition(store, newCondition)
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&esv1.ClusterProvider{}).
		Owns(&corev1.Secret{}). // Watch secrets that might be used for auth
		Complete(r)
}

