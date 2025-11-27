// /*
// Copyright © 2025 ESO Maintainer Team
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

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

// Package generator implements the controller for Generator resources.
package generator

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
)

const (
	// GeneratorGroup is the group name for the generator API.
	GeneratorGroup = "generators.external-secrets.io"
	// GeneratorVersion is the version name for the generator API.
	GeneratorVersion = "v1alpha1"
)

// Reconciler is the controller for Generator resources.
type Reconciler struct {
	client.Client

	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
	recorder   record.EventRecorder

	Kind string
}

// Reconcile updates a given generator outputs.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, err error) {
	genericGenerator, err := BuildGeneratorObject(r.Scheme, r.Kind)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error building generator object: %w", err)
	}

	if err := r.Get(ctx, req.NamespacedName, genericGenerator); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	gvk := genericGenerator.GetObjectKind().GroupVersionKind()

	generator, found := genv1alpha1.GetGeneratorByKind(gvk.Kind)
	if !found {
		return ctrl.Result{}, fmt.Errorf("generator of kind %s not found", gvk.Kind)
	}

	err = genericGenerator.SetOutputs(generator.GetKeys())
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error setting outputs: %w", err)
	}

	err = r.Status().Update(ctx, genericGenerator)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("error updating generic generator: %w", err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager returns a new controller builder that will be started by the provided Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager, obj client.Object, opts controller.Options) error {
	r.recorder = mgr.GetEventRecorderFor("generators")
	return ctrl.NewControllerManagedBy(mgr).
		WithOptions(opts).
		For(obj).
		Complete(r)
}

// BuildGeneratorObject builds a new GenericGenerator object.
func BuildGeneratorObject(scheme *runtime.Scheme, kind string) (genv1alpha1.GenericGenerator, error) {
	gvk := schema.GroupVersionKind{Group: GeneratorGroup, Version: GeneratorVersion, Kind: kind}
	obj, err := scheme.New(gvk)
	if err != nil {
		return nil, fmt.Errorf("failed to create object %v: %w", gvk, err)
	}
	co, ok := obj.(genv1alpha1.GenericGenerator)
	if !ok {
		return nil, fmt.Errorf("invalid object: %T", obj)
	}
	return co, nil
}
