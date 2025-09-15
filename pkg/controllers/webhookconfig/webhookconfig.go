/*
Copyright © 2025 ESO Maintainer Team

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

// Package webhookconfig contains the controller for the WebhookConfig resource.
package webhookconfig

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/external-secrets/external-secrets/pkg/esutils"
	"github.com/go-logr/logr"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/pkg/constants"
)

// Reconciler reconciles a ValidatingWebhookConfiguration object
// and updates it with the CA bundle from the given secret.
type Reconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueDuration time.Duration
	SvcName         string
	SvcNamespace    string
	SecretName      string
	SecretNamespace string

	// store state for the readiness probe.
	// we're ready when we're not the leader or
	// if we've reconciled the webhook config when we're the leader.
	leaderChan     <-chan struct{}
	leaderElected  bool
	webhookReadyMu *sync.Mutex
	webhookReady   bool
}

// Opts are the options for the webhookconfig controller Reconciler.
type Opts struct {
	SvcName         string
	SvcNamespace    string
	SecretName      string
	SecretNamespace string
	RequeueInterval time.Duration
}

// New returns a new Reconciler.
// The controller will watch ValidatingWebhookConfiguration resources
// and update them with the CA bundle from the given secret.
func New(k8sClient client.Client, scheme *runtime.Scheme, leaderChan <-chan struct{}, log logr.Logger, opts Opts) *Reconciler {
	return &Reconciler{
		Client:          k8sClient,
		Scheme:          scheme,
		Log:             log,
		RequeueDuration: opts.RequeueInterval,
		SvcName:         opts.SvcName,
		SvcNamespace:    opts.SvcNamespace,
		SecretName:      opts.SecretName,
		SecretNamespace: opts.SecretNamespace,
		leaderChan:      leaderChan,
		leaderElected:   false,
		webhookReadyMu:  &sync.Mutex{},
		webhookReady:    false,
	}
}

const (
	// ReasonUpdateFailed is used when we fail to update the webhook config.
	ReasonUpdateFailed = "UpdateFailed"
	errWebhookNotReady = "webhook not ready"
	errCACertNotReady  = "ca cert not yet ready"

	caCertName = "ca.crt"
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// In this case, we reconcile ValidatingWebhookConfiguration resources
// that are labeled with the well-known label key and value.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Webhookconfig", req.NamespacedName)
	var cfg admissionregistration.ValidatingWebhookConfiguration
	err := r.Get(ctx, req.NamespacedName, &cfg)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get Webhookconfig")
		return ctrl.Result{}, err
	}

	if cfg.Labels[constants.WellKnownLabelKey] != constants.WellKnownLabelValueWebhook {
		log.Info("ignoring webhook due to missing labels", constants.WellKnownLabelKey, constants.WellKnownLabelValueWebhook)
		return ctrl.Result{}, nil
	}

	log.Info("updating webhook config")
	err = r.updateConfig(logr.NewContext(ctx, log), &cfg)
	if err != nil {
		log.Error(err, "could not update webhook config")
		r.recorder.Eventf(&cfg, v1.EventTypeWarning, ReasonUpdateFailed, err.Error())
		return ctrl.Result{
			RequeueAfter: time.Minute,
		}, err
	}

	// right now we only have one single
	// webhook config we care about
	r.webhookReadyMu.Lock()
	defer r.webhookReadyMu.Unlock()
	r.webhookReady = true
	return ctrl.Result{
		RequeueAfter: r.RequeueDuration,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
// Also initializes the event recorder.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("validating-webhook-configuration")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&admissionregistration.ValidatingWebhookConfiguration{}).
		Complete(r)
}

// ReadyCheck does a readiness check for the webhook using the endpoint slices.
func (r *Reconciler) ReadyCheck(_ *http.Request) error {
	// skip readiness check if we're not leader
	// as we depend on caches and being able to reconcile Webhooks
	if !r.leaderElected {
		select {
		case <-r.leaderChan:
			r.leaderElected = true
		default:
			return nil
		}
	}
	r.webhookReadyMu.Lock()
	defer r.webhookReadyMu.Unlock()
	if !r.webhookReady {
		return errors.New(errWebhookNotReady)
	}

	return esutils.CheckEndpointSlicesReady(context.TODO(), r.Client, r.SvcName, r.SvcNamespace)
}

// reads the ca cert and updates the webhook config.
func (r *Reconciler) updateConfig(ctx context.Context, cfg *admissionregistration.ValidatingWebhookConfiguration) error {
	log := logr.FromContextOrDiscard(ctx)
	before := cfg.DeepCopyObject()

	secret := v1.Secret{}
	secretName := types.NamespacedName{
		Name:      r.SecretName,
		Namespace: r.SecretNamespace,
	}
	err := r.Get(context.Background(), secretName, &secret)
	if err != nil {
		return err
	}

	crt, ok := secret.Data[caCertName]
	if !ok {
		return errors.New(errCACertNotReady)
	}

	r.inject(cfg, r.SvcName, r.SvcNamespace, crt)

	if !equality.Semantic.DeepEqual(before, cfg) {
		if err := r.Update(ctx, cfg); err != nil {
			return err
		}
		log.Info("updated webhook config")
		return nil
	}
	log.V(1).Info("webhook config unchanged")
	return nil
}

func (r *Reconciler) inject(cfg *admissionregistration.ValidatingWebhookConfiguration, svcName, svcNamespace string, certData []byte) {
	r.Log.Info("injecting ca certificate and service names", "cacrt", base64.StdEncoding.EncodeToString(certData), "name", cfg.Name)
	for idx, w := range cfg.Webhooks {
		if !strings.HasSuffix(w.Name, "external-secrets.io") {
			r.Log.Info("skipping webhook", "name", cfg.Name, "webhook-name", w.Name)
			continue
		}
		// we just patch the relevant fields
		cfg.Webhooks[idx].ClientConfig.Service.Name = svcName
		cfg.Webhooks[idx].ClientConfig.Service.Namespace = svcNamespace
		cfg.Webhooks[idx].ClientConfig.CABundle = certData
	}
}
