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

package webhookconfig

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	admissionregistration "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/external-secrets/external-secrets/pkg/constants"
)

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

func New(k8sClient client.Client, scheme *runtime.Scheme, leaderChan <-chan struct{},
	log logr.Logger, svcName, svcNamespace, secretName, secretNamespace string,
	requeueInterval time.Duration) *Reconciler {
	return &Reconciler{
		Client:          k8sClient,
		Scheme:          scheme,
		Log:             log,
		RequeueDuration: requeueInterval,
		SvcName:         svcName,
		SvcNamespace:    svcNamespace,
		SecretName:      secretName,
		SecretNamespace: secretNamespace,
		leaderChan:      leaderChan,
		leaderElected:   false,
		webhookReadyMu:  &sync.Mutex{},
		webhookReady:    false,
	}
}

const (
	ReasonUpdateFailed   = "UpdateFailed"
	errWebhookNotReady   = "webhook not ready"
	errSubsetsNotReady   = "subsets not ready"
	errAddressesNotReady = "addresses not ready"
	errCACertNotReady    = "ca cert not yet ready"

	caCertName = "ca.crt"
)

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
	err = r.updateConfig(ctx, &cfg)
	if err != nil {
		log.Error(err, "could not update webhook config")
		r.recorder.Eventf(&cfg, v1.EventTypeWarning, ReasonUpdateFailed, err.Error())
		return ctrl.Result{
			RequeueAfter: time.Minute,
		}, err
	}
	log.Info("updated webhook config")

	// right now we only have one single
	// webhook config we care about
	r.webhookReadyMu.Lock()
	defer r.webhookReadyMu.Unlock()
	r.webhookReady = true
	return ctrl.Result{
		RequeueAfter: r.RequeueDuration,
	}, nil
}

func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("validating-webhook-configuration")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(&admissionregistration.ValidatingWebhookConfiguration{}).
		Complete(r)
}

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
		return fmt.Errorf(errWebhookNotReady)
	}
	var eps v1.Endpoints
	err := r.Get(context.TODO(), types.NamespacedName{
		Name:      r.SvcName,
		Namespace: r.SvcNamespace,
	}, &eps)
	if err != nil {
		return err
	}
	if len(eps.Subsets) == 0 {
		return fmt.Errorf(errSubsetsNotReady)
	}
	if len(eps.Subsets[0].Addresses) == 0 {
		return fmt.Errorf(errAddressesNotReady)
	}
	return nil
}

// reads the ca cert and updates the webhook config.
func (r *Reconciler) updateConfig(ctx context.Context, cfg *admissionregistration.ValidatingWebhookConfiguration) error {
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
		return fmt.Errorf(errCACertNotReady)
	}
	if err := r.inject(cfg, r.SvcName, r.SvcNamespace, crt); err != nil {
		return err
	}
	return r.Update(ctx, cfg)
}

func (r *Reconciler) inject(cfg *admissionregistration.ValidatingWebhookConfiguration, svcName, svcNamespace string, certData []byte) error {
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
	return nil
}
