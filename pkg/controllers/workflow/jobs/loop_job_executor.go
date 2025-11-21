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

// Package jobs provides workflow job executors.
package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
	"github.com/external-secrets/external-secrets/pkg/controllers/secretstore"
	"github.com/external-secrets/external-secrets/pkg/controllers/workflow/templates"
)

// LoopJobExecutor handles execution of loop jobs.
type LoopJobExecutor struct {
	job     *workflows.LoopJob
	log     logr.Logger
	scheme  *runtime.Scheme
	manager secretstore.ManagerInterface
}

// NewLoopJobExecutor creates a new LoopJobExecutor.
func NewLoopJobExecutor(job *workflows.LoopJob, scheme *runtime.Scheme, log logr.Logger, manager secretstore.ManagerInterface) *LoopJobExecutor {
	return &LoopJobExecutor{
		job:     job,
		scheme:  scheme,
		log:     log,
		manager: manager,
	}
}

// copyMap creates a shallow copy of a map[string]interface{}.
func copyMap(original map[string]interface{}) map[string]interface{} {
	newMap := make(map[string]interface{}, len(original))
	for k, v := range original {
		newMap[k] = v
	}
	return newMap
}

// prepareLoopRangeData extracts keys and values from the range data.
func prepareLoopRangeData(rangeData interface{}) ([]string, []interface{}, error) {
	var rangeKeys []string
	var rangeValues []interface{}

	switch v := rangeData.(type) {
	case map[string]interface{}:
		for k, val := range v {
			rangeKeys = append(rangeKeys, k)
			rangeValues = append(rangeValues, val)
		}
	case []interface{}:
		for i, val := range v {
			rangeKeys = append(rangeKeys, strconv.Itoa(i))
			rangeValues = append(rangeValues, val)
		}
	default:
		return nil, nil, fmt.Errorf("range value must be a JSON object or an array")
	}

	return rangeKeys, rangeValues, nil
}

// executeIteration encapsulates common logic for processing an iteration of the loop.
func executeIteration(
	ctx context.Context,
	baseCtx *JobExecutionContext,
	key string,
	value interface{},
	loopJob *workflows.LoopJob,
) error {
	// Create a copy of the base data and inject the current range iteration
	iterationData := copyMap(baseCtx.Data)
	iterationData["range"] = map[string]interface{}{
		"key":   key,
		"value": value,
	}

	iterationCtx := &JobExecutionContext{
		Client:    baseCtx.Client,
		Workflow:  baseCtx.Workflow,
		JobName:   baseCtx.JobName,
		JobStatus: baseCtx.JobStatus,
		Scheme:    baseCtx.Scheme,
		Logger:    baseCtx.Logger,
		Data:      iterationData,
		Manager:   baseCtx.Manager,
	}

	// Process each step sequentially within this iteration.
	for _, step := range loopJob.Steps {
		// Needs to be done everytime otherwise it's lost on a second step
		iterationCtx.Data["range"] = map[string]interface{}{
			"key":   key,
			"value": value,
		}
		stepStatus := baseCtx.JobStatus.StepStatuses[step.Name]

		// Skip steps that have already succeeded or failed.
		// TODO[gusfcarvalho] - this blocks concurrent jobs to actually run
		// Need to add steps for each range input here.
		// if stepStatus.Phase == workflows.StepPhaseSucceeded || stepStatus.Phase == workflows.StepPhaseFailed {
		// 	continue
		// }

		// Execute the step.
		if err := ExecuteStepWithContext(ctx, iterationCtx, step, step.Name); err != nil {
			return fmt.Errorf("failed to execute step %s for key %s: %w", step.Name, key, err)
		}

		// Get the updated step status after execution
		stepStatus = baseCtx.JobStatus.StepStatuses[step.Name]

		// Ensure outputs map exists
		if stepStatus.Outputs == nil {
			stepStatus.Outputs = make(map[string]string)
		}

		// Update the step status in the job status
		baseCtx.JobStatus.StepStatuses[step.Name] = stepStatus
	}

	return nil
}

// executeSequentialLoop executes loop steps sequentially.
func executeSequentialLoop(
	ctx context.Context,
	jobCtx *JobExecutionContext,
	loopJob *workflows.LoopJob,
	rangeKeys []string,
	rangeValues []interface{},
) error {
	// Iterate over each key-value pair and process sequentially.
	for i, key := range rangeKeys {
		value := rangeValues[i]
		if err := executeIteration(ctx, jobCtx, key, value, loopJob); err != nil {
			return err
		}
	}
	return nil
}

// executeConcurrentLoop executes loop steps concurrently using errgroup.
func executeConcurrentLoop(
	ctx context.Context,
	jobCtx *JobExecutionContext,
	loopJob *workflows.LoopJob,
	rangeKeys []string,
	rangeValues []interface{},
) error {
	g, ctx := errgroup.WithContext(ctx)
	for i, key := range rangeKeys {
		keyCopy := key
		valueCopy := rangeValues[i]
		g.Go(func() error {
			return executeIteration(ctx, jobCtx, keyCopy, valueCopy, loopJob)
		})
	}
	return g.Wait()
}

// Execute processes a loop job by executing its steps for each item in the range.
func (e *LoopJobExecutor) Execute(ctx context.Context, client client.Client, wf *workflows.Workflow, jobName string, jobStatus *workflows.JobStatus) error {
	e.log.Info("Executing loop job", "job", jobName)

	if e.job == nil {
		return fmt.Errorf("loop job is nil")
	}

	// Create job execution context.
	jobCtx, err := NewJobExecutionContext(client, wf, jobName, jobStatus, e.scheme, e.log, e.manager)
	if err != nil {
		return fmt.Errorf("error creating new job execution context: %w", err)
	}

	// Resolve the range template.
	resolvedRange, err := templates.ResolveTemplate(e.job.Range, jobCtx.Data)
	if err != nil {
		return fmt.Errorf("failed to resolve range template: %w", err)
	}

	// Unmarshal the resolved value into an interface{}.
	var rangeData interface{}
	if err := json.Unmarshal([]byte(resolvedRange), &rangeData); err != nil {
		return fmt.Errorf("range value must resolve to a valid JSON: %w", err)
	}

	// Prepare keys and values slices from the range data.
	rangeKeys, rangeValues, err := prepareLoopRangeData(rangeData)
	if err != nil {
		return err
	}

	// Initialize outputs for all steps
	for _, step := range e.job.Steps {
		stepStatus := jobStatus.StepStatuses[step.Name]
		if stepStatus.Outputs == nil {
			stepStatus.Outputs = make(map[string]string)
		}
		jobStatus.StepStatuses[step.Name] = stepStatus
	}

	// Execute iterations either concurrently or sequentially based on Concurrency.
	if e.job.Concurrency > 1 {
		if err := executeConcurrentLoop(ctx, jobCtx, e.job, rangeKeys, rangeValues); err != nil {
			return err
		}
	} else {
		if err := executeSequentialLoop(ctx, jobCtx, e.job, rangeKeys, rangeValues); err != nil {
			return err
		}
	}

	// After all iterations, manually add all range keys to the outputs
	// This ensures that all keys from the range are present in the outputs
	if mapData, ok := rangeData.(map[string]interface{}); ok {
		for _, step := range e.job.Steps {
			stepStatus := jobStatus.StepStatuses[step.Name]
			for k, v := range mapData {
				stepStatus.Outputs[k] = fmt.Sprintf("%v", v)
			}
			jobStatus.StepStatuses[step.Name] = stepStatus
		}
	}

	// All iterations completed successfully.
	return CompleteJob(jobStatus)
}
