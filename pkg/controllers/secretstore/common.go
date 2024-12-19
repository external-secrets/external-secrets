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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore/metrics"
)

const (
	// condition messages for "Valid" reason.
	msgValid = "store validated"

	// log messages.
	logErrorUpdateStatus = "unable to update %s status"

	// error formats.
	errStoreClient      = "could not get provider client: %w"
	errValidationFailed = "validation failed: %w"
)

type Opts struct {
	ControllerClass string
	GaugeVecGetter  metrics.GaugeVevGetter
	Recorder        record.EventRecorder
	RequeueInterval time.Duration
}

func reconcile(ctx context.Context, req ctrl.Request, ss esapi.GenericStore, cl client.Client, log logr.Logger, opts Opts) (result ctrl.Result, err error) {
	if !ShouldProcessStore(ss, opts.ControllerClass) {
		log.V(1).Info("skip store")
		return ctrl.Result{}, nil
	}

	requeueInterval := opts.RequeueInterval

	if ss.GetSpec().RefreshInterval != 0 {
		requeueInterval = time.Second * time.Duration(ss.GetSpec().RefreshInterval)
	}

	// patch status when done processing
	// update status of the SecretStore when this function returns, if needed.
	// NOTE: we use the ability of deferred functions to update named return values `result` and `err`
	// NOTE: we dereference the DeepCopy of the status field because status fields are NOT pointers,
	//       so otherwise the `equality.Semantic.DeepEqual` will always return false.
	ssStatus := ss.GetStatus()
	currentStatus := *ssStatus.DeepCopy()
	defer func() {
		// if the status has not changed, we don't need to update it
		// WARNING: be VERY careful to ensure that you haven't set empty `omitempty` fields to their empty values,
		//          as when we get the SecretStore from the API server, these fields will be seen as different
		//          from your true empty values, and the status will be updated every time.
		if equality.Semantic.DeepEqual(currentStatus, ss.GetStatus()) {
			return
		}

		// update the status of the SecretStore, storing any error in a new variable
		// if there was no new error, we don't need to change the `result` or `err` values
		updateErr := cl.Status().Update(ctx, ss)
		if updateErr == nil {
			return
		}

		// if we got an update conflict, we should requeue immediately
		if apierrors.IsConflict(updateErr) {
			log.V(1).Info("conflict while updating status, will requeue")

			// we only explicitly request a requeue if the main function did not return an `err`.
			// otherwise, we get an annoying log saying that results are ignored when there is an error,
			// as errors are always retried.
			if err == nil {
				result = ctrl.Result{Requeue: true}
			}
			return
		}

		// for other errors, log and update the `err` variable if there is no error already
		// so the reconciler will requeue the request
		log.Error(updateErr, logErrorUpdateStatus, ss.GetKind())
		if err == nil {
			err = updateErr
		}
	}()

	// validate the store, updating the status and returning an error if the store is invalid
	err = validateStore(ctx, req.Namespace, opts.ControllerClass, ss, cl, opts.GaugeVecGetter)
	if err != nil {
		log.Error(err, "store validation failed")
		return ctrl.Result{}, err
	}

	// update the capabilities of the store
	storeProvider, err := esapi.GetProvider(ss)
	if err != nil {
		return ctrl.Result{}, err
	}
	ssStatus = ss.GetStatus()
	ssStatus.Capabilities = storeProvider.Capabilities()
	ss.SetStatus(ssStatus)

	// update the status of the SecretStore to indicate that it is valid
	cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionTrue, esapi.ReasonStoreValid, msgValid)
	SetExternalSecretCondition(ss, *cond, opts.GaugeVecGetter)

	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// validateStore tries to construct a new client
// if it fails sets a condition and writes events.
func validateStore(ctx context.Context, namespace, controllerClass string, store esapi.GenericStore, client client.Client, gaugeVecGetter metrics.GaugeVevGetter) error {
	mgr := NewManager(client, controllerClass, false)
	defer mgr.Close(ctx)
	cl, err := mgr.GetFromStore(ctx, store, namespace)
	if err != nil {
		err = fmt.Errorf(errStoreClient, err)
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidProviderConfig, err.Error())
		SetExternalSecretCondition(store, *cond, gaugeVecGetter)
		return err
	}
	validationResult, err := cl.Validate()
	if err != nil && validationResult != esapi.ValidationResultUnknown {
		err = fmt.Errorf(errValidationFailed, err)
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonValidationFailed, err.Error())
		SetExternalSecretCondition(store, *cond, gaugeVecGetter)
		return err
	}

	return nil
}

// ShouldProcessStore returns true if the store should be processed.
func ShouldProcessStore(store esapi.GenericStore, class string) bool {
	if store == nil || store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}

	return false
}
