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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// PushStepExecutor executes a push step.
type PushStepExecutor struct {
	Step    *workflows.PushStep
	Client  client.Client
	Manager secretstore.ManagerInterface
}

// NewPushStepExecutor creates a new push step executor.
func NewPushStepExecutor(step *workflows.PushStep, c client.Client, manager secretstore.ManagerInterface) *PushStepExecutor {
	if manager == nil {
		panic("manager cannot be nil")
	}

	return &PushStepExecutor{
		Step:    step.DeepCopy(), // Otherwise we are modifying possible templates.
		Client:  c,
		Manager: manager,
	}
}

// Execute pushes secret values to the destination store.
func (e *PushStepExecutor) Execute(ctx context.Context, _ client.Client, wf *workflows.Workflow, inputData map[string]interface{}, _ string) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	if e.Manager == nil {
		return nil, fmt.Errorf("secret store manager is required")
	}
	defer func() {
		_ = e.Manager.Close(ctx)
	}()

	templates.ProcessTemplates(reflect.ValueOf(e.Step), inputData)

	// Find the source transform data
	secretSource := e.Step.SecretSource
	if secretSource == "" {
		return nil, fmt.Errorf("sourceTransform is required")
	}

	var secret corev1.Secret
	byteData := make(map[string][]byte)

	// For each data item, resolve its value
	for _, data := range e.Step.Data {
		// We need to convert our whole secretSource as a Kubernetes Secret
		// For that, we must ask the user that `secretSource` is a valid map.
		// Pointing to a pull or generator step, this is always the case.
		sourceKeys := strings.Split(secretSource, ".")[1:]
		temp := inputData
		for _, key := range sourceKeys {
			a, ok := temp[key]
			if !ok {
				// Temporary Failure
				return nil, fmt.Errorf("error getting value for key %s: no key found", key)
			}
			d, err := json.Marshal(a)
			if err != nil {
				return nil, fmt.Errorf("error getting value for key %s: %w", key, err)
			}
			v := map[string]interface{}{}
			err = json.Unmarshal(d, &v)
			if err != nil {
				return nil, fmt.Errorf("error getting value for key %s: %w", key, err)
			}
			temp = v
		}
		for k, v := range temp {
			switch val := v.(type) {
			case nil, string, bool, float64, int, int64:
				// Simple types can be converted to string directly
				byteData[k] = []byte(fmt.Sprintf("%v", val))
			default:
				// Complex types need JSON serialization
				jsonBytes, err := json.Marshal(val)
				if err != nil {
					return nil, fmt.Errorf("error serializing value for key %s: %w", data.Match.SecretKey, err)
				}
				byteData[k] = jsonBytes
			}
			continue
		}
	}
	// NOTE: We don't verify secretKey logic here as `PushSecret` will already fail
	// if the Secret does not contain the key
	// This approch also allows for pushing the entire Secret as opposed to just one key

	secret = corev1.Secret{
		Data: byteData,
	}

	for _, data := range e.Step.Data {
		destClient, err := e.Manager.Get(ctx, e.Step.Destination.SecretStoreRef, wf.Namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("error getting destination store client: %w", err)
		}
		err = destClient.PushSecret(ctx, &secret, data)
		if err != nil {
			return nil, fmt.Errorf("error pushing secret data: %w", err)
		}
		output[data.Match.SecretKey] = data.Match.RemoteRef.RemoteKey
	}

	return output, nil
}
