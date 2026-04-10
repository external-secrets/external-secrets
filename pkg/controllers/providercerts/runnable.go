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

package providercerts

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProviderCertReconciler periodically reconciles provider certificates.
type ProviderCertReconciler struct {
	client.Client
	Log             logr.Logger
	ProviderConfig  *ProviderCertConfig
	RequeueInterval time.Duration
	CAName          string
	CAOrganization  string
	leaderElected   <-chan struct{}
}

// New creates a new ProviderCertReconciler.
func New(
	k8sClient client.Client,
	logger logr.Logger,
	config *ProviderCertConfig,
	interval time.Duration,
	leaderChan <-chan struct{},
) *ProviderCertReconciler {
	return &ProviderCertReconciler{
		Client:          k8sClient,
		Log:             logger,
		ProviderConfig:  config,
		RequeueInterval: interval,
		CAName:          "external-secrets",
		CAOrganization:  "external-secrets",
		leaderElected:   leaderChan,
	}
}

// Start implements manager.Runnable.
func (r *ProviderCertReconciler) Start(ctx context.Context) error {
	r.Log.Info("starting provider certificate reconciler")

	// Wait for leader election
	select {
	case <-ctx.Done():
		return nil
	case <-r.leaderElected:
		r.Log.Info("leader elected, starting provider certificate reconciliation")
	}

	// Run initial reconciliation
	if err := r.ReconcileProviderCert(ctx, r.ProviderConfig); err != nil {
		r.Log.Error(err, "failed to reconcile provider certificates")
	}

	// Start periodic reconciliation
	ticker := time.NewTicker(r.RequeueInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			r.Log.Info("stopping provider certificate reconciler")
			return nil
		case <-ticker.C:
			if err := r.ReconcileProviderCert(ctx, r.ProviderConfig); err != nil {
				r.Log.Error(err, "failed to reconcile provider certificates")
			}
		}
	}
}
