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

// Package workflow implements workflow controllers.
package workflow

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	workflows "github.com/external-secrets/external-secrets/apis/workflows/v1alpha1"
)

// TODO[onakin]: Move this into a validator webhook later

// Regular expressions to find template references.
// Matches: {{ .global.jobs.<jobName>.<stepName> }}.
var jobStepReferenceRegex = regexp.MustCompile(`{{\s*\.global\.jobs\.([^.]+)\.([^. \}]+)`)

// Matches: {{ .global.variables.<varName> }}.
var globalVariableReferenceRegex = regexp.MustCompile(`{{\s*\.global\.variables\.([^. \}]+)`)

// validateWorkflowSpec validates the workflow spec for issues such as duplicate step names
// and invalid template references.
func validateWorkflowSpec(wf *workflows.Workflow) error {
	// 1. Check each job for duplicate step names.
	for jobName, job := range wf.Spec.Jobs {
		seenSteps := make(map[string]bool)

		// Get steps based on which job configuration is set
		var steps []workflows.Step
		if job.Standard != nil {
			steps = job.Standard.Steps
		} else if job.Loop != nil {
			steps = job.Loop.Steps
		}

		for _, step := range steps {
			if seenSteps[step.Name] {
				return fmt.Errorf("job %q has duplicate step name %q", jobName, step.Name)
			}
			seenSteps[step.Name] = true
		}
	}

	// 2. Validate global variables.
	toParseVariables, err := json.Marshal(wf.Spec.Variables)
	if err != nil {
		return fmt.Errorf("error marshaling variables from workflow %s: %w", wf.Name, err)
	}
	var parsedVariables map[string]interface{}
	err = json.Unmarshal(toParseVariables, &parsedVariables)
	if err != nil {
		return fmt.Errorf("error unmarshalling variables from workflow %s: %w", wf.Name, err)
	}

	for key, value := range parsedVariables {
		strValue, ok := value.(string)
		if !ok {
			continue
		}

		if err := validateTemplateReferencesInString(strValue, parsedVariables, wf, fmt.Sprintf("global variable %q", key)); err != nil {
			return err
		}
	}

	// 3. Validate each job's variables and step fields.
	for jobName, job := range wf.Spec.Jobs {
		// Validate job-level variables.
		for key, value := range job.Variables {
			if err := validateTemplateReferencesInString(value, parsedVariables, wf, fmt.Sprintf("job %q variable %q", jobName, key)); err != nil {
				return err
			}
		}

		// Get steps based on which job configuration is set
		var steps []workflows.Step
		if job.Standard != nil {
			steps = job.Standard.Steps
		} else if job.Loop != nil {
			steps = job.Loop.Steps
		}

		// Validate fields in each step.
		for _, step := range steps {
			// Validate output definitions
			if err := validateStepOutputs(step, jobName); err != nil {
				return err
			}

			// If the step is a Debug step, check the Message field.
			if step.Debug != nil {
				if err := validateTemplateReferencesInString(step.Debug.Message, parsedVariables, wf, fmt.Sprintf("job %q step %q (debug message)", jobName, step.Name)); err != nil {
					return err
				}
			}
			// If the step is a Transform step, check the mappings and template.
			if step.Transform != nil {
				for mappingKey, mappingValue := range step.Transform.Mappings {
					if err := validateTemplateReferencesInString(mappingValue, parsedVariables, wf, fmt.Sprintf("job %q step %q (transform mapping %q)", jobName, step.Name, mappingKey)); err != nil {
						return err
					}
				}
				if err := validateTemplateReferencesInString(step.Transform.Template, parsedVariables, wf, fmt.Sprintf("job %q step %q (transform template)", jobName, step.Name)); err != nil {
					return err
				}
			}
			// Extend here for other step types that may contain templates.
		}
	}

	return nil
}

// validateTemplateReferencesInString scans a string for template references and
// validates that any referenced global objects exist in the workflow.
func validateTemplateReferencesInString(templateStr string, parsedVariables map[string]interface{}, wf *workflows.Workflow, contextStr string) error {
	// If there's no template marker, skip validation.
	if !strings.Contains(templateStr, "{{") {
		return nil
	}

	// 1. Validate references of the form: {{ .global.jobs.<jobName>.<stepName> ... }}
	jobMatches := jobStepReferenceRegex.FindAllStringSubmatch(templateStr, -1)
	for _, match := range jobMatches {
		// match[1] is the jobName, match[2] is the stepName.
		if len(match) < 3 {
			continue
		}
		refJob := match[1]
		refStep := match[2]
		job, exists := wf.Spec.Jobs[refJob]
		if !exists {
			return fmt.Errorf("%s: references job %q which does not exist", contextStr, refJob)
		}
		if !jobHasStep(job, refStep) {
			return fmt.Errorf("%s: references step %q in job %q which does not exist", contextStr, refStep, refJob)
		}
	}

	// 2. Validate references of the form: {{ .global.variables.<varName> }}
	globalVarMatches := globalVariableReferenceRegex.FindAllStringSubmatch(templateStr, -1)
	for _, match := range globalVarMatches {
		if len(match) < 2 {
			continue
		}
		varName := match[1]
		if _, exists := parsedVariables[varName]; !exists {
			return fmt.Errorf("%s: references global variable %q which does not exist", contextStr, varName)
		}
	}

	return nil
}

// validateStepOutputs validates that output names are unique within a step.
func validateStepOutputs(step workflows.Step, jobName string) error {
	// Check for duplicate output names
	seenOutputs := make(map[string]bool)
	for _, output := range step.Outputs {
		if seenOutputs[output.Name] {
			return fmt.Errorf("job %q step %q has duplicate output name %q", jobName, step.Name, output.Name)
		}
		seenOutputs[output.Name] = true
	}

	return nil
}

// jobHasStep returns true if the given job has a step with the specified name.
func jobHasStep(job workflows.Job, stepName string) bool {
	// Get steps based on which job configuration is set
	var steps []workflows.Step
	if job.Standard != nil {
		steps = job.Standard.Steps
	} else if job.Loop != nil {
		steps = job.Loop.Steps
	}

	for _, step := range steps {
		if step.Name == stepName {
			return true
		}
	}
	return false
}
