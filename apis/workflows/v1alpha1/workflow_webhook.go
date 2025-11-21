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

// Package v1alpha1 contains API Schema definitions for the workflows v1alpha1 API group
package v1alpha1

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
)

// Compile regular expressions to capture references in a template string.
// The first regex captures references like: {{ .global.jobs.<jobName>.<stepName> }}.
var jobStepReferenceRegex = regexp.MustCompile(`{{\s*\.global\.jobs\.([^.]+)\.([^. \}]+)`)

// The second regex captures references like: {{ .global.variables.<varName> }}.
var globalVariableReferenceRegex = regexp.MustCompile(`{{\s*\.global\.variables\.([^. \}]+)`)

// ValidateCreate implements webhook.Validator so that a webhook will be registered for the type.
func (wf *Workflow) ValidateCreate() error {
	return wf.validateWorkflow()
}

// ValidateUpdate implements webhook.Validator so that a webhook will be registered for the type.
func (wf *Workflow) ValidateUpdate(_ runtime.Object) error {
	return wf.validateWorkflow()
}

// ValidateDelete implements webhook.Validator; no custom logic needed on delete.
func (wf *Workflow) ValidateDelete() error {
	return nil
}

// validateWorkflow runs our custom validation logic over the Workflow spec.
func (wf *Workflow) validateWorkflow() error {
	// 1. Check each job for duplicate step names.
	for jobName, job := range wf.Spec.Jobs {
		seenSteps := make(map[string]bool)

		// Get steps based on which job configuration is set
		var steps []Step
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

	// 2. Scan all strings that can contain templates.
	//    (a) Global variables
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
	//    (b) Job variables and step fields
	for jobName, job := range wf.Spec.Jobs {
		// Validate job-level variables.
		for key, value := range job.Variables {
			if err := validateTemplateReferencesInString(value, parsedVariables, wf, fmt.Sprintf("job %q variable %q", jobName, key)); err != nil {
				return err
			}
		}
		// Get steps based on which job configuration is set
		var steps []Step
		if job.Standard != nil {
			steps = job.Standard.Steps
		} else if job.Loop != nil {
			steps = job.Loop.Steps
		}

		// Validate fields in each step.
		for _, step := range steps {
			// For Debug steps: check the Message field.
			if step.Debug != nil {
				if err := validateTemplateReferencesInString(step.Debug.Message, parsedVariables, wf, fmt.Sprintf("job %q step %q (debug message)", jobName, step.Name)); err != nil {
					return err
				}
			}
			// For Transform steps: check the mappings and full template.
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
			// If in the future other step types support templates, add them here.
		}
	}

	return nil
}

// validateTemplateReferencesInString scans a string for template references (if any)
// and validates that any global references to jobs or variables exist in the workflow.
func validateTemplateReferencesInString(templateStr string, parsedVariables map[string]interface{}, wf *Workflow, context string) error {
	// If there is no template marker, skip validation.
	if !strings.Contains(templateStr, "{{") {
		return nil
	}

	// 1. Validate any references of the form: {{ .global.jobs.<jobName>.<stepName> ... }}
	jobMatches := jobStepReferenceRegex.FindAllStringSubmatch(templateStr, -1)
	for _, match := range jobMatches {
		// match[1] is jobName, match[2] is stepName.
		if len(match) < 3 {
			continue
		}
		refJob := match[1]
		refStep := match[2]
		job, exists := wf.Spec.Jobs[refJob]
		if !exists {
			return fmt.Errorf("%s: references job %q which does not exist", context, refJob)
		}
		if !jobHasStep(job, refStep) {
			return fmt.Errorf("%s: references step %q in job %q which does not exist", context, refStep, refJob)
		}
	}

	// 2. Validate any references of the form: {{ .global.variables.<varName> }}
	globalVarMatches := globalVariableReferenceRegex.FindAllStringSubmatch(templateStr, -1)
	for _, match := range globalVarMatches {
		if len(match) < 2 {
			continue
		}
		varName := match[1]
		if _, exists := parsedVariables[varName]; !exists {
			return fmt.Errorf("%s: references global variable %q which does not exist", context, varName)
		}
	}

	return nil
}

// jobHasStep returns true if the given job has a step with the specified name.
func jobHasStep(job Job, stepName string) bool {
	// Get steps based on which job configuration is set
	var steps []Step
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
