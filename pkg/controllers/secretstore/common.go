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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esapi "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

const (
	errStoreProvider       = "could not get store provider: %w"
	errStoreClient         = "could not get provider client: %w"
	errValidationFailed    = "could not validate provider: %w"
	errPatchStatus         = "unable to patch status: %w"
	errUnableCreateClient  = "unable to create client"
	errUnableValidateStore = "unable to validate store"
	errUnableGetProvider   = "unable to get store provider"

	msgStoreValidated = "store validated"
)

func reconcile(ctx context.Context, req ctrl.Request, ss esapi.GenericStore, cl client.Client,
	log logr.Logger, controllerClass string, recorder record.EventRecorder, requeueInterval time.Duration) (ctrl.Result, error) {
	if !ShouldProcessStore(ss, controllerClass) {
		log.V(1).Info("skip store")
		return ctrl.Result{}, nil
	}

	if ss.GetSpec().RefreshInterval != 0 {
		requeueInterval = time.Second * time.Duration(ss.GetSpec().RefreshInterval)
	}

	// patch status when done processing
	p := client.MergeFrom(ss.Copy())
	defer func() {
		err := cl.Status().Patch(ctx, ss, p)
		if err != nil {
			log.Error(err, errPatchStatus)
		}
	}()

	// validateStore modifies the store conditions
	// we have to patch the status
	log.V(1).Info("validating")
	err := validateStore(ctx, req.Namespace, ss, cl, recorder)
	if err != nil {
		return ctrl.Result{}, err
	}

	recorder.Event(ss, v1.EventTypeNormal, esapi.ReasonStoreValid, msgStoreValidated)
	cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionTrue, esapi.ReasonStoreValid, msgStoreValidated)
	SetExternalSecretCondition(ss, *cond)

	return ctrl.Result{
		RequeueAfter: requeueInterval,
	}, err
}

// validateStore tries to construct a new client
// if it fails sets a condition and writes events.
func validateStore(ctx context.Context, namespace string, store esapi.GenericStore,
	client client.Client, recorder record.EventRecorder) error {
	storeProvider, err := esapi.GetProvider(store)
	if err != nil {
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidStore, errUnableGetProvider)
		SetExternalSecretCondition(store, *cond)
		recorder.Event(store, v1.EventTypeWarning, esapi.ReasonInvalidStore, err.Error())
		return fmt.Errorf(errStoreProvider, err)
	}

	cl, err := storeProvider.NewClient(ctx, store, client, namespace)
	if err != nil {
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonInvalidProviderConfig, errUnableCreateClient)
		SetExternalSecretCondition(store, *cond)
		recorder.Event(store, v1.EventTypeWarning, esapi.ReasonInvalidProviderConfig, err.Error())
		return fmt.Errorf(errStoreClient, err)
	}
	defer cl.Close(ctx)

	validationResult, err := cl.Validate()
	if err != nil && validationResult != esapi.ValidationResultUnknown {
		cond := NewSecretStoreCondition(esapi.SecretStoreReady, v1.ConditionFalse, esapi.ReasonValidationFailed, errUnableValidateStore)
		SetExternalSecretCondition(store, *cond)
		recorder.Event(store, v1.EventTypeWarning, esapi.ReasonValidationFailed, err.Error())
		return fmt.Errorf(errValidationFailed, err)
	}

	return nil
}

// ShouldProcessStore returns true if the store should be processed.
func ShouldProcessStore(store esapi.GenericStore, class string) bool {
	if store.GetSpec().Controller == "" || store.GetSpec().Controller == class {
		return true
	}

	return false
}
