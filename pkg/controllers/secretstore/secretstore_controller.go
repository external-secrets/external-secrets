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

package secretstore

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"

	// Loading registered providers.
	_ "github.com/external-secrets/external-secrets/pkg/provider/register"
)

// StoreReconciler reconciles a SecretStore object.
type StoreReconciler struct {
	client.Client
	Log             logr.Logger
	Scheme          *runtime.Scheme
	recorder        record.EventRecorder
	RequeueInterval time.Duration
	ControllerClass string
}

func (r *StoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("secretstore", req.NamespacedName)
	var ss esapi.SecretStore
	err := r.Get(ctx, req.NamespacedName, &ss)
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "unable to get SecretStore")
		return ctrl.Result{}, err
	}

	return reconcile(ctx, req, &ss, r.Client, log, r.ControllerClass, r.recorder, r.RequeueInterval)
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *StoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("secret-store")

	return ctrl.NewControllerManagedBy(mgr).
		For(&esapi.SecretStore{}).
		Complete(r)
}
