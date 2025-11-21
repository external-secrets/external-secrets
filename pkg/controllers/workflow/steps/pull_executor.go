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
	"errors"
	"fmt"
	"reflect"

	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	esv1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1"
	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	utils "github.com/external-secrets/external-secrets/runtime/esutils"
)

// PullStepExecutor pulls secret values from external providers and applies post‐processing.
type PullStepExecutor struct {
	Step    *workflows.PullStep
	Client  client.Client
	Manager secretstore.ManagerInterface
}

// NewPullStepExecutor creates a new PullStepExecutor with a SecretStoreManager.
func NewPullStepExecutor(step *workflows.PullStep, c client.Client, manager secretstore.ManagerInterface) *PullStepExecutor {
	if manager == nil {
		panic("manager cannot be nil")
	}

	return &PullStepExecutor{
		Step:    step.DeepCopy(),
		Client:  c,
		Manager: manager,
	}
}

// Execute retrieves secret values from both spec.data and spec.dataFrom entries,
// applies rewriting, conversion, validation, and decoding steps, and returns a map
// of key/value pairs.
func (e *PullStepExecutor) Execute(ctx context.Context, _ client.Client, wf *workflows.Workflow, inputData map[string]interface{}, _ string) (map[string]interface{}, error) {
	output := make(map[string]interface{})

	templates.ProcessTemplates(reflect.ValueOf(e.Step), inputData)

	defer func() {
		_ = e.Manager.Close(ctx)
	}()

	// Use the workflow's namespace.
	namespace := wf.ObjectMeta.Namespace

	// Process spec.data entries.
	for i, dataItem := range e.Step.Data {
		secClient, err := e.Manager.Get(ctx, e.Step.Source.SecretStoreRef, namespace, nil)
		if err != nil {
			return nil, fmt.Errorf("error getting client for spec.data entry [%d]: %w", i, err)
		}
		// Get the secret value using the RemoteRef settings.
		rawSecret, err := secClient.GetSecret(ctx, dataItem.RemoteRef)
		if err != nil {
			return nil, fmt.Errorf("error fetching secret for spec.data entry [%d]: %w", i, err)
		}

		// Decode the secret value according to the specified decoding strategy.
		decodedSecret, err := utils.Decode(dataItem.RemoteRef.DecodingStrategy, rawSecret)
		if err != nil {
			return nil, fmt.Errorf("error decoding secret for spec.data entry [%d]: %w", i, err)
		}

		// Default to string value
		output[dataItem.SecretKey] = string(decodedSecret)

		// Try to parse as JSON
		var jsonValue map[string]interface{}
		if err := json.Unmarshal(decodedSecret, &jsonValue); err == nil {
			output[dataItem.SecretKey] = jsonValue
		}
		// Try to parse as JSON array
		var arrayValue []interface{}
		if err := json.Unmarshal(decodedSecret, &arrayValue); err == nil {
			output[dataItem.SecretKey] = arrayValue
		}
	}

	// Process spec.dataFrom entries.
	for i, remoteRef := range e.Step.DataFrom {
		var secretMap map[string][]byte
		var err error

		// Get the secret client
		secClient, err := e.Manager.Get(ctx, e.Step.Source.SecretStoreRef, namespace, remoteRef.SourceRef)
		if err != nil {
			return nil, fmt.Errorf("error getting client for spec.dataFrom entry [%d]: %w", i, err)
		}

		if remoteRef.Find != nil {
			secretMap, err = findAllSecrets(ctx, remoteRef, secClient, i)
		} else if remoteRef.Extract != nil {
			secretMap, err = extractSecrets(ctx, remoteRef, secClient, i)
		} else {
			err = errors.New("spec.dataFrom entry does not specify a valid method (Find or Extract)")
		}
		if err != nil {
			return nil, err
		}

		// Merge the returned secret map into our output, attempting JSON parsing
		for k, v := range secretMap {
			// Default to string value
			output[k] = string(v)
		}
	}

	return output, nil
}

// findAllSecrets processes a spec.dataFrom entry that uses the "find" method.
func findAllSecrets(ctx context.Context,
	remoteRef esv1.ExternalSecretDataFromRemoteRef,
	client esv1.SecretsClient,
	index int) (map[string][]byte, error) {
	// Fetch all matching secrets.
	secretMap, err := client.GetAllSecrets(ctx, *remoteRef.Find)
	if err != nil {
		return nil, fmt.Errorf("error fetching secrets with find for spec.dataFrom entry [%d]: %w", index, err)
	}

	// Rewrite keys as specified.
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf("error rewriting keys for spec.dataFrom[find] entry [%d]: %w", index, err)
	}

	// If no rewrite rules are provided, convert keys using the conversion strategy.
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Find.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf("error converting keys for spec.dataFrom[find] entry [%d] using strategy %s: %w", index, remoteRef.Find.ConversionStrategy, err)
		}
	}

	// Validate the keys.
	logger := ctrl.Log.WithValues("findAllSecrets", fmt.Sprintf("spec.dataFrom[find] entry [%d]", index))
	if err = utils.ValidateKeys(logger, secretMap); err != nil {
		return nil, fmt.Errorf("invalid keys for spec.dataFrom[find] entry [%d]: %w", index, err)
	}

	// Decode secret values.
	secretMap, err = utils.DecodeMap(remoteRef.Find.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf("error decoding secret values for spec.dataFrom[find] entry [%d] using strategy %s: %w", index, remoteRef.Find.DecodingStrategy, err)
	}

	return secretMap, nil
}

// extractSecrets processes a spec.dataFrom entry that uses the "extract" method.
func extractSecrets(ctx context.Context,
	remoteRef esv1.ExternalSecretDataFromRemoteRef,
	client esv1.SecretsClient,
	index int) (map[string][]byte, error) {
	// Fetch secrets using the extract criteria.
	secretMap, err := client.GetSecretMap(ctx, *remoteRef.Extract)
	if err != nil {
		return nil, fmt.Errorf("error fetching secrets with extract for spec.dataFrom entry [%d]: %w", index, err)
	}

	// Rewrite keys.
	secretMap, err = utils.RewriteMap(remoteRef.Rewrite, secretMap)
	if err != nil {
		return nil, fmt.Errorf("error rewriting keys for spec.dataFrom[extract] entry [%d]: %w", index, err)
	}

	// If no rewrite rules, apply conversion.
	if len(remoteRef.Rewrite) == 0 {
		secretMap, err = utils.ConvertKeys(remoteRef.Extract.ConversionStrategy, secretMap)
		if err != nil {
			return nil, fmt.Errorf("error converting keys for spec.dataFrom[extract] entry [%d] using strategy %s: %w", index, remoteRef.Extract.ConversionStrategy, err)
		}
	}

	// Validate keys.
	logger := ctrl.Log.WithValues("extractSecrets", fmt.Sprintf("spec.dataFrom[extract] entry [%d]", index))
	if err = utils.ValidateKeys(logger, secretMap); err != nil {
		return nil, fmt.Errorf("invalid keys for spec.dataFrom[extract] entry [%d]: %w", index, err)
	}

	// Decode secret values.
	secretMap, err = utils.DecodeMap(remoteRef.Extract.DecodingStrategy, secretMap)
	if err != nil {
		return nil, fmt.Errorf("error decoding secret values for spec.dataFrom[extract] entry [%d] using strategy %s: %w", index, remoteRef.Extract.DecodingStrategy, err)
	}

	return secretMap, nil
}
