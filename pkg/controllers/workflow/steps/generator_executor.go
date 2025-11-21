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

// 2025
// Copyright External Secrets Inc.
// All Rights Reserved.

// Package steps provides workflow step executors.
package steps

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"

	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	genv1alpha1 "github.com/external-secrets/external-secrets/apis/generators/v1alpha1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
	utils "github.com/external-secrets/external-secrets/runtime/esutils"
	"github.com/external-secrets/external-secrets/runtime/esutils/resolvers"
	"github.com/external-secrets/external-secrets/runtime/statemanager"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	errGenerate    = "error using generator: %w"
	errRewrite     = "error applying rewrite to keys: %w"
	errInvalidKeys = "invalid secret keys (TIP: use rewrite or conversionStrategy to change keys): %w"
)

// GeneratorStepExecutor executes a generator step.
type GeneratorStepExecutor struct {
	Step    *workflows.GeneratorStep
	Client  client.Client
	Scheme  *runtime.Scheme
	Manager secretstore.ManagerInterface
}

// NewGeneratorStepExecutor creates a new generator step executor.
func NewGeneratorStepExecutor(step *workflows.GeneratorStep, c client.Client, scheme *runtime.Scheme, manager secretstore.ManagerInterface) *GeneratorStepExecutor {
	if manager == nil {
		panic("manager cannot be nil")
	}

	return &GeneratorStepExecutor{
		Step:    step,
		Client:  c,
		Scheme:  scheme,
		Manager: manager,
	}
}

// Execute generates secret values using the configured generator,
// applies post-processing, and returns a map of key/value pairs.
// Execute generates secret values using the configured generator.
func (e *GeneratorStepExecutor) Execute(ctx context.Context, _ client.Client, wf *workflows.Workflow, inputData map[string]interface{}, jobName string) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	log := ctrl.Log.WithName("controllers").WithName("Workflow")

	defer func() {
		_ = e.Manager.Close(ctx)
	}()
	templates.ProcessTemplates(reflect.ValueOf(e.Step), inputData)

	// Use the workflow's namespace
	namespace := wf.ObjectMeta.Namespace

	var gen genv1alpha1.Generator
	var obj *apiextensions.JSON
	var err error

	if e.Step.GeneratorRef != nil {
		// Handle existing generator reference
		gen, obj, err = resolvers.GeneratorRef(ctx, e.Client, e.Scheme, namespace, e.Step.GeneratorRef)
		if err != nil {
			return nil, err
		}
	} else if e.Step.Kind != "" && e.Step.Generator != nil {
		// Handle inline generator configuration
		var ok bool
		gen, ok = genv1alpha1.GetGeneratorByKind(string(e.Step.Kind))
		if !ok {
			return nil, fmt.Errorf("unknown generator kind: %s", e.Step.Kind)
		}

		// Convert generator spec to JSON
		jsonData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(e.Step.Generator)
		if err != nil {
			return nil, fmt.Errorf("failed to convert generator spec to unstructured: %w", err)
		}

		// Convert to JSON bytes
		jsonBytes, err := json.Marshal(jsonData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal generator spec: %w", err)
		}

		obj = &apiextensions.JSON{Raw: jsonBytes}
	} else {
		return nil, fmt.Errorf("either generatorRef or kind+generator must be specified")
	}

	// Use the generator
	secretMap, newGenState, err := gen.Generate(ctx, obj, e.Client, namespace)
	if err != nil {
		return nil, fmt.Errorf(errGenerate, err)
	}

	// Handle generator state
	if e.Step.AutoCleanup {
		var statefulResource genv1alpha1.StatefulResource
		runTemplate, err := getWorkflowRunTemplateFromWorkflow(ctx, e.Client, wf)
		if err == nil {
			statefulResource = runTemplate
		} else {
			statefulResource = wf
			log.Info(fmt.Sprintf("error getting workflow run template: %v", err))
		}

		generatorState := statemanager.New(ctx, e.Client, e.Scheme, namespace, statefulResource)
		defer func() {
			if err != nil {
				if err := generatorState.Rollback(); err != nil {
					log.Error(err, "error rolling back generator state")
				}

				return
			}
			if err := generatorState.Commit(); err != nil {
				log.Error(err, "error committing generator state")
			}
		}()
		cleanupPolicy, err := gen.GetCleanupPolicy(obj)
		if err != nil {
			return nil, err
		}
		generatorState.SetCleanupPolicy(cleanupPolicy)

		genStateKey := fmt.Sprintf("%s.%s.%s.%s", statefulResource.GetObjectKind().GroupVersionKind().String(), wf.Namespace, statefulResource.GetName(), jobName)
		hash := sha512.Sum512([]byte(genStateKey))
		if generatorState != nil {
			generatorState.EnqueueCreateState(hex.EncodeToString(hash[:]), namespace, obj, gen, newGenState)
		}
	}

	// rewrite the keys if needed
	secretMap, err = utils.RewriteMap(e.Step.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errRewrite, err)
	}

	// validate the keys
	err = utils.ValidateKeys(log, secretMap)
	if err != nil {
		return nil, fmt.Errorf(errInvalidKeys, err)
	}

	// Try to parse values as JSON, fallback to string if not valid JSON
	for k, v := range secretMap {
		var jsonValue interface{}
		if err := json.Unmarshal(v, &jsonValue); err == nil {
			output[k] = jsonValue
		} else {
			output[k] = string(v)
		}
	}

	return output, nil
}
func getWorkflowRunTemplateFromWorkflow(
	ctx context.Context,
	c client.Client,
	wf *workflows.Workflow,
) (*workflows.WorkflowRunTemplate, error) {
	runTemplate, ok := wf.GetLabels()["workflows.external-secrets.io/runtemplate"]
	if !ok {
		return nil, fmt.Errorf("workflow %q has no WorkflowRunTemplate", wf.Name)
	}
	tmpl := &workflows.WorkflowRunTemplate{}
	if err := c.Get(ctx,
		client.ObjectKey{Namespace: wf.Namespace, Name: runTemplate},
		tmpl,
	); err != nil {
		return nil, fmt.Errorf("error fetching WorkflowRunTemplate %q: %w", runTemplate, err)
	}

	return tmpl, nil
}
